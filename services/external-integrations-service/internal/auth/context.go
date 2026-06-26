package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type AuthenticatedUser struct {
	ID          uuid.UUID
	Email       string
	AccessToken string
}

type contextKey struct{}

func WithUser(ctx context.Context, user AuthenticatedUser) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

func UserFromContext(ctx context.Context) (AuthenticatedUser, bool) {
	user, ok := ctx.Value(contextKey{}).(AuthenticatedUser)
	return user, ok
}

func MustUserFromContext(ctx context.Context) (AuthenticatedUser, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return AuthenticatedUser{}, errors.New("authenticated user missing from context")
	}
	return user, nil
}
