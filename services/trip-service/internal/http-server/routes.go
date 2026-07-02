package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

// NewRouter builds the application's chi router with middleware and routes.
func NewRouter(
	log *zap.Logger,
	tripHandler *handler.Handler,
	readinessHandler http.Handler,
	corsCfg config.CORSConfig,
	authCfg config.AuthConfig,
) http.Handler {
	r := chi.NewRouter()

	r.Use(observability.RequestIDMiddleware)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(observability.HTTPMetricsMiddleware(observability.DefaultHTTPMetrics("trip-service")))
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(corsCfg))

	r.Get("/health", healthHandler)
	if readinessHandler != nil {
		r.Get("/ready", readinessHandler.ServeHTTP)
	}
	r.Handle("/metrics", observability.MetricsHandler(nil))

	tripHandler.RegisterPublicRoutes(r)

	devUserID, err := uuid.Parse(authCfg.DevUserID)
	if err != nil {
		log.Panic("invalid dev user id", zap.String("dev_user_id", authCfg.DevUserID), zap.Error(err))
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(auth.MiddlewareConfig{
			Required:        authCfg.Required,
			JWTAccessSecret: authCfg.JWTAccessSecret,
			HeaderName:      authCfg.HeaderName,
			DevUserID:       devUserID,
		}))
		tripHandler.RegisterRoutes(r)
	})

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
				zap.String("service", "trip-service"),
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
