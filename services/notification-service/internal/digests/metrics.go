package digests

import "github.com/prometheus/client_golang/prometheus"

var (
	digestBatchesCreated = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_digest_batches_created_total", Help: "Notification digest batches created."}, []string{"channel", "mode"})
	digestBatchesSent    = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_digest_batches_sent_total", Help: "Notification digest batches sent."}, []string{"channel", "mode"})
	digestBatchesFailed  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_digest_batches_failed_total", Help: "Notification digest batches failed."}, []string{"channel", "mode"})
	digestItemsGrouped   = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_digest_items_grouped_total", Help: "Notification digest items grouped."}, []string{"channel", "mode"})
)

func init() {
	prometheus.MustRegister(digestBatchesCreated, digestBatchesSent, digestBatchesFailed, digestItemsGrouped)
}
func recordDigestQueued(channel, mode string, grouped, batchCreated bool) {
	if batchCreated {
		digestBatchesCreated.WithLabelValues(channel, mode).Inc()
	}
	if grouped {
		digestItemsGrouped.WithLabelValues(channel, mode).Inc()
	}
}
func recordDigestSent(channel, mode string) { digestBatchesSent.WithLabelValues(channel, mode).Inc() }
func recordDigestFailed(channel, mode string) {
	digestBatchesFailed.WithLabelValues(channel, mode).Inc()
}
