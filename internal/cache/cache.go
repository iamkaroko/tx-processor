// internal/cache/cache.go
package cache

import (
	"context"
	"tx-processor/internal/models"
)

type AnalyticsCache interface {
	UserInfo(ctx context.Context, userID string) (*models.UserAnalytics, error)
	SetUserInfo(ctx context.Context, analytics *models.UserAnalytics) error
	InvalidateUser(ctx context.Context, userID string) error
}
