package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/trip"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// container holds the wired dependencies. It is a small, explicit composition
// root — no DI framework — assembled in buildContainer.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires all dependencies in order:
// postgres (with auto-migrations) -> validator -> repository -> service ->
// handler -> router. Long-lived resources register themselves with the closer.
func buildContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	validator, err := validation.NewValidator(
		validation.BeforeNowTag(),
		validation.OriginTag(),
	)
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	repo := trip.NewRepository(db)
	service := trip.NewService(repo, log)
	handler := trip.NewHandler(service, validator, log)
	router := httpserver.NewRouter(log, handler)

	return &container{
		db:     db,
		router: router,
	}, nil
}
