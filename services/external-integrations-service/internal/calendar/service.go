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
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
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
	repo                 Repository
	provider             CalendarProvider
	cipher               *tokencrypto.StringCipher
	stateTTL             time.Duration
	publicWebBaseURL     string
	defaultTimeZone      string
	enabled              bool
	freeBusyEnabled      bool
	freeBusyMaxRangeDays int
	freeBusyTimeout      time.Duration
	freeBusyPrimaryOnly  bool
	guard                *providerlimits.Guard
	providerName         string
	log                  *zap.Logger
	now                  func() time.Time
}

type Config struct {
	Enabled              bool
	StateTTL             time.Duration
	PublicWebBaseURL     string
	DefaultTimeZone      string
	FreeBusyEnabled      bool
	FreeBusyMaxRangeDays int
	FreeBusyTimeout      time.Duration
	FreeBusyPrimaryOnly  bool
	// ProviderName is the active calendar provider name used for limit metrics
	// and Ops display (e.g. "google" or "mock").
	ProviderName string
}

func NewService(repo Repository, provider CalendarProvider, cipher *tokencrypto.StringCipher, cfg Config, guard *providerlimits.Guard, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:                 repo,
		provider:             provider,
		cipher:               cipher,
		stateTTL:             cfg.StateTTL,
		publicWebBaseURL:     strings.TrimRight(strings.TrimSpace(cfg.PublicWebBaseURL), "/"),
		defaultTimeZone:      strings.TrimSpace(cfg.DefaultTimeZone),
		enabled:              cfg.Enabled,
		freeBusyEnabled:      cfg.FreeBusyEnabled,
		freeBusyMaxRangeDays: cfg.FreeBusyMaxRangeDays,
		freeBusyTimeout:      cfg.FreeBusyTimeout,
		freeBusyPrimaryOnly:  cfg.FreeBusyPrimaryOnly,
		guard:                guard,
		providerName:         strings.TrimSpace(cfg.ProviderName),
		log:                  log,
		now:                  func() time.Time { return time.Now().UTC() },
	}
}

// reserveCalendar applies the provider limit guard for a calendar write. Calendar
// writes never fall back to mock, so a limited call is reported to the caller as
// a controlled failure and the real provider is not called.
func (s *Service) reserveCalendar(ctx context.Context, operation string) (providerlimits.Decision, bool) {
	if s.guard == nil {
		return providerlimits.Decision{Allowed: true}, true
	}
	decision, _ := s.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      s.providerName,
		Operation:     operation,
		Cost:          1,
		AllowFallback: false,
	})
	return decision, decision.Allowed
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
		scopes = "https://www.googleapis.com/auth/calendar.events https://www.googleapis.com/auth/calendar.freebusy"
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

