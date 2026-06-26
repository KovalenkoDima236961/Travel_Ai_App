package calendar

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	tokencrypto "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/crypto"
)

type Repository interface {
	UpsertCalendarConnection(ctx context.Context, conn CalendarConnection) (*CalendarConnection, error)
	GetCalendarConnectionByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*CalendarConnection, error)
	GetActiveCalendarConnection(ctx context.Context, userID uuid.UUID, provider string) (*CalendarConnection, error)
	DisconnectCalendarConnection(ctx context.Context, userID uuid.UUID, provider string) error
	UpdateCalendarTokens(ctx context.Context, userID uuid.UUID, provider, accessTokenEncrypted string, refreshTokenEncrypted *string, expiresAt *time.Time, scopes string) error
	CreateOAuthState(ctx context.Context, state OAuthState) error
	GetOAuthState(ctx context.Context, state string) (*OAuthState, error)
	MarkOAuthStateUsed(ctx context.Context, state string) (bool, error)
	DeleteExpiredOAuthStates(ctx context.Context, now time.Time) error
}

type Service struct {
	repo             Repository
	provider         CalendarProvider
	cipher           *tokencrypto.StringCipher
	stateTTL         time.Duration
	publicWebBaseURL string
	defaultTimeZone  string
	enabled          bool
	log              *zap.Logger
	now              func() time.Time
}

type Config struct {
	Enabled          bool
	StateTTL         time.Duration
	PublicWebBaseURL string
	DefaultTimeZone  string
}

