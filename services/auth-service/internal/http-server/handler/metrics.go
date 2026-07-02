package handler

import "github.com/prometheus/client_golang/prometheus"

var (
	authRegisterTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "auth_register_total", Help: "Total auth register requests."},
		[]string{"result"},
	)
	authLoginTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "auth_login_total", Help: "Total auth login requests."},
		[]string{"result"},
	)
	authRefreshTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "auth_refresh_total", Help: "Total auth token refresh requests."},
		[]string{"result"},
	)
	authLogoutTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "auth_logout_total", Help: "Total auth logout requests."},
		[]string{"result"},
	)
)

func init() {
	prometheus.MustRegister(authRegisterTotal, authLoginTotal, authRefreshTotal, authLogoutTotal)
}

func recordAuthRegister(result string) {
	authRegisterTotal.WithLabelValues(result).Inc()
}

func recordAuthLogin(result string) {
	authLoginTotal.WithLabelValues(result).Inc()
}

func recordAuthRefresh(result string) {
	authRefreshTotal.WithLabelValues(result).Inc()
}

func recordAuthLogout(result string) {
	authLogoutTotal.WithLabelValues(result).Inc()
}
