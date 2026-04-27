package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryAndRequestLoggerCapturePanicAs500(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	handler := RequestLogger(logger)(Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %q", len(lines), logs.String())
	}

	foundPanic := false
	foundHTTP500 := false
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("unmarshal log: %v", err)
		}
		if entry["msg"] == "panic recovered" {
			foundPanic = true
		}
		if entry["msg"] == "http" {
			if status, ok := entry["status"].(float64); ok && int(status) == http.StatusInternalServerError {
				foundHTTP500 = true
			}
		}
	}

	if !foundPanic {
		t.Fatal("expected panic recovery log entry")
	}
	if !foundHTTP500 {
		t.Fatal("expected request log entry with status 500")
	}
}

func TestRequestLoggerCapturesExplicitStatus(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/created", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusCreated {
		t.Fatalf("expected logged status 201, got %+v", entry["status"])
	}
}
