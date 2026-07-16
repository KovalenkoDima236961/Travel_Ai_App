package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	auth "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/httpserver"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/httpserver/handler"
	authrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

// container holds the wired dependencies. It is a small, explicit composition
// root, assembled in buildContainer to match the Trip Service style.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// postgres (with auto-migrations) -> repository -> password/tokens -> service
// -> handler -> router. Long-lived resources register themselves with shutdown.
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

	repo := authrepo.New(db)
	password := auth.NewPasswordHasher()
	tokens := auth.NewTokenManager(
		cfg.JWT.AccessSecret,
		cfg.AccessTokenTTL(),
		cfg.RefreshTokenTTL(),
	)
	svc := auth.New(repo, password, tokens, log)
	authHandler := handler.New(svc, log).EnableRateLimits(
		cfg.RateLimits.LoginPerMinute,
		cfg.RateLimits.RegisterPerMinute,
		cfg.RateLimits.RefreshPerMinute,
	)
	readinessHandler := httpserver.NewReadinessHandler(db, log)
	router := httpserver.NewRouter(log, authHandler, readinessHandler, cfg.CORS, cfg.Internal)

	return &container{
		db:     db,
		router: router,
	}, nil
}
