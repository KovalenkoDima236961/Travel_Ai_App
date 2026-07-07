package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	workerconfig "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/rabbitmq"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/pkg/logger"
)

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

// App owns worker-service configuration, dependency lifecycle, and runtime
// coordination.
type App struct {
	cfg      *workerconfig.Config
	log      *zap.Logger
	server   server
	consumer *rabbitmq.Consumer
	shutdown *closer.Stack
}

// New loads configuration and wires the worker. It returns errors to main so
// startup failures are visible without hiding control flow behind log.Fatal.
func New(configPath string) (*App, error) {
	cfg, err := workerconfig.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.InitLogger()
	log.Info("configuration loaded",
		zap.String("service", cfg.Runtime.ServiceName),
		zap.String("http_address", cfg.Runtime.HTTPAddress),
	)

	shutdown := closer.New()
	app := &App{
		cfg:      cfg,
		log:      log,
		shutdown: shutdown,
	}
	if !cfg.Runtime.Enabled {
		return app, nil
	}

	c, err := buildContainer(context.Background(), cfg, log, shutdown)
	if err != nil {
		_ = shutdown.CloseAll(context.Background())
		return nil, fmt.Errorf("build worker: %w", err)
	}
	app.server = c.server
	app.consumer = c.consumer
	return app, nil
}

// Run starts the RabbitMQ consumer and HTTP server, then blocks until a signal
// or runtime failure triggers graceful shutdown.
func (a *App) Run() error {
	if !a.cfg.Runtime.Enabled {
		a.log.Info("worker disabled")
		_ = a.log.Sync()
		return nil
	}
	if a.server == nil || a.consumer == nil {
		return fmt.Errorf("worker is not fully initialized")
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	consumerCtx, cancelConsumer := context.WithCancel(rootCtx)
	defer cancelConsumer()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- a.consumer.Run(consumerCtx)
	}()

	serverDone := make(chan error, 1)
	go func() {
		a.log.Info("worker http server starting", zap.String("address", a.cfg.Runtime.HTTPAddress))
		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	var runErr error
	consumerStopped := false
	select {
	case <-rootCtx.Done():
		a.log.Info("shutdown signal received")
	case err := <-consumerDone:
		consumerStopped = true
		if err != nil {
			runErr = fmt.Errorf("consumer stopped: %w", err)
			a.log.Error("consumer stopped", zap.Error(err))
		}
	case err := <-serverDone:
		if err != nil {
			runErr = fmt.Errorf("http server stopped: %w", err)
			a.log.Error("http server stopped", zap.Error(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout())
	defer cancel()
	cancelConsumer()

	if !consumerStopped {
		select {
		case err := <-consumerDone:
			if err != nil && runErr == nil {
				runErr = fmt.Errorf("consumer stopped during shutdown: %w", err)
				a.log.Warn("consumer stopped during shutdown", zap.Error(err))
			}
		case <-shutdownCtx.Done():
			a.log.Warn("consumer shutdown timed out", zap.Error(shutdownCtx.Err()))
		}
	}

	closeErr := a.shutdown.CloseAll(shutdownCtx)
	if closeErr != nil {
		a.log.Error("error during graceful shutdown", zap.Error(closeErr))
	}
	a.log.Info("shutdown complete")
	_ = a.log.Sync()

	if runErr != nil {
		return runErr
	}
	return closeErr
}
