package activity

import "go.uber.org/zap"

// New constructs the activity service from its persistence port and a logger.
func New(repo Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}
