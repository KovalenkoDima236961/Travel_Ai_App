package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
)

// mockRepo is a hand-written tripRepository that captures arguments and the
// order of status transitions so tests can assert on use-case behaviour without
// a database.
type mockRepo struct {
	createdTrip *entity.Trip
	createErr   error

	getByIDResult *entity.Trip
	getByIDErr    error
	getByIDUserID uuid.UUID

	listResult []entity.Trip
	listErr    error
	listLimit  int
	listOffset int
	listUserID uuid.UUID

	updateStatusErr  error
	statusSeq        []entity.Status
	statusUserIDs    []uuid.UUID
	updateItinStatus entity.Status
	updateItinRaw    json.RawMessage
	updateItinErr    error
	updateItinUserID uuid.UUID
	updateItinSource entity.ItineraryVersionSource
	updateItinMeta   map[string]any

	versions          []entity.ItineraryVersion
	listVersionsTrip  uuid.UUID
	listVersionsUser  uuid.UUID
	listVersionsLimit int
	listVersionsOff   int
	getVersionID      uuid.UUID
	getVersionTripID  uuid.UUID
	getVersionUserID  uuid.UUID
}

func (m *mockRepo) Create(_ context.Context, t *entity.Trip) (*entity.Trip, error) {
	m.createdTrip = t
	if m.createErr != nil {
		return nil, m.createErr
	}
	out := *t
	out.ID = uuid.New()
	out.CreatedAt = time.Now()
	out.UpdatedAt = time.Now()
	return &out, nil
}

func (m *mockRepo) GetByIDAndUserID(_ context.Context, _, userID uuid.UUID) (*entity.Trip, error) {
	m.getByIDUserID = userID
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.getByIDResult, nil
}

func (m *mockRepo) ListByUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]entity.Trip, error) {
	m.listUserID = userID
	m.listLimit = limit
	m.listOffset = offset
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockRepo) UpdateStatusByUserID(_ context.Context, id, userID uuid.UUID, status entity.Status) (*entity.Trip, error) {
	m.statusSeq = append(m.statusSeq, status)
	m.statusUserIDs = append(m.statusUserIDs, userID)
	if m.updateStatusErr != nil {
		return nil, m.updateStatusErr
	}
	return &entity.Trip{ID: id, Status: status}, nil
}

func (m *mockRepo) UpdateItineraryByUserIDAndCreateVersion(
	_ context.Context,
	id, userID uuid.UUID,
	itinerary json.RawMessage,
	status entity.Status,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, *entity.ItineraryVersion, error) {
	m.updateItinRaw = itinerary
	m.updateItinStatus = status
	m.updateItinUserID = userID
	m.updateItinSource = source
	m.updateItinMeta = metadata
	if m.updateItinErr != nil {
		return nil, nil, m.updateItinErr
	}
	version := entity.ItineraryVersion{
		ID:            uuid.New(),
		TripID:        id,
		UserID:        userID,
		VersionNumber: countTripVersions(m.versions, id) + 1,
		Source:        source,
		Itinerary:     itinerary,
		Metadata:      metadata,
		CreatedAt:     time.Now(),
	}
	m.versions = append(m.versions, version)
	return &entity.Trip{ID: id, Status: status, Itinerary: itinerary}, &version, nil
}

func (m *mockRepo) ListItineraryVersionsByTripAndUser(_ context.Context, tripID, userID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error) {
	m.listVersionsTrip = tripID
	m.listVersionsUser = userID
	m.listVersionsLimit = limit
	m.listVersionsOff = offset
	out := make([]entity.ItineraryVersion, 0)
	for _, version := range m.versions {
		if version.TripID == tripID && version.UserID == userID {
			out = append(out, version)
		}
	}
	return out, nil
}

func (m *mockRepo) GetItineraryVersionByIDTripAndUser(_ context.Context, id, tripID, userID uuid.UUID) (*entity.ItineraryVersion, error) {
	m.getVersionID = id
	m.getVersionTripID = tripID
	m.getVersionUserID = userID
	for i := range m.versions {
		version := m.versions[i]
		if version.ID == id && version.TripID == tripID && version.UserID == userID {
			return &version, nil
		}
	}
	return nil, domainerrs.ErrNotFound
}

func countTripVersions(versions []entity.ItineraryVersion, tripID uuid.UUID) int {
	count := 0
	for _, version := range versions {
		if version.TripID == tripID {
			count++
		}
	}
	return count
}

// mockGenerator is an application.ItineraryGenerator test double.
type mockGenerator struct {
	result               *aggregate.Itinerary
	err                  error
	called               bool
	capturedInput        application.GenerateItineraryInput
	dayResult            *aggregate.ItineraryDay
	dayErr               error
	regenerateDayCalled  bool
	capturedDayInput     application.RegenerateDayInput
	itemResult           *aggregate.ItineraryItem
	itemErr              error
	regenerateItemCalled bool
	capturedItemInput    application.RegenerateItemInput
}

func (g *mockGenerator) Generate(_ context.Context, input application.GenerateItineraryInput) (*aggregate.Itinerary, error) {
	g.called = true
	g.capturedInput = input
	if g.err != nil {
		return nil, g.err
	}
	if g.result != nil {
		return g.result, nil
	}
	trip := input.Trip
	return &aggregate.Itinerary{Destination: trip.Destination}, nil
}

func (g *mockGenerator) RegenerateDay(_ context.Context, input application.RegenerateDayInput) (*aggregate.ItineraryDay, error) {
	g.regenerateDayCalled = true
	g.capturedDayInput = input
	if g.dayErr != nil {
		return nil, g.dayErr
	}
	if g.dayResult != nil {
		return g.dayResult, nil
	}
	return &aggregate.ItineraryDay{
		Day:   input.DayNumber,
		Title: "Regenerated day",
		Items: []aggregate.ItineraryItem{{
			Time: "10:00",
			Type: "activity",
			Name: "Replacement activity",
		}},
	}, nil
}

func (g *mockGenerator) RegenerateItem(_ context.Context, input application.RegenerateItemInput) (*aggregate.ItineraryItem, error) {
	g.regenerateItemCalled = true
	g.capturedItemInput = input
	if g.itemErr != nil {
		return nil, g.itemErr
	}
	if g.itemResult != nil {
		return g.itemResult, nil
	}
	return &aggregate.ItineraryItem{
		Time: "12:30",
		Type: "food",
		Name: "Replacement item",
	}, nil
}

type mockUserContextProvider struct {
	result        *usercontext.UserContext
	err           error
	called        bool
	capturedToken string
}

func (p *mockUserContextProvider) GetUserContext(_ context.Context, accessToken string) (*usercontext.UserContext, error) {
	p.called = true
	p.capturedToken = accessToken
	if p.err != nil {
		return nil, p.err
	}
	if p.result != nil {
		return p.result, nil
	}
	return &usercontext.UserContext{}, nil
}

type mockWeatherContextProvider struct {
	result              *weathercontext.WeatherForecast
	err                 error
	called              bool
	capturedDestination string
	capturedStartDate   string
	capturedDays        int
}

type mockPlaceEnrichmentProvider struct {
	result        *placeenrichment.EnrichItineraryResult
	err           error
	called        bool
	capturedInput placeenrichment.EnrichItineraryInput
}

func (p *mockPlaceEnrichmentProvider) EnrichItinerary(_ context.Context, input placeenrichment.EnrichItineraryInput) (*placeenrichment.EnrichItineraryResult, error) {
	p.called = true
	p.capturedInput = input
	if p.err != nil {
		return nil, p.err
	}
	if p.result != nil {
		return p.result, nil
	}
	return &placeenrichment.EnrichItineraryResult{Itinerary: input.Itinerary}, nil
}

