package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type accessClaims struct {
	Subject   string
	Email     string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type jwtPayload struct {
	Subject   string      `json:"sub"`
	Email     string      `json:"email"`
	ExpiresAt json.Number `json:"exp"`
	IssuedAt  json.Number `json:"iat"`
}

func parseAccessClaims(payload []byte) (accessClaims, error) {
	var raw jwtPayload
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return accessClaims{}, fmt.Errorf("decode claims: %w", err)
	}

	exp, err := raw.ExpiresAt.Int64()
	if err != nil || exp <= 0 {
		return accessClaims{}, fmt.Errorf("invalid exp claim")
	}

	var issuedAt time.Time
	if raw.IssuedAt != "" {
		iat, err := raw.IssuedAt.Int64()
		if err != nil {
			return accessClaims{}, fmt.Errorf("invalid iat claim")
		}
		issuedAt = time.Unix(iat, 0).UTC()
	}

	return accessClaims{
		Subject:   raw.Subject,
		Email:     raw.Email,
		ExpiresAt: time.Unix(exp, 0).UTC(),
		IssuedAt:  issuedAt,
	}, nil
}
