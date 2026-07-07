package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
)

// Server wraps an *http.Server with start/graceful-shutdown helpers.
type Server struct {
	srv *http.Server
	log *zap.Logger
}

// New builds the HTTP server from configuration and a router.
func New(cfg config.HTTPServer, log *zap.Logger, handler http.Handler) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.Address,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: 10 * time.Second,
		},
		log: log,
	}
}

// Start binds the listener (failing fast on bind errors) and serves requests
// in a background goroutine.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}

	s.log.Info("http server listening", zap.String("addr", s.srv.Addr))
	go func() {
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("http server stopped unexpectedly", zap.Error(err))
		}
	}()
	return nil
}

// Shutdown gracefully drains in-flight requests. It is registered with the
// closer so it runs on shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.srv.Shutdown(ctx)
}