func (p *mockWeatherContextProvider) GetForecast(_ context.Context, destination string, startDate string, days int) (*weathercontext.WeatherForecast, error) {
	p.called = true
	p.capturedDestination = destination
	p.capturedStartDate = startDate
	p.capturedDays = days
	if p.err != nil {
		return nil, p.err
	}
	if p.result != nil {
		return p.result, nil
	}
	return testWeatherForecast(), nil
}

func newTestService(repo tripRepository, gen *mockGenerator) *Service {
	return New(repo, gen, zap.NewNop())
}

func validCreateInput() appdto.CreateTripInput {
	return appdto.CreateTripInput{
		Destination: "Rome",
		Days:        3,
		Travelers:   2,
	}
}

func authContext() context.Context {
	return authContextWithToken("")
}

func authContextWithToken(accessToken string) context.Context {
	return auth.WithUser(context.Background(), auth.AuthenticatedUser{
		ID:          testUserID(),
		Email:       "traveler@example.com",
		AccessToken: accessToken,
	})
}

func testUserID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func assertInvalidInput(t *testing.T, err error) {
	t.Helper()
	var invalid *apperrs.InvalidInputError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected *InvalidInputError, got %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.Create(authContext(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID == uuid.Nil {
		t.Fatalf("expected a persisted trip with an ID, got %+v", got)
	}
	if repo.createdTrip == nil {
		t.Fatal("expected repository Create to be called")
	}
	if repo.createdTrip.Status != entity.StatusDraft {
		t.Errorf("expected status DRAFT, got %s", repo.createdTrip.Status)
	}
	if repo.createdTrip.UserID == nil || *repo.createdTrip.UserID != testUserID() {
		t.Fatalf("expected authenticated user id %s, got %v", testUserID(), repo.createdTrip.UserID)
	}
}

func TestCreate_EmptyDestination(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateInput()
	in.Destination = "   "

	_, err := svc.Create(authContext(), in)
	assertInvalidInput(t, err)
	if repo.createdTrip != nil {
		t.Error("repository Create must not be called on invalid input")
	}
}

func TestCreate_DaysTooLow(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateInput()
	in.Days = 0

	_, err := svc.Create(authContext(), in)
	assertInvalidInput(t, err)
	if repo.createdTrip != nil {
		t.Error("repository Create must not be called on invalid input")
	}
}

func TestCreate_DaysTooHigh(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateInput()
	in.Days = 31

	_, err := svc.Create(authContext(), in)
	assertInvalidInput(t, err)
	if repo.createdTrip != nil {
		t.Error("repository Create must not be called on invalid input")
	}
}

func TestCreate_DefaultsCurrencyToEUR(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateInput()
	in.BudgetCurrency = ""

	if _, err := svc.Create(authContext(), in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdTrip.BudgetCurrency != "EUR" {
		t.Errorf("expected currency EUR, got %q", repo.createdTrip.BudgetCurrency)
	}
}

func TestCreate_DefaultsPaceToBalanced(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	in := validCreateInput()
	in.Pace = ""

	if _, err := svc.Create(authContext(), in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createdTrip.Pace != "balanced" {
		t.Errorf("expected pace balanced, got %q", repo.createdTrip.Pace)
	}
}

func TestGenerate_Success_SetsCompleted(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	svc := newTestService(repo, gen)

	got, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gen.called {
		t.Error("expected the generator to be invoked")
	}
	if got.Status != entity.StatusCompleted {
		t.Errorf("expected returned status COMPLETED, got %s", got.Status)
	}
	if repo.updateItinStatus != entity.StatusCompleted {
		t.Errorf("expected persisted status COMPLETED, got %s", repo.updateItinStatus)
	}
	if len(repo.updateItinRaw) == 0 {
		t.Error("expected the itinerary to be persisted as JSON")
	}
	// PROCESSING is set before generation.
	if len(repo.statusSeq) != 1 || repo.statusSeq[0] != entity.StatusProcessing {
		t.Errorf("expected status sequence [PROCESSING], got %v", repo.statusSeq)
	}
	if repo.getByIDUserID != testUserID() || repo.updateItinUserID != testUserID() {
		t.Fatalf("expected generate repository calls for user %s", testUserID())
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one itinerary version, got %d", len(repo.versions))
	}
	if repo.versions[0].Source != entity.ItineraryVersionSourceGenerated {
		t.Fatalf("expected GENERATED version, got %s", repo.versions[0].Source)
	}
	if repo.versions[0].VersionNumber != 1 {
		t.Fatalf("expected version number 1, got %d", repo.versions[0].VersionNumber)
	}
	if repo.versions[0].Metadata["generator"] != "full" {
		t.Fatalf("expected full generator metadata, got %+v", repo.versions[0].Metadata)
	}
}

func TestGenerate_PlaceEnrichmentEnabled_SavesEnrichedGeneratedVersion(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	generated := aggregate.Itinerary{
		Destination: "Rome",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Historic Rome",
			Items: []aggregate.ItineraryItem{{
				Time: "09:00",
				Type: "place",
				Name: "Colosseum",
			}},
		}},
	}
	enriched := generated
	enriched.Days = append([]aggregate.ItineraryDay(nil), generated.Days...)
	enriched.Days[0].Items = append([]aggregate.ItineraryItem(nil), generated.Days[0].Items...)
	enriched.Days[0].Items[0].Place = validPlaceRef()
	enriched.Days[0].Items[0].PlaceEnrichment = &aggregate.PlaceEnrichmentMeta{
		Status:     placeenrichment.StatusMatched,
		Confidence: 0.9,
		Query:      "Colosseum",
		Provider:   "mock",
		MatchedAt:  "2026-06-23T12:00:00Z",
		Reason:     "exact_name_match",
	}
	enricher := &mockPlaceEnrichmentProvider{
		result: &placeenrichment.EnrichItineraryResult{
			Itinerary: enriched,
			Stats:     placeenrichment.PlaceEnrichmentStats{Attempted: 1, Matched: 1},
		},
	}
	svc := New(repo, &mockGenerator{result: &generated}, zap.NewNop(), WithPlaceEnrichment(enricher, true, true))

	got, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Status != entity.StatusCompleted {
		t.Fatalf("expected completed trip, got %s", got.Status)
	}
	if !enricher.called {
		t.Fatal("expected place enrichment to be called")
	}
	if enricher.capturedInput.Destination != "Rome" {
		t.Fatalf("expected enrichment destination Rome, got %q", enricher.capturedInput.Destination)
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one generated version, got %d", len(repo.versions))
	}
	saved := decodeItinerary(t, repo.versions[0].Itinerary)
	item := saved.Days[0].Items[0]
	if item.Place == nil || item.Place.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected version to store enriched place, got %+v", item.Place)
	}
	if item.PlaceEnrichment == nil || item.PlaceEnrichment.Status != placeenrichment.StatusMatched {
		t.Fatalf("expected version to store matched enrichment metadata, got %+v", item.PlaceEnrichment)
	}
}

func TestGenerate_PlaceEnrichmentDisabled_SkipsEnrichment(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	enricher := &mockPlaceEnrichmentProvider{err: errors.New("should not be called")}
	svc := New(repo, &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}, zap.NewNop(), WithPlaceEnrichment(enricher, false, false))

	if _, err := svc.Generate(authContext(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enricher.called {
		t.Fatal("place enrichment must not be called when disabled")
	}
}

func TestGenerate_PlaceEnrichmentFailOpen_ContinuesWithOriginalItinerary(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	generated := &aggregate.Itinerary{
		Destination: "Rome",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Historic Rome",
			Items: []aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}},
		}},
	}
	enricher := &mockPlaceEnrichmentProvider{err: errors.New("place service down")}
	svc := New(repo, &mockGenerator{result: generated}, zap.NewNop(), WithPlaceEnrichment(enricher, true, true))

	got, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != entity.StatusCompleted {
		t.Fatalf("expected completed trip, got %s", got.Status)
	}
	saved := decodeItinerary(t, repo.updateItinRaw)
	if saved.Days[0].Items[0].Place != nil {
		t.Fatalf("expected original itinerary without place, got %+v", saved.Days[0].Items[0].Place)
	}
}

