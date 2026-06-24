package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type fakeActivityService struct {
	recorded   []activity.RecordActivityInput
	listInput  *activity.ListActivityInput
	listResult *activity.ListActivityResult
}

func (f *fakeActivityService) Record(_ context.Context, in activity.RecordActivityInput) error {
	f.recorded = append(f.recorded, in)
	return nil
}

func (f *fakeActivityService) List(_ context.Context, in activity.ListActivityInput) (*activity.ListActivityResult, error) {
	f.listInput = &in
	if f.listResult != nil {
		return f.listResult, nil
	}
	return &activity.ListActivityResult{Events: []entity.TripActivityEvent{}}, nil
}

func newTestServiceWithActivity(repo tripRepository, gen *mockGenerator, act activityService) *Service {
	return New(repo, gen, zap.NewNop(), WithActivity(act))
}

func ownedTripRepo() (*mockRepo, uuid.UUID, uuid.UUID) {
	owner := testUserID()
	tripID := uuid.New()
	repo := &mockRepo{getByIDResult: &entity.Trip{
		ID:     tripID,
		UserID: &owner,
		Status: entity.StatusCompleted,
		Days:   2,
	}}
	return repo, tripID, owner
}

func TestListActivity_OwnerCanView(t *testing.T) {
	repo, tripID, owner := ownedTripRepo()
	act := &fakeActivityService{}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, act)

	if _, err := svc.ListActivity(ctxWithUserID(owner), tripID, 0, ""); err != nil {
		t.Fatalf("owner should view activity, got %v", err)
	}
	if act.listInput == nil {
		t.Fatalf("expected activity.List to be called")
	}
}

func TestListActivity_EditorAndViewerCanView(t *testing.T) {
	for _, role := range []entity.CollaboratorRole{entity.CollaboratorRoleEditor, entity.CollaboratorRoleViewer} {
		repo, tripID, _ := ownedTripRepo()
		repo.collaboratorByUser = acceptedCollaborator(role)
		svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

		if _, err := svc.ListActivity(ctxWithUserID(uuid.New()), tripID, 0, ""); err != nil {
			t.Fatalf("%s collaborator should view activity, got %v", role, err)
		}
	}
}

func TestListActivity_PendingCollaboratorCannotView(t *testing.T) {
	repo, tripID, _ := ownedTripRepo()
	repo.collaboratorByUser = &entity.TripCollaborator{
		Role:   entity.CollaboratorRoleEditor,
		Status: entity.CollaboratorStatusPending,
	}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

	_, err := svc.ListActivity(ctxWithUserID(uuid.New()), tripID, 0, "")
	assertNotFound(t, err)
}

func TestListActivity_NonCollaboratorCannotView(t *testing.T) {
	repo, tripID, _ := ownedTripRepo()
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

	_, err := svc.ListActivity(ctxWithUserID(uuid.New()), tripID, 0, "")
	assertNotFound(t, err)
}

func TestListActivity_UnauthenticatedCannotView(t *testing.T) {
	repo, tripID, _ := ownedTripRepo()
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

	if _, err := svc.ListActivity(context.Background(), tripID, 0, ""); err == nil {
		t.Fatalf("expected an error without an authenticated user")
	}
}

func TestListActivity_InvalidCursorRejected(t *testing.T) {
	repo, tripID, owner := ownedTripRepo()
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

	_, err := svc.ListActivity(ctxWithUserID(owner), tripID, 0, "not-a-valid-cursor$$$")
	assertInvalidInput(t, err)
}

func TestListActivity_InvalidLimitRejected(t *testing.T) {
	repo, tripID, owner := ownedTripRepo()
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, &fakeActivityService{})

	_, err := svc.ListActivity(ctxWithUserID(owner), tripID, activity.MaxLimit+1, "")
	assertInvalidInput(t, err)
}

func TestListActivity_NilActivityReturnsEmpty(t *testing.T) {
	repo, tripID, owner := ownedTripRepo()
	svc := newTestService(repo, &mockGenerator{}) // no activity configured

	result, err := svc.ListActivity(ctxWithUserID(owner), tripID, 0, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || len(result.Events) != 0 || result.NextCursor != "" {
		t.Fatalf("expected an empty activity page, got %+v", result)
	}
}

func TestCreate_RecordsTripCreated(t *testing.T) {
	repo := &mockRepo{}
	act := &fakeActivityService{}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, act)

	created, err := svc.Create(authContext(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(act.recorded) != 1 {
		t.Fatalf("expected exactly one recorded event, got %d", len(act.recorded))
	}
	event := act.recorded[0]
	if event.EventType != activity.EventTripCreated {
		t.Fatalf("expected trip_created, got %q", event.EventType)
	}
	if event.TripID != created.ID {
		t.Fatalf("recorded event trip id mismatch")
	}
	if event.Metadata["destination"] != "Rome" {
		t.Fatalf("expected destination in metadata, got %+v", event.Metadata)
	}
}

func TestCreateComment_RecordsCommentCreated(t *testing.T) {
	repo, tripID := commentRepo(t)
	act := &fakeActivityService{}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, act)

	if _, err := svc.CreateComment(ctxWithUserID(testUserID()), tripID, validCreateComment()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(act.recorded) != 1 || act.recorded[0].EventType != activity.EventCommentCreated {
		t.Fatalf("expected one comment_created event, got %+v", act.recorded)
	}
	if act.recorded[0].Metadata["dayNumber"] != 1 {
		t.Fatalf("expected dayNumber in metadata, got %+v", act.recorded[0].Metadata)
	}
}

func TestUnauthorizedAction_DoesNotRecord(t *testing.T) {
	repo, tripID := commentRepo(t)
	act := &fakeActivityService{}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, act)

	// A non-collaborator cannot comment; no activity must be recorded.
	if _, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment()); err == nil {
		t.Fatalf("expected a permission error")
	}
	if len(act.recorded) != 0 {
		t.Fatalf("failed action must not record activity, got %+v", act.recorded)
	}
}

func TestInvalidInput_DoesNotRecord(t *testing.T) {
	repo := &mockRepo{}
	act := &fakeActivityService{}
	svc := newTestServiceWithActivity(repo, &mockGenerator{}, act)

	bad := validCreateInput()
	bad.Destination = ""
	if _, err := svc.Create(authContext(), bad); err == nil {
		t.Fatalf("expected an invalid-input error")
	}
	if len(act.recorded) != 0 {
		t.Fatalf("invalid action must not record activity, got %+v", act.recorded)
	}
}
