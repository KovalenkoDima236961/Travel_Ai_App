package emailnotifications

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/email"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
)

type fakeLookup struct {
	profiles map[uuid.UUID]users.UserProfile
	err      error
	gotIDs   []uuid.UUID
}

func (f *fakeLookup) LookupByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]users.UserProfile, error) {
	f.gotIDs = ids
	if f.err != nil {
		return nil, f.err
	}
	return f.profiles, nil
}

type fakeSender struct {
	sent     []email.EmailMessage
	failAll  bool
	failTo   map[string]bool
	attempts int
}

type fakeEmailPreferenceGate struct {
	allowed map[string]bool
}

func (f fakeEmailPreferenceGate) AllowEmail(_ uuid.UUID, notificationType string) bool {
	allowed, ok := f.allowed[notificationType]
	if !ok {
		return true
	}
	return allowed
}

func (f *fakeSender) Send(_ context.Context, msg email.EmailMessage) error {
	f.attempts++
	if f.failAll || f.failTo[msg.ToEmail] {
		return errors.New("simulated send failure")
	}
	f.sent = append(f.sent, msg)
	return nil
}

func notify(user uuid.UUID, actor uuid.UUID, typ string) entity.Notification {
	a := actor
	return entity.Notification{
		ID:          uuid.New(),
		UserID:      user,
		ActorUserID: &a,
		Type:        typ,
		Metadata:    map[string]any{"tripId": "trip-1", "destination": "Paris", "role": "editor"},
	}
}

func newService(t *testing.T, failOpen bool, lookup UserLookup, sender email.EmailSender) *Service {
	t.Helper()
	return New(Config{
		Enabled:          true,
		FailOpen:         failOpen,
		PublicWebBaseURL: "http://localhost:3000",
		Types:            defaultAllowlist(),
	}, lookup, sender, nil)
}

func TestSendEmails_SendsAllowlistedSkipsOthers(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com", DisplayName: "Anna"},
	}}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	in := []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),   // allowlisted -> sent
		notify(recipient, actor, notifications.TypeItineraryUpdated), // not allowlisted -> skipped
	}
	res, err := svc.SendEmailsForNotifications(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Attempted != 1 || res.Sent != 1 || res.Skipped != 1 || res.Failed != 0 {
		t.Fatalf("unexpected result %+v", res)
	}
	if len(sender.sent) != 1 || sender.sent[0].ToEmail != "anna@example.com" {
		t.Fatalf("unexpected sent messages %+v", sender.sent)
	}
}

func TestSendEmails_SkipsSelfNotification(t *testing.T) {
	user := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		user: {UserID: user, Email: "self@example.com"},
	}}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	// Actor == recipient.
	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(user, user, notifications.TypeCommentCreated),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Attempted != 0 || res.Sent != 0 || res.Skipped != 1 {
		t.Fatalf("expected self-notification skipped, got %+v", res)
	}
	if len(lookup.gotIDs) != 0 {
		t.Fatal("expected no recipient lookup for a self-only batch")
	}
}

func TestSendEmails_SkipsDisabledPreference(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com"},
	}}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(
		context.Background(),
		[]entity.Notification{notify(recipient, actor, notifications.TypeCommentCreated)},
		fakeEmailPreferenceGate{allowed: map[string]bool{notifications.TypeCommentCreated: false}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Skipped != 1 || res.SkippedByPreference != 1 || res.Attempted != 0 {
		t.Fatalf("expected preference skip with no send attempt, got %+v", res)
	}
	if len(lookup.gotIDs) != 0 || sender.attempts != 0 {
		t.Fatal("expected no lookup or send when preference disables email")
	}
}

func TestSendEmails_SendsEnabledPreference(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com"},
	}}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(
		context.Background(),
		[]entity.Notification{notify(recipient, actor, notifications.TypeCommentCreated)},
		fakeEmailPreferenceGate{allowed: map[string]bool{notifications.TypeCommentCreated: true}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Attempted != 1 || res.Sent != 1 || res.SkippedByPreference != 0 {
		t.Fatalf("expected enabled preference email sent, got %+v", res)
	}
}

func TestSendEmails_SkipsMissingRecipientEmail(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	// Lookup returns no profile for the recipient.
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{}}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Attempted != 0 || res.Skipped != 1 || res.Sent != 0 {
		t.Fatalf("expected skip for missing email, got %+v", res)
	}
}

func TestSendEmails_FailOpenSwallowsSendError(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com"},
	}}
	sender := &fakeSender{failAll: true}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
	})
	if err != nil {
		t.Fatalf("fail-open should not return a hard error, got %v", err)
	}
	if res.Attempted != 1 || res.Failed != 1 || res.Sent != 0 {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestSendEmails_FailClosedReturnsErrorOnSendFailure(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com"},
	}}
	sender := &fakeSender{failAll: true}
	svc := newService(t, false, lookup, sender)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
	})
	if err == nil {
		t.Fatal("fail-closed should return an error on send failure")
	}
	if res.Failed != 1 {
		t.Fatalf("expected failed=1, got %+v", res)
	}
}

func TestSendEmails_MixedSuccessAndFailure(t *testing.T) {
	good := uuid.New()
	bad := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		good: {UserID: good, Email: "good@example.com"},
		bad:  {UserID: bad, Email: "bad@example.com"},
	}}
	sender := &fakeSender{failTo: map[string]bool{"bad@example.com": true}}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(good, actor, notifications.TypeCommentCreated),
		notify(bad, actor, notifications.TypeCommentCreated),
	})
	if err != nil {
		t.Fatalf("fail-open should not return error: %v", err)
	}
	if res.Attempted != 2 || res.Sent != 1 || res.Failed != 1 {
		t.Fatalf("unexpected mixed result %+v", res)
	}
}

func TestSendEmails_LookupFailureFailOpen(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{err: errors.New("auth service down")}
	sender := &fakeSender{}
	svc := newService(t, true, lookup, sender)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
	})
	if err != nil {
		t.Fatalf("fail-open lookup failure should not return error: %v", err)
	}
	if res.Skipped != 1 || res.Attempted != 0 {
		t.Fatalf("expected eligible skipped on lookup failure, got %+v", res)
	}
	if sender.attempts != 0 {
		t.Fatal("expected no send attempts when lookup failed")
	}
}

func TestSendEmails_LookupFailureFailClosed(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{err: errors.New("auth service down")}
	sender := &fakeSender{}
	svc := newService(t, false, lookup, sender)

	_, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
	})
	if err == nil {
		t.Fatal("fail-closed lookup failure should return an error")
	}
}

func TestSendEmails_DisabledSkipsEverything(t *testing.T) {
	recipient := uuid.New()
	actor := uuid.New()
	lookup := &fakeLookup{profiles: map[uuid.UUID]users.UserProfile{
		recipient: {UserID: recipient, Email: "anna@example.com"},
	}}
	sender := &fakeSender{}
	svc := New(Config{Enabled: false, FailOpen: true, Types: defaultAllowlist()}, lookup, sender, nil)

	res, err := svc.SendEmailsForNotifications(context.Background(), []entity.Notification{
		notify(recipient, actor, notifications.TypeCommentCreated),
		notify(recipient, actor, notifications.TypeCollaborationInvited),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Skipped != 2 || res.Attempted != 0 || res.Sent != 0 {
		t.Fatalf("expected all skipped when disabled, got %+v", res)
	}
	if sender.attempts != 0 {
		t.Fatal("expected no send attempts when disabled")
	}
}