func TestGenerate_PlaceEnrichmentFailClosed_ReturnsDependencyError(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	generated := &aggregate.Itinerary{
		Destination: "Rome",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Historic Rome",
			Items: []aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}},
		}},
	}
	enricher := &mockPlaceEnrichmentProvider{err: errors.New("place service down")}
	svc := New(repo, &mockGenerator{result: generated}, zap.NewNop(), WithPlaceEnrichment(enricher, true, false))

	_, err := svc.Generate(authContext(), id)
	if err == nil {
		t.Fatal("expected enrichment dependency error")
	}
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected DependencyError, got %v", err)
	}
	if dependencyErr.Error() != "failed to enrich itinerary places" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if len(repo.versions) != 0 {
		t.Fatalf("fail-closed enrichment must not create versions, got %+v", repo.versions)
	}
	if len(repo.statusSeq) != 2 || repo.statusSeq[0] != entity.StatusProcessing || repo.statusSeq[1] != entity.StatusFailed {
		t.Fatalf("expected status sequence [PROCESSING FAILED], got %v", repo.statusSeq)
	}
}

func TestGenerate_UserContextSuccess_PassesProfileAndPreferencesToGenerator(t *testing.T) {
	id := uuid.New()
	userID := testUserID()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	displayName := "Test Traveler"
	walking := 8.0
	userContextProvider := &mockUserContextProvider{
		result: &usercontext.UserContext{
			Profile: &usercontext.UserProfile{
				UserID:            userID,
				DisplayName:       &displayName,
				PreferredCurrency: "EUR",
				PreferredLanguage: "en",
			},
			Preferences: &usercontext.UserPreferences{
				UserID:             userID,
				TravelStyles:       []string{"budget", "food", "hidden_gems"},
				MaxWalkingKmPerDay: &walking,
				Avoid:              []string{"nightclubs"},
			},
		},
	}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, true, true))

	_, err := svc.Generate(authContextWithToken("access-token-for-forwarding"), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !userContextProvider.called {
		t.Fatal("expected user context provider to be called")
	}
	if userContextProvider.capturedToken != "access-token-for-forwarding" {
		t.Fatalf("expected raw access token to be forwarded, got %q", userContextProvider.capturedToken)
	}
	if gen.capturedInput.UserProfile == nil || gen.capturedInput.UserProfile.DisplayName == nil || *gen.capturedInput.UserProfile.DisplayName != "Test Traveler" {
		t.Fatalf("expected profile in generator input, got %+v", gen.capturedInput.UserProfile)
	}
	if gen.capturedInput.UserPreferences == nil || len(gen.capturedInput.UserPreferences.Avoid) != 1 || gen.capturedInput.UserPreferences.Avoid[0] != "nightclubs" {
		t.Fatalf("expected preferences in generator input, got %+v", gen.capturedInput.UserPreferences)
	}
}

func TestGenerate_UserContextFailOpen_ContinuesWithoutPersonalization(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	userContextProvider := &mockUserContextProvider{err: errors.New("user service down")}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, true, true))

	got, err := svc.Generate(authContextWithToken("access-token-for-forwarding"), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Status != entity.StatusCompleted {
		t.Fatalf("expected completed trip, got %s", got.Status)
	}
	if !gen.called {
		t.Fatal("expected generator to be called when user context fails open")
	}
	if gen.capturedInput.UserProfile != nil || gen.capturedInput.UserPreferences != nil {
		t.Fatalf("expected generator input without context, got %+v", gen.capturedInput)
	}
}

func TestGenerate_UserContextFailClosed_ReturnsDependencyError(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	userContextProvider := &mockUserContextProvider{err: errors.New("user service down")}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, true, false))

	_, err := svc.Generate(authContextWithToken("access-token-for-forwarding"), id)
	if err == nil {
		t.Fatal("expected error")
	}
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected DependencyError, got %v", err)
	}
	if dependencyErr.Error() != "failed to load user preferences" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if gen.called {
		t.Fatal("generator must not be called when user context fails closed")
	}
	if len(repo.statusSeq) != 0 {
		t.Fatalf("trip should not enter PROCESSING before fail-closed context load, got %v", repo.statusSeq)
	}
}

func TestGenerate_UserContextDisabled_DoesNotCallProvider(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	userContextProvider := &mockUserContextProvider{err: errors.New("should not be called")}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, false, false))

	_, err := svc.Generate(authContextWithToken("access-token-for-forwarding"), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userContextProvider.called {
		t.Fatal("user context provider should not be called when disabled")
	}
}

func TestGenerate_WeatherContextSuccess_PassesForecastToGenerator(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:          id,
			Destination: "Rome",
			StartDate:   &startDate,
			Days:        3,
			Pace:        "balanced",
		},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	weatherProvider := &mockWeatherContextProvider{result: testWeatherForecast()}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, true))

	_, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !weatherProvider.called {
		t.Fatal("expected weather context provider to be called")
	}
	if weatherProvider.capturedDestination != "Rome" || weatherProvider.capturedStartDate != "2026-08-10" || weatherProvider.capturedDays != 3 {
		t.Fatalf("unexpected weather request: destination=%q startDate=%q days=%d", weatherProvider.capturedDestination, weatherProvider.capturedStartDate, weatherProvider.capturedDays)
	}
	if gen.capturedInput.WeatherForecast == nil || len(gen.capturedInput.WeatherForecast.Days) != 1 {
		t.Fatalf("expected weather forecast in generator input, got %+v", gen.capturedInput.WeatherForecast)
	}
}

func TestGenerate_WeatherContextFailOpen_ContinuesWithoutWeather(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", StartDate: &startDate, Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	weatherProvider := &mockWeatherContextProvider{err: errors.New("weather service down")}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, true))

	got, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Status != entity.StatusCompleted {
		t.Fatalf("expected completed trip, got %s", got.Status)
	}
	if !gen.called {
		t.Fatal("expected generator to be called when weather context fails open")
	}
	if gen.capturedInput.WeatherForecast != nil {
		t.Fatalf("expected generator input without weather, got %+v", gen.capturedInput.WeatherForecast)
	}
}

func TestGenerate_WeatherContextFailClosed_ReturnsDependencyError(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", StartDate: &startDate, Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	weatherProvider := &mockWeatherContextProvider{err: errors.New("weather service down")}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, false))

	_, err := svc.Generate(authContext(), id)
	if err == nil {
		t.Fatal("expected error")
	}
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected DependencyError, got %v", err)
	}
	if dependencyErr.Error() != "failed to load weather forecast" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if gen.called {
		t.Fatal("generator must not be called when weather context fails closed")
	}
	if len(repo.statusSeq) != 0 {
		t.Fatalf("trip should not enter PROCESSING before fail-closed weather load, got %v", repo.statusSeq)
	}
}

