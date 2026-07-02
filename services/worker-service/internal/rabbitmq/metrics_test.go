package rabbitmq

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMessageMetricsRecordRetryAndDeadLetter(t *testing.T) {
	queue := "metrics.test.queue"
	messageType := "metrics_test_message"
	reason := "metrics_test_reason"
	retriesBefore := testutil.ToFloat64(workerMessagesRetried.WithLabelValues(queue, messageType))
	deadBefore := testutil.ToFloat64(workerMessagesDeadLettered.WithLabelValues(queue, messageType, reason))

	recordRetried(queue, messageType)
	recordDeadLettered(queue, messageType, reason)

	if got := testutil.ToFloat64(workerMessagesRetried.WithLabelValues(queue, messageType)); got != retriesBefore+1 {
		t.Fatalf("retries = %v, want %v", got, retriesBefore+1)
	}
	if got := testutil.ToFloat64(workerMessagesDeadLettered.WithLabelValues(queue, messageType, reason)); got != deadBefore+1 {
		t.Fatalf("dead letters = %v, want %v", got, deadBefore+1)
	}
}
