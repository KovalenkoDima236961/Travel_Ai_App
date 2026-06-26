package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/handler"
	internalmw "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/middleware"
)

// NewRouter builds the application's chi router with middleware and routes.
func NewRouter(
	log *zap.Logger,
	placesHandler *handler.PlacesHandler,
	routesHandler *handler.RoutesHandler,
	weatherHandler *handler.WeatherHandler,
	calendarHandler *handler.CalendarHandler,
	internalCalendarHandler *handler.InternalCalendarHandler,
	readinessHandler http.Handler,
	corsCfg config.CORSConfig,
	authCfg config.AuthConfig,
	internalCfg config.InternalConfig,
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
	placesHandler.RegisterRoutes(r)
	routesHandler.RegisterRoutes(r)
	weatherHandler.RegisterRoutes(r)
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