func TestGenerate_WeatherContextDisabled_DoesNotCallProvider(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", StartDate: &startDate, Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	weatherProvider := &mockWeatherContextProvider{err: errors.New("should not be called")}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, false, false))

	_, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if weatherProvider.called {
		t.Fatal("weather context provider should not be called when disabled")
	}
}

func TestGenerate_MissingStartDateSkipsWeatherContext(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	weatherProvider := &mockWeatherContextProvider{err: errors.New("should not be called")}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, false))

	_, err := svc.Generate(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if weatherProvider.called {
		t.Fatal("weather context provider should not be called without startDate")
	}
	if gen.capturedInput.WeatherForecast != nil {
		t.Fatalf("expected no weather forecast in generator input, got %+v", gen.capturedInput.WeatherForecast)
	}
}

func TestGenerate_UserContextLogging_DoesNotLogAccessToken(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2, Pace: "balanced"},
	}
	gen := &mockGenerator{result: &aggregate.Itinerary{Destination: "Rome"}}
	observedCore, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(observedCore)
	userContextProvider := &mockUserContextProvider{err: errors.New("user service down")}
	svc := New(repo, gen, logger, WithUserContext(userContextProvider, true, true))

	_, err := svc.Generate(authContextWithToken("secret-access-token"), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, "secret-access-token") {
			t.Fatalf("access token leaked in log message: %q", entry.Message)
		}
		for _, field := range entry.Context {
			if strings.Contains(field.String, "secret-access-token") {
				t.Fatalf("access token leaked in log field %s", field.Key)
			}
		}
	}
}

func TestGenerate_GeneratorError_SetsFailed(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Days: 2},
	}
	gen := &mockGenerator{err: errors.New("generation boom")}
	svc := newTestService(repo, gen)

	_, err := svc.Generate(authContext(), id)
	if err == nil {
		t.Fatal("expected an error when the generator fails")
	}
	want := []entity.Status{entity.StatusProcessing, entity.StatusFailed}
	if len(repo.statusSeq) != len(want) {
		t.Fatalf("expected status sequence %v, got %v", want, repo.statusSeq)
	}
	for i := range want {
		if repo.statusSeq[i] != want[i] {
			t.Fatalf("expected status sequence %v, got %v", want, repo.statusSeq)
		}
		if repo.statusUserIDs[i] != testUserID() {
			t.Fatalf("expected status update %d for user %s, got %s", i, testUserID(), repo.statusUserIDs[i])
		}
	}
	if len(repo.versions) != 0 {
		t.Fatalf("failed generation must not create itinerary versions, got %+v", repo.versions)
	}
}

func TestUpdateItinerary_CreatesManualEditVersion(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		Itinerary: validExistingItineraryRaw(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != entity.StatusCompleted || repo.updateItinStatus != entity.StatusCompleted {
		t.Fatalf("expected completed update, got returned=%s persisted=%s", got.Status, repo.updateItinStatus)
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one itinerary version, got %d", len(repo.versions))
	}
	if repo.versions[0].Source != entity.ItineraryVersionSourceManualEdit {
		t.Fatalf("expected MANUAL_EDIT version, got %s", repo.versions[0].Source)
	}
	if len(repo.versions[0].Metadata) != 0 {
		t.Fatalf("expected empty metadata, got %+v", repo.versions[0].Metadata)
	}
}

func TestUpdateItinerary_WithItemPlaceSucceedsAndVersionsPlaceMetadata(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		Itinerary: validItineraryWithPlaceRaw(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := decodeItinerary(t, got.Itinerary)
	place := updated.Days[0].Items[0].Place
	if place == nil || place.ProviderPlaceID != "mock-colosseum-rome" || place.Name != "Colosseum" {
		t.Fatalf("expected persisted place metadata, got %+v", place)
	}
	if len(place.OpeningHours) == 0 || place.OpeningHours[0].DayOfWeek != 1 {
		t.Fatalf("expected persisted opening hours, got %+v", place.OpeningHours)
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one itinerary version, got %d", len(repo.versions))
	}
	version := decodeItinerary(t, repo.versions[0].Itinerary)
	versionPlace := version.Days[0].Items[0].Place
	if versionPlace == nil || versionPlace.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected version to store place metadata, got %+v", versionPlace)
	}
	if len(versionPlace.OpeningHours) == 0 || versionPlace.OpeningHours[0].Open != "08:30" {
		t.Fatalf("expected version to store opening hours, got %+v", versionPlace.OpeningHours)
	}
}

func TestUpdateItinerary_WithOldPlaceWithoutOpeningHoursStillSucceeds(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		Itinerary: validItineraryWithPlaceWithoutOpeningHoursRaw(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := decodeItinerary(t, got.Itinerary)
	place := updated.Days[0].Items[0].Place
	if place == nil || place.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected old place metadata to persist, got %+v", place)
	}
	if len(place.OpeningHours) != 0 {
		t.Fatalf("expected old place metadata without opening hours to remain valid, got %+v", place.OpeningHours)
	}
}

func TestUpdateItinerary_WithPlaceEnrichmentSucceedsAndVersionsMetadata(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		Itinerary: validItineraryWithPlaceEnrichmentRaw(t),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := decodeItinerary(t, got.Itinerary)
	meta := updated.Days[0].Items[0].PlaceEnrichment
	if meta == nil || meta.Status != placeenrichment.StatusMatched || meta.Query != "Colosseum" {
		t.Fatalf("expected persisted place enrichment metadata, got %+v", meta)
	}
	version := decodeItinerary(t, repo.versions[0].Itinerary)
	versionMeta := version.Days[0].Items[0].PlaceEnrichment
	if versionMeta == nil || versionMeta.Status != placeenrichment.StatusMatched {
		t.Fatalf("expected version to store place enrichment metadata, got %+v", versionMeta)
	}
}

func TestUpdateItinerary_WithPlaceEnrichmentReviewStatusSucceedsAndVersionsMetadata(t *testing.T) {
	statuses := []string{
		placeenrichment.ReviewStatusPending,
		placeenrichment.ReviewStatusAccepted,
		placeenrichment.ReviewStatusChanged,
		placeenrichment.ReviewStatusRemoved,
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			id := uuid.New()
			repo := &mockRepo{}
			svc := newTestService(repo, &mockGenerator{})

			got, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
				Itinerary: validItineraryWithPlaceEnrichmentReviewRaw(t, status),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			updated := decodeItinerary(t, got.Itinerary)
			meta := updated.Days[0].Items[0].PlaceEnrichment
			if meta == nil || meta.ReviewStatus != status {
				t.Fatalf("expected persisted review status %q, got %+v", status, meta)
			}

			version := decodeItinerary(t, repo.versions[0].Itinerary)
			versionMeta := version.Days[0].Items[0].PlaceEnrichment
			if versionMeta == nil || versionMeta.ReviewStatus != status {
				t.Fatalf("expected version review status %q, got %+v", status, versionMeta)
			}
		})
	}
}

func TestUpdateItinerary_InvalidPlaceEnrichmentReturnsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*aggregate.PlaceEnrichmentMeta)
	}{
		{
			name: "invalid status",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Status = "manual"
			},
		},
		{
			name: "invalid review status",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.ReviewStatus = "ignored"
			},
		},
		{
			name: "negative confidence",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Confidence = -0.1
			},
		},
		{
			name: "confidence over one",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Confidence = 1.1
			},
		},
		{
			name: "query too long",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Query = strings.Repeat("q", maxPlaceEnrichmentQuery+1)
			},
		},
		{
			name: "provider too long",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Provider = strings.Repeat("p", maxPlaceEnrichmentProvider+1)
			},
		},
		{
			name: "reason too long",
			mutate: func(meta *aggregate.PlaceEnrichmentMeta) {
				meta.Reason = strings.Repeat("r", maxPlaceEnrichmentReason+1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{}
			svc := newTestService(repo, &mockGenerator{})

			_, err := svc.UpdateItinerary(authContext(), uuid.New(), appdto.UpdateItineraryInput{
				Itinerary: itineraryWithMutatedPlaceEnrichmentRaw(t, tt.mutate),
			})
			assertInvalidInput(t, err)
			if len(repo.versions) != 0 {
				t.Fatalf("invalid place enrichment metadata must not create versions, got %+v", repo.versions)
			}
		})
	}
}

