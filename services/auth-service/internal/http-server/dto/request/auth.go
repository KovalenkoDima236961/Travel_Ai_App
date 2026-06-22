package request

import appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"

// Register is the POST /auth/register request body.
type Register struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r Register) ToInput() appdto.RegisterInput {
	return appdto.RegisterInput{
		Email:    r.Email,
		Password: r.Password,
	}
}

// Login is the POST /auth/login request body.
type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r Login) ToInput() appdto.LoginInput {
	return appdto.LoginInput{
		Email:    r.Email,
		Password: r.Password,
	}
}

// Refresh is the POST /auth/refresh request body.
type Refresh struct {
	RefreshToken string `json:"refreshToken"`
}

func (r Refresh) ToInput() appdto.RefreshInput {
	return appdto.RefreshInput{RefreshToken: r.RefreshToken}
}

// Logout is the POST /auth/logout request body.
type Logout struct {
	RefreshToken string `json:"refreshToken"`
}

func (r Logout) ToInput() appdto.LogoutInput {
	return appdto.LogoutInput{RefreshToken: r.RefreshToken}
}
