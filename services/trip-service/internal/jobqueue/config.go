package jobqueue

import "time"

type Config struct {
	URL                  string
	Exchange             string
	DLX                  string
	QueueName            string
	RoutingKey           string
	DeadLetterQueueName  string
	DeadLetterRoutingKey string
	RetryQueueName       string
	RetryRoutingKey      string
	RetryDelay           time.Duration
	PublishTimeout       time.Duration
}

func NormalizeConfig(cfg Config) Config {
	if cfg.Exchange == "" {
		cfg.Exchange = "trip.jobs.exchange"
	}
	if cfg.DLX == "" {
		cfg.DLX = "trip.jobs.dlx"
	}
	if cfg.QueueName == "" {
		cfg.QueueName = "trip.generation.jobs"
	}
	if cfg.RoutingKey == "" {
		cfg.RoutingKey = "trip.generation"
	}
	if cfg.DeadLetterQueueName == "" {
		cfg.DeadLetterQueueName = "trip.generation.dead_letter"
	}
	if cfg.DeadLetterRoutingKey == "" {
		cfg.DeadLetterRoutingKey = "trip.generation.dead"
	}
	if cfg.RetryQueueName == "" {
		cfg.RetryQueueName = "trip.generation.retry"
	}
	if cfg.RetryRoutingKey == "" {
		cfg.RetryRoutingKey = "trip.generation.retry"
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 10 * time.Second
	}
	if cfg.PublishTimeout <= 0 {
		cfg.PublishTimeout = 5 * time.Second
	}
	return cfg
}