func TestUpdateItinerary_InvalidPlaceMetadataReturnsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*aggregate.PlaceRef)
	}{
		{
			name: "invalid latitude",
			mutate: func(place *aggregate.PlaceRef) {
				value := 91.0
				place.Latitude = &value
			},
		},
		{
			name: "invalid longitude",
			mutate: func(place *aggregate.PlaceRef) {
				value := -181.0
				place.Longitude = &value
			},
		},
		{
			name: "invalid rating",
			mutate: func(place *aggregate.PlaceRef) {
				value := 5.1
				place.Rating = &value
			},
		},
		{
			name: "invalid opening hours day",
			mutate: func(place *aggregate.PlaceRef) {
				place.OpeningHours[0].DayOfWeek = 8
			},
		},
		{
			name: "invalid opening time format",
			mutate: func(place *aggregate.PlaceRef) {
				place.OpeningHours[0].Open = "9:00"
			},
		},
		{
			name: "invalid closing time format",
			mutate: func(place *aggregate.PlaceRef) {
				place.OpeningHours[0].Close = "24:00"
			},
		},
		{
			name: "opening after close",
			mutate: func(place *aggregate.PlaceRef) {
				place.OpeningHours[0].Open = "19:15"
				place.OpeningHours[0].Close = "08:30"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{}
			svc := newTestService(repo, &mockGenerator{})

			_, err := svc.UpdateItinerary(authContext(), uuid.New(), appdto.UpdateItineraryInput{
				Itinerary: itineraryWithMutatedPlaceRaw(t, tt.mutate),
			})
			assertInvalidInput(t, err)
			if len(repo.versions) != 0 {
				t.Fatalf("invalid place metadata must not create versions, got %+v", repo.versions)
			}
		})
	}
}

func TestUpdateItinerary_InvalidPayloadDoesNotCreateVersion(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		Itinerary: json.RawMessage(`{"days":[]}`),
	})
	assertInvalidInput(t, err)
	if len(repo.versions) != 0 {
		t.Fatalf("invalid manual edit must not create versions, got %+v", repo.versions)
	}
}

