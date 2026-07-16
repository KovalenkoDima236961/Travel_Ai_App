package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/httpserver/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/httpserver/dto/response"
)

type authService interface {
	Register(ctx context.Context, in appdto.RegisterInput) (*appdto.AuthResult, error)
	Login(ctx context.Context, in appdto.LoginInput) (*appdto.AuthResult, error)
	Refresh(ctx context.Context, in appdto.RefreshInput) (*appdto.TokenPair, error)
	Logout(ctx context.Context, in appdto.LogoutInput) error
	CurrentUser(ctx context.Context, accessToken string) (*entity.User, error)
	UserByEmail(ctx context.Context, email string) (*entity.User, error)
	UsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error)
}

// maxInternalUserBatch caps how many ids a single internal batch lookup may
// request, bounding the work a trusted caller can trigger.
const maxInternalUserBatch = 200

// Handler wires the auth use case to HTTP.
type Handler struct {
	svc             authService
	log             *zap.Logger
	loginLimiter    *authRateLimiter
	registerLimiter *authRateLimiter
	refreshLimiter  *authRateLimiter
}

// New constructs the auth HTTP handler.
func New(svc authService, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		svc: svc, log: log,
		loginLimiter:    newAuthRateLimiter(10),
		registerLimiter: newAuthRateLimiter(10),
		refreshLimiter:  newAuthRateLimiter(30),
	}
}

func (h *Handler) EnableRateLimits(login, register, refresh int) *Handler {
	h.loginLimiter = newAuthRateLimiter(login)
	h.registerLimiter = newAuthRateLimiter(register)
	h.refreshLimiter = newAuthRateLimiter(refresh)
	return h
}

// RegisterRoutes mounts the auth routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
		r.Get("/me", h.Me)
	})
}

// RegisterInternalRoutes mounts service-to-service endpoints that require the
// internal service token. The caller wraps these in the internal-token
// middleware; they must never be exposed to browsers and never require a user
// JWT.
func (h *Handler) RegisterInternalRoutes(r chi.Router) {
	r.Get("/internal/users/by-email", h.InternalUserByEmail)
	r.Post("/internal/users/batch", h.InternalUsersBatch)
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.allowSensitive(w, r, "register", h.registerLimiter) {
		return
	}
	var req request.Register
	if !decodeJSON(w, r, &req) {
		recordAuthRegister("invalid_request")
		return
	}

	resp, err := h.svc.Register(r.Context(), req.ToInput())
	if err != nil {
		recordAuthRegister("error")
		h.writeServiceError(w, err)
		return
	}

	recordAuthRegister("success")
	writeJSON(w, http.StatusCreated, response.NewAuth(resp))
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.allowSensitive(w, r, "login", h.loginLimiter) {
		return
	}
	var req request.Login
	if !decodeJSON(w, r, &req) {
		recordAuthLogin("invalid_request")
		return
	}

	resp, err := h.svc.Login(r.Context(), req.ToInput())
	if err != nil {
		recordAuthLogin("error")
		h.writeServiceError(w, err)
		return
	}

	recordAuthLogin("success")
	writeJSON(w, http.StatusOK, response.NewAuth(resp))
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if !h.allowSensitive(w, r, "refresh", h.refreshLimiter) {
		return
	}
	var req request.Refresh
	if !decodeJSON(w, r, &req) {
		recordAuthRefresh("invalid_request")
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.ToInput())
	if err != nil {
		recordAuthRefresh("error")
		h.writeServiceError(w, err)
		return
	}

	recordAuthRefresh("success")
	writeJSON(w, http.StatusOK, response.NewToken(resp))
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req request.Logout
	if !decodeJSON(w, r, &req) {
		recordAuthLogout("invalid_request")
		return
	}

	if err := h.svc.Logout(r.Context(), req.ToInput()); err != nil {
		recordAuthLogout("error")
		h.writeServiceError(w, err)
		return
	}

	recordAuthLogout("success")
	writeJSON(w, http.StatusOK, response.NewLogout())
}

// Me handles GET /auth/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	resp, err := h.svc.CurrentUser(r.Context(), token)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewUser(resp))
}

// InternalUserByEmail handles exact registered-user lookup behind the internal
// service-token middleware.
func (h *Handler) InternalUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	user, err := h.svc.UserByEmail(r.Context(), email)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewInternalUserLookup(user))
}

// InternalUsersBatch resolves a set of registered users by id for trusted
// internal callers (e.g. Notification Service resolving recipient emails). The
// route is mounted behind the internal service-token middleware. The response
// contains only the users that exist; ids with no matching account are omitted.
func (h *Handler) InternalUsersBatch(w http.ResponseWriter, r *http.Request) {
	var req request.InternalUsersBatch
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.UserIDs) == 0 {
		writeError(w, http.StatusBadRequest, "userIds is required")
		return
	}
	if len(req.UserIDs) > maxInternalUserBatch {
		writeError(w, http.StatusBadRequest, "userIds exceeds maximum batch size")
		return
	}

	ids := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, raw := range req.UserIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			writeError(w, http.StatusBadRequest, "userIds must be valid uuids")
			return
		}
		ids = append(ids, id)
	}

	users, err := h.svc.UsersByIDs(r.Context(), ids)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewInternalUsersBatch(users))
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	value := strings.TrimSpace(header)
	if len(value) <= len(prefix) || strings.ToLower(value[:len(prefix)]) != prefix {
		return "", false
	}
	token := strings.TrimSpace(value[len(prefix):])
	return token, token != ""
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, apperrs.ErrEmailAlreadyExists):
		writeError(w, http.StatusConflict, "email already exists")
	case errors.Is(err, apperrs.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, apperrs.ErrInvalidAccessToken), errors.Is(err, apperrs.ErrInvalidRefreshToken):
		writeError(w, http.StatusUnauthorized, "invalid token")
	case errors.Is(err, domainerrs.ErrNotFound):
		writeError(w, http.StatusNotFound, "registered user not found")
	default:
		h.log.Error("unhandled auth service error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}