func (s *Service) FreeBusy(ctx context.Context, userID uuid.UUID, req FreeBusyRequest) (*FreeBusyResponse, error) {
	started := time.Now()
	if !s.enabled {
		return nil, ErrCalendarDisabled
	}
	if !s.freeBusyEnabled {
		return nil, ErrCalendarFreeBusyDisabled
	}
	normalized, start, endExclusive, loc, err := s.normalizeFreeBusyRequest(req)
	if err != nil {
		return nil, err
	}
	providerName := s.providerName
	if strings.TrimSpace(providerName) == "" {
		providerName = ProviderGoogle
	}
	if decision, ok := s.reserveCalendar(ctx, providerlimits.OpCalendarFreeBusy); !ok {
		limitErr := providerlimits.LimitErrorFrom(decision)
		if limitErr != nil {
			recordCalendarFreeBusyFailure(providerName, limitErr.Code)
			return nil, limitErr
		}
		recordCalendarFreeBusyFailure(providerName, "provider_rate_limited")
		return nil, ErrCalendarFreeBusyUnavailable
	}

	callCtx := ctx
	cancel := func() {}
	if s.freeBusyTimeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, s.freeBusyTimeout)
	}
	defer cancel()

	accessToken, err := s.accessToken(callCtx, userID)
	if err != nil {
		recordCalendarFreeBusyFailure(providerName, calendarErrorCode(err))
		return nil, err
	}
	blocks, err := s.provider.GetFreeBusy(callCtx, accessToken, ProviderFreeBusyRequest{
		Start:       start,
		End:         endExclusive,
		TimeZone:    normalized.TimeZone,
		CalendarIDs: normalized.CalendarIDs,
	})
	if err != nil {
		code := calendarErrorCode(err)
		if errors.Is(err, ErrCalendarReauthRequired) {
			_ = s.repo.DisconnectCalendarConnection(ctx, userID, ProviderGoogle)
			recordCalendarFreeBusyFailure(providerName, code)
			return nil, err
		}
		recordCalendarFreeBusyFailure(providerName, code)
		if errors.Is(err, ErrCalendarFreeBusyMalformedResponse) {
			return nil, ErrCalendarFreeBusyMalformedResponse
		}
		if errors.Is(err, ErrCalendarFreeBusyUnavailable) {
			return nil, ErrCalendarFreeBusyUnavailable
		}
		return nil, ErrCalendarFreeBusyUnavailable
	}
	for i := range blocks {
		blocks[i].Source = "google_calendar"
	}
	summary := buildFreeBusySummary(normalized, blocks, loc)
	recordCalendarFreeBusyRequest(providerName, "success", time.Since(started))
	recordCalendarFreeBusyBlocks(providerName, len(blocks))
	s.log.Info("calendar_free_busy_completed",
		zap.String("userId", userID.String()),
		zap.String("startDate", normalized.StartDate),
		zap.String("endDate", normalized.EndDate),
		zap.Int("busyBlockCount", len(blocks)),
		zap.Float64("durationMs", float64(time.Since(started).Microseconds())/1000),
	)
	return &FreeBusyResponse{
		BusyBlocks: blocks,
		Summary:    summary,
		Warnings: []string{
			"Only busy/free information was imported. Event details are not stored.",
		},
	}, nil
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
		isUpdate := strings.TrimSpace(item.ExistingEventID) != ""
		operation := providerlimits.OpCalendarEventCreate
		if isUpdate {
			operation = providerlimits.OpCalendarEventUpdate
		}
		// Calendar writes never fall back to mock: a limited call must not
		// pretend an event was created/updated. Report a controlled failure and
		// skip the provider call so we do not hammer a limited provider.
		if decision, ok := s.reserveCalendar(ctx, operation); !ok {
			limitErr := providerlimits.LimitErrorFrom(decision)
			result.Status = "failed"
			result.Error = limitErr.Code
			s.log.Warn("calendar event sync limited",
				zap.String("sync_key", item.SyncKey),
				zap.String("operation", operation),
				zap.String("reason", decision.Reason),
			)
			out.Items = append(out.Items, result)
			continue
		}
		var providerResult *CalendarEventResult
		var providerErr error
		if isUpdate {
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

func (s *Service) normalizeFreeBusyRequest(req FreeBusyRequest) (FreeBusyRequest, time.Time, time.Time, *time.Location, error) {
	timezone := strings.TrimSpace(req.TimeZone)
	if timezone == "" {
		timezone = s.defaultTimeZone
	}
	if timezone == "" {
		timezone = "UTC"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		if s.defaultTimeZone != "" && timezone != s.defaultTimeZone {
			loc, err = time.LoadLocation(s.defaultTimeZone)
			timezone = s.defaultTimeZone
		}
		if err != nil {
			return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyInvalidTimeZone
		}
	}
	start, err := parseCalendarDateInLocation(req.StartDate, loc)
	if err != nil {
		return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyInvalidRange
	}
	end, err := parseCalendarDateInLocation(req.EndDate, loc)
	if err != nil {
		return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyInvalidRange
	}
	if end.Before(start) {
		return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyInvalidRange
	}
	maxRange := s.freeBusyMaxRangeDays
	if maxRange <= 0 {
		maxRange = 180
	}
	rangeDays := int(end.Sub(start).Hours()/24) + 1
	if rangeDays > maxRange {
		return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyRangeTooLarge
	}
	calendarIDs := normalizeCalendarIDs(req.CalendarIDs)
	if len(calendarIDs) > 5 {
		return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyUnsupportedCalendar
	}
	if s.freeBusyPrimaryOnly {
		for _, id := range calendarIDs {
			if id != "primary" {
				return req, time.Time{}, time.Time{}, nil, ErrCalendarFreeBusyUnsupportedCalendar
			}
		}
		calendarIDs = []string{"primary"}
	}
	req.StartDate = start.Format("2006-01-02")
	req.EndDate = end.Format("2006-01-02")
	req.TimeZone = timezone
	req.CalendarIDs = calendarIDs
	return req, start, end.AddDate(0, 0, 1), loc, nil
}

func normalizeCalendarIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{"primary"}
	}
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			trimmed = "primary"
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{"primary"}
	}
	return out
}

func parseCalendarDateInLocation(value string, loc *time.Location) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := parsed.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
}

func buildFreeBusySummary(req FreeBusyRequest, blocks []FreeBusyBlock, loc *time.Location) FreeBusySummary {
	busyHoursByDay := map[string]float64{}
	fullDays := map[string]struct{}{}
	for _, block := range blocks {
		if !block.End.After(block.Start) {
			continue
		}
		startDay := dayStart(block.Start.In(loc))
		endDay := dayStart(block.End.Add(-time.Nanosecond).In(loc))
		for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
			next := day.AddDate(0, 0, 1)
			overlapStart := maxCalendarTime(block.Start.In(loc), day)
			overlapEnd := minCalendarTime(block.End.In(loc), next)
			if overlapEnd.After(overlapStart) {
				key := day.Format("2006-01-02")
				busyHoursByDay[key] += overlapEnd.Sub(overlapStart).Hours()
				if block.AllDay || overlapEnd.Sub(overlapStart) >= 23*time.Hour {
					fullDays[key] = struct{}{}
				}
			}
		}
	}
	fullyBusy := len(fullDays)
	partiallyBusy := 0
	for day := range busyHoursByDay {
		if _, ok := fullDays[day]; !ok {
			partiallyBusy++
		}
	}
	return FreeBusySummary{
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		TimeZone:          req.TimeZone,
		BusyBlockCount:    len(blocks),
		BusyDays:          len(busyHoursByDay),
		FullyBusyDays:     fullyBusy,
		PartiallyBusyDays: partiallyBusy,
		CalendarCount:     len(req.CalendarIDs),
	}
}

func dayStart(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, value.Location())
}

func maxCalendarTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minCalendarTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func calendarErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrCalendarNotConnected):
		return "calendar_not_connected"
	case errors.Is(err, ErrCalendarReauthRequired):
		return "calendar_connection_revoked"
	case errors.Is(err, ErrCalendarFreeBusyMalformedResponse):
		return "calendar_free_busy_malformed_response"
	case errors.Is(err, ErrCalendarFreeBusyUnavailable):
		return "calendar_free_busy_unavailable"
	default:
		return "calendar_free_busy_unavailable"
	}
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
		if decision, ok := s.reserveCalendar(ctx, providerlimits.OpCalendarEventDelete); !ok {
			out.Failed++
			s.log.Warn("calendar event delete limited",
				zap.String("event_id", event.EventID),
				zap.String("reason", decision.Reason),
			)
			continue
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
