package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"tx-processor/internal/models"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newMockRepo(t *testing.T) (*AnalyticsRepo, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	cleanup := func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
		_ = db.Close()
	}

	return NewAnalyticsRepo(sqlx.NewDb(db, "sqlmock")), mock, cleanup
}

func TestUserAnalyticsReturnsZeroValueForMissingUser(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT user_id, total_orders, total_spent FROM user_analytics WHERE user_id = $1")).
		WithArgs("user_1").
		WillReturnError(sql.ErrNoRows)

	got, err := repo.UserAnalytics(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("UserAnalytics() error = %v", err)
	}
	if got.UserID != "user_1" || got.TotalOrders != 0 || got.TotalSpent != 0 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTopUsersReturnsRows(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"user_id", "total_orders", "total_spent"}).
		AddRow("user_2", 5, 12.5).
		AddRow("user_1", 3, 9.0)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT user_id, total_orders, total_spent
		FROM user_analytics
		ORDER BY total_orders DESC
		LIMIT $1
	`)).
		WithArgs(2).
		WillReturnRows(rows)

	got, err := repo.TopUsers(context.Background(), 2)
	if err != nil {
		t.Fatalf("TopUsers() error = %v", err)
	}
	if len(got) != 2 || got[0].UserID != "user_2" || got[1].UserID != "user_1" {
		t.Fatalf("unexpected users: %+v", got)
	}
}

func TestUpdateAnalyticsExecutesBulkUpsert(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO user_analytics (user_id, total_orders, total_spent)
			VALUES ($1,$2,$3)
			ON CONFLICT (user_id) DO UPDATE SET
				total_orders = user_analytics.total_orders + EXCLUDED.total_orders,
				total_spent  = user_analytics.total_spent  + EXCLUDED.total_spent
		`)).
		WithArgs("user_1", 2, 19.5).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateAnalytics(context.Background(), map[string]*models.UserAnalytics{
		"user_1": {UserID: "user_1", TotalOrders: 2, TotalSpent: 19.5},
	})
	if err != nil {
		t.Fatalf("UpdateAnalytics() error = %v", err)
	}
}
