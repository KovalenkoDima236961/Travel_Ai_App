package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
)

type testHasher struct{}

func (testHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (testHasher) Verify(password, passwordHash string) bool {
	return passwordHash == "hashed:"+password
}

type memoryRepository struct {
	mu            sync.Mutex
	now           func() time.Time
	usersByID     map[uuid.UUID]entity.User
	usersByEmail  map[string]entity.User
	tokensByHash  map[string]entity.RefreshToken
	createCounter int
}

func newMemoryRepository(now func() time.Time) *memoryRepository {
	return &memoryRepository{
		now:          now,
		usersByID:    map[uuid.UUID]entity.User{},
		usersByEmail: map[string]entity.User{},
		tokensByHash: map[string]entity.RefreshToken{},
	}
}

func (r *memoryRepository) CreateUser(_ context.Context, user *entity.User) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByEmail[user.Email]; exists {
		return nil, domainerrs.ErrAlreadyExists
	}

	now := r.now()
	created := *user
	created.CreatedAt = now
	created.UpdatedAt = now
	r.usersByID[created.ID] = created
	r.usersByEmail[created.Email] = created
	r.createCounter++
	return &created, nil
}

func (r *memoryRepository) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.usersByEmail[email]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &user, nil
}

func (r *memoryRepository) GetUserByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.usersByID[id]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &user, nil
}

func (r *memoryRepository) CreateRefreshToken(_ context.Context, token *entity.RefreshToken) (*entity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tokensByHash[token.TokenHash]; exists {
		return nil, fmt.Errorf("duplicate token")
	}
	created := *token
	created.CreatedAt = r.now()
	r.tokensByHash[created.TokenHash] = created
	return &created, nil
}

func (r *memoryRepository) GetRefreshTokenByHash(_ context.Context, tokenHash string) (*entity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.tokensByHash[tokenHash]
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	return &token, nil
}

func (r *memoryRepository) RotateRefreshToken(
	_ context.Context,
	oldTokenID uuid.UUID,
	revokedAt time.Time,
	newToken *entity.RefreshToken,
) (*entity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var oldHash string
	var old entity.RefreshToken
	for hash, token := range r.tokensByHash {
		if token.ID == oldTokenID {
			oldHash = hash
			old = token
			break
		}
	}
	if oldHash == "" {
		return nil, domainerrs.ErrNotFound
	}

	old.RevokedAt = &revokedAt
	r.tokensByHash[oldHash] = old

	created := *newToken
	created.CreatedAt = r.now()
	r.tokensByHash[created.TokenHash] = created
	return &created, nil
}

func (r *memoryRepository) RevokeRefreshTokenByHash(_ context.Context, tokenHash string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.tokensByHash[tokenHash]
	if !ok {
		return nil
	}
	if token.RevokedAt == nil {
		token.RevokedAt = &revokedAt
		r.tokensByHash[tokenHash] = token
	}
	return nil
}

func (r *memoryRepository) tokenByRaw(raw string) (entity.RefreshToken, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.tokensByHash[HashRefreshToken(raw)]
	return token, ok
}

func (r *memoryRepository) expireRefreshToken(raw string, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash := HashRefreshToken(raw)
	token := r.tokensByHash[hash]
	token.ExpiresAt = at
	r.tokensByHash[hash] = token
}

func newTestService() (*Service, *memoryRepository) {
	now := time.Now().UTC().Truncate(time.Second)
	repo := newMemoryRepository(func() time.Time { return now })
	tokens := NewTokenManager("test-secret-that-is-long-enough", 15*time.Minute, 30*24*time.Hour)
	tokens.now = func() time.Time { return now }
	svc := New(repo, testHasher{}, tokens, zap.NewNop())
	svc.now = func() time.Time { return now }
	return svc, repo
}

