package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"xugeaneeu/pollify/internal/platform/auth"
	"xugeaneeu/pollify/internal/platform/config"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func newHandler(verifier auth.TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.Handle("/api/v1/", auth.Middleware(verifier)(http.NotFoundHandler()))

	return mux
}

func New(cfg config.Config, logger *slog.Logger, verifier auth.TokenVerifier) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.HTTPAddress,
			Handler:           newHandler(verifier),
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
