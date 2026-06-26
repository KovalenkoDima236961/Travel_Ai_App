package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenValidator struct {
	secret []byte
	now    func() time.Time
}

func NewTokenValidator(secret string) *TokenValidator {
	return &TokenValidator{
		secret: []byte(secret),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (v *TokenValidator) ValidateAccessToken(raw string) (AuthenticatedUser, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return AuthenticatedUser{}, ErrInvalidToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AuthenticatedUser{}, ErrInvalidToken
	}

	header, err := decodeSegment(parts[0])
	if err != nil {
		return AuthenticatedUser{}, ErrInvalidToken
	}
	if !isHS256(header) {
		return AuthenticatedUser{}, ErrInvalidToken
	}
	if !v.validSignature(parts[0], parts[1], parts[2]) {
		return AuthenticatedUser{}, ErrInvalidToken
	}

	payload, err := decodeSegment(parts[1])
	if err != nil {
		return AuthenticatedUser{}, ErrInvalidToken
	}
	claims, err := parseAccessClaims(payload)
	if err != nil {
		return AuthenticatedUser{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" || !claims.ExpiresAt.After(v.now()) {
		return AuthenticatedUser{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return AuthenticatedUser{}, ErrInvalidToken
	}

	return AuthenticatedUser{ID: userID, Email: claims.Email}, nil
}

func (v *TokenValidator) validSignature(header, payload, signature string) bool {
	got, err := decodeSegment(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(fmt.Sprintf("%s.%s", header, payload)))
	want := mac.Sum(nil)
	return hmac.Equal(got, want)
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}

func isHS256(header []byte) bool {
	var parsed struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(header, &parsed); err != nil {
		return false
	}
	return parsed.Alg == "HS256"
}
