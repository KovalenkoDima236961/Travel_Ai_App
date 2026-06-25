package emailnotifications

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
)

const testWebBase = "http://localhost:3000"

func build(t *testing.T, n entity.Notification, profile users.UserProfile) (subject, text, htmlBody string) {
	t.Helper()
	msg, err := BuildEmailForNotification(BuildEmailInput{
		Notification:     n,
		Recipient:        profile,
		PublicWebBaseURL: testWebBase,
	})
	if err != nil {
		t.Fatalf("build email for %s: %v", n.Type, err)
	}
	if msg.ToEmail != profile.Email {
		t.Fatalf("expected recipient %q, got %q", profile.Email, msg.ToEmail)
	}
	return msg.Subject, msg.TextBody, msg.HTMLBody
}

func TestTemplateCollaborationInvited(t *testing.T) {
	n := entity.Notification{
		UserID: uuid.New(),
		Type:   notifications.TypeCollaborationInvited,
		Metadata: map[string]any{
			"tripId":      "trip-123",
			"destination": "Paris",
			"role":        "editor",
		},
	}
	subject, text, htmlBody := build(t, n, users.UserProfile{Email: "anna@example.com", DisplayName: "Anna"})

	if subject != "You were invited to collaborate on a trip" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	for _, want := range []string{"Hi Anna,", "collaborate on Paris as editor", testWebBase + "/trips?tab=invitations"} {
		if !strings.Contains(text, want) {
			t.Errorf("text missing %q\n%s", want, text)
		}
	}
	if !strings.Contains(htmlBody, "Paris") {
		t.Errorf("html missing destination\n%s", htmlBody)
	}
}

func TestTemplateCommentCreatedNoDestination(t *testing.T) {
	// comment_created metadata has no destination — must degrade gracefully.
	n := entity.Notification{
		UserID: uuid.New(),
		Type:   notifications.TypeCommentCreated,
		Metadata: map[string]any{
			"tripId":    "trip-123",
			"dayNumber": float64(2), // arrives as float64 over the wire
			"itemName":  "Louvre Museum",
		},
	}
	subject, text, _ := build(t, n, users.UserProfile{Email: "anna@example.com"})

	if subject != "New comment on a trip" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(text, "Day 2 · Louvre Museum") {
		t.Errorf("text missing location\n%s", text)
	}
	if strings.Contains(text, " in ") {
		t.Errorf("should not render ' in <destination>' when destination is absent\n%s", text)
	}
	// No display name -> neutral greeting.
	if !strings.Contains(text, "Hi there,") {
		t.Errorf("expected neutral greeting\n%s", text)
	}
	if !strings.Contains(text, testWebBase+"/trips/trip-123") {
		t.Errorf("text missing trip link\n%s", text)
	}
}

func TestTemplateRoleChanged(t *testing.T) {
	n := entity.Notification{
		UserID: uuid.New(),
		Type:   notifications.TypeCollaboratorRoleChange,
		Metadata: map[string]any{
			"tripId":      "trip-9",
			"destination": "Tokyo",
			"oldRole":     "viewer",
			"newRole":     "editor",
		},
	}
	subject, text, _ := build(t, n, users.UserProfile{Email: "b@example.com"})
	if subject != "Your trip role changed" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(text, "changed from viewer to editor") {
		t.Errorf("text missing role change\n%s", text)
	}
}

func TestTemplateRemovedHasNoLink(t *testing.T) {
	n := entity.Notification{
		UserID:   uuid.New(),
		Type:     notifications.TypeCollaboratorRemoved,
		Metadata: map[string]any{"destination": "Rome"},
	}
	subject, text, _ := build(t, n, users.UserProfile{Email: "b@example.com"})
	if subject != "You were removed from a trip" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if !strings.Contains(text, "no longer have access to Rome") {
		t.Errorf("text missing removal sentence\n%s", text)
	}
	if strings.Contains(text, "/trips") {
		t.Errorf("removed email should not contain a trip link\n%s", text)
	}
}

func TestTemplateMissingMetadataDegradesGracefully(t *testing.T) {
	// role changed with no roles and no destination at all.
	n := entity.Notification{UserID: uuid.New(), Type: notifications.TypeCollaboratorRoleChange}
	subject, text, _ := build(t, n, users.UserProfile{Email: "b@example.com"})
	if subject == "" || text == "" {
		t.Fatal("expected non-empty subject/body even with no metadata")
	}
	if !strings.Contains(text, "your trip") {
		t.Errorf("expected neutral destination fallback\n%s", text)
	}
	if strings.Contains(text, "Day 0") || strings.Contains(text, "from  to ") {
		t.Errorf("expected no malformed fragments\n%s", text)
	}
}

func TestTemplateDoesNotLeakSecretMetadata(t *testing.T) {
	// Even if a secret-looking value sneaks into metadata, templates only read
	// known rendering keys, so it must never appear in the email.
	n := entity.Notification{
		UserID: uuid.New(),
		Type:   notifications.TypeCommentCreated,
		Metadata: map[string]any{
			"tripId":       "trip-123",
			"dayNumber":    float64(1),
			"shareToken":   "super-secret-token-value",
			"accessSecret": "jwt.header.payload.sig",
		},
	}
	_, text, htmlBody := build(t, n, users.UserProfile{Email: "anna@example.com"})
	for _, secret := range []string{"super-secret-token-value", "jwt.header.payload.sig"} {
		if strings.Contains(text, secret) || strings.Contains(htmlBody, secret) {
			t.Fatalf("email leaked secret metadata %q", secret)
		}
	}
}

func TestTemplatePrefersEntityTripIDForLink(t *testing.T) {
	tripID := uuid.New()
	// No metadata tripId at all; the authoritative entity.TripID must drive the link.
	n := entity.Notification{
		UserID:   uuid.New(),
		TripID:   &tripID,
		Type:     notifications.TypeCollaboratorRoleChange,
		Metadata: map[string]any{"destination": "Tokyo", "oldRole": "viewer", "newRole": "editor"},
	}
	_, text, _ := build(t, n, users.UserProfile{Email: "b@example.com"})
	if !strings.Contains(text, testWebBase+"/trips/"+tripID.String()) {
		t.Errorf("expected deep link from entity.TripID\n%s", text)
	}
}

func TestTemplateUnknownTypeReturnsErrNoTemplate(t *testing.T) {
	n := entity.Notification{UserID: uuid.New(), Type: "totally_unknown_type"}
	_, err := BuildEmailForNotification(BuildEmailInput{Notification: n, PublicWebBaseURL: testWebBase})
	if !errors.Is(err, ErrNoTemplate) {
		t.Fatalf("expected ErrNoTemplate, got %v", err)
	}
}
