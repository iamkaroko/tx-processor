// internal/handler/handler.go
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"tx-processor/internal/models"
)

type analyticsService interface {
	UserAnalytics(ctx context.Context, userID string) (*models.UserAnalytics, error)
	TopUsers(ctx context.Context, limit int) ([]models.UserAnalytics, error)
	Anomalies(ctx context.Context) ([]models.AnomalyUser, error)
}

type Handler struct {
	svc    analyticsService
	logger *slog.Logger
}

func New(svc analyticsService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/total_orders", h.totalOrdersHandler())
	mux.HandleFunc("/total_spendings", h.totalSpendingsHandler())
	mux.HandleFunc("/top_users", h.topUsersHandler())
	mux.HandleFunc("/anomalies", h.anomaliesHandler())
}

func writeJSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}
