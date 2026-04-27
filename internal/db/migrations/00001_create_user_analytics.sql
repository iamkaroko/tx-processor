-- internal/db/migrations/00001_create_user_analytics.sql

-- +goose Up
CREATE TABLE IF NOT EXISTS user_analytics
(
    user_id      VARCHAR(255) PRIMARY KEY,
    total_orders INTEGER        DEFAULT 0,
    total_spent  DECIMAL(15, 2) DEFAULT 0.0,
    last_updated TIMESTAMP      DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_analytics_orders ON user_analytics (total_orders DESC);
CREATE INDEX IF NOT EXISTS idx_user_analytics_spent ON user_analytics (total_spent DESC);

-- +goose Down
DROP TABLE IF EXISTS user_analytics;