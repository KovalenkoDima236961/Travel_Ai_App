// Package service contains the trip use cases. It depends on ports (interfaces)
// it owns, not on concrete adapters.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
)

const (
	defaultCurrency = "EUR"
	defaultPace     = "balanced"

	maxDays                 = 30
	maxItineraryDays        = 60
	maxItineraryItemsPerDay = 30
	maxInstructionLength    = 500
	defaultLimit            = 20
	maxLimit                = 100
)

type editableItinerary struct {
	Destination string                   `json:"destination,omitempty"`
	Summary     string                   `json:"summary,omitempty"`
	Travelers   int32                    `json:"travelers,omitempty"`
	Pace        string                   `json:"pace,omitempty"`
	Currency    string                   `json:"currency,omitempty"`
	TotalBudget *float64                 `json:"totalBudget,omitempty"`
	Days        []aggregate.ItineraryDay `json:"days"`
	GeneratedAt *time.Time               `json:"generatedAt,omitempty"`
	Source      string                   `json:"source,omitempty"`
}

// tripRepository is the persistence port the use case depends on. The concrete
// postgres adapter satisfies it; tests substitute a mock. It is intentionally
// unexported — the use case owns the abstraction it consumes.
type tripRepository interface {
	Create(ctx context.Context, t *entity.Trip) (*entity.Trip, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*entity.Trip, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Trip, error)
	UpdateStatusByUserID(ctx context.Context, id, userID uuid.UUID, status entity.Status) (*entity.Trip, error)
	UpdateItineraryByUserID(ctx context.Context, id, userID uuid.UUID, itinerary json.RawMessage, status entity.Status) (*entity.Trip, error)
}

type userContextProvider interface {
	GetUserContext(ctx context.Context, accessToken string) (*usercontext.UserContext, error)
}

// Option customizes Service dependencies that are not required for the core
// trip CRUD flow.
type Option func(*Service)

// WithUserContext enables optional user profile/preferences loading during
// itinerary generation.
func WithUserContext(provider userContextProvider, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.userContextProvider = provider
		s.userContextEnabled = enabled
		s.userContextFailOpen = failOpen
	}
}

// Service holds the trip business logic. It depends on the repository and
// generator ports and a logger.
type Service struct {
	repo                tripRepository
	generator           application.ItineraryGenerator
	userContextProvider userContextProvider
	userContextEnabled  bool
	userContextFailOpen bool
	log                 *zap.Logger
}

