package search

import (
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

func NewModule(db *postgres.DB, workspaceProvider WorkspaceProvider, cfg Config, log *zap.Logger) *Handler {
	repo := NewRepository(db)
	service := NewService(repo, workspaceProvider, cfg, log)
	normalized := NormalizeConfig(cfg)
	return NewHandler(service, log, tripsecurity.NewRateLimiter(normalized.RateLimitPerMin, time.Minute))
}
