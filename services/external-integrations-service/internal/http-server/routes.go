package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/middleware"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/ops"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/prices"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

// NewRouter builds the application's chi router with middleware and routes.
func NewRouter(
	log *zap.Logger,
	placesHandler *handler.PlacesHandler,
	routesHandler *handler.RoutesHandler,
	weatherHandler *handler.WeatherHandler,
	exchangeRateHandler *handler.ExchangeRateHandler,
	priceHandler *prices.Handler,
	calendarHandler *handler.CalendarHandler,
	internalCalendarHandler *handler.InternalCalendarHandler,
	providerOpsHandler *handler.ProviderOpsHandler,
	readinessHandler http.Handler,
	corsCfg config.CORSConfig,
	authCfg config.AuthConfig,
	internalCfg config.InternalConfig,
	opsCfg config.OpsConfig,
) http.Handler {
	r := chi.NewRouter()

	r.Use(observability.RequestIDMiddleware)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(observability.HTTPMetricsMiddleware(observability.DefaultHTTPMetrics("external-integrations-service")))
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(corsCfg))

	r.Get("/health", healthHandler)
	if readinessHandler != nil {
		r.Get("/ready", readinessHandler.ServeHTTP)
	}
	r.Handle("/metrics", observability.MetricsHandler(nil))
	placesHandler.RegisterRoutes(r)
	routesHandler.RegisterRoutes(r)
	weatherHandler.RegisterRoutes(r)
	if exchangeRateHandler != nil {
		exchangeRateHandler.RegisterRoutes(r)
	}
	if priceHandler != nil {
		r.Group(func(r chi.Router) {
			r.Use(internalmw.InternalServiceToken(internalCfg.ServiceToken))
			priceHandler.RegisterRoutes(r)
		})
	}
	if calendarHandler != nil {
		r.Get("/calendar/google/callback", calendarHandler.Callback)
	}

	if calendarHandler != nil {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(auth.MiddlewareConfig{
				JWTAccessSecret: authCfg.JWTAccessSecret,
				HeaderName:      authCfg.HeaderName,
			}))
			r.Get("/calendar/google/status", calendarHandler.Status)
			r.Post("/calendar/google/connect", calendarHandler.Connect)
			r.Delete("/calendar/google/disconnect", calendarHandler.Disconnect)
		})
	}

	if internalCalendarHandler != nil {
		r.Group(func(r chi.Router) {
			r.Use(internalmw.InternalServiceToken(internalCfg.ServiceToken))
			r.Post("/internal/calendar/google/events/sync", internalCalendarHandler.SyncGoogleEvents)
			r.Post("/internal/calendar/google/events/delete", internalCalendarHandler.DeleteGoogleEvents)
		})
	}

	if opsCfg.DashboardEnabled && providerOpsHandler != nil {
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(auth.MiddlewareConfig{
				JWTAccessSecret: authCfg.JWTAccessSecret,
				HeaderName:      authCfg.HeaderName,
			}))
			r.Use(ops.NewAdminChecker(opsCfg, log).Middleware)
			providerOpsHandler.RegisterRoutes(r)
		})
	}

	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// requestLogger logs one structured line per request using Zap.
func requestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			fields := []zap.Field{
				zap.String("service", "external-integrations-service"),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("route", observability.RoutePattern(r)),
				zap.Int("status", status),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Float64("durationMs", float64(time.Since(start).Microseconds())/1000),
			}
			fields = append(fields, observability.RequestIDFields(r.Context())...)
			log.Info("http_request", fields...)
		})
	}
}
