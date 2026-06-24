package sharing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	publicShareTokenType = "public_share"
	publicShareAudience  = "public-trip-share"
	publicShareIssuer    = "trip-service"
)

var ErrInvalidPublicShareAccessToken = errors.New("invalid public share access token")

type PublicShareTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type PublicShareClaims struct {
	Type       string `json:"typ"`
	ShareToken string `json:"shareToken"`
	Audience   string `json:"aud"`
	Issuer     string `json:"iss"`
	ExpiresAt  int64  `json:"exp"`
	IssuedAt   int64  `json:"iat"`
}

func NewPublicShareTokenManager(secret string, ttl time.Duration) *PublicShareTokenManager {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &PublicShareTokenManager{
		secret: []byte(secret),
		ttl:    ttl,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (m *PublicShareTokenManager) CreatePublicShareAccessToken(shareToken string) (string, time.Time, error) {
	token := strings.TrimSpace(shareToken)
	if token == "" {
		return "", time.Time{}, ErrInvalidPublicShareAccessToken
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(PublicShareClaims{
		Type:       publicShareTokenType,
		ShareToken: token,
		Audience:   publicShareAudience,
		Issuer:     publicShareIssuer,
		IssuedAt:   now.Unix(),
		ExpiresAt:  expiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("marshal public share claims: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := m.sign(header, payload)
	return header + "." + payload + "." + signature, expiresAt, nil
}

func (m *PublicShareTokenManager) ValidatePublicShareAccessToken(raw string, expectedShareToken string) error {
	token := strings.TrimSpace(raw)
	expected := strings.TrimSpace(expectedShareToken)
	if token == "" || expected == "" {
		return ErrInvalidPublicShareAccessToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrInvalidPublicShareAccessToken
	}

	header, err := decodeSegment(parts[0])
	if err != nil || !isHS256(header) {
		return ErrInvalidPublicShareAccessToken
	}
	if !m.validSignature(parts[0], parts[1], parts[2]) {
		return ErrInvalidPublicShareAccessToken
	}

	payload, err := decodeSegment(parts[1])
	if err != nil {
		return ErrInvalidPublicShareAccessToken
	}
	var claims PublicShareClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ErrInvalidPublicShareAccessToken
	}
	if claims.Type != publicShareTokenType ||
		claims.ShareToken != expected ||
		claims.Audience != publicShareAudience ||
		claims.Issuer != publicShareIssuer ||
		claims.ExpiresAt <= 0 ||
		!time.Unix(claims.ExpiresAt, 0).UTC().After(m.now().UTC()) {
		return ErrInvalidPublicShareAccessToken
	}
	return nil
}

func (m *PublicShareTokenManager) sign(header, payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(header + "." + payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *PublicShareTokenManager) validSignature(header, payload, signature string) bool {
	got, err := decodeSegment(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(header + "." + payload))
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
