package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dbQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "trip_db_query_duration_seconds",
		Help:    "Trip database query duration by low-cardinality SQL operation.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"operation"})
	dbQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trip_db_query_errors_total",
		Help: "Trip database query errors by low-cardinality SQL operation.",
	}, []string{"operation"})
	dbPoolConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trip_db_pool_connections",
		Help: "Current trip database pool connections by state.",
	}, []string{"state"})
)

func recordDBQuery(operation string, duration time.Duration, err error, stats *pgxpool.Stat) {
	dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
	if err != nil {
		dbQueryErrors.WithLabelValues(operation).Inc()
	}
	if stats != nil {
		dbPoolConnections.WithLabelValues("acquired").Set(float64(stats.AcquiredConns()))
		dbPoolConnections.WithLabelValues("idle").Set(float64(stats.IdleConns()))
		dbPoolConnections.WithLabelValues("total").Set(float64(stats.TotalConns()))
		dbPoolConnections.WithLabelValues("max").Set(float64(stats.MaxConns()))
	}
}
