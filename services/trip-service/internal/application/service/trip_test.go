package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
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

func (m *mockRepo) UpdateItineraryByUserID(_ context.Context, id, userID uuid.UUID, itinerary json.RawMessage, status entity.Status) (*entity.Trip, error) {
	m.updateItinRaw = itinerary
	m.updateItinStatus = status
	m.updateItinUserID = userID
	if m.updateItinErr != nil {
		return nil, m.updateItinErr
	}
	return &entity.Trip{ID: id, Status: status, Itinerary: itinerary}, nil
}

// mockGenerator is an application.ItineraryGenerator test double.
type mockGenerator struct {
	result *aggregate.Itinerary
	err    error
	called bool
}

func (g *mockGenerator) Generate(_ context.Context, trip entity.Trip) (*aggregate.Itinerary, error) {
	g.called = true
	if g.err != nil {
		return nil, g.err
	}
	if g.result != nil {
		return g.result, nil
	}
	return &aggregate.Itinerary{Destination: trip.Destination}, nil
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
	return auth.WithUser(context.Background(), auth.AuthenticatedUser{
		ID:    testUserID(),
		Email: "traveler@example.com",
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
