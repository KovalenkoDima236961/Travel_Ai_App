package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
)

// authRepository is the persistence port the use case depends on. The concrete
// postgres adapter satisfies it; tests substitute a memory implementation.
type authRepository interface {
	CreateUser(ctx context.Context, user *entity.User) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error)
	CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) (*entity.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, revokedAt time.Time, newToken *entity.RefreshToken) (*entity.RefreshToken, error)
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
}

type Service struct {
	repo     authRepository
	password PasswordHasher
	tokens   *TokenManager
	log      *zap.Logger
	now      func() time.Time
}

func New(repo authRepository, password PasswordHasher, tokens *TokenManager, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:     repo,
		password: password,
		tokens:   tokens,
		log:      log,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Register(ctx context.Context, in appdto.RegisterInput) (*appdto.AuthResult, error) {
	email, err := normalizeAndValidateEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if err := ValidatePasswordStrength(in.Password); err != nil {
		return nil, err
	}

	passwordHash, err := s.password.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, &entity.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, domainerrs.ErrAlreadyExists) {
			return nil, apperrs.ErrEmailAlreadyExists
		}
		return nil, err
	}

	resp, err := s.issueAuthResponse(ctx, *user)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) Login(ctx context.Context, in appdto.LoginInput) (*appdto.AuthResult, error) {
	email, err := normalizeAndValidateEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Password) == "" {
		return nil, apperrs.NewInvalidInput("password is required")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, apperrs.ErrInvalidCredentials
		}
		return nil, err
	}

	if !s.password.Verify(in.Password, user.PasswordHash) {
		return nil, apperrs.ErrInvalidCredentials
	}

	resp, err := s.issueAuthResponse(ctx, *user)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) Refresh(ctx context.Context, in appdto.RefreshInput) (*appdto.TokenPair, error) {
	raw := strings.TrimSpace(in.RefreshToken)
	if raw == "" {
		return nil, apperrs.NewInvalidInput("refreshToken is required")
	}

	now := s.now()
	stored, err := s.repo.GetRefreshTokenByHash(ctx, HashRefreshToken(raw))
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, apperrs.ErrInvalidRefreshToken
		}
		return nil, err
	}
	if stored.RevokedAt != nil || !stored.ExpiresAt.After(now) {
		return nil, apperrs.ErrInvalidRefreshToken
	}

	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, apperrs.ErrInvalidRefreshToken
		}
		return nil, err
	}

	newRefreshToken, newRefreshParams, err := s.newRefreshTokenParams(user.ID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.RotateRefreshToken(ctx, stored.ID, now, newRefreshParams); err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, apperrs.ErrInvalidRefreshToken
		}
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(*user)
	if err != nil {
		return nil, err
	}

	return &appdto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, in appdto.LogoutInput) error {
	raw := strings.TrimSpace(in.RefreshToken)
	if raw == "" {
		return apperrs.NewInvalidInput("refreshToken is required")
	}
	if err := s.repo.RevokeRefreshTokenByHash(ctx, HashRefreshToken(raw), s.now()); err != nil {
		return err
	}
	return nil
}

func (s *Service) CurrentUser(ctx context.Context, accessToken string) (*entity.User, error) {
	raw := strings.TrimSpace(accessToken)
	if raw == "" {
		return nil, apperrs.ErrInvalidAccessToken
	}

	claims, err := s.tokens.ValidateAccessToken(raw)
	if err != nil {
		return nil, apperrs.ErrInvalidAccessToken
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, apperrs.ErrInvalidAccessToken
	}

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, apperrs.ErrInvalidAccessToken
		}
		return nil, err
	}

	return user, nil
}

func (s *Service) UserByEmail(ctx context.Context, email string) (*entity.User, error) {
	normalized, err := normalizeAndValidateEmail(email)
	if err != nil {
		return nil, err
	}
	return s.repo.GetUserByEmail(ctx, normalized)
}

// UsersByIDs resolves a set of registered users by id for trusted internal
// callers (e.g. Notification Service resolving recipient emails). It returns
// only the users that exist; ids with no matching account are simply omitted so
// the caller can decide how to handle a partial result. Duplicate ids are
// de-duplicated before the lookup.
func (s *Service) UsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	return s.repo.GetUsersByIDs(ctx, unique)
}

func (s *Service) issueAuthResponse(ctx context.Context, user entity.User) (*appdto.AuthResult, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshParams, err := s.newRefreshTokenParams(user.ID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CreateRefreshToken(ctx, refreshParams); err != nil {
		return nil, err
	}

	return &appdto.AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Service) newRefreshTokenParams(userID uuid.UUID) (string, *entity.RefreshToken, error) {
	raw, err := GenerateRefreshToken()
	if err != nil {
		return "", nil, err
	}
	return raw, &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: HashRefreshToken(raw),
		ExpiresAt: s.now().Add(s.tokens.RefreshTokenTTL()),
	}, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeAndValidateEmail(email string) (string, error) {
	normalized := NormalizeEmail(email)
	if normalized == "" {
		return "", apperrs.NewInvalidInput("email is required")
	}
	addr, err := mail.ParseAddress(normalized)
	if err != nil || addr.Address != normalized {
		return "", apperrs.NewInvalidInput("email must be a valid email address")
	}
	return normalized, nil
}