func TestGet_ReturnsSavedPlaceMetadata(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{getByIDResult: &entity.Trip{ID: id, Itinerary: validItineraryWithPlaceRaw(t)}}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.Get(authContext(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	itinerary := decodeItinerary(t, got.Itinerary)
	place := itinerary.Days[0].Items[0].Place
	if place == nil || place.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected GET trip to return saved place metadata, got %+v", place)
	}
	if len(place.OpeningHours) == 0 || place.OpeningHours[0].Close != "19:15" {
		t.Fatalf("expected GET trip to return saved opening hours, got %+v", place.OpeningHours)
	}
}

func TestRegenerateDay_ReplacesOnlySelectedDay(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{
		dayResult: &aggregate.ItineraryDay{
			Day:   99,
			Title: "  Cheaper food day  ",
			Items: []aggregate.ItineraryItem{
				{Time: " 10:00 ", Type: " food ", Name: " Local bakery ", Note: "  Budget start  "},
			},
		},
	}
	svc := newTestService(repo, gen)

	got, err := svc.RegenerateDay(authContext(), id, 2, appdto.RegenerateItineraryPartInput{Instruction: " make it cheaper "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Status != entity.StatusCompleted || repo.updateItinStatus != entity.StatusCompleted {
		t.Fatalf("expected completed update, got returned=%s persisted=%s", got.Status, repo.updateItinStatus)
	}
	if !gen.regenerateDayCalled {
		t.Fatal("expected RegenerateDay to be called")
	}
	if gen.capturedDayInput.DayNumber != 2 || gen.capturedDayInput.Instruction != "make it cheaper" {
		t.Fatalf("unexpected generator input: %+v", gen.capturedDayInput)
	}

	updated := decodeItinerary(t, repo.updateItinRaw)
	if len(updated.Days) != 2 {
		t.Fatalf("expected two days, got %+v", updated.Days)
	}
	if updated.Days[0].Title != "Original Day 1" || updated.Days[0].Items[0].Name != "Original Item 1A" {
		t.Fatalf("day 1 should be preserved, got %+v", updated.Days[0])
	}
	if updated.Days[1].Day != 2 || updated.Days[1].Title != "Cheaper food day" {
		t.Fatalf("day 2 should be replaced and normalized, got %+v", updated.Days[1])
	}
	if updated.Days[1].Items[0].Name != "Local bakery" {
		t.Fatalf("expected replacement item, got %+v", updated.Days[1].Items[0])
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one itinerary version, got %d", len(repo.versions))
	}
	if repo.versions[0].Source != entity.ItineraryVersionSourceRegenerateDay {
		t.Fatalf("expected REGENERATE_DAY version, got %s", repo.versions[0].Source)
	}
	if repo.versions[0].Metadata["dayNumber"] != float64(2) && repo.versions[0].Metadata["dayNumber"] != 2 {
		t.Fatalf("expected dayNumber metadata, got %+v", repo.versions[0].Metadata)
	}
	if repo.versions[0].Metadata["instructionPresent"] != true {
		t.Fatalf("expected instructionPresent metadata, got %+v", repo.versions[0].Metadata)
	}
}

func TestRegenerateItem_ReplacesOnlySelectedItem(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{
		itemResult: &aggregate.ItineraryItem{
			Time: " 12:30 ",
			Type: " food ",
			Name: " Local trattoria ",
			Note: "  Cheaper local option  ",
		},
	}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateItem(authContext(), id, 1, 1, appdto.RegenerateItineraryPartInput{Instruction: "avoid museums"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !gen.regenerateItemCalled {
		t.Fatal("expected RegenerateItem to be called")
	}
	if gen.capturedItemInput.DayNumber != 1 || gen.capturedItemInput.ItemIndex != 1 || gen.capturedItemInput.Instruction != "avoid museums" {
		t.Fatalf("unexpected generator input: %+v", gen.capturedItemInput)
	}

	updated := decodeItinerary(t, repo.updateItinRaw)
	if updated.Days[0].Items[0].Name != "Original Item 1A" {
		t.Fatalf("item 0 should be preserved, got %+v", updated.Days[0].Items[0])
	}
	if updated.Days[0].Items[1].Name != "Local trattoria" || updated.Days[0].Items[1].Type != "food" {
		t.Fatalf("item 1 should be replaced and normalized, got %+v", updated.Days[0].Items[1])
	}
	if updated.Days[1].Title != "Original Day 2" || updated.Days[1].Items[0].Name != "Original Item 2A" {
		t.Fatalf("day 2 should be preserved, got %+v", updated.Days[1])
	}
	if len(repo.versions) != 1 {
		t.Fatalf("expected one itinerary version, got %d", len(repo.versions))
	}
	if repo.versions[0].Source != entity.ItineraryVersionSourceRegenerateItem {
		t.Fatalf("expected REGENERATE_ITEM version, got %s", repo.versions[0].Source)
	}
	if repo.versions[0].Metadata["dayNumber"] != float64(1) && repo.versions[0].Metadata["dayNumber"] != 1 {
		t.Fatalf("expected dayNumber metadata, got %+v", repo.versions[0].Metadata)
	}
	if repo.versions[0].Metadata["itemIndex"] != float64(1) && repo.versions[0].Metadata["itemIndex"] != 1 {
		t.Fatalf("expected itemIndex metadata, got %+v", repo.versions[0].Metadata)
	}
	if repo.versions[0].Metadata["instructionPresent"] != true {
		t.Fatalf("expected instructionPresent metadata, got %+v", repo.versions[0].Metadata)
	}
}

func TestRegenerateDay_WeatherContextSuccess_PassesForecastToGenerator(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:          id,
			Destination: "Rome",
			StartDate:   &startDate,
			Days:        2,
			Itinerary:   validExistingItineraryRaw(t),
		},
	}
	gen := &mockGenerator{}
	weatherProvider := &mockWeatherContextProvider{result: testWeatherForecast()}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, true))

	_, err := svc.RegenerateDay(authContext(), id, 1, appdto.RegenerateItineraryPartInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !weatherProvider.called {
		t.Fatal("expected weather context provider to be called")
	}
	if weatherProvider.capturedDestination != "Rome" || weatherProvider.capturedStartDate != "2026-08-10" || weatherProvider.capturedDays != 2 {
		t.Fatalf("unexpected weather request: destination=%q startDate=%q days=%d", weatherProvider.capturedDestination, weatherProvider.capturedStartDate, weatherProvider.capturedDays)
	}
	if gen.capturedDayInput.WeatherForecast == nil {
		t.Fatal("expected weather forecast in regenerate day input")
	}
}

func TestRegenerateItem_WeatherContextSuccess_PassesForecastToGenerator(t *testing.T) {
	id := uuid.New()
	startDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{
			ID:          id,
			Destination: "Rome",
			StartDate:   &startDate,
			Days:        2,
			Itinerary:   validExistingItineraryRaw(t),
		},
	}
	gen := &mockGenerator{}
	weatherProvider := &mockWeatherContextProvider{result: testWeatherForecast()}
	svc := New(repo, gen, zap.NewNop(), WithWeatherContext(weatherProvider, true, true))

	_, err := svc.RegenerateItem(authContext(), id, 1, 0, appdto.RegenerateItineraryPartInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !weatherProvider.called {
		t.Fatal("expected weather context provider to be called")
	}
	if gen.capturedItemInput.WeatherForecast == nil {
		t.Fatal("expected weather forecast in regenerate item input")
	}
}

func TestRegenerateDay_MissingItineraryReturnsInvalidInput(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{getByIDResult: &entity.Trip{ID: id, Destination: "Rome"}}
	gen := &mockGenerator{}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateDay(authContext(), id, 1, appdto.RegenerateItineraryPartInput{})
	assertInvalidInput(t, err)
	if gen.regenerateDayCalled {
		t.Fatal("generator must not be called for missing current itinerary")
	}
	if repo.updateItinRaw != nil {
		t.Fatal("itinerary must not be saved for missing current itinerary")
	}
}

func TestRegenerateDay_InvalidDayNumberReturnsInvalidInput(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateDay(authContext(), id, 3, appdto.RegenerateItineraryPartInput{})
	assertInvalidInput(t, err)
	if gen.regenerateDayCalled {
		t.Fatal("generator must not be called for invalid day number")
	}
}

func TestRegenerateItem_InvalidItemIndexReturnsInvalidInput(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateItem(authContext(), id, 1, 9, appdto.RegenerateItineraryPartInput{})
	assertInvalidInput(t, err)
	if gen.regenerateItemCalled {
		t.Fatal("generator must not be called for invalid item index")
	}
}

func TestRegenerateDay_InstructionTooLongReturnsInvalidInput(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateDay(authContext(), id, 1, appdto.RegenerateItineraryPartInput{Instruction: strings.Repeat("x", maxInstructionLength+1)})
	assertInvalidInput(t, err)
	if gen.regenerateDayCalled {
		t.Fatal("generator must not be called for overlong instruction")
	}
	if repo.updateItinRaw != nil {
		t.Fatal("itinerary must not be saved for overlong instruction")
	}
}

func TestRegenerateDay_InvalidAIReplacementDoesNotSave(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{
		dayResult: &aggregate.ItineraryDay{Day: 1, Title: " ", Items: []aggregate.ItineraryItem{{Time: "10:00", Type: "activity", Name: "Walk"}}},
	}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateDay(authContext(), id, 1, appdto.RegenerateItineraryPartInput{})
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected dependency error, got %v", err)
	}
	if dependencyErr.Error() != "AI returned invalid replacement" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if repo.updateItinRaw != nil {
		t.Fatal("invalid replacement must not be saved")
	}
}

func TestRegenerateItem_InvalidAIReplacementDoesNotSave(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{
		itemResult: &aggregate.ItineraryItem{Time: "", Type: "food", Name: "Lunch"},
	}
	svc := newTestService(repo, gen)

	_, err := svc.RegenerateItem(authContext(), id, 1, 0, appdto.RegenerateItineraryPartInput{})
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected dependency error, got %v", err)
	}
	if dependencyErr.Error() != "AI returned invalid replacement" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if repo.updateItinRaw != nil {
		t.Fatal("invalid replacement must not be saved")
	}
}

func TestRegenerateDay_UserContextFailOpen_ContinuesWithoutPersonalization(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{}
	userContextProvider := &mockUserContextProvider{err: errors.New("user service down")}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, true, true))

	_, err := svc.RegenerateDay(authContextWithToken("access-token-for-forwarding"), id, 1, appdto.RegenerateItineraryPartInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gen.regenerateDayCalled {
		t.Fatal("expected generator to be called when user context fails open")
	}
	if gen.capturedDayInput.UserProfile != nil || gen.capturedDayInput.UserPreferences != nil {
		t.Fatalf("expected generator input without context, got %+v", gen.capturedDayInput)
	}
}

func TestRegenerateItem_UserContextFailClosed_ReturnsDependencyError(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: id, Destination: "Rome", Itinerary: validExistingItineraryRaw(t)},
	}
	gen := &mockGenerator{}
	userContextProvider := &mockUserContextProvider{err: errors.New("user service down")}
	svc := New(repo, gen, zap.NewNop(), WithUserContext(userContextProvider, true, false))

	_, err := svc.RegenerateItem(authContextWithToken("access-token-for-forwarding"), id, 1, 0, appdto.RegenerateItineraryPartInput{})
	var dependencyErr *apperrs.DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("expected dependency error, got %v", err)
	}
	if dependencyErr.Error() != "failed to load user preferences" {
		t.Fatalf("unexpected dependency error: %v", dependencyErr)
	}
	if gen.regenerateItemCalled {
		t.Fatal("generator must not be called when user context fails closed")
	}
	if repo.updateItinRaw != nil {
		t.Fatal("itinerary must not be saved when user context fails closed")
	}
}

func TestGet_NotFound(t *testing.T) {
	wantErr := errors.New("trip not found")
	repo := &mockRepo{getByIDErr: wantErr}
	svc := newTestService(repo, &mockGenerator{})

	_, err := svc.Get(authContext(), uuid.New())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the repository error to propagate, got %v", err)
	}
}

