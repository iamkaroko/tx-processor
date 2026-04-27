// internal/service/analytics.go
package service

import (
	"context"
	"fmt"

	"tx-processor/internal/cache"
	"tx-processor/internal/models"
	"tx-processor/internal/repository"
)

type Analytics struct {
	repo  repository.Analytics
	cache cache.AnalyticsCache
}

func NewAnalytics(repo repository.Analytics, cache cache.AnalyticsCache) *Analytics {
	return &Analytics{repo: repo, cache: cache}
}

func (s *Analytics) UserAnalytics(ctx context.Context, userID string) (*models.UserAnalytics, error) {
	if s.cache != nil {
		if cached, err := s.cache.UserInfo(ctx, userID); err == nil && cached != nil {
			return cached, nil
		}
	}

	data, err := s.repo.UserAnalytics(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.SetUserInfo(ctx, data)
	}

	return data, nil
}

func (s *Analytics) TopUsers(ctx context.Context, limit int) ([]models.UserAnalytics, error) {
	return s.repo.TopUsers(ctx, limit)
}

func (s *Analytics) Anomalies(ctx context.Context) ([]models.AnomalyUser, error) {
	return s.repo.UserAnomalies(ctx)
}

func (s *Analytics) UpdateAnalytics(ctx context.Context, updates map[string]*models.UserAnalytics) error {
	if err := s.repo.UpdateAnalytics(ctx, updates); err != nil {
		return err
	}

	if s.cache == nil {
		return nil
	}

	for userID := range updates {
		if err := s.cache.InvalidateUser(ctx, userID); err != nil {
			return fmt.Errorf("invalidate cache for %s: %w", userID, err)
		}
	}

	return nil
}
