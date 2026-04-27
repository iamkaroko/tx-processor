// internal/repository/repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"tx-processor/internal/models"

	"github.com/jmoiron/sqlx"
)

type Analytics interface {
	UpdateAnalytics(ctx context.Context, updates map[string]*models.UserAnalytics) error
	UserAnalytics(ctx context.Context, userID string) (*models.UserAnalytics, error)
	TopUsers(ctx context.Context, limit int) ([]models.UserAnalytics, error)
	UserAnomalies(ctx context.Context) ([]models.AnomalyUser, error)
}

type AnalyticsRepo struct {
	db *sqlx.DB
}

func NewAnalyticsRepo(db *sqlx.DB) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

const bulkChunkSize = 1000

func (r *AnalyticsRepo) UpdateAnalytics(ctx context.Context, updates map[string]*models.UserAnalytics) error {
	if len(updates) == 0 {
		return nil
	}

	rows := make([]*models.UserAnalytics, 0, len(updates))
	for _, ua := range updates {
		rows = append(rows, ua)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			fmt.Printf("rollback error: %v\n", err)
		}
	}()

	for i := 0; i < len(rows); i += bulkChunkSize {
		end := i + bulkChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*3)
		for j, ua := range chunk {
			base := j * 3
			placeholders[j] = fmt.Sprintf("($%d,$%d,$%d)", base+1, base+2, base+3)
			args = append(args, ua.UserID, ua.TotalOrders, ua.TotalSpent)
		}

		query := fmt.Sprintf(`
            INSERT INTO user_analytics (user_id, total_orders, total_spent)
            VALUES %s
            ON CONFLICT (user_id) DO UPDATE SET
                total_orders = user_analytics.total_orders + EXCLUDED.total_orders,
                total_spent  = user_analytics.total_spent  + EXCLUDED.total_spent
        `, strings.Join(placeholders, ","))

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("bulk insert chunk %d: %w", i/bulkChunkSize, err)
		}
	}

	return tx.Commit()
}

func (r *AnalyticsRepo) UserAnalytics(ctx context.Context, userID string) (*models.UserAnalytics, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	var analytics models.UserAnalytics
	query := "SELECT user_id, total_orders, total_spent FROM user_analytics WHERE user_id = $1"

	if err := r.db.GetContext(ctx, &analytics, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return &models.UserAnalytics{UserID: userID}, nil
		}
		return nil, fmt.Errorf("select user analytics: %w", err)
	}

	return &analytics, nil
}

func (r *AnalyticsRepo) TopUsers(ctx context.Context, limit int) ([]models.UserAnalytics, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	var users []models.UserAnalytics
	query := `
        SELECT user_id, total_orders, total_spent
        FROM user_analytics
        ORDER BY total_orders DESC
        LIMIT $1
    `

	if err := r.db.SelectContext(ctx, &users, query, limit); err != nil {
		return nil, fmt.Errorf("select top users: %w", err)
	}

	return users, nil
}

func (r *AnalyticsRepo) UserAnomalies(ctx context.Context) ([]models.AnomalyUser, error) {
	query := `
        WITH stats AS (
            SELECT
                AVG(total_orders)::FLOAT    AS avg_orders,
                STDDEV(total_orders)::FLOAT AS stddev_orders,
                AVG(total_spent)::FLOAT     AS avg_spent,
                STDDEV(total_spent)::FLOAT  AS stddev_spent
            FROM user_analytics
            WHERE total_orders > 0
        )
        SELECT
            ua.user_id,
            ua.total_orders,
            ua.total_spent,
            (ua.total_orders > stats.avg_orders + 2 * stats.stddev_orders) AS order_anomaly,
            (ua.total_spent  > stats.avg_spent  + 2 * stats.stddev_spent)  AS spending_anomaly
        FROM user_analytics ua, stats
        WHERE ua.total_orders > stats.avg_orders + 2 * stats.stddev_orders
           OR ua.total_spent  > stats.avg_spent  + 2 * stats.stddev_spent
        ORDER BY ua.total_orders DESC, ua.total_spent DESC
    `

	var anomalies []models.AnomalyUser
	if err := r.db.SelectContext(ctx, &anomalies, query); err != nil {
		return nil, fmt.Errorf("select anomalies: %w", err)
	}

	return anomalies, nil
}
