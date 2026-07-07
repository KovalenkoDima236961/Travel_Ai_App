package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/httpserver"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/logger"
)

// App is the composition root and lifecycle owner for the external integrations
// service.
type App struct {
	cfg      *config.Config
	log      *zap.Logger
	server   *httpserver.Server
	shutdown *closer.Stack
}

// New loads configuration, builds the logger, wires dependencies, and returns a
// ready-to-run App.
func New(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.InitLogger()
	log.Info("configuration loaded", zap.String("address", cfg.HTTPServer.Address))

	shutdown := closer.New()
	c, err := buildContainer(context.Background(), cfg, log, shutdown)
	if err != nil {
		return nil, fmt.Errorf("build application: %w", err)
	}

	return &App{
		cfg:      cfg,
		log:      log,
		server:   httpserver.New(cfg.HTTPServer, log, c.router),
		shutdown: shutdown,
	}, nil
}

// Run starts the HTTP server and blocks until an interrupt/terminate signal is
// received, then gracefully shuts down registered resources.
func (a *App) Run() error {
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	a.shutdown.Add("http-server", a.server.Shutdown)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	a.log.Info("shutdown signal received, releasing resources")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTPServer.ShutdownTimeout)
	defer cancel()

	closeErr := a.shutdown.CloseAll(shutdownCtx)
	if closeErr != nil {
		a.log.Error("error during graceful shutdown", zap.Error(closeErr))
	}

	a.log.Info("shutdown complete")
	_ = a.log.Sync()
	return closeErr
}
