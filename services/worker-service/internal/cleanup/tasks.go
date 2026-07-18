package cleanup

import (
	workerconfig "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/config"
)

// DefaultTasks is intentionally explicit. Adding a growing table or file type
// means adding it here and to the retention-policy document, rather than
// silently sweeping a service database.
func DefaultTasks(cfg workerconfig.Cleanup, token string) []Task {
	timeout := cfg.Timeout()
	tasks := make([]Task, 0, 10)
	add := func(name, description, owner, retention, baseURL string) {
		tasks = append(tasks, NewHTTPTask(Descriptor{Name: name, Description: description, OwningService: owner, DefaultRetention: retention, DryRunSupported: true}, baseURL, token, timeout))
	}
	add("expired_refresh_tokens", "Expired refresh tokens past the retention window.", "auth-service", "30 days", cfg.AuthServiceURL)
	add("revoked_refresh_tokens", "Revoked refresh tokens past the retention window.", "auth-service", "30 days", cfg.AuthServiceURL)
	add("read_notifications", "Read in-app notifications past the retention window.", "notification-service", "180 days", cfg.NotificationServiceURL)
	add("unread_notifications", "Unread notifications past the longer retention window.", "notification-service", "365 days", cfg.NotificationServiceURL)
	add("notification_digests", "Final notification digest batches and items.", "notification-service", "180 days", cfg.NotificationServiceURL)
	add("inactive_push_subscriptions", "Disabled push subscriptions only; active subscriptions are preserved.", "notification-service", "180 days", cfg.NotificationServiceURL)
	add("provider_cache", "Expired persisted provider cache rows; in-memory caches self-evict.", "external-integrations-service", "expiry + 7 days", cfg.ExternalServiceURL)
	add("oauth_states", "Used or expired OAuth state values.", "external-integrations-service", "1 day", cfg.ExternalServiceURL)
	add("provider_quota_counters", "Daily quota counter rows past reporting retention.", "external-integrations-service", "400 days", cfg.ExternalServiceURL)
	return tasks
}
