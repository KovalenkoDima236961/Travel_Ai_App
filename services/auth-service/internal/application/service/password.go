package service

import (
	"strings"
	"unicode"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, passwordHash string) bool
}

type BcryptPasswordHasher struct{}

func NewPasswordHasher() PasswordHasher {
	return BcryptPasswordHasher{}
}

func (BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (BcryptPasswordHasher) Verify(password, passwordHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func ValidatePasswordStrength(password string) error {
	if strings.TrimSpace(password) == "" {
		return apperrs.NewInvalidInput("password is required")
	}
	if len(password) < 8 {
		return apperrs.NewInvalidInput("password must be at least 8 characters")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return apperrs.NewInvalidInput("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}
