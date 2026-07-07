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

type InternalUserLookup struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

// InternalUsersBatch is the response for POST /internal/users/batch. It contains
// only the users that exist; ids with no matching account are omitted so the
// caller can decide how to handle a partial result.
type InternalUsersBatch struct {
	Items []InternalUserLookup `json:"items"`
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

func NewInternalUserLookup(user *entity.User) InternalUserLookup {
	return InternalUserLookup{
		UserID:      user.ID.String(),
		Email:       user.Email,
		DisplayName: "",
	}
}

// NewInternalUsersBatch maps a set of resolved users into the batch response.
// DisplayName is empty in v1 (Auth Service owns email, not profile display
// names); callers fall back to a neutral greeting when it is blank.
func NewInternalUsersBatch(users []*entity.User) InternalUsersBatch {
	items := make([]InternalUserLookup, 0, len(users))
	for _, user := range users {
		items = append(items, NewInternalUserLookup(user))
	}
	return InternalUsersBatch{Items: items}
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
