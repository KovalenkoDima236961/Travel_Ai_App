package rabbitmq

import "github.com/prometheus/client_golang/prometheus"

var (
	workerMessagesConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_consumed_total",
			Help: "Total RabbitMQ messages consumed by worker.",
		},
		[]string{"queue", "message_type"},
	)
	workerMessagesAcked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_acked_total",
			Help: "Total RabbitMQ messages acked by worker.",
		},
		[]string{"queue", "message_type"},
	)
	workerMessagesNacked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_nacked_total",
			Help: "Total RabbitMQ messages nacked by worker.",
		},
		[]string{"queue", "message_type", "reason"},
	)
	workerMessagesRetried = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_retried_total",
			Help: "Total RabbitMQ messages scheduled for retry by worker.",
		},
		[]string{"queue", "message_type"},
	)
	workerMessagesDeadLettered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_dead_lettered_total",
			Help: "Total RabbitMQ messages dead-lettered by worker.",
		},
		[]string{"queue", "message_type", "reason"},
	)
)

func init() {
	prometheus.MustRegister(
		workerMessagesConsumed,
		workerMessagesAcked,
		workerMessagesNacked,
		workerMessagesRetried,
		workerMessagesDeadLettered,
	)
}

func recordConsumed(queue, messageType string) {
	workerMessagesConsumed.WithLabelValues(queue, messageType).Inc()
}

func recordAcked(queue, messageType string) {
	workerMessagesAcked.WithLabelValues(queue, messageType).Inc()
}

func recordNacked(queue, messageType, reason string) {
	workerMessagesNacked.WithLabelValues(queue, messageType, reason).Inc()
}

func recordRetried(queue, messageType string) {
	workerMessagesRetried.WithLabelValues(queue, messageType).Inc()
}

func recordDeadLettered(queue, messageType, reason string) {
	workerMessagesDeadLettered.WithLabelValues(queue, messageType, reason).Inc()
}
