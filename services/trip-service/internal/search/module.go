package search

import (
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

func NewModule(db *postgres.DB, workspaceProvider WorkspaceProvider, cfg Config, log *zap.Logger) *Handler {
	repo := NewRepository(db)
	service := NewService(repo, workspaceProvider, cfg, log)
	return NewHandler(service, log)
}
