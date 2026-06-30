package activity

import "go.uber.org/zap"

type Option func(*Service)

// New constructs the activity service from its persistence port and a logger.
func New(repo Repository, log *zap.Logger, opts ...Option) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	svc := &Service{repo: repo, log: log}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func WithPublisher(publisher Publisher) Option {
	return func(s *Service) {
		s.publisher = publisher
	}
}
