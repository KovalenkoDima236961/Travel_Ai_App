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
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/middleware"
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

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(corsCfg))

	r.Get("/health", healthHandler)
	if readinessHandler != nil {
		r.Get("/ready", readinessHandler.ServeHTTP)
	}

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
		r.Use(internalmw.InternalServiceToken(internalCfg.ServiceToken))
		internalHandler.RegisterRoutes(r)
	})

	return r
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

			log.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("request_id", chimiddleware.GetReqID(r.Context())),
			)
		})
	}
}