func NewService(repo Repository, provider CalendarProvider, cipher *tokencrypto.StringCipher, cfg Config, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:             repo,
		provider:         provider,
		cipher:           cipher,
		stateTTL:         cfg.StateTTL,
		publicWebBaseURL: strings.TrimRight(strings.TrimSpace(cfg.PublicWebBaseURL), "/"),
		defaultTimeZone:  strings.TrimSpace(cfg.DefaultTimeZone),
		enabled:          cfg.Enabled,
		log:              log,
		now:              func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Status(ctx context.Context, userID uuid.UUID) (ConnectionStatus, error) {
	if !s.enabled {
		return ConnectionStatus{Connected: false, Provider: ProviderGoogle}, nil
	}
	conn, err := s.repo.GetActiveCalendarConnection(ctx, userID, ProviderGoogle)
	if err != nil {
		if errors.Is(err, ErrCalendarNotConnected) {
			return ConnectionStatus{Connected: false, Provider: ProviderGoogle}, nil
		}
		return ConnectionStatus{}, err
	}
	return ConnectionStatus{
		Connected:            true,
		Provider:             ProviderGoogle,
		ProviderAccountEmail: conn.ProviderAccountEmail,
		ConnectedAt:          &conn.ConnectedAt,
		Scopes:               conn.Scopes,
	}, nil
}

func (s *Service) StartConnect(ctx context.Context, userID uuid.UUID, returnURL string) (string, error) {
	if !s.enabled {
		return "", ErrCalendarDisabled
	}
	state, err := randomState()
	if err != nil {
		return "", err
	}
	safeReturn := s.safeReturnURL(returnURL)
	stateRow := OAuthState{
		State:     state,
		UserID:    userID,
		Provider:  ProviderGoogle,
		ReturnURL: &safeReturn,
		ExpiresAt: s.now().Add(s.stateTTL),
	}
	if err := s.repo.CreateOAuthState(ctx, stateRow); err != nil {
		return "", err
	}
	return s.provider.BuildAuthURL(ctx, state)
}

func (s *Service) HandleCallback(ctx context.Context, code, state, googleError string) (string, error) {
	row, err := s.repo.GetOAuthState(ctx, strings.TrimSpace(state))
	if err != nil {
		return s.withStatusParam(s.defaultReturnURL(), "calendar_error", "invalid_state"), ErrInvalidOAuthState
	}
	returnURL := s.safeReturnURL(valueOrEmpty(row.ReturnURL))
	if row.Provider != ProviderGoogle || row.UsedAt != nil || !row.ExpiresAt.After(s.now()) {
		return s.withStatusParam(returnURL, "calendar_error", "invalid_state"), ErrInvalidOAuthState
	}
	if googleError != "" {
		used, markErr := s.repo.MarkOAuthStateUsed(ctx, row.State)
		if markErr != nil {
			return s.withStatusParam(returnURL, "calendar_error", "state_update_failed"), markErr
		}
		if !used {
			return s.withStatusParam(returnURL, "calendar_error", "invalid_state"), ErrInvalidOAuthState
		}
		return s.withStatusParam(returnURL, "calendar_error", "access_denied"), nil
	}
	if strings.TrimSpace(code) == "" {
		return s.withStatusParam(returnURL, "calendar_error", "missing_code"), nil
	}

	used, err := s.repo.MarkOAuthStateUsed(ctx, row.State)
	if err != nil {
		return s.withStatusParam(returnURL, "calendar_error", "state_update_failed"), err
	}
	if !used {
		return s.withStatusParam(returnURL, "calendar_error", "invalid_state"), ErrInvalidOAuthState
	}

	token, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return s.withStatusParam(returnURL, "calendar_error", "token_exchange_failed"), err
	}
	account, err := s.provider.GetAccountInfo(ctx, token.AccessToken)
	if err != nil {
		s.log.Warn("google account lookup failed", zap.Error(err))
	}
	accessEncrypted, err := s.cipher.EncryptString(token.AccessToken)
	if err != nil {
		return s.withStatusParam(returnURL, "calendar_error", "token_encrypt_failed"), err
	}
	var refreshEncrypted *string
	if strings.TrimSpace(token.RefreshToken) != "" {
		encrypted, err := s.cipher.EncryptString(token.RefreshToken)
		if err != nil {
			return s.withStatusParam(returnURL, "calendar_error", "token_encrypt_failed"), err
		}
		refreshEncrypted = &encrypted
	}
	var email *string
	if account != nil && strings.TrimSpace(account.Email) != "" {
		emailValue := strings.TrimSpace(account.Email)
		email = &emailValue
	}
	scopes := token.Scopes
	if strings.TrimSpace(scopes) == "" {
		scopes = "https://www.googleapis.com/auth/calendar.events"
	}
	_, err = s.repo.UpsertCalendarConnection(ctx, CalendarConnection{
		ID:                    uuid.New(),
		UserID:                row.UserID,
		Provider:              ProviderGoogle,
		ProviderAccountEmail:  email,
		AccessTokenEncrypted:  accessEncrypted,
		RefreshTokenEncrypted: refreshEncrypted,
		TokenExpiresAt:        token.ExpiresAt,
		Scopes:                &scopes,
		Status:                "active",
	})
	if err != nil {
		return s.withStatusParam(returnURL, "calendar_error", "store_failed"), err
	}
	return s.withStatusParam(returnURL, "calendar_connected", "1"), nil
}

func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	if !s.enabled {
		return ErrCalendarDisabled
	}
	return s.repo.DisconnectCalendarConnection(ctx, userID, ProviderGoogle)
}

