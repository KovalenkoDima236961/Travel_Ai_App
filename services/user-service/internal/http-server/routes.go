package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/http-server/handler"
)

// NewRouter builds the application's chi router with middleware and routes.
func NewRouter(
	log *zap.Logger,
	userHandler *handler.Handler,
	readinessHandler http.Handler,
	corsCfg config.CORSConfig,
	authCfg config.AuthConfig,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(corsMiddleware(corsCfg))

	r.Get("/health", healthHandler)
	if readinessHandler != nil {
		r.Get("/ready", readinessHandler.ServeHTTP)
	}

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
		userHandler.RegisterRoutes(r)
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			log.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
