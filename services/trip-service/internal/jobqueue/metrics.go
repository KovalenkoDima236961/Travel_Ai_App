package jobqueue

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	rabbitMQMessagesPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_published_total",
			Help: "Total RabbitMQ messages successfully published.",
		},
		[]string{"queue", "routing_key", "message_type"},
	)
	rabbitMQPublishFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_publish_failures_total",
			Help: "Total RabbitMQ publish failures.",
		},
		[]string{"queue", "routing_key", "error_code"},
	)
	rabbitMQPublishDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rabbitmq_publish_duration_seconds",
			Help:    "RabbitMQ publish duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue", "routing_key"},
	)
)

func init() {
	prometheus.MustRegister(
		rabbitMQMessagesPublished,
		rabbitMQPublishFailures,
		rabbitMQPublishDuration,
	)
}

func recordPublishSuccess(queue, routingKey, messageType string, duration time.Duration) {
	rabbitMQMessagesPublished.WithLabelValues(queue, routingKey, messageType).Inc()
	rabbitMQPublishDuration.WithLabelValues(queue, routingKey).Observe(duration.Seconds())
}

func recordPublishFailure(queue, routingKey, errorCode string, duration time.Duration) {
	rabbitMQPublishFailures.WithLabelValues(queue, routingKey, errorCode).Inc()
	rabbitMQPublishDuration.WithLabelValues(queue, routingKey).Observe(duration.Seconds())
}