// New constructs the trip service.
func New(repo tripRepository, generator application.ItineraryGenerator, log *zap.Logger, opts ...Option) *Service {
	s := &Service{repo: repo, generator: generator, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create validates input, applies defaults, and stores a new DRAFT trip.
func (s *Service) Create(ctx context.Context, in appdto.CreateTripInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	destination := strings.TrimSpace(in.Destination)
	if destination == "" {
		return nil, apperrs.NewInvalidInput("destination is required")
	}
	if in.Days < 1 || in.Days > maxDays {
		return nil, apperrs.NewInvalidInput("days must be between 1 and %d", maxDays)
	}
	if in.Travelers < 1 {
		return nil, apperrs.NewInvalidInput("travelers must be at least 1")
	}

	currency := strings.ToUpper(strings.TrimSpace(in.BudgetCurrency))
	if currency == "" {
		currency = defaultCurrency
	}
	pace := strings.TrimSpace(in.Pace)
	if pace == "" {
		pace = defaultPace
	}

	var startDate *time.Time
	if strings.TrimSpace(in.StartDate) != "" {
		parsed, err := time.Parse("2006-01-02", in.StartDate)
		if err != nil {
			return nil, apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
		}
		startDate = &parsed
	}

	interests := in.Interests
	if interests == nil {
		interests = []string{}
	}

	created, err := s.repo.Create(ctx, &entity.Trip{
		UserID:         &user.ID,
		Destination:    destination,
		StartDate:      startDate,
		Days:           in.Days,
		BudgetAmount:   in.BudgetAmount,
		BudgetCurrency: currency,
		Travelers:      in.Travelers,
		Interests:      interests,
		Pace:           pace,
		Status:         entity.StatusDraft,
	})
	if err != nil {
		return nil, err
	}

	s.log.Info("trip created",
		zap.String("trip_id", created.ID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("destination", created.Destination),
	)
	return created, nil
}

// Get returns a trip by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByIDAndUserID(ctx, id, user.ID)
}

// List returns trips ordered by created_at DESC. It normalises and validates the
// pagination parameters: limit defaults to 20 (when 0) and must be 1..100;
// offset must be >= 0.
func (s *Service) List(ctx context.Context, limit, offset int) ([]entity.Trip, int, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}
	if offset < 0 {
		return nil, 0, 0, apperrs.NewInvalidInput("offset must be >= 0")
	}

	trips, err := s.repo.ListByUser(ctx, user.ID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return trips, limit, offset, nil
}

// Generate runs the planning flow: PROCESSING -> generate itinerary -> COMPLETED
// (or FAILED on error). The itinerary itself is produced by the injected
// ItineraryGenerator port.
func (s *Service) Generate(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	current, err := s.repo.GetByIDAndUserID(ctx, id, user.ID)
	if err != nil {
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpdateStatusByUserID(ctx, id, user.ID, entity.StatusProcessing); err != nil {
		return nil, err
	}
	s.log.Info("trip processing started",
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	)

	itinerary, err := s.generator.Generate(ctx, application.GenerateItineraryInput{
		Trip:            *current,
		UserProfile:     userContext.Profile,
		UserPreferences: userContext.Preferences,
	})
	if err != nil {
		s.markFailed(ctx, id, user.ID)
		return nil, err
	}

	raw, err := json.Marshal(itinerary)
	if err != nil {
		s.markFailed(ctx, id, user.ID)
		return nil, err
	}

	updated, err := s.repo.UpdateItineraryByUserID(ctx, id, user.ID, raw, entity.StatusCompleted)
	if err != nil {
		s.markFailed(ctx, id, user.ID)
		return nil, err
	}

	s.log.Info("trip completed",
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	)
	return updated, nil
}

// UpdateItinerary validates and replaces the full itinerary JSON for a trip
// owned by the authenticated user. It does not call the itinerary generator.
func (s *Service) UpdateItinerary(ctx context.Context, id uuid.UUID, in appdto.UpdateItineraryInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	fields := []zap.Field{
		zap.String("action", "itinerary_update"),
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	}

	normalized, err := validateAndNormalizeItinerary(in.Itinerary)
	if err != nil {
		s.log.Warn("itinerary update failed",
			append(fields,
				zap.Bool("success", false),
				zap.Error(err),
			)...,
		)
		return nil, err
	}

	updated, err := s.repo.UpdateItineraryByUserID(ctx, id, user.ID, normalized, entity.StatusCompleted)
	if err != nil {
		s.log.Warn("itinerary update failed",
			append(fields,
				zap.Bool("success", false),
				zap.Error(err),
			)...,
		)
		return nil, err
	}

	s.log.Info("itinerary updated",
		append(fields,
			zap.Bool("success", true),
		)...,
	)
	return updated, nil
}

// RegenerateDay replaces only one existing itinerary day with an AI-generated
// replacement. DayNumber is one-based and matched against itinerary.days[].day.
func (s *Service) RegenerateDay(ctx context.Context, id uuid.UUID, dayNumber int, in appdto.RegenerateItineraryPartInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	instruction, err := normalizeRegenerationInstruction(in.Instruction)
	fields := regenerateLogFields("regenerate_day", id, user.ID, dayNumber, nil, instruction)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}

	current, err := s.repo.GetByIDAndUserID(ctx, id, user.ID)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}

	currentItinerary, dayIndex, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	userContextLoaded := userContext.Profile != nil || userContext.Preferences != nil

	replacement, err := s.generator.RegenerateDay(ctx, application.RegenerateDayInput{
		Trip:             *current,
		CurrentItinerary: currentItinerary,
		DayNumber:        dayNumber,
		Instruction:      instruction,
		UserProfile:      userContext.Profile,
		UserPreferences:  userContext.Preferences,
	})
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	normalizedReplacement, err := normalizeReplacementDay(replacement, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, apperrs.NewDependencyError("AI returned invalid replacement")
	}

	currentItinerary.Days[dayIndex] = normalizedReplacement
	updated, err := s.saveRegeneratedItinerary(ctx, id, user.ID, currentItinerary)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	s.logRegenerationSuccess("itinerary day regenerated", fields, started, userContextLoaded)
	return updated, nil
}

// RegenerateItem replaces only one item in one itinerary day. DayNumber is
// one-based; ItemIndex is zero-based to match the items array index.
func (s *Service) RegenerateItem(ctx context.Context, id uuid.UUID, dayNumber, itemIndex int, in appdto.RegenerateItineraryPartInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	instruction, err := normalizeRegenerationInstruction(in.Instruction)
	fields := regenerateLogFields("regenerate_item", id, user.ID, dayNumber, &itemIndex, instruction)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}

	current, err := s.repo.GetByIDAndUserID(ctx, id, user.ID)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}

	currentItinerary, dayIndex, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	if itemIndex < 0 || itemIndex >= len(currentItinerary.Days[dayIndex].Items) {
		err := currentItineraryInvalidError()
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	userContextLoaded := userContext.Profile != nil || userContext.Preferences != nil

	replacement, err := s.generator.RegenerateItem(ctx, application.RegenerateItemInput{
		Trip:             *current,
		CurrentItinerary: currentItinerary,
		DayNumber:        dayNumber,
		ItemIndex:        itemIndex,
		Instruction:      instruction,
		UserProfile:      userContext.Profile,
		UserPreferences:  userContext.Preferences,
	})
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	normalizedReplacement, err := normalizeReplacementItem(replacement)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, apperrs.NewDependencyError("AI returned invalid replacement")
	}

	currentItinerary.Days[dayIndex].Items[itemIndex] = normalizedReplacement
	updated, err := s.saveRegeneratedItinerary(ctx, id, user.ID, currentItinerary)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	s.logRegenerationSuccess("itinerary item regenerated", fields, started, userContextLoaded)
	return updated, nil
}

