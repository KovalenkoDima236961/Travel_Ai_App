package handler

import "github.com/prometheus/client_golang/prometheus"

var (
	notificationsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_created_total", Help: "Total notifications created."},
		[]string{"type", "channel"},
	)
	notificationsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_failed_total", Help: "Total notification delivery failures."},
		[]string{"type", "channel", "error_code"},
	)
	notificationsEmailSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_email_sent_total", Help: "Total notification email send outcomes."},
		[]string{"type", "result"},
	)
	notificationsPushSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "push_notifications_sent_total", Help: "Total notification push send outcomes."},
		[]string{"type", "category", "result"},
	)
	notificationsPushFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "push_notifications_failed_total", Help: "Total notification push failures."},
		[]string{"type", "category", "error_code"},
	)
	notificationsPushSubscriptionsDisabled = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "push_subscriptions_disabled_total", Help: "Total browser push subscriptions disabled."},
		[]string{"reason"},
	)
	notificationsSSEConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "notifications_sse_connections", Help: "Notification SSE connections."},
		[]string{"status"},
	)
	notificationsSSEEventsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_sse_events_sent_total", Help: "Notification SSE events sent."},
		[]string{"event_type"},
	)
	notificationsSSEEventsDropped = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_sse_events_dropped_total", Help: "Notification SSE events dropped."},
		[]string{"event_type", "reason"},
	)
)

func init() {
	prometheus.MustRegister(
		notificationsCreated,
		notificationsFailed,
		notificationsEmailSent,
		notificationsPushSent,
		notificationsPushFailed,
		notificationsPushSubscriptionsDisabled,
		notificationsSSEConnections,
		notificationsSSEEventsSent,
		notificationsSSEEventsDropped,
	)
}

func recordNotificationCreated(notificationType, channel string) {
	notificationsCreated.WithLabelValues(notificationType, channel).Inc()
}

func recordNotificationFailed(notificationType, channel, errorCode string) {
	notificationsFailed.WithLabelValues(notificationType, channel, errorCode).Inc()
}

func recordNotificationEmail(typeLabel, result string, count int) {
	if count <= 0 {
		return
	}
	notificationsEmailSent.WithLabelValues(typeLabel, result).Add(float64(count))
}

func recordNotificationPush(typeLabel, category, result string, count int) {
	if count <= 0 {
		return
	}
	notificationsPushSent.WithLabelValues(typeLabel, category, result).Add(float64(count))
	if result == "failed" {
		notificationsPushFailed.WithLabelValues(typeLabel, category, "send_failed").Add(float64(count))
	}
	if result == "disabled" {
		notificationsPushSubscriptionsDisabled.WithLabelValues("gone_or_invalid").Add(float64(count))
	}
}

func recordNotificationSSEConnection(status string, delta float64) {
	notificationsSSEConnections.WithLabelValues(status).Add(delta)
}

func recordNotificationSSEEventSent(eventType string) {
	notificationsSSEEventsSent.WithLabelValues(eventType).Inc()
}

func recordNotificationSSEEventDropped(eventType, reason string) {
	notificationsSSEEventsDropped.WithLabelValues(eventType, reason).Inc()
}
