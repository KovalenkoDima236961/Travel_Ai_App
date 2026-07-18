package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/middleware"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/observability"
)

// NewRouter builds the application's chi router with middleware and routes.
//
// Route groups:
//   - /health, /ready are open (no auth).
//   - /notifications/* require a valid user JWT; user_id always comes from the
//     token so a user can only see their own notifications.
//   - /internal/notifications/* require the internal service token (no user JWT)
//     and are intended for the private service network only.
func NewRouter(
	log *zap.Logger,
	notificationHandler *handler.Handler,
	internalHandler *handler.InternalHandler,
	readinessHandler http.Handler,
	corsCfg config.CORSConfig,
	jwtCfg config.JWTConfig,
	internalCfg config.InternalConfig,
) http.Handler {
	r := chi.NewRouter()

	r.Use(observability.RequestIDMiddleware)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(observability.HTTPMetricsMiddleware(observability.DefaultHTTPMetrics("notification-service")))
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(corsCfg))

	r.Get("/health", healthHandler)
	if readinessHandler != nil {
		r.Get("/ready", readinessHandler.ServeHTTP)
	}
	r.Handle("/metrics", observability.MetricsHandler(nil))
	notificationHandler.RegisterPublicRoutes(r)

	// User-facing routes: require a valid access token.
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(auth.MiddlewareConfig{
			JWTAccessSecret: jwtCfg.AccessSecret,
			HeaderName:      jwtCfg.HeaderName,
		}))
		notificationHandler.RegisterRoutes(r)
	})

	// Internal service-to-service routes: require the internal token only.
	r.Group(func(r chi.Router) {
		r.Use(internalmw.InternalServiceToken(internalCfg.ActiveServiceTokens(), log))
		internalHandler.RegisterRoutes(r)
	})

	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "notification-service"})
}

// requestLogger logs one structured line per request using Zap. It never logs
// Authorization headers or the internal service token.
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
				zap.String("service", "notification-service"),
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
