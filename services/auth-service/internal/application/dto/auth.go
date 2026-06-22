package dto

import "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"

// RegisterInput is the application-level representation of a registration request.
type RegisterInput struct {
	Email    string
	Password string
}

// LoginInput is the application-level representation of a login request.
type LoginInput struct {
	Email    string
	Password string
}

// RefreshInput is the application-level representation of a token refresh request.
type RefreshInput struct {
	RefreshToken string
}

// LogoutInput is the application-level representation of a logout request.
type LogoutInput struct {
	RefreshToken string
}

// AuthResult contains the authenticated user and newly issued tokens.
type AuthResult struct {
	User         entity.User
	AccessToken  string
	RefreshToken string
}

// TokenPair contains a new access token and refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
