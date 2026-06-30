package activity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

var errBoom = errors.New("boom")

type fakeRepo struct {
	created    []*entity.TripActivityEvent
	listResult []entity.TripActivityEvent
	listErr    error
	createErr  error

	gotLimit           int
	gotCursorCreatedAt *time.Time
	gotCursorID        *uuid.UUID
}

type fakePublisher struct {
	events []EventDTO
}

func (f *fakePublisher) Publish(_ context.Context, _ uuid.UUID, event EventDTO) {
	f.events = append(f.events, event)
}

func (f *fakeRepo) CreateTripActivityEvent(_ context.Context, event *entity.TripActivityEvent) (*entity.TripActivityEvent, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	f.created = append(f.created, event)
	return event, nil
}

func (f *fakeRepo) ListTripActivityEvents(
	_ context.Context,
	_ uuid.UUID,
	limit int,
	cursorCreatedAt *time.Time,
	cursorID *uuid.UUID,
) ([]entity.TripActivityEvent, error) {
	f.gotLimit = limit
	f.gotCursorCreatedAt = cursorCreatedAt
	f.gotCursorID = cursorID
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func newTestService(repo Repository) *Service {
	return New(repo, zap.NewNop())
}

func TestRecord_StoresSanitizedEvent(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo)

	actor := uuid.New()
	entityType := EntityItineraryItem
	longName := strings.Repeat("x", maxMetadataStringLen+50)
	err := svc.Record(context.Background(), RecordActivityInput{
		TripID:      uuid.New(),
		ActorUserID: &actor,
		EventType:   EventItemRegenerated,
		EntityType:  &entityType,
		Metadata: map[string]any{
			"dayNumber": 2,
			"itemName":  longName,
			"dropped":   nil,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(repo.created))
	}
	stored := repo.created[0]
	if stored.EventType != EventItemRegenerated {
		t.Fatalf("unexpected event type: %q", stored.EventType)
	}
	if stored.ID == uuid.Nil {
		t.Fatalf("expected a generated event id")
	}
	if _, ok := stored.Metadata["dropped"]; ok {
		t.Fatalf("nil metadata value should have been dropped")
	}
	name, _ := stored.Metadata["itemName"].(string)
	if len(name) != maxMetadataStringLen {
		t.Fatalf("expected itemName truncated to %d, got %d", maxMetadataStringLen, len(name))
	}
}

func TestRecord_PublishesAfterSuccessfulCreate(t *testing.T) {
	repo := &fakeRepo{}
	publisher := &fakePublisher{}
	svc := New(repo, zap.NewNop(), WithPublisher(publisher))
	tripID := uuid.New()

	if err := svc.Record(context.Background(), RecordActivityInput{
		TripID:    tripID,
		EventType: EventCommentCreated,
		Metadata:  map[string]any{"itemName": "Lunch"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("expected one published event, got %d", len(publisher.events))
	}
	got := publisher.events[0]
	if got.TripID != tripID || got.EventType != EventCommentCreated {
		t.Fatalf("unexpected published event: %+v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("expected published event to include stored createdAt")
	}
}

func TestRecord_DoesNotPublishWhenCreateFails(t *testing.T) {
	repo := &fakeRepo{createErr: errBoom}
	publisher := &fakePublisher{}
	svc := New(repo, zap.NewNop(), WithPublisher(publisher))

	if err := svc.Record(context.Background(), RecordActivityInput{
		TripID:    uuid.New(),
		EventType: EventCommentCreated,
	}); err == nil {
		t.Fatal("expected create error")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no published events, got %+v", publisher.events)
	}
}

func TestRecord_RequiresTripAndEventType(t *testing.T) {
	svc := newTestService(&fakeRepo{})

	if err := svc.Record(context.Background(), RecordActivityInput{EventType: EventTripCreated}); err == nil {
		t.Fatalf("expected error for missing trip id")
	}
	if err := svc.Record(context.Background(), RecordActivityInput{TripID: uuid.New()}); err == nil {
		t.Fatalf("expected error for missing event type")
	}
}

func TestList_NoNextCursorWhenUnderLimit(t *testing.T) {
	repo := &fakeRepo{listResult: makeEvents(3)}
	svc := newTestService(repo)

	result, err := svc.List(context.Background(), ListActivityInput{TripID: uuid.New(), Limit: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotLimit != 31 {
		t.Fatalf("expected repo queried with limit+1 (31), got %d", repo.gotLimit)
	}
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result.Events))
	}
	if result.NextCursor != "" {
		t.Fatalf("expected no next cursor, got %q", result.NextCursor)
	}
}

func TestList_ReturnsNextCursorWhenMore(t *testing.T) {
	events := makeEvents(3)
	repo := &fakeRepo{listResult: events}
	svc := newTestService(repo)

	result, err := svc.List(context.Background(), ListActivityInput{TripID: uuid.New(), Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected page trimmed to 2 events, got %d", len(result.Events))
	}
	if result.NextCursor == "" {
		t.Fatalf("expected a next cursor")
	}

	gotCreatedAt, gotID, err := DecodeCursor(result.NextCursor)
	if err != nil {
		t.Fatalf("next cursor should decode: %v", err)
	}
	// The cursor must point at the last returned row (index limit-1).
	if *gotID != events[1].ID {
		t.Fatalf("cursor id should be the last returned event id")
	}
	if !gotCreatedAt.Equal(events[1].CreatedAt.UTC()) {
		t.Fatalf("cursor created_at should match the last returned event")
	}
}

func TestList_PassesDecodedCursorToRepo(t *testing.T) {
	repo := &fakeRepo{listResult: makeEvents(1)}
	svc := newTestService(repo)

	createdAt := time.Now().UTC()
	id := uuid.New()
	if _, err := svc.List(context.Background(), ListActivityInput{
		TripID:          uuid.New(),
		Limit:           10,
		CursorCreatedAt: &createdAt,
		CursorID:        &id,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotCursorID == nil || *repo.gotCursorID != id {
		t.Fatalf("cursor id was not passed through to the repo")
	}
	if repo.gotCursorCreatedAt == nil || !repo.gotCursorCreatedAt.Equal(createdAt) {
		t.Fatalf("cursor created_at was not passed through to the repo")
	}
}

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 6, 24, 10, 30, 15, 123456000, time.UTC)
	id := uuid.New()

	encoded := EncodeCursor(createdAt, id)
	if encoded == "" {
		t.Fatalf("expected a non-empty cursor")
	}
	gotCreatedAt, gotID, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotCreatedAt.Equal(createdAt) {
		t.Fatalf("created_at did not round-trip: got %v want %v", gotCreatedAt, createdAt)
	}
	if *gotID != id {
		t.Fatalf("id did not round-trip")
	}
}

func TestDecodeCursor_EmptyIsFirstPage(t *testing.T) {
	createdAt, id, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("empty cursor should not error: %v", err)
	}
	if createdAt != nil || id != nil {
		t.Fatalf("empty cursor should yield nil values")
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	for _, raw := range []string{"not-base64-$$$", "YWJj"} { // second decodes to "abc" (not JSON)
		if _, _, err := DecodeCursor(raw); err != ErrInvalidCursor {
			t.Fatalf("expected ErrInvalidCursor for %q, got %v", raw, err)
		}
	}
}

func TestNormalizeLimit(t *testing.T) {
	cases := map[int]int{
		0:            DefaultLimit,
		-5:           DefaultLimit,
		10:           10,
		MaxLimit:     MaxLimit,
		MaxLimit + 1: MaxLimit,
		1000:         MaxLimit,
	}
	for in, want := range cases {
		if got := NormalizeLimit(in); got != want {
			t.Fatalf("NormalizeLimit(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestSanitizeMetadata(t *testing.T) {
	out := sanitizeMetadata(map[string]any{
		"keep":   "value",
		"nilled": nil,
		"long":   strings.Repeat("a", maxMetadataStringLen+10),
		"number": 7,
	})
	if _, ok := out["nilled"]; ok {
		t.Fatalf("nil values must be dropped")
	}
	if out["keep"] != "value" {
		t.Fatalf("plain string value should be kept")
	}
	if out["number"] != 7 {
		t.Fatalf("non-string value should be kept")
	}
	if len(out["long"].(string)) != maxMetadataStringLen {
		t.Fatalf("long string should be truncated")
	}
}

// makeEvents builds n events newest-first with strictly decreasing timestamps,
// mirroring how the repository returns rows.
func makeEvents(n int) []entity.TripActivityEvent {
	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	events := make([]entity.TripActivityEvent, 0, n)
	for i := 0; i < n; i++ {
		events = append(events, entity.TripActivityEvent{
			ID:        uuid.New(),
			TripID:    uuid.New(),
			EventType: EventCommentCreated,
			Metadata:  map[string]any{},
			CreatedAt: base.Add(time.Duration(-i) * time.Minute),
		})
	}
	return events
}
