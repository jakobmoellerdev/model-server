package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/open-component-model/model-server/internal/config"
)

// Server wraps an http.Server with graceful shutdown.
type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

// New creates a Server with the provided config, handler, and logger.
func New(cfg config.ServerConfig, handler http.Handler, log *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Listen,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout.Duration,
			WriteTimeout: cfg.WriteTimeout.Duration,
			IdleTimeout:  cfg.IdleTimeout.Duration,
		},
		log: log,
	}
}

// Start begins serving. Returns when the server stops.
func (s *Server) Start() error {
	s.log.Info("starting server", slog.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown drains connections with the given timeout.
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	s.log.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
