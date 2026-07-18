package service

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
)

func TestAccessTokenClaims(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	manager := NewTokenManager("test-secret-that-is-long-enough", 15*time.Minute, 30*24*time.Hour)
	manager.now = func() time.Time { return now }
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(entity.User{ID: userID, Email: "user@example.com"})
	if err != nil {
		t.Fatalf("generate access token returned error: %v", err)
	}

	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("validate access token returned error: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Fatalf("expected subject %s, got %s", userID, claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("expected email claim, got %s", claims.Email)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatal("expected iat and exp claims")
	}
}

func TestAccessTokenValidationRejectsInvalidAndExpiredTokens(t *testing.T) {
	manager := NewTokenManager("test-secret-that-is-long-enough", -time.Minute, 30*24*time.Hour)
	token, err := manager.GenerateAccessToken(entity.User{ID: uuid.New(), Email: "user@example.com"})
	if err != nil {
		t.Fatalf("generate access token returned error: %v", err)
	}

	if _, err := manager.ValidateAccessToken(token); !errors.Is(err, apperrs.ErrInvalidAccessToken) {
		t.Fatalf("expected expired token to be invalid, got %v", err)
	}

	if _, err := manager.ValidateAccessToken("not-a-jwt"); !errors.Is(err, apperrs.ErrInvalidAccessToken) {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestAccessTokenValidationRejectsUnexpectedSigningAlgorithm(t *testing.T) {
	const secret = "test-secret-that-is-long-enough"
	manager := NewTokenManager(secret, 15*time.Minute, 30*24*time.Hour)
	claims := AccessClaims{
		Email: "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
		},
	}
	// HS512 is correctly signed with the service secret, but it is not an
	// algorithm this service issues or accepts.
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign HS512 token: %v", err)
	}

	if _, err := manager.ValidateAccessToken(token); !errors.Is(err, apperrs.ErrInvalidAccessToken) {
		t.Fatalf("expected unexpected signing algorithm to be invalid, got %v", err)
	}
}

func TestRefreshTokenHashing(t *testing.T) {
	raw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token returned error: %v", err)
	}
	hash := HashRefreshToken(raw)
	if hash == "" {
		t.Fatal("expected hash")
	}
	if hash == raw {
		t.Fatal("refresh token hash matches raw token")
	}
	if HashRefreshToken(raw) != hash {
		t.Fatal("refresh token hashing is not stable")
	}
}
