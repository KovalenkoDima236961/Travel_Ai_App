package sharing

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	minSharePasswordLength = 6
	maxSharePasswordLength = 128
)

func HashSharePassword(password string) (string, error) {
	if err := ValidateSharePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifySharePassword(hash string, password string) bool {
	if strings.TrimSpace(hash) == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func ValidateSharePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}
	if len(password) < minSharePasswordLength {
		return errors.New("password must be at least 6 characters")
	}
	if len(password) > maxSharePasswordLength {
		return errors.New("password must be 128 characters or fewer")
	}
	return nil
}
