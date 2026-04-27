// internal/handler/analytics.go
package handler

import (
	"fmt"
	"net/http"
	"strconv"
)

func (h *Handler) totalOrdersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, http.StatusBadRequest, "user_id parameter is required")
			return
		}

		analytics, err := h.svc.UserAnalytics(r.Context(), userID)
		if err != nil {
			h.logger.Error("failed to get user analytics", "error", err, "user_id", userID)
			writeError(w, http.StatusInternalServerError, "failed to get user analytics")
			return
		}

		writeJSON(w, http.StatusOK, struct {
			UserID      string `json:"user_id"`
			TotalOrders int    `json:"total_orders"`
			Message     string `json:"message"`
		}{
			UserID:      userID,
			TotalOrders: analytics.TotalOrders,
			Message:     fmt.Sprintf("User %s has placed %d orders", userID, analytics.TotalOrders),
		})
	}
}

func (h *Handler) totalSpendingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, http.StatusBadRequest, "user_id parameter is required")
			return
		}

		analytics, err := h.svc.UserAnalytics(r.Context(), userID)
		if err != nil {
			h.logger.Error("failed to get user analytics", "error", err, "user_id", userID)
			writeError(w, http.StatusInternalServerError, "failed to get user analytics")
			return
		}

		writeJSON(w, http.StatusOK, struct {
			UserID     string  `json:"user_id"`
			TotalSpent float64 `json:"total_spent"`
			Message    string  `json:"message"`
		}{
			UserID:     userID,
			TotalSpent: analytics.TotalSpent,
			Message:    fmt.Sprintf("User %s has spent a total of %.2f", userID, analytics.TotalSpent),
		})
	}
}

func (h *Handler) topUsersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			parsed, err := strconv.Atoi(l)
			if err != nil || parsed <= 0 {
				writeError(w, http.StatusBadRequest, "limit must be a positive integer")
				return
			}
			limit = parsed
		}

		users, err := h.svc.TopUsers(r.Context(), limit)
		if err != nil {
			h.logger.Error("failed to get top users", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to get top users")
			return
		}

		writeJSON(w, http.StatusOK, users)
	}
}

func (h *Handler) anomaliesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		anomalies, err := h.svc.Anomalies(r.Context())
		if err != nil {
			h.logger.Error("failed to detect anomalies", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to detect anomalies")
			return
		}

		writeJSON(w, http.StatusOK, anomalies)
	}
}