func validateAndNormalizeItinerary(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil, apperrs.NewInvalidInput("itinerary is required")
	}

	var itinerary editableItinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		return nil, apperrs.NewInvalidInput("invalid itinerary")
	}

	itinerary.Destination = strings.TrimSpace(itinerary.Destination)
	itinerary.Summary = strings.TrimSpace(itinerary.Summary)
	itinerary.Pace = strings.TrimSpace(itinerary.Pace)
	itinerary.Currency = strings.ToUpper(strings.TrimSpace(itinerary.Currency))
	itinerary.Source = strings.TrimSpace(itinerary.Source)
	if itinerary.TotalBudget != nil && *itinerary.TotalBudget < 0 {
		return nil, apperrs.NewInvalidInput("itinerary.totalBudget must be >= 0")
	}

	if len(itinerary.Days) == 0 {
		return nil, apperrs.NewInvalidInput("itinerary.days must contain at least 1 day")
	}
	if len(itinerary.Days) > maxItineraryDays {
		return nil, apperrs.NewInvalidInput("itinerary.days must contain at most %d days", maxItineraryDays)
	}

	for dayIndex := range itinerary.Days {
		day := &itinerary.Days[dayIndex]
		if day.Day < 1 {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].day must be >= 1", dayIndex)
		}
		day.Title = strings.TrimSpace(day.Title)
		if day.Title == "" {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].title is required", dayIndex)
		}
		if len(day.Items) == 0 {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].items must contain at least 1 item", dayIndex)
		}
		if len(day.Items) > maxItineraryItemsPerDay {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].items must contain at most %d items", dayIndex, maxItineraryItemsPerDay)
		}

		for itemIndex := range day.Items {
			item := &day.Items[itemIndex]
			item.Time = strings.TrimSpace(item.Time)
			if item.Time == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].time is required", dayIndex, itemIndex)
			}
			item.Type = strings.TrimSpace(item.Type)
			if item.Type == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].type is required", dayIndex, itemIndex)
			}
			item.Name = strings.TrimSpace(item.Name)
			if item.Name == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].name is required", dayIndex, itemIndex)
			}
			item.Note = strings.TrimSpace(item.Note)
			if item.EstimatedCost != nil && *item.EstimatedCost < 0 {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].estimatedCost must be >= 0", dayIndex, itemIndex)
			}
		}
	}

	normalized, err := json.Marshal(itinerary)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizeRegenerationInstruction(raw string) (string, error) {
	instruction := strings.TrimSpace(raw)
	if len(instruction) > maxInstructionLength {
		return "", apperrs.NewInvalidInput("instruction must be at most %d characters", maxInstructionLength)
	}
	return instruction, nil
}