func TestServiceRegisterSuccess(t *testing.T) {
	svc, repo := newTestService()

	resp, err := svc.Register(context.Background(), appdto.RegisterInput{
		Email:    " USER@Example.COM ",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	if resp.User.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", resp.User.Email)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	user, err := repo.GetUserByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("expected user in repo: %v", err)
	}
	if user.PasswordHash == "StrongPassword123!" {
		t.Fatal("plaintext password was stored")
	}

	storedToken, ok := repo.tokenByRaw(resp.RefreshToken)
	if !ok {
		t.Fatal("refresh token hash was not stored")
	}
	if storedToken.TokenHash == resp.RefreshToken {
		t.Fatal("raw refresh token was stored")
	}
}

func TestServiceRegisterDuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	req := appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"}

	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("first register returned error: %v", err)
	}
	_, err := svc.Register(context.Background(), req)
	if !errors.Is(err, apperrs.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestServiceRegisterValidation(t *testing.T) {
	tests := []struct {
		name string
		req  appdto.RegisterInput
	}{
		{name: "invalid email", req: appdto.RegisterInput{Email: "not-an-email", Password: "StrongPassword123!"}},
		{name: "weak password", req: appdto.RegisterInput{Email: "user@example.com", Password: "weakpass"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()
			_, err := svc.Register(context.Background(), tt.req)
			var validationErr *apperrs.InvalidInputError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestServiceLogin(t *testing.T) {
	svc, _ := newTestService()
	req := appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"}
	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	resp, err := svc.Login(context.Background(), appdto.LoginInput{Email: "USER@example.com", Password: "StrongPassword123!"})
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
}

func TestServiceLoginInvalidCredentials(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.Register(context.Background(), appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"}); err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	tests := []appdto.LoginInput{
		{Email: "user@example.com", Password: "WrongPassword123!"},
		{Email: "unknown@example.com", Password: "StrongPassword123!"},
	}
	for _, req := range tests {
		_, err := svc.Login(context.Background(), req)
		if !errors.Is(err, apperrs.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials for %+v, got %v", req, err)
		}
	}
}

func TestServiceRefreshRotatesToken(t *testing.T) {
	svc, repo := newTestService()
	authResp, err := svc.Register(context.Background(), appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"})
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	tokenResp, err := svc.Refresh(context.Background(), appdto.RefreshInput{RefreshToken: authResp.RefreshToken})
	if err != nil {
		t.Fatalf("refresh returned error: %v", err)
	}
	if tokenResp.RefreshToken == "" || tokenResp.RefreshToken == authResp.RefreshToken {
		t.Fatal("expected rotated refresh token")
	}

	oldToken, ok := repo.tokenByRaw(authResp.RefreshToken)
	if !ok || oldToken.RevokedAt == nil {
		t.Fatal("expected old token to be revoked")
	}
	if _, ok := repo.tokenByRaw(tokenResp.RefreshToken); !ok {
		t.Fatal("expected new refresh token hash to be stored")
	}

	_, err = svc.Refresh(context.Background(), appdto.RefreshInput{RefreshToken: authResp.RefreshToken})
	if !errors.Is(err, apperrs.ErrInvalidRefreshToken) {
		t.Fatalf("expected old refresh token to be invalid after rotation, got %v", err)
	}
}

func TestServiceRefreshInvalidStates(t *testing.T) {
	t.Run("revoked token", func(t *testing.T) {
		svc, _ := newTestService()
		authResp, err := svc.Register(context.Background(), appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"})
		if err != nil {
			t.Fatalf("register returned error: %v", err)
		}
		if err := svc.Logout(context.Background(), appdto.LogoutInput{RefreshToken: authResp.RefreshToken}); err != nil {
			t.Fatalf("logout returned error: %v", err)
		}
		_, err = svc.Refresh(context.Background(), appdto.RefreshInput{RefreshToken: authResp.RefreshToken})
		if !errors.Is(err, apperrs.ErrInvalidRefreshToken) {
			t.Fatalf("expected invalid refresh token, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		svc, repo := newTestService()
		authResp, err := svc.Register(context.Background(), appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"})
		if err != nil {
			t.Fatalf("register returned error: %v", err)
		}
		repo.expireRefreshToken(authResp.RefreshToken, svc.now().Add(-time.Minute))
		_, err = svc.Refresh(context.Background(), appdto.RefreshInput{RefreshToken: authResp.RefreshToken})
		if !errors.Is(err, apperrs.ErrInvalidRefreshToken) {
			t.Fatalf("expected invalid refresh token, got %v", err)
		}
	})
}

func TestServiceLogoutAlwaysSucceedsForUnknownToken(t *testing.T) {
	svc, _ := newTestService()

	if err := svc.Logout(context.Background(), appdto.LogoutInput{RefreshToken: "unknown-token"}); err != nil {
		t.Fatalf("logout returned error: %v", err)
	}
}

func TestServiceCurrentUser(t *testing.T) {
	svc, _ := newTestService()
	authResp, err := svc.Register(context.Background(), appdto.RegisterInput{Email: "user@example.com", Password: "StrongPassword123!"})
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}

	user, err := svc.CurrentUser(context.Background(), authResp.AccessToken)
	if err != nil {
		t.Fatalf("current user returned error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %s", user.Email)
	}
}
