package tripdiscovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
)

func TestSurpriseCreatesSuggestionsButNotTrip(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepository{}
	trips := &fakeTripCreator{}
	service := newTestService(repo, trips, &fakeJobCreator{})
	ctx := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID})

	session, err := service.Surprise(ctx, DiscoverInput{
		Scope:          "personal",
		Travelers:      1,
		OutputLanguage: "en",
	})
	if err != nil {
		t.Fatalf("Surprise() error = %v", err)
	}
	if session == nil || len(session.Response.Suggestions) != 1 {
		t.Fatalf("expected a persisted suggestion session, got %+v", session)
	}
	if trips.createCalls != 0 {
		t.Fatalf("surprise must not create a trip; calls = %d", trips.createCalls)
	}
	if session.CreatedTripID != nil {
		t.Fatal("surprise session unexpectedly has createdTripId")
	}
}

func TestCreateTripRequiresConfirmationAndStoresDiscoveryMetadata(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	repo := &fakeRepository{
		session: &Session{
			ID:             sessionID,
			UserID:         userID,
			Mode:           ModePrompt,
			Prompt:         "warm food weekend",
			OutputLanguage: "uk",
			Status:         "completed",
			Request: AIRequest{
				TripContext: TripContext{Travelers: 2, Scope: "personal"},
			},
			Response: SuggestionResponse{Suggestions: []Suggestion{testSuggestion()}},
		},
	}
	trips := &fakeTripCreator{}
	jobs := &fakeJobCreator{}
	service := newTestService(repo, trips, jobs)
	ctx := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID})

	result, err := service.CreateTrip(ctx, sessionID, "valencia-spain", CreateTripInput{
		DurationDays:          4,
		Travelers:             2,
		AutoGenerateItinerary: true,
	})
	if err != nil {
		t.Fatalf("CreateTrip() error = %v", err)
	}
	if result.Trip.CreationMetadata["creationSource"] != "trip_discovery" {
		t.Fatalf("missing discovery metadata: %+v", result.Trip.CreationMetadata)
	}
	if result.GenerationJob == nil || jobs.createCalls != 1 {
		t.Fatalf("expected one generation job, got %+v", result.GenerationJob)
	}
	if repo.markedTripID != result.Trip.ID {
		t.Fatalf("session was not marked with trip id %s", result.Trip.ID)
	}
}

func TestPromptValidationRejectsEmptyAndUnsupportedLanguage(t *testing.T) {
	service := newTestService(&fakeRepository{}, &fakeTripCreator{}, &fakeJobCreator{})
	ctx := auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: uuid.New()})

	if _, err := service.Discover(ctx, DiscoverInput{Scope: "personal"}); err == nil {
		t.Fatal("expected empty prompt validation error")
	}
	if _, err := service.Discover(ctx, DiscoverInput{
		Prompt:         "weekend",
		Scope:          "personal",
		OutputLanguage: "de",
	}); err == nil {
		t.Fatal("expected unsupported language validation error")
	}
}

func TestPreviousTripsAreSummarizedWithoutItinerary(t *testing.T) {
	amount := 450.0
	summaries := summarizeTrips([]entity.Trip{{
		Destination:    "Prague, Czechia",
		Days:           3,
		BudgetAmount:   &amount,
		BudgetCurrency: "EUR",
		Interests:      []string{"food"},
		Pace:           "balanced",
		Itinerary:      []byte(`{"private":"full itinerary"}`),
		CreatedAt:      time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
	}})

	if len(summaries) != 1 || summaries[0].Destination != "Prague" {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	if summaries[0].Country != "Czechia" || summaries[0].Budget == nil {
		t.Fatalf("expected country and budget summary: %+v", summaries[0])
	}
}

func newTestService(
	repo *fakeRepository,
	trips *fakeTripCreator,
	jobs *fakeJobCreator,
) *Service {
	return NewService(
		repo,
		fakeAI{},
		trips,
		jobs,
		nil,
		nil,
		nil,
		Config{Enabled: true},
		zap.NewNop(),
	)
}

type fakeAI struct{}

func (fakeAI) SuggestDestinations(
	context.Context,
	AIRequest,
) (*SuggestionResponse, error) {
	return &SuggestionResponse{
		SessionTitle: "Ideas",
		Suggestions:  []Suggestion{testSuggestion()},
	}, nil
}

func testSuggestion() Suggestion {
	return Suggestion{
		ID:                      "valencia-spain",
		Destination:             "Valencia, Spain",
		City:                    "Valencia",
		Country:                 "Spain",
		MatchScore:              87,
		RecommendedDurationDays: 4,
		EstimatedBudget: BudgetEstimate{
			Amount: 520, Currency: "EUR", Confidence: "medium",
		},
		TripPreview: TripPreview{
			Title: "Valencia escape", Summary: "Food and architecture", SampleDay: []string{"Market"},
		},
		Tags:                        []string{"food", "city_break"},
		SuggestedPromptForItinerary: "Create a four-day Valencia trip.",
	}
}

type fakeRepository struct {
	session      *Session
	markedTripID uuid.UUID
}

func (r *fakeRepository) CreateTripDiscoverySession(
	_ context.Context,
	session *Session,
) (*Session, error) {
	copy := *session
	copy.CreatedAt = time.Now()
	r.session = &copy
	return &copy, nil
}

func (r *fakeRepository) GetTripDiscoverySessionByIDAndUser(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (*Session, error) {
	return r.session, nil
}

func (r *fakeRepository) ListTripDiscoverySessionsByUser(
	context.Context,
	uuid.UUID,
	int,
) ([]Session, error) {
	if r.session == nil {
		return []Session{}, nil
	}
	return []Session{*r.session}, nil
}

func (r *fakeRepository) MarkTripDiscoverySessionCreatedTrip(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	tripID uuid.UUID,
) (*Session, error) {
	r.markedTripID = tripID
	r.session.CreatedTripID = &tripID
	return r.session, nil
}

func (r *fakeRepository) ListByUser(
	context.Context,
	uuid.UUID,
	int,
	int,
) ([]entity.Trip, error) {
	return []entity.Trip{}, nil
}

func (r *fakeRepository) UpdateTripCreationMetadata(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
	metadata map[string]any,
) (*entity.Trip, error) {
	return &entity.Trip{
		ID:                uuid.New(),
		Destination:       "Valencia, Spain",
		BudgetCurrency:    "EUR",
		Interests:         []string{},
		Status:            entity.StatusDraft,
		CreationMetadata:  metadata,
		ItineraryRevision: 0,
	}, nil
}

type fakeTripCreator struct {
	createCalls int
}

func (f *fakeTripCreator) Create(
	_ context.Context,
	input appdto.CreateTripInput,
) (*entity.Trip, error) {
	f.createCalls++
	return &entity.Trip{
		ID:                uuid.New(),
		Destination:       input.Destination,
		BudgetCurrency:    input.BudgetCurrency,
		Interests:         input.Interests,
		Status:            entity.StatusDraft,
		ItineraryRevision: 0,
	}, nil
}

type fakeJobCreator struct {
	createCalls int
}

func (f *fakeJobCreator) Create(
	_ context.Context,
	tripID uuid.UUID,
	request generationjobs.CreateRequest,
) (*entity.GenerationJob, error) {
	f.createCalls++
	return &entity.GenerationJob{
		ID:                        uuid.New(),
		TripID:                    tripID,
		JobType:                   request.JobType,
		Status:                    entity.GenerationJobStatusQueued,
		ExpectedItineraryRevision: 0,
	}, nil
}
