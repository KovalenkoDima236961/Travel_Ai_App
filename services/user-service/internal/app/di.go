package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/http-server/handler"
	userrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/infrastructure/repository/postgres"
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
// handler -> router. Long-lived resources register themselves with closer.
func buildContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	validator, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	repo := userrepo.New(db)
	svc := service.New(repo, log)
	userHandler := handler.New(svc, validator, log)
	readinessHandler := httpserver.NewReadinessHandler(db, log)
	router := httpserver.NewRouter(log, userHandler, readinessHandler, cfg.CORS, cfg.Auth)

	return &container{
		db:     db,
		router: router,
	}, nil
}
