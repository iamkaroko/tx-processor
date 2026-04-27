// internal/models/models.go
package models

import "time"

type Transaction struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

type UserAnalytics struct {
	UserID      string  `json:"user_id"      db:"user_id"`
	TotalOrders int     `json:"total_orders" db:"total_orders"`
	TotalSpent  float64 `json:"total_spent"  db:"total_spent"`
}

type AnomalyUser struct {
	UserID          string  `json:"user_id"          db:"user_id"`
	TotalOrders     int     `json:"total_orders"     db:"total_orders"`
	TotalSpent      float64 `json:"total_spent"      db:"total_spent"`
	OrderAnomaly    bool    `json:"order_anomaly"    db:"order_anomaly"`
	SpendingAnomaly bool    `json:"spending_anomaly" db:"spending_anomaly"`
}
