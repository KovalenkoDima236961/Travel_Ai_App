package trip

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	defaultCurrency = "EUR"
	defaultPace     = "balanced"
)

// Service holds the business logic for trips. It depends only on the
// repository and a logger.
type Service struct {
	repo *Repository
	log  *zap.Logger
}

// NewService constructs the trip service.
func NewService(repo *Repository, log *zap.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Create applies defaults and stores a new DRAFT trip. Input is assumed to be
// already validated by the transport layer.
func (s *Service) Create(ctx context.Context, in CreateTripInput) (*Trip, error) {
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
			return nil, fmt.Errorf("parse start date: %w", err)
		}
		startDate = &parsed
	}

	interests := in.Interests
	if interests == nil {
		interests = []string{}
	}

	created, err := s.repo.Create(ctx, &Trip{
		Destination:    strings.TrimSpace(in.Destination),
		StartDate:      startDate,
		Days:           in.Days,
		BudgetAmount:   in.BudgetAmount,
		BudgetCurrency: currency,
		Travelers:      in.Travelers,
		Interests:      interests,
		Pace:           pace,
		Status:         StatusDraft,
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
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Trip, error) {
	return s.repo.GetByID(ctx, id)
}

// Generate runs the mock planning flow: PROCESSING -> build itinerary ->
// COMPLETED (or FAILED on error). It is a local stand-in for the future async
// AI Planning Service integration.
func (s *Service) Generate(ctx context.Context, id uuid.UUID) (*Trip, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpdateStatus(ctx, id, StatusProcessing); err != nil {
		return nil, err
	}
	s.log.Info("trip processing started", zap.String("trip_id", id.String()))

	itinerary, err := json.Marshal(buildMockItinerary(current))
	if err != nil {
		s.markFailed(ctx, id)
		return nil, fmt.Errorf("marshal itinerary: %w", err)
	}

	updated, err := s.repo.UpdateItinerary(ctx, id, itinerary, StatusCompleted)
	if err != nil {
		s.markFailed(ctx, id)
		return nil, err
	}

	s.log.Info("trip completed", zap.String("trip_id", id.String()))
	return updated, nil
}

func (s *Service) markFailed(ctx context.Context, id uuid.UUID) {
	if _, err := s.repo.UpdateStatus(ctx, id, StatusFailed); err != nil {
		s.log.Error("failed to mark trip as FAILED",
			zap.String("trip_id", id.String()),
			zap.Error(err),
		)
	}
}

// buildMockItinerary produces a deterministic, interest-aware sample plan.
func buildMockItinerary(t *Trip) Itinerary {
	interests := t.Interests
	if len(interests) == 0 {
		interests = []string{"sightseeing"}
	}

	days := make([]ItineraryDay, 0, t.Days)
	for i := int32(0); i < t.Days; i++ {
		focus := interests[int(i)%len(interests)]
		days = append(days, ItineraryDay{
			Day:   int(i) + 1,
			Title: fmt.Sprintf("Day %d in %s — %s", i+1, t.Destination, titleCase(focus)),
			Activities: []string{
				fmt.Sprintf("Morning: explore %s highlights focused on %s", t.Destination, focus),
				fmt.Sprintf("Afternoon: a %s-paced %s experience", t.Pace, focus),
				"Evening: local dinner recommendation",
			},
		})
	}

	return Itinerary{
		Destination: t.Destination,
		Summary: fmt.Sprintf("A %d-day %s trip to %s for %d traveler(s).",
			t.Days, t.Pace, t.Destination, t.Travelers),
		Travelers:   t.Travelers,
		Pace:        t.Pace,
		Currency:    t.BudgetCurrency,
		TotalBudget: t.BudgetAmount,
		Days:        days,
		GeneratedAt: time.Now().UTC(),
		Source:      "mock-local-generator",
	}
}

// titleCase upper-cases the first rune of s.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
