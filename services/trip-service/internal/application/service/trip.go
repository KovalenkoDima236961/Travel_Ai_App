// Package service contains the trip use cases. It depends on ports (interfaces)
// it owns, not on concrete adapters.
package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	defaultCurrency = "EUR"
	defaultPace     = "balanced"

	maxDays      = 30
	defaultLimit = 20
	maxLimit     = 100
)

// tripRepository is the persistence port the use case depends on. The concrete
// postgres adapter satisfies it; tests substitute a mock. It is intentionally
// unexported — the use case owns the abstraction it consumes.
type tripRepository interface {
	Create(ctx context.Context, t *entity.Trip) (*entity.Trip, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Trip, error)
	List(ctx context.Context, limit, offset int) ([]entity.Trip, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.Status) (*entity.Trip, error)
	UpdateItinerary(ctx context.Context, id uuid.UUID, itinerary json.RawMessage, status entity.Status) (*entity.Trip, error)
}

// Service holds the trip business logic. It depends on the repository and
// generator ports and a logger.
type Service struct {
	repo      tripRepository
	generator application.ItineraryGenerator
	log       *zap.Logger
}

// New constructs the trip service.
func New(repo tripRepository, generator application.ItineraryGenerator, log *zap.Logger) *Service {
	return &Service{repo: repo, generator: generator, log: log}
}

// Create validates input, applies defaults, and stores a new DRAFT trip.
func (s *Service) Create(ctx context.Context, in appdto.CreateTripInput) (*entity.Trip, error) {
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
		zap.String("destination", created.Destination),
	)
	return created, nil
}

// Get returns a trip by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns trips ordered by created_at DESC. It normalises and validates the
// pagination parameters: limit defaults to 20 (when 0) and must be 1..100;
// offset must be >= 0.
func (s *Service) List(ctx context.Context, limit, offset int) ([]entity.Trip, int, int, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}
	if offset < 0 {
		return nil, 0, 0, apperrs.NewInvalidInput("offset must be >= 0")
	}

	trips, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return trips, limit, offset, nil
}

// Generate runs the planning flow: PROCESSING -> generate itinerary -> COMPLETED
// (or FAILED on error). The itinerary itself is produced by the injected
// ItineraryGenerator port, a local stand-in for the future AI Planning Service.
func (s *Service) Generate(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpdateStatus(ctx, id, entity.StatusProcessing); err != nil {
		return nil, err
	}
	s.log.Info("trip processing started", zap.String("trip_id", id.String()))

	itinerary, err := s.generator.Generate(ctx, *current)
	if err != nil {
		s.markFailed(ctx, id)
		return nil, err
	}

	raw, err := json.Marshal(itinerary)
	if err != nil {
		s.markFailed(ctx, id)
		return nil, err
	}

	updated, err := s.repo.UpdateItinerary(ctx, id, raw, entity.StatusCompleted)
	if err != nil {
		s.markFailed(ctx, id)
		return nil, err
	}

	s.log.Info("trip completed", zap.String("trip_id", id.String()))
	return updated, nil
}

func (s *Service) markFailed(ctx context.Context, id uuid.UUID) {
	if _, err := s.repo.UpdateStatus(ctx, id, entity.StatusFailed); err != nil {
		s.log.Error("failed to mark trip as FAILED",
			zap.String("trip_id", id.String()),
			zap.Error(err),
		)
	}
}
