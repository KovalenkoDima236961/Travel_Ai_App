package handler

import "github.com/prometheus/client_golang/prometheus"

var (
	userProfileRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "user_profile_requests_total", Help: "Total user profile requests."},
		[]string{"operation", "result"},
	)
	userPreferencesRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "user_preferences_requests_total", Help: "Total user preferences requests."},
		[]string{"operation", "result"},
	)
)

func init() {
	prometheus.MustRegister(userProfileRequests, userPreferencesRequests)
}

func recordUserProfileRequest(operation, result string) {
	userProfileRequests.WithLabelValues(operation, result).Inc()
}

func recordUserPreferencesRequest(operation, result string) {
	userPreferencesRequests.WithLabelValues(operation, result).Inc()
}