func TestList_AppliesDefaults(t *testing.T) {
	repo := &mockRepo{listResult: []entity.Trip{}}
	svc := newTestService(repo, &mockGenerator{})

	_, limit, offset, err := svc.List(authContext(), 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != defaultLimit {
		t.Errorf("expected default limit %d, got %d", defaultLimit, limit)
	}
	if offset != 0 {
		t.Errorf("expected offset 0, got %d", offset)
	}
	if repo.listLimit != defaultLimit || repo.listOffset != 0 {
		t.Errorf("expected repo called with (%d, 0), got (%d, %d)", defaultLimit, repo.listLimit, repo.listOffset)
	}
	if repo.listUserID != testUserID() {
		t.Errorf("expected list for user %s, got %s", testUserID(), repo.listUserID)
	}
}

func TestList_RejectsInvalidLimit(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	_, _, _, err := svc.List(authContext(), maxLimit+1, 0)
	assertInvalidInput(t, err)
}

func TestList_RejectsNegativeOffset(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	_, _, _, err := svc.List(authContext(), 20, -1)
	assertInvalidInput(t, err)
}

func TestListItineraryVersions_ReturnsOwnedTripVersions(t *testing.T) {
	tripID := uuid.New()
	otherTripID := uuid.New()
	userID := testUserID()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: tripID, UserID: &userID},
		versions: []entity.ItineraryVersion{
			{ID: uuid.New(), TripID: tripID, UserID: userID, VersionNumber: 2, Source: entity.ItineraryVersionSourceManualEdit},
			{ID: uuid.New(), TripID: otherTripID, UserID: userID, VersionNumber: 1, Source: entity.ItineraryVersionSourceGenerated},
		},
	}
	svc := newTestService(repo, &mockGenerator{})

	versions, limit, offset, err := svc.ListItineraryVersions(authContext(), tripID, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limit != defaultLimit || offset != 0 {
		t.Fatalf("expected default pagination, got limit=%d offset=%d", limit, offset)
	}
	if len(versions) != 1 || versions[0].TripID != tripID {
		t.Fatalf("expected only requested trip versions, got %+v", versions)
	}
	if repo.getByIDUserID != userID || repo.listVersionsUser != userID || repo.listVersionsTrip != tripID {
		t.Fatalf("expected owner-scoped repository calls, got trip=%s user=%s", repo.listVersionsTrip, repo.listVersionsUser)
	}
}

func TestListItineraryVersions_RejectsInvalidPagination(t *testing.T) {
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	_, _, _, err := svc.ListItineraryVersions(authContext(), uuid.New(), maxLimit+1, 0)
	assertInvalidInput(t, err)

	_, _, _, err = svc.ListItineraryVersions(authContext(), uuid.New(), 20, -1)
	assertInvalidInput(t, err)
}

func TestGetItineraryVersion_ReturnsDetailForOwner(t *testing.T) {
	tripID := uuid.New()
	versionID := uuid.New()
	userID := testUserID()
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: tripID, UserID: &userID},
		versions: []entity.ItineraryVersion{
			{
				ID:            versionID,
				TripID:        tripID,
				UserID:        userID,
				VersionNumber: 1,
				Source:        entity.ItineraryVersionSourceGenerated,
				Itinerary:     validExistingItineraryRaw(t),
			},
		},
	}
	svc := newTestService(repo, &mockGenerator{})

	version, err := svc.GetItineraryVersion(authContext(), tripID, versionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version.ID != versionID || len(version.Itinerary) == 0 {
		t.Fatalf("expected version detail with itinerary, got %+v", version)
	}
	if repo.getVersionID != versionID || repo.getVersionTripID != tripID || repo.getVersionUserID != userID {
		t.Fatalf("expected owner-scoped version lookup, got version=%s trip=%s user=%s", repo.getVersionID, repo.getVersionTripID, repo.getVersionUserID)
	}
}

func TestRestoreItineraryVersion_UpdatesTripAndCreatesRestoredVersion(t *testing.T) {
	tripID := uuid.New()
	versionID := uuid.New()
	userID := testUserID()
	original := validExistingItineraryRaw(t)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: tripID, UserID: &userID},
		versions: []entity.ItineraryVersion{
			{
				ID:            versionID,
				TripID:        tripID,
				UserID:        userID,
				VersionNumber: 1,
				Source:        entity.ItineraryVersionSourceGenerated,
				Itinerary:     original,
			},
		},
	}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.RestoreItineraryVersion(authContext(), tripID, versionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != entity.StatusCompleted || repo.updateItinStatus != entity.StatusCompleted {
		t.Fatalf("expected completed restore, got returned=%s persisted=%s", got.Status, repo.updateItinStatus)
	}
	if len(repo.versions) != 2 {
		t.Fatalf("restore should append a new version without deleting old ones, got %d", len(repo.versions))
	}
	restored := repo.versions[1]
	if restored.Source != entity.ItineraryVersionSourceRestored {
		t.Fatalf("expected RESTORED version, got %s", restored.Source)
	}
	if restored.VersionNumber != 2 {
		t.Fatalf("expected next version number 2, got %d", restored.VersionNumber)
	}
	if restored.Metadata["restoredFromVersionId"] != versionID.String() || restored.Metadata["restoredFromVersionNumber"] != 1 {
		t.Fatalf("unexpected restore metadata: %+v", restored.Metadata)
	}
}

func TestRestoreItineraryVersion_RestoresPlaceMetadata(t *testing.T) {
	tripID := uuid.New()
	versionID := uuid.New()
	userID := testUserID()
	original := validItineraryWithPlaceRaw(t)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: tripID, UserID: &userID},
		versions: []entity.ItineraryVersion{
			{
				ID:            versionID,
				TripID:        tripID,
				UserID:        userID,
				VersionNumber: 1,
				Source:        entity.ItineraryVersionSourceManualEdit,
				Itinerary:     original,
			},
		},
	}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.RestoreItineraryVersion(authContext(), tripID, versionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	itinerary := decodeItinerary(t, got.Itinerary)
	place := itinerary.Days[0].Items[0].Place
	if place == nil || place.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected restored trip to include place metadata, got %+v", place)
	}
	if len(place.OpeningHours) == 0 || place.OpeningHours[0].Open != "08:30" {
		t.Fatalf("expected restored trip to include opening hours, got %+v", place.OpeningHours)
	}
	restoredVersion := decodeItinerary(t, repo.versions[1].Itinerary)
	restoredPlace := restoredVersion.Days[0].Items[0].Place
	if restoredPlace == nil || restoredPlace.ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected restored version to include place metadata, got %+v", restoredPlace)
	}
	if len(restoredPlace.OpeningHours) == 0 || restoredPlace.OpeningHours[0].Close != "19:15" {
		t.Fatalf("expected restored version to include opening hours, got %+v", restoredPlace.OpeningHours)
	}
}

