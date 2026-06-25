package app

import (
	"context"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/logger"
)

// App is the composition root and lifecycle owner for the notification service.
type App struct {
	cfg    *config.Config
	log    *zap.Logger
	server *httpserver.Server
}

// New loads configuration, builds the logger, wires dependencies, and returns
// a ready-to-run App. It logs fatally on any bootstrap failure.
func New(configPath string) *App {
	cfg := config.MustLoad(configPath)

	log := logger.InitLogger()
	log.Info("configuration loaded", zap.String("address", cfg.HTTPServer.Address))
	warnWeakDevelopmentSecrets(cfg, log)

	c, err := buildContainer(context.Background(), cfg, log)
	if err != nil {
		log.Fatal("failed to build application", zap.Error(err))
	}

	return &App{
		cfg:    cfg,
		log:    log,
		server: httpserver.New(cfg.HTTPServer, log, c.router),
	}
}

// Run starts the HTTP server and blocks until an interrupt/terminate signal is
// received, then gracefully shuts down all registered resources (LIFO).
func (a *App) Run() {
	if err := a.server.Start(); err != nil {
		a.log.Fatal("failed to start http server", zap.Error(err))
	}
	closer.Add("http-server", a.server.Shutdown)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	a.log.Info("shutdown signal received, releasing resources")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTPServer.ShutdownTimeout)
	defer cancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		a.log.Error("error during graceful shutdown", zap.Error(err))
	}

	a.log.Info("shutdown complete")
	_ = a.log.Sync()
}

// warnWeakDevelopmentSecrets surfaces development-only secrets at startup so an
// operator notices before promoting the service. It never logs the secret value.
func warnWeakDevelopmentSecrets(cfg *config.Config, log *zap.Logger) {
	if cfg.IsProduction() {
		return
	}
	secret := strings.TrimSpace(cfg.JWT.AccessSecret)
	if secret == config.DefaultDevelopmentJWTSecret || len(secret) < config.MinProductionJWTSecretLength {
		log.Warn(
			"JWT access secret is suitable only for development",
			zap.Int("length", len(secret)),
		)
	}
	if strings.TrimSpace(cfg.Internal.ServiceToken) == config.DefaultDevelopmentInternalToken {
		log.Warn("internal service token is the development default")
	}
}
