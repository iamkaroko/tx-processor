// internal/server/server.go
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"tx-processor/internal/handler"
	"tx-processor/internal/middleware"
)

type Server struct {
	http   *http.Server
	logger *slog.Logger
}

func New(port string, h *handler.Handler, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	var chain http.Handler = mux
	chain = middleware.Recovery(logger)(chain)
	chain = middleware.RequestLogger(logger)(chain)

	return &Server{
		http: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: chain,
		},
		logger: logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server started", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.http.Shutdown(shutdownCtx)
	}
}