func currentItineraryAndDayIndex(t *entity.Trip, dayNumber int) (aggregate.Itinerary, int, error) {
	if t == nil || len(t.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(t.Itinerary)), "null") {
		return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
	}

	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(t.Itinerary, &itinerary); err != nil {
		return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
	}
	if err := validateCurrentItinerary(itinerary); err != nil {
		return aggregate.Itinerary{}, -1, err
	}

	for index := range itinerary.Days {
		if itinerary.Days[index].Day == dayNumber {
			return itinerary, index, nil
		}
	}

	return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
}

func validateCurrentItinerary(itinerary aggregate.Itinerary) error {
	if len(itinerary.Days) == 0 || len(itinerary.Days) > maxItineraryDays {
		return currentItineraryInvalidError()
	}

	seenDays := make(map[int]struct{}, len(itinerary.Days))
	for _, day := range itinerary.Days {
		if day.Day < 1 {
			return currentItineraryInvalidError()
		}
		if _, exists := seenDays[day.Day]; exists {
			return currentItineraryInvalidError()
		}
		seenDays[day.Day] = struct{}{}

		if strings.TrimSpace(day.Title) == "" {
			return currentItineraryInvalidError()
		}
		if len(day.Items) == 0 || len(day.Items) > maxItineraryItemsPerDay {
			return currentItineraryInvalidError()
		}
		for _, item := range day.Items {
			if strings.TrimSpace(item.Time) == "" ||
				strings.TrimSpace(item.Type) == "" ||
				strings.TrimSpace(item.Name) == "" {
				return currentItineraryInvalidError()
			}
			if item.EstimatedCost != nil && *item.EstimatedCost < 0 {
				return currentItineraryInvalidError()
			}
		}
	}

	return nil
}

func normalizeReplacementDay(day *aggregate.ItineraryDay, dayNumber int) (aggregate.ItineraryDay, error) {
	if day == nil {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day is required")
	}

	normalized := *day
	normalized.Day = dayNumber
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day title is required")
	}
	if len(normalized.Items) == 0 || len(normalized.Items) > maxItineraryItemsPerDay {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day item count is invalid")
	}
	for index := range normalized.Items {
		item, err := normalizeReplacementItem(&normalized.Items[index])
		if err != nil {
			return aggregate.ItineraryDay{}, err
		}
		normalized.Items[index] = item
	}

	return normalized, nil
}

func normalizeReplacementItem(item *aggregate.ItineraryItem) (aggregate.ItineraryItem, error) {
	if item == nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item is required")
	}

	normalized := *item
	normalized.Time = strings.TrimSpace(normalized.Time)
	normalized.Type = strings.TrimSpace(normalized.Type)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Note = strings.TrimSpace(normalized.Note)
	if normalized.Time == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item time is required")
	}
	if normalized.Type == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item type is required")
	}
	if normalized.Name == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item name is required")
	}
	if normalized.EstimatedCost != nil && *normalized.EstimatedCost < 0 {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item estimated cost must be >= 0")
	}

	return normalized, nil
}

func (s *Service) saveRegeneratedItinerary(ctx context.Context, tripID, userID uuid.UUID, itinerary aggregate.Itinerary) (*entity.Trip, error) {
	raw, err := json.Marshal(itinerary)
	if err != nil {
		return nil, err
	}
	return s.repo.UpdateItineraryByUserID(ctx, tripID, userID, raw, entity.StatusCompleted)
}

