package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/http-server/dto/response"
)

type stubAuthService struct {
	register func(context.Context, appdto.RegisterInput) (*appdto.AuthResult, error)
	login    func(context.Context, appdto.LoginInput) (*appdto.AuthResult, error)
	refresh  func(context.Context, appdto.RefreshInput) (*appdto.TokenPair, error)
	logout   func(context.Context, appdto.LogoutInput) error
	me       func(context.Context, string) (*entity.User, error)
	byEmail  func(context.Context, string) (*entity.User, error)
}

func (s stubAuthService) Register(ctx context.Context, req appdto.RegisterInput) (*appdto.AuthResult, error) {
	return s.register(ctx, req)
}

func (s stubAuthService) Login(ctx context.Context, req appdto.LoginInput) (*appdto.AuthResult, error) {
	return s.login(ctx, req)
}

func (s stubAuthService) Refresh(ctx context.Context, req appdto.RefreshInput) (*appdto.TokenPair, error) {
	return s.refresh(ctx, req)
}

func (s stubAuthService) Logout(ctx context.Context, req appdto.LogoutInput) error {
	return s.logout(ctx, req)
}

func (s stubAuthService) CurrentUser(ctx context.Context, accessToken string) (*entity.User, error) {
	return s.me(ctx, accessToken)
}

func (s stubAuthService) UserByEmail(ctx context.Context, email string) (*entity.User, error) {
	return s.byEmail(ctx, email)
}

func newTestRouter(svc stubAuthService) http.Handler {
	r := chi.NewRouter()
	New(svc, zap.NewNop()).RegisterRoutes(r)
	return r
}

func TestHandlerRegisterDuplicateReturnsConflict(t *testing.T) {
	router := newTestRouter(stubAuthService{
		register: func(context.Context, appdto.RegisterInput) (*appdto.AuthResult, error) {
			return nil, apperrs.ErrEmailAlreadyExists
		},
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodPost,
		"/auth/register",
		bytes.NewReader([]byte(`{"email":"user@example.com","password":"StrongPassword123!"}`)),
	))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected register 409, got %d: %s", rec.Code, rec.Body.String())
	}

	var errBody errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errBody.Error != "email already exists" {
		t.Fatalf("unexpected error body: %+v", errBody)
	}
}

func TestHandlerMeReturnsCurrentUser(t *testing.T) {
	router := newTestRouter(stubAuthService{
		me: func(_ context.Context, accessToken string) (*entity.User, error) {
			if accessToken != "access-token" {
				return nil, apperrs.ErrInvalidAccessToken
			}
			return &entity.User{
				ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				Email:     "user@example.com",
				CreatedAt: time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /auth/me 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var user response.User
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode user response: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %s", user.Email)
	}
}

func TestHandlerInvalidLoginReturnsUnauthorized(t *testing.T) {
	router := newTestRouter(stubAuthService{
		login: func(context.Context, appdto.LoginInput) (*appdto.AuthResult, error) {
			return nil, apperrs.ErrInvalidCredentials
		},
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewReader([]byte(`{"email":"user@example.com","password":"WrongPassword123!"}`)),
	))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected login 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerInvalidJSONErrorShape(t *testing.T) {
	router := newTestRouter(stubAuthService{})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{`))))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var errBody errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errBody.Error == "" {
		t.Fatal("expected error message")
	}
}
