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
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/http-server/middleware"
)

type stubAuthService struct {
	register   func(context.Context, appdto.RegisterInput) (*appdto.AuthResult, error)
	login      func(context.Context, appdto.LoginInput) (*appdto.AuthResult, error)
	refresh    func(context.Context, appdto.RefreshInput) (*appdto.TokenPair, error)
	logout     func(context.Context, appdto.LogoutInput) error
	me         func(context.Context, string) (*entity.User, error)
	byEmail    func(context.Context, string) (*entity.User, error)
	usersByIDs func(context.Context, []uuid.UUID) ([]*entity.User, error)
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

func (s stubAuthService) UsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	return s.usersByIDs(ctx, ids)
}

func newTestRouter(svc stubAuthService) http.Handler {
	r := chi.NewRouter()
	New(svc, zap.NewNop()).RegisterRoutes(r)
	return r
}

const testInternalToken = "test-internal-token"

// newInternalTestRouter mounts the internal routes behind the internal
// service-token middleware, mirroring how NewRouter wires them.
func newInternalTestRouter(svc stubAuthService) http.Handler {
	r := chi.NewRouter()
	h := New(svc, zap.NewNop())
	r.Group(func(r chi.Router) {
		r.Use(middleware.InternalServiceToken(testInternalToken))
		h.RegisterInternalRoutes(r)
	})
	return r
}

func TestInternalUsersBatchRequiresToken(t *testing.T) {
	router := newInternalTestRouter(stubAuthService{})
	body := []byte(`{"userIds":["11111111-1111-1111-1111-111111111111"]}`)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/users/batch", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without internal token, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/users/batch", bytes.NewReader(body))
	req.Header.Set(middleware.InternalServiceTokenHeader, "wrong-token")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong internal token, got %d", rec.Code)
	}
}

func TestInternalUsersBatchReturnsOnlyExistingUsers(t *testing.T) {
	present := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	absent := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	router := newInternalTestRouter(stubAuthService{
		usersByIDs: func(_ context.Context, ids []uuid.UUID) ([]*entity.User, error) {
			users := make([]*entity.User, 0, len(ids))
			for _, id := range ids {
				if id == present {
					users = append(users, &entity.User{ID: present, Email: "anna@example.com"})
				}
			}
			return users, nil
		},
	})

	body := []byte(`{"userIds":["` + present.String() + `","` + absent.String() + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/users/batch", bytes.NewReader(body))
	req.Header.Set(middleware.InternalServiceTokenHeader, testInternalToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp response.InternalUsersBatch
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode batch response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 resolved user (absent omitted), got %+v", resp.Items)
	}
	if resp.Items[0].Email != "anna@example.com" || resp.Items[0].UserID != present.String() {
		t.Fatalf("unexpected item: %+v", resp.Items[0])
	}
}

func TestInternalUsersBatchRejectsInvalidUUID(t *testing.T) {
	router := newInternalTestRouter(stubAuthService{})
	body := []byte(`{"userIds":["not-a-uuid"]}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/users/batch", bytes.NewReader(body))
	req.Header.Set(middleware.InternalServiceTokenHeader, testInternalToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid uuid, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInternalUsersBatchRejectsEmpty(t *testing.T) {
	router := newInternalTestRouter(stubAuthService{})
	body := []byte(`{"userIds":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/users/batch", bytes.NewReader(body))
	req.Header.Set(middleware.InternalServiceTokenHeader, testInternalToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty userIds, got %d: %s", rec.Code, rec.Body.String())
	}
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
