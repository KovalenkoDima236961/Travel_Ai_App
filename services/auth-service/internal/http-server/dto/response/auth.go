package response

import (
	"time"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
)

// User is the public user payload.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// Auth contains a user and token pair.
type Auth struct {
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// Token contains a rotated token pair.
type Token struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// Logout acknowledges logout completion.
type Logout struct {
	Success bool `json:"success"`
}

func NewUser(user *entity.User) User {
	return User{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func NewAuth(result *appdto.AuthResult) Auth {
	return Auth{
		User:         NewUser(&result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
}

func NewToken(pair *appdto.TokenPair) Token {
	return Token{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}
}

func NewLogout() Logout {
	return Logout{Success: true}
}
