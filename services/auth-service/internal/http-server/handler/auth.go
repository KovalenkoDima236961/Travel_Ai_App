package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/http-server/dto/response"
)

type authService interface {
	Register(ctx context.Context, in appdto.RegisterInput) (*appdto.AuthResult, error)
	Login(ctx context.Context, in appdto.LoginInput) (*appdto.AuthResult, error)
	Refresh(ctx context.Context, in appdto.RefreshInput) (*appdto.TokenPair, error)
	Logout(ctx context.Context, in appdto.LogoutInput) error
	CurrentUser(ctx context.Context, accessToken string) (*entity.User, error)
	UserByEmail(ctx context.Context, email string) (*entity.User, error)
}

// Handler wires the auth use case to HTTP.
type Handler struct {
	svc authService
	log *zap.Logger
}

// New constructs the auth HTTP handler.
func New(svc authService, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{svc: svc, log: log}
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
	r.Get("/internal/users/by-email", h.InternalUserByEmail)
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req request.Register
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.Register(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, response.NewAuth(resp))
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.Login
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.Login(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewAuth(resp))
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req request.Refresh
	if !decodeJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewToken(resp))
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req request.Logout
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := h.svc.Logout(r.Context(), req.ToInput()); err != nil {
		h.writeServiceError(w, err)
		return
	}

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

// InternalUserByEmail handles exact registered-user lookup for service-to-service
// calls. TODO: protect this route with internal service auth before exposing
// auth-service outside the private service network.
func (h *Handler) InternalUserByEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	user, err := h.svc.UserByEmail(r.Context(), email)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewInternalUserLookup(user))
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