func TestRestoreItineraryVersion_RestoresPlaceEnrichmentReviewStatus(t *testing.T) {
	tripID := uuid.New()
	versionID := uuid.New()
	userID := testUserID()
	original := validItineraryWithPlaceEnrichmentReviewRaw(t, placeenrichment.ReviewStatusChanged)
	repo := &mockRepo{
		getByIDResult: &entity.Trip{ID: tripID, UserID: &userID},
		versions: []entity.ItineraryVersion{
			{
				ID:            versionID,
				TripID:        tripID,
				UserID:        userID,
				VersionNumber: 1,
				Source:        entity.ItineraryVersionSourceManualEdit,
				Itinerary:     original,
			},
		},
	}
	svc := newTestService(repo, &mockGenerator{})

	got, err := svc.RestoreItineraryVersion(authContext(), tripID, versionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	itinerary := decodeItinerary(t, got.Itinerary)
	meta := itinerary.Days[0].Items[0].PlaceEnrichment
	if meta == nil || meta.ReviewStatus != placeenrichment.ReviewStatusChanged {
		t.Fatalf("expected restored trip to include review status, got %+v", meta)
	}

	restoredVersion := decodeItinerary(t, repo.versions[1].Itinerary)
	restoredMeta := restoredVersion.Days[0].Items[0].PlaceEnrichment
	if restoredMeta == nil || restoredMeta.ReviewStatus != placeenrichment.ReviewStatusChanged {
		t.Fatalf("expected restored version to include review status, got %+v", restoredMeta)
	}
}

func TestItineraryVersionNumbersIncrementPerTrip(t *testing.T) {
	userID := testUserID()
	firstTripID := uuid.New()
	secondTripID := uuid.New()
	repo := &mockRepo{}
	svc := newTestService(repo, &mockGenerator{})

	if _, err := svc.UpdateItinerary(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID}), firstTripID, appdto.UpdateItineraryInput{Itinerary: validExistingItineraryRaw(t)}); err != nil {
		t.Fatalf("first trip first update: %v", err)
	}
	if _, err := svc.UpdateItinerary(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID}), firstTripID, appdto.UpdateItineraryInput{Itinerary: validExistingItineraryRaw(t)}); err != nil {
		t.Fatalf("first trip second update: %v", err)
	}
	if _, err := svc.UpdateItinerary(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID}), secondTripID, appdto.UpdateItineraryInput{Itinerary: validExistingItineraryRaw(t)}); err != nil {
		t.Fatalf("second trip first update: %v", err)
	}

	if repo.versions[0].VersionNumber != 1 || repo.versions[1].VersionNumber != 2 || repo.versions[2].VersionNumber != 1 {
		t.Fatalf("expected per-trip version numbering [1,2,1], got [%d,%d,%d]", repo.versions[0].VersionNumber, repo.versions[1].VersionNumber, repo.versions[2].VersionNumber)
	}
}

func validExistingItineraryRaw(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(aggregate.Itinerary{
		Destination: "Rome",
		Summary:     "Original summary",
		Travelers:   2,
		Pace:        "balanced",
		Currency:    "EUR",
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Original Day 1",
				Items: []aggregate.ItineraryItem{
					{Time: "09:00", Type: "activity", Name: "Original Item 1A", Note: "Keep 1A"},
					{Time: "12:00", Type: "food", Name: "Original Item 1B", Note: "Keep 1B"},
				},
			},
			{
				Day:   2,
				Title: "Original Day 2",
				Items: []aggregate.ItineraryItem{
					{Time: "09:30", Type: "place", Name: "Original Item 2A", Note: "Keep 2A"},
					{Time: "13:00", Type: "food", Name: "Original Item 2B", Note: "Keep 2B"},
				},
			},
		},
		GeneratedAt: time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC),
		Source:      "test",
	})
	if err != nil {
		t.Fatalf("marshal itinerary: %v", err)
	}
	return raw
}

func validItineraryWithPlaceRaw(t *testing.T) json.RawMessage {
	t.Helper()
	raw := validExistingItineraryRaw(t)
	itinerary := decodeItinerary(t, raw)
	itinerary.Days[0].Items[0].Place = validPlaceRef()

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal itinerary with place: %v", err)
	}
	return out
}

func validItineraryWithPlaceWithoutOpeningHoursRaw(t *testing.T) json.RawMessage {
	t.Helper()
	raw := validExistingItineraryRaw(t)
	itinerary := decodeItinerary(t, raw)
	place := validPlaceRef()
	place.OpeningHours = nil
	itinerary.Days[0].Items[0].Place = place

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal itinerary with old place: %v", err)
	}
	return out
}

func validItineraryWithPlaceEnrichmentRaw(t *testing.T) json.RawMessage {
	t.Helper()
	raw := validItineraryWithPlaceRaw(t)
	itinerary := decodeItinerary(t, raw)
	itinerary.Days[0].Items[0].PlaceEnrichment = validPlaceEnrichmentMeta()

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal itinerary with place enrichment: %v", err)
	}
	return out
}

func validItineraryWithPlaceEnrichmentReviewRaw(t *testing.T, status string) json.RawMessage {
	t.Helper()
	raw := validItineraryWithPlaceEnrichmentRaw(t)
	itinerary := decodeItinerary(t, raw)
	itinerary.Days[0].Items[0].PlaceEnrichment.ReviewStatus = status

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal itinerary with place enrichment review status: %v", err)
	}
	return out
}

func itineraryWithMutatedPlaceRaw(t *testing.T, mutate func(*aggregate.PlaceRef)) json.RawMessage {
	t.Helper()
	raw := validExistingItineraryRaw(t)
	itinerary := decodeItinerary(t, raw)
	place := validPlaceRef()
	mutate(place)
	itinerary.Days[0].Items[0].Place = place

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal mutated place itinerary: %v", err)
	}
	return out
}

func itineraryWithMutatedPlaceEnrichmentRaw(t *testing.T, mutate func(*aggregate.PlaceEnrichmentMeta)) json.RawMessage {
	t.Helper()
	raw := validItineraryWithPlaceRaw(t)
	itinerary := decodeItinerary(t, raw)
	meta := validPlaceEnrichmentMeta()
	mutate(meta)
	itinerary.Days[0].Items[0].PlaceEnrichment = meta

	out, err := json.Marshal(itinerary)
	if err != nil {
		t.Fatalf("marshal mutated place enrichment itinerary: %v", err)
	}
	return out
}

func validPlaceRef() *aggregate.PlaceRef {
	lat := 41.8902
	lng := 12.4922
	rating := 4.7
	ratingCount := 120000
	return &aggregate.PlaceRef{
		Provider:        "mock",
		ProviderPlaceID: "mock-colosseum-rome",
		Name:            "Colosseum",
		Address:         "Piazza del Colosseo, 1, 00184 Roma RM, Italy",
		Latitude:        &lat,
		Longitude:       &lng,
		Rating:          &rating,
		RatingCount:     &ratingCount,
		MapURL:          "https://maps.example.com/mock-colosseum-rome",
		Category:        "landmark",
		Website:         "https://example.com/colosseum",
		OpeningHours: []aggregate.OpeningHoursInterval{
			{DayOfWeek: 1, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 2, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 3, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 4, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 5, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 6, Open: "08:30", Close: "19:15"},
			{DayOfWeek: 7, Open: "08:30", Close: "19:15"},
		},
	}
}

func validPlaceEnrichmentMeta() *aggregate.PlaceEnrichmentMeta {
	return &aggregate.PlaceEnrichmentMeta{
		Status:     placeenrichment.StatusMatched,
		Confidence: 0.9,
		Query:      "Colosseum",
		Provider:   "mock",
		MatchedAt:  "2026-06-23T12:00:00Z",
		Reason:     "exact_name_match",
	}
}

func testWeatherForecast() *weathercontext.WeatherForecast {
	return &weathercontext.WeatherForecast{
		Destination: "Rome",
		Provider:    "mock",
		Days: []weathercontext.WeatherDay{
			{
				Date:                "2026-08-10",
				Condition:           "hot",
				TemperatureMinC:     24,
				TemperatureMaxC:     35,
				PrecipitationChance: 5,
				WindSpeedKph:        10,
				Summary:             "Hot and sunny",
				Warnings:            []string{"High heat: avoid long outdoor walks at midday"},
			},
		},
	}
}

func decodeItinerary(t *testing.T, raw json.RawMessage) aggregate.Itinerary {
	t.Helper()
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		t.Fatalf("decode itinerary: %v", err)
	}
	return itinerary
}
