package ops

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

type AdminChecker struct {
	enabled bool
	emails  map[string]struct{}
	log     *zap.Logger
}

func NewAdminChecker(cfg config.OpsConfig, log *zap.Logger) AdminChecker {
	if log == nil {
		log = zap.NewNop()
	}
	return AdminChecker{
		enabled: cfg.DashboardEnabled,
		emails:  ParseAdminEmails(cfg.AdminEmails),
		log:     log,
	}
}

func ParseAdminEmails(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		email := strings.ToLower(strings.TrimSpace(part))
		if email != "" {
			out[email] = struct{}{}
		}
	}
	return out
}

func (c AdminChecker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.enabled {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		user, ok := auth.UserFromContext(r.Context())
		if !ok || strings.TrimSpace(user.Email) == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if _, allowed := c.emails[email]; !allowed {
			c.log.Warn("ops access denied",
				zap.String("opsAdminEmail", email),
				zap.String("userId", user.ID.String()),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.Any("requestIds", observability.RequestIDFields(r.Context())),
			)
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
