package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/handler"
	notificationrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

// container holds the wired dependencies. It is a small, explicit composition
// root — no DI framework — assembled in buildContainer to match the existing
// Go services' style.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// postgres (with auto-migrations) -> repository -> service -> handlers ->
// router. Long-lived resources register themselves with the closer.
func buildContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	repo := notificationrepo.New(db)
	svc := notifications.New(repo, log)

	notificationHandler := handler.New(svc, log)
	internalHandler := handler.NewInternal(svc, log)
	readinessHandler := httpserver.NewReadinessHandler(db, log)

	router := httpserver.NewRouter(
		log,
		notificationHandler,
		internalHandler,
		readinessHandler,
		cfg.CORS,
		cfg.JWT,
		cfg.Internal,
	)

	return &container{
		db:     db,
		router: router,
	}, nil
}