func currentItineraryInvalidError() error {
	return apperrs.NewInvalidInput("current itinerary is invalid")
}

func regenerateLogFields(action string, tripID, userID uuid.UUID, dayNumber int, itemIndex *int, instruction string) []zap.Field {
	fields := []zap.Field{
		zap.String("action", action),
		zap.String("tripId", tripID.String()),
		zap.String("userId", userID.String()),
		zap.Int("dayNumber", dayNumber),
		zap.Bool("instructionPresent", instruction != ""),
	}
	if itemIndex != nil {
		fields = append(fields, zap.Int("itemIndex", *itemIndex))
	}
	return fields
}

func (s *Service) logRegenerationSuccess(message string, fields []zap.Field, started time.Time, userContextLoaded bool) {
	s.log.Info(message,
		append(fields,
			zap.Bool("userContextLoaded", userContextLoaded),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
			zap.Bool("success", true),
		)...,
	)
}

func (s *Service) logRegenerationFailure(message string, fields []zap.Field, started time.Time, userContextLoaded bool, err error) {
	s.log.Warn(message,
		append(fields,
			zap.Bool("userContextLoaded", userContextLoaded),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
			zap.Bool("success", false),
			zap.Error(err),
		)...,
	)
}

func (s *Service) loadUserContext(ctx context.Context, user auth.AuthenticatedUser, tripID uuid.UUID) (usercontext.UserContext, error) {
	fields := []zap.Field{
		zap.Bool("userContextEnabled", s.userContextEnabled),
		zap.Bool("userContextFailOpen", s.userContextFailOpen),
		zap.String("userId", user.ID.String()),
		zap.String("tripId", tripID.String()),
	}

	if !s.userContextEnabled {
		s.log.Info("user context disabled",
			append(fields,
				zap.Bool("userContextLoaded", false),
				zap.String("userContextErrorType", ""),
			)...,
		)
		return usercontext.UserContext{}, nil
	}

	if s.userContextProvider == nil {
		err := userContextError(usercontext.ErrorTypeService, "user context provider is not configured")
		return s.handleUserContextError(err, fields)
	}

	accessToken, ok := auth.AccessTokenFromContext(ctx)
	if !ok {
		err := userContextError(usercontext.ErrorTypeAuth, "access token missing from request context")
		return s.handleUserContextError(err, fields)
	}

	loaded, err := s.userContextProvider.GetUserContext(ctx, accessToken)
	if err != nil {
		return s.handleUserContextError(err, fields)
	}
	if loaded == nil {
		loaded = &usercontext.UserContext{}
	}

	contextLoaded := loaded.Profile != nil || loaded.Preferences != nil
	s.log.Info("user context loaded",
		append(fields,
			zap.Bool("userContextLoaded", contextLoaded),
			zap.String("userContextErrorType", ""),
		)...,
	)

	return *loaded, nil
}

func (s *Service) handleUserContextError(err error, fields []zap.Field) (usercontext.UserContext, error) {
	errorType := classifyUserContextError(err)
	logFields := append(fields,
		zap.Bool("userContextLoaded", false),
		zap.String("userContextErrorType", errorType),
		zap.Error(err),
	)

	if s.userContextFailOpen {
		s.log.Warn("failed to load user context; continuing without personalization", logFields...)
		return usercontext.UserContext{}, nil
	}

	s.log.Warn("failed to load user context; generation blocked", logFields...)
	return usercontext.UserContext{}, apperrs.NewDependencyError("failed to load user preferences")
}

func classifyUserContextError(err error) string {
	var userContextErr *usercontext.Error
	if err != nil && errors.As(err, &userContextErr) {
		return string(userContextErr.Type)
	}
	return string(usercontext.ErrorTypeService)
}

func userContextError(errorType usercontext.ErrorType, message string) error {
	return &usercontext.Error{Type: errorType, Message: message}
}

func (s *Service) markFailed(ctx context.Context, id, userID uuid.UUID) {
	if _, err := s.repo.UpdateStatusByUserID(ctx, id, userID, entity.StatusFailed); err != nil {
		s.log.Error("failed to mark trip as FAILED",
			zap.String("trip_id", id.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
	}
}
