package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/authusers"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/dataexport"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/httpserver"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/httpserver/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/notifications"
	userrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/workspaces"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/validation"
)

// container holds the wired dependencies. It is a small, explicit composition
// root, matching the Auth Service and Trip Service style.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// postgres (with auto-migrations) -> validator -> repository -> service ->
// handler -> router. Long-lived resources register themselves with shutdown.
func buildContainer(
	ctx context.Context,
	cfg *config.Config,
	log *zap.Logger,
	shutdown *closer.Stack,
) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	shutdown.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	validator, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	repo := userrepo.New(db)
	exportStorage, err := dataexport.NewLocalStorage(cfg.DataExports.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("init data export storage: %w", err)
	}
	serviceOptions := []service.Option{service.WithDataExports(exportStorage, dataexport.Config{
		Enabled: cfg.DataExports.Enabled, StorageDir: cfg.DataExports.StorageDir,
		TTL:             time.Duration(cfg.DataExports.TTLHours) * time.Hour,
		MaxAccountBytes: int64(cfg.DataExports.MaxAccountExportMB) * 1024 * 1024,
	})}
	if cfg.TripExports.Enabled {
		tripPackageClient, clientErr := dataexport.NewTripPackageClient(cfg.TripExports.TripServiceURL, cfg.TripExports.ServiceToken, time.Duration(cfg.TripExports.TimeoutSeconds)*time.Second, int64(cfg.DataExports.MaxAccountExportMB)*1024*1024)
		if clientErr != nil {
			return nil, fmt.Errorf("init account trip export client: %w", clientErr)
		}
		serviceOptions = append(serviceOptions, service.WithAccountTripPackageProvider(tripPackageClient))
	}
	svc := service.New(repo, log, serviceOptions...)
	if cfg.DataExports.CleanupEnabled {
		shutdown.Add("data-export-cleanup", service.StartAccountExportCleanupLoop(context.Background(), svc, time.Duration(cfg.DataExports.CleanupIntervalMinutes)*time.Minute, log))
	}
	userHandler := handler.New(svc, validator, log)
	authUsersClient, err := authusers.New(authusers.Config{
		BaseURL:        cfg.AuthUsers.AuthServiceURL,
		Token:          cfg.Internal.ServiceToken,
		TimeoutSeconds: cfg.AuthUsers.TimeoutSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("init auth users client: %w", err)
	}
	workspaceRepo := workspaces.NewRepository(db)
	workspaceOptions := []workspaces.Option{workspaces.WithUserLookup(authUsersClient)}
	if cfg.Notifications.Enabled {
		notificationClient, err := notifications.New(notifications.Config{
			BaseURL:        cfg.Notifications.NotificationServiceURL,
			Token:          cfg.Notifications.NotificationServiceToken,
			TimeoutSeconds: cfg.Notifications.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init notification client: %w", err)
		}
		workspaceOptions = append(workspaceOptions, workspaces.WithNotifications(
			notificationClient,
			cfg.Notifications.Enabled,
			cfg.Notifications.FailOpen,
			cfg.Notifications.PublicWebBaseURL,
		))
	}
	workspaceSvc := workspaces.NewService(workspaceRepo, log, workspaceOptions...)
	workspaceHandler := workspaces.NewHandler(workspaceSvc, log)
	readinessHandler := httpserver.NewReadinessHandler(db, log)
	router := httpserver.NewRouter(log, userHandler, workspaceHandler, readinessHandler, cfg.CORS, cfg.Auth, cfg.Internal)

	return &container{
		db:     db,
		router: router,
	}, nil
}
