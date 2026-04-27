// internal/cache/redis/cache.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tx-processor/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisAnalyticsCache struct {
	client     *redis.Client
	prefix     string
	defaultTTL time.Duration
}

func NewRedisAnalyticsCache(client *redis.Client) *RedisAnalyticsCache {
	return &RedisAnalyticsCache{
		client:     client,
		prefix:     "analytics:",
		defaultTTL: 5 * time.Hour,
	}
}

func (r *RedisAnalyticsCache) buildKey(userID string) string {
	return fmt.Sprintf("%s%s", r.prefix, userID)
}

func (r *RedisAnalyticsCache) UserInfo(ctx context.Context, userID string) (*models.UserAnalytics, error) {
	val, err := r.client.Get(ctx, r.buildKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}

	var analytics models.UserAnalytics
	if err := json.Unmarshal([]byte(val), &analytics); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}

	return &analytics, nil
}

func (r *RedisAnalyticsCache) SetUserInfo(ctx context.Context, analytics *models.UserAnalytics) error {
	data, err := json.Marshal(analytics)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}

	if err := r.client.Set(ctx, r.buildKey(analytics.UserID), data, r.defaultTTL).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}

	return nil
}

func (r *RedisAnalyticsCache) InvalidateUser(ctx context.Context, userID string) error {
	if err := r.client.Del(ctx, r.buildKey(userID)).Err(); err != nil {
		return fmt.Errorf("cache invalidate: %w", err)
	}

	return nil
}
