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
	notificationsDeliveryDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_delivery_decisions_total", Help: "Deterministic notification delivery decisions."},
		[]string{"channel", "category", "priority", "mode", "decision", "reason"},
	)
	notificationsMuted             = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notifications_muted_total", Help: "Notifications muted by policy."}, []string{"channel", "category", "priority"})
	notificationsDigested          = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notifications_digested_total", Help: "Notifications queued for digests."}, []string{"channel", "category", "priority", "mode"})
	notificationsInstantSent       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notifications_instant_sent_total", Help: "Notifications selected for instant delivery."}, []string{"channel", "category", "priority"})
	notificationsQuietHoursDelayed = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_quiet_hours_delayed_total", Help: "Notifications delayed by quiet hours."}, []string{"channel", "category"})
	notificationDedupeDropped      = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "notification_dedupe_dropped_total", Help: "Duplicate notification events grouped or dropped."}, []string{"result"})
	notificationCleanupDeleted     = prometheus.NewCounter(prometheus.CounterOpts{Name: "notification_cleanup_deleted_total", Help: "Notifications permanently removed by explicit cleanup."})
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
		notificationsDeliveryDecisions,
		notificationsMuted,
		notificationsDigested,
		notificationsInstantSent,
		notificationsQuietHoursDelayed,
		notificationDedupeDropped,
		notificationCleanupDeleted,
	)
}

func recordNotificationCleanupDeleted(count int) {
	if count > 0 {
		notificationCleanupDeleted.Add(float64(count))
	}
}

func recordDeliveryDecision(channel, category, priority, mode, decision, reason string) {
	notificationsDeliveryDecisions.WithLabelValues(channel, category, priority, mode, decision, reason).Inc()
	switch decision {
	case "mute":
		notificationsMuted.WithLabelValues(channel, category, priority).Inc()
	case "digest", "delay_until_quiet_hours_end":
		notificationsDigested.WithLabelValues(channel, category, priority, mode).Inc()
	case "create_in_app_only":
		if mode != "instant" {
			notificationsDigested.WithLabelValues(channel, category, priority, mode).Inc()
		}
	case "send_instant":
		notificationsInstantSent.WithLabelValues(channel, category, priority).Inc()
	}
}
func recordQuietHoursDelayed(channel, category string) {
	notificationsQuietHoursDelayed.WithLabelValues(channel, category).Inc()
}
func recordDedupeDropped(count int) {
	if count > 0 {
		notificationDedupeDropped.WithLabelValues("grouped").Add(float64(count))
	}
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
