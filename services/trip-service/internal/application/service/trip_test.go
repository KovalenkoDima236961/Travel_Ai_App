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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
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

func decodeItinerary(t *testing.T, raw json.RawMessage) aggregate.Itinerary {
	t.Helper()
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		t.Fatalf("decode itinerary: %v", err)
	}
	return itinerary
}
