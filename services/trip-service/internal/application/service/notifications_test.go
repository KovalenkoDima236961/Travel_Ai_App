package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
)

// fakeNotifier records the notification batches it is asked to send and can be
// configured to fail, so tests can assert recipients and fail-open behavior.
type fakeNotifier struct {
	batches [][]notifications.NotificationCreateInput
	err     error
}

func (f *fakeNotifier) CreateNotifications(_ context.Context, batch []notifications.NotificationCreateInput) error {
	f.batches = append(f.batches, batch)
	return f.err
}

func (f *fakeNotifier) allInputs() []notifications.NotificationCreateInput {
	var all []notifications.NotificationCreateInput
	for _, b := range f.batches {
		all = append(all, b...)
	}
	return all
}

func (f *fakeNotifier) recipients() map[uuid.UUID]bool {
	out := map[uuid.UUID]bool{}
	for _, in := range f.allInputs() {
		out[in.UserID] = true
	}
	return out
}

func newTestServiceWithNotifications(repo tripRepository, gen *mockGenerator, n notifier, enabled, failOpen bool) *Service {
	return New(repo, gen, zap.NewNop(), WithNotifications(n, enabled, failOpen))
}

func accepted(userID uuid.UUID, role entity.CollaboratorRole) entity.TripCollaborator {
	return entity.TripCollaborator{UserID: userID, Role: role, Status: entity.CollaboratorStatusAccepted}
}

func TestCreateComment_NotifiesOwnerAndCollaboratorsExceptAuthor(t *testing.T) {
	repo, tripID := commentRepo(t) // owner == testUserID(), completed trip with itinerary
	owner := testUserID()
	author := uuid.New()
	otherEditor := uuid.New()
	pendingUser := uuid.New()

	// Author is an accepted viewer (so the access check passes).
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleViewer)
	repo.listCollaborators = []entity.TripCollaborator{
		accepted(author, entity.CollaboratorRoleViewer),
		accepted(otherEditor, entity.CollaboratorRoleEditor),
		{UserID: pendingUser, Role: entity.CollaboratorRoleEditor, Status: entity.CollaboratorStatusPending},
	}

	notifier := &fakeNotifier{}
	svc := newTestServiceWithNotifications(repo, &mockGenerator{}, notifier, true, true)

	if _, err := svc.CreateComment(ctxWithUserID(author), tripID, validCreateComment()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := notifier.recipients()
	if !got[owner] || !got[otherEditor] {
		t.Fatalf("expected owner and other editor notified, got %v", got)
	}
	if got[author] {
		t.Fatal("comment author must not be notified about their own comment")
	}
	if got[pendingUser] {
		t.Fatal("pending collaborator must not be notified")
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly 2 recipients, got %d (%v)", len(got), got)
	}
	for _, in := range notifier.allInputs() {
		if in.Type != notifications.TypeCommentCreated {
			t.Fatalf("expected comment_created type, got %s", in.Type)
		}
		if in.ActorUserID == nil || *in.ActorUserID != author {
			t.Fatalf("expected actor to be the comment author")
		}
	}
}

func TestUpdateItinerary_NotifiesCollaboratorsExceptActor(t *testing.T) {
	id := uuid.New()
	editor := uuid.New()
	repo := &mockRepo{}
	repo.listCollaborators = []entity.TripCollaborator{accepted(editor, entity.CollaboratorRoleEditor)}

	notifier := &fakeNotifier{}
	svc := newTestServiceWithNotifications(repo, &mockGenerator{}, notifier, true, true)

	// authContext() acts as the owner (testUserID); GetByID returns a trip owned
	// by testUserID, so the owner is the actor and must be excluded.
	if _, err := svc.UpdateItinerary(authContext(), id, appdto.UpdateItineraryInput{
		ExpectedItineraryRevision: intPtr(0),
		Itinerary:                 validExistingItineraryRaw(t),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := notifier.recipients()
	if !got[editor] {
		t.Fatalf("expected the editor collaborator notified, got %v", got)
	}
	if got[testUserID()] {
		t.Fatal("the acting owner must not be notified about their own update")
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 recipient, got %d (%v)", len(got), got)
	}
	if notifier.allInputs()[0].Type != notifications.TypeItineraryUpdated {
		t.Fatalf("expected itinerary_updated type, got %s", notifier.allInputs()[0].Type)
	}
}

func TestNotificationFailureDoesNotFailAction(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleEditor)
	repo.listCollaborators = []entity.TripCollaborator{accepted(uuid.New(), entity.CollaboratorRoleEditor)}

	notifier := &fakeNotifier{err: errors.New("notification service unavailable")}
	svc := newTestServiceWithNotifications(repo, &mockGenerator{}, notifier, true, true)

	// Fail-open: the comment is created successfully even though notifying fails.
	if _, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment()); err != nil {
		t.Fatalf("fail-open: expected comment to succeed despite notifier error, got %v", err)
	}
	if len(repo.comments) != 1 {
		t.Fatalf("expected the comment to be persisted, got %d", len(repo.comments))
	}
	if len(notifier.batches) == 0 {
		t.Fatal("expected the notifier to have been attempted")
	}
}

func TestNotificationsDisabled_NoCall(t *testing.T) {
	repo, tripID := commentRepo(t)
	repo.collaboratorByUser = acceptedCollaborator(entity.CollaboratorRoleEditor)
	repo.listCollaborators = []entity.TripCollaborator{accepted(uuid.New(), entity.CollaboratorRoleEditor)}

	notifier := &fakeNotifier{}
	svc := newTestServiceWithNotifications(repo, &mockGenerator{}, notifier, false, true)

	if _, err := svc.CreateComment(ctxWithUserID(uuid.New()), tripID, validCreateComment()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.batches) != 0 {
		t.Fatal("notifications disabled: notifier must not be called")
	}
}
