package service

import (
	"context"
	"errors"
	"testing"

	"tx-processor/internal/models"
)

type fakeRepo struct {
	userAnalyticsFn   func(ctx context.Context, userID string) (*models.UserAnalytics, error)
	topUsersFn        func(ctx context.Context, limit int) ([]models.UserAnalytics, error)
	userAnomaliesFn   func(ctx context.Context) ([]models.AnomalyUser, error)
	updateAnalyticsFn func(ctx context.Context, updates map[string]*models.UserAnalytics) error
}

func (f *fakeRepo) UpdateAnalytics(ctx context.Context, updates map[string]*models.UserAnalytics) error {
	return f.updateAnalyticsFn(ctx, updates)
}

func (f *fakeRepo) UserAnalytics(ctx context.Context, userID string) (*models.UserAnalytics, error) {
	return f.userAnalyticsFn(ctx, userID)
}

func (f *fakeRepo) TopUsers(ctx context.Context, limit int) ([]models.UserAnalytics, error) {
	return f.topUsersFn(ctx, limit)
}

func (f *fakeRepo) UserAnomalies(ctx context.Context) ([]models.AnomalyUser, error) {
	return f.userAnomaliesFn(ctx)
}

type fakeCache struct {
	userInfoFn       func(ctx context.Context, userID string) (*models.UserAnalytics, error)
	setUserInfoFn    func(ctx context.Context, analytics *models.UserAnalytics) error
	invalidateUserFn func(ctx context.Context, userID string) error
}

func (f *fakeCache) UserInfo(ctx context.Context, userID string) (*models.UserAnalytics, error) {
	return f.userInfoFn(ctx, userID)
}

func (f *fakeCache) SetUserInfo(ctx context.Context, analytics *models.UserAnalytics) error {
	return f.setUserInfoFn(ctx, analytics)
}

func (f *fakeCache) InvalidateUser(ctx context.Context, userID string) error {
	return f.invalidateUserFn(ctx, userID)
}

func TestUserAnalyticsReturnsCachedValue(t *testing.T) {
	t.Parallel()

	repoCalled := false
	svc := NewAnalytics(&fakeRepo{
		userAnalyticsFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			repoCalled = true
			return &models.UserAnalytics{UserID: userID, TotalOrders: 99}, nil
		},
	}, &fakeCache{
		userInfoFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			return &models.UserAnalytics{UserID: userID, TotalOrders: 7}, nil
		},
		setUserInfoFn: func(ctx context.Context, analytics *models.UserAnalytics) error {
			return nil
		},
		invalidateUserFn: func(ctx context.Context, userID string) error {
			return nil
		},
	})

	got, err := svc.UserAnalytics(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("UserAnalytics() error = %v", err)
	}
	if repoCalled {
		t.Fatal("expected repository not to be called on cache hit")
	}
	if got.TotalOrders != 7 {
		t.Fatalf("expected cached result, got %+v", got)
	}
}

func TestUserAnalyticsCachesRepositoryResultOnMiss(t *testing.T) {
	t.Parallel()

	var cached *models.UserAnalytics
	svc := NewAnalytics(&fakeRepo{
		userAnalyticsFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			return &models.UserAnalytics{UserID: userID, TotalOrders: 3, TotalSpent: 14.5}, nil
		},
	}, &fakeCache{
		userInfoFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			return nil, nil
		},
		setUserInfoFn: func(ctx context.Context, analytics *models.UserAnalytics) error {
			cached = analytics
			return nil
		},
		invalidateUserFn: func(ctx context.Context, userID string) error {
			return nil
		},
	})

	got, err := svc.UserAnalytics(context.Background(), "user_2")
	if err != nil {
		t.Fatalf("UserAnalytics() error = %v", err)
	}
	if got.UserID != "user_2" || got.TotalOrders != 3 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if cached == nil || cached.UserID != "user_2" {
		t.Fatalf("expected repository result to be cached, got %+v", cached)
	}
}

func TestUpdateAnalyticsInvalidatesAffectedUsers(t *testing.T) {
	t.Parallel()

	invalidated := make(map[string]bool)
	svc := NewAnalytics(&fakeRepo{
		updateAnalyticsFn: func(ctx context.Context, updates map[string]*models.UserAnalytics) error {
			return nil
		},
	}, &fakeCache{
		userInfoFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			return nil, nil
		},
		setUserInfoFn: func(ctx context.Context, analytics *models.UserAnalytics) error {
			return nil
		},
		invalidateUserFn: func(ctx context.Context, userID string) error {
			invalidated[userID] = true
			return nil
		},
	})

	updates := map[string]*models.UserAnalytics{
		"user_1": {UserID: "user_1", TotalOrders: 2},
		"user_2": {UserID: "user_2", TotalOrders: 4},
	}

	if err := svc.UpdateAnalytics(context.Background(), updates); err != nil {
		t.Fatalf("UpdateAnalytics() error = %v", err)
	}
	for userID := range updates {
		if !invalidated[userID] {
			t.Fatalf("expected cache invalidation for %s", userID)
		}
	}
}

func TestUpdateAnalyticsReturnsInvalidateError(t *testing.T) {
	t.Parallel()

	svc := NewAnalytics(&fakeRepo{
		updateAnalyticsFn: func(ctx context.Context, updates map[string]*models.UserAnalytics) error {
			return nil
		},
	}, &fakeCache{
		userInfoFn: func(ctx context.Context, userID string) (*models.UserAnalytics, error) {
			return nil, nil
		},
		setUserInfoFn: func(ctx context.Context, analytics *models.UserAnalytics) error {
			return nil
		},
		invalidateUserFn: func(ctx context.Context, userID string) error {
			return errors.New("boom")
		},
	})

	err := svc.UpdateAnalytics(context.Background(), map[string]*models.UserAnalytics{
		"user_1": {UserID: "user_1", TotalOrders: 1},
	})
	if err == nil {
		t.Fatal("expected invalidation error")
	}
}
