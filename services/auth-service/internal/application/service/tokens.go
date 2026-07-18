package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
)

const refreshTokenBytes = 32

type AccessClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	accessSecret []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	now          func() time.Time
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret: []byte(secret),
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (m *TokenManager) GenerateAccessToken(user entity.User) (string, error) {
	now := m.now()
	claims := AccessClaims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (m *TokenManager) ValidateAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		// The service only issues HS256 tokens. Accepting every HMAC variant is
		// unnecessary algorithm agility and makes the verification contract less
		// explicit than the issuing contract.
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}
		return m.accessSecret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, apperrs.ErrInvalidAccessToken
	}
	if claims.Subject == "" || claims.Email == "" {
		return nil, apperrs.ErrInvalidAccessToken
	}
	return claims, nil
}

func (m *TokenManager) RefreshTokenTTL() time.Duration {
	return m.refreshTTL
}

func GenerateRefreshToken() (string, error) {
	var b [refreshTokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
