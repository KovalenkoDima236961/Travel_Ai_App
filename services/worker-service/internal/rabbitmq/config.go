package rabbitmq

import (
	"time"

	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
)

// GenerationQueueConfig maps Trip Service generation-job settings to the
// RabbitMQ topology used by the worker.
func GenerationQueueConfig(cfg *tripconfig.Config) jobqueue.Config {
	return jobqueue.Config{
		URL:                  cfg.GenerationJobs.RabbitMQURL,
		Exchange:             cfg.GenerationJobs.RabbitMQExchange,
		DLX:                  cfg.GenerationJobs.RabbitMQDLX,
		QueueName:            cfg.GenerationJobs.QueueName,
		RoutingKey:           cfg.GenerationJobs.RoutingKey,
		DeadLetterQueueName:  cfg.GenerationJobs.DeadLetterQueueName,
		DeadLetterRoutingKey: cfg.GenerationJobs.DeadLetterRoutingKey,
		RetryQueueName:       cfg.GenerationJobs.RetryQueueName,
		RetryRoutingKey:      cfg.GenerationJobs.RetryRoutingKey,
		RetryDelay:           time.Duration(cfg.GenerationJobs.RetryDelaySeconds) * time.Second,
		PublishTimeout:       cfg.GenerationJobPublishTimeout(),
	}
}
