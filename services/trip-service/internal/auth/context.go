package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// AuthenticatedUser is the identity extracted from a validated access token.
type AuthenticatedUser struct {
	ID          uuid.UUID
	Email       string
	AccessToken string
}

type contextKey struct{}

// WithUser stores an authenticated user in the request context.
func WithUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// UserFromContext returns the authenticated user, when present.
func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(contextKey{}).(AuthenticatedUser)
	return user, ok
}

// MustUserFromContext returns the authenticated user or an error.
func MustUserFromContext(ctx context.Context) (AuthenticatedUser, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return AuthenticatedUser{}, errors.New("authenticated user missing from context")
	}
	return user, nil
}

// AccessTokenFromContext returns the raw access token captured by auth
// middleware for internal service-to-service forwarding.
func AccessTokenFromContext(ctx context.Context) (string, bool) {
	user, ok := UserFromContext(ctx)
	if !ok || user.AccessToken == "" {
		return "", false
	}
	return user.AccessToken, true
}