func (s *Service) SyncEvents(ctx context.Context, req SyncEventsRequest) (*SyncEventsResponse, error) {
	if !s.enabled {
		return nil, ErrCalendarDisabled
	}
	accessToken, err := s.accessToken(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	timeZone := strings.TrimSpace(req.TimeZone)
	if timeZone == "" {
		timeZone = s.defaultTimeZone
	}
	out := &SyncEventsResponse{Items: make([]SyncEventItemResult, 0, len(req.Items))}
	for _, item := range req.Items {
		input := CalendarEventInput{
			Title:       item.Title,
			Description: item.Description,
			Location:    item.Location,
			Start:       item.Start,
			End:         item.End,
			TimeZone:    timeZone,
		}
		result := SyncEventItemResult{
			SyncKey:   item.SyncKey,
			DayNumber: item.DayNumber,
			ItemIndex: item.ItemIndex,
		}
		var providerResult *CalendarEventResult
		var providerErr error
		if strings.TrimSpace(item.ExistingEventID) != "" {
			calendarID := item.ExistingCalendarID
			if strings.TrimSpace(calendarID) == "" {
				calendarID = "primary"
			}
			providerResult, providerErr = s.provider.UpdateEvent(ctx, accessToken, calendarID, item.ExistingEventID, input)
			result.Status = "updated"
		} else {
			providerResult, providerErr = s.provider.CreateEvent(ctx, accessToken, input)
			result.Status = "created"
		}
		if providerErr != nil {
			result.Status = "failed"
			result.Error = "provider_error"
			s.log.Warn("calendar event sync failed", zap.String("sync_key", item.SyncKey), zap.Error(providerErr))
		} else if providerResult != nil {
			result.CalendarID = providerResult.CalendarID
			result.EventID = providerResult.EventID
			result.HtmlLink = providerResult.HtmlLink
		}
		out.Items = append(out.Items, result)
	}
	return out, nil
}

func (s *Service) DeleteEvents(ctx context.Context, req DeleteEventsRequest) (*DeleteEventsResponse, error) {
	if !s.enabled {
		return nil, ErrCalendarDisabled
	}
	accessToken, err := s.accessToken(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	out := &DeleteEventsResponse{}
	for _, event := range req.Events {
		calendarID := event.CalendarID
		if strings.TrimSpace(calendarID) == "" {
			calendarID = "primary"
		}
		if err := s.provider.DeleteEvent(ctx, accessToken, calendarID, event.EventID); err != nil {
			out.Failed++
			s.log.Warn("calendar event delete failed", zap.Error(err))
			continue
		}
		out.Deleted++
	}
	return out, nil
}

func (s *Service) accessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	conn, err := s.repo.GetActiveCalendarConnection(ctx, userID, ProviderGoogle)
	if err != nil {
		return "", err
	}
	if conn.TokenExpiresAt != nil && conn.TokenExpiresAt.After(s.now().Add(time.Minute)) {
		return s.cipher.DecryptString(conn.AccessTokenEncrypted)
	}
	if conn.RefreshTokenEncrypted == nil || strings.TrimSpace(*conn.RefreshTokenEncrypted) == "" {
		_ = s.repo.DisconnectCalendarConnection(ctx, userID, ProviderGoogle)
		return "", ErrCalendarReauthRequired
	}
	refreshToken, err := s.cipher.DecryptString(*conn.RefreshTokenEncrypted)
	if err != nil {
		return "", ErrCalendarReauthRequired
	}
	token, err := s.provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		_ = s.repo.DisconnectCalendarConnection(ctx, userID, ProviderGoogle)
		return "", ErrCalendarReauthRequired
	}
	accessEncrypted, err := s.cipher.EncryptString(token.AccessToken)
	if err != nil {
		return "", err
	}
	var refreshEncrypted *string
	if strings.TrimSpace(token.RefreshToken) != "" {
		value, err := s.cipher.EncryptString(token.RefreshToken)
		if err != nil {
			return "", err
		}
		refreshEncrypted = &value
	}
	if err := s.repo.UpdateCalendarTokens(ctx, userID, ProviderGoogle, accessEncrypted, refreshEncrypted, token.ExpiresAt, token.Scopes); err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func randomState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Service) defaultReturnURL() string {
	if s.publicWebBaseURL == "" {
		return "http://localhost:3000/settings"
	}
	return s.publicWebBaseURL + "/settings"
}

func (s *Service) safeReturnURL(raw string) string {
	baseURL := s.publicWebBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "http://localhost:3000/settings"
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return base.ResolveReference(&url.URL{Path: "/settings"}).String()
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return base.ResolveReference(&url.URL{Path: "/settings"}).String()
	}
	if parsed.IsAbs() {
		if parsed.Scheme == base.Scheme && parsed.Host == base.Host {
			return parsed.String()
		}
		return base.ResolveReference(&url.URL{Path: "/settings"}).String()
	}
	if strings.HasPrefix(value, "//") {
		return base.ResolveReference(&url.URL{Path: "/settings"}).String()
	}
	return base.ResolveReference(parsed).String()
}

func (s *Service) withStatusParam(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return s.defaultReturnURL()
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
