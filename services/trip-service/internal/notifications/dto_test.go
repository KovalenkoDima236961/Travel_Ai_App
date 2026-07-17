package notifications

import (
	"testing"

	"github.com/google/uuid"
)

func TestToPayloadDerivesRouteNoiseControlMetadata(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New()
	payload := toPayload(NotificationCreateInput{
		UserID: userID, TripID: &tripID, Type: TypeRouteChanged,
		Title: "Route changed", Message: "The route was updated.",
	})

	if payload.Priority != PriorityHigh || payload.Category != "trip_updates" {
		t.Fatalf("unexpected route defaults: priority=%q category=%q", payload.Priority, payload.Category)
	}
	if payload.DigestKey != "trip:"+tripID.String()+":trip_updates" {
		t.Fatalf("unexpected digest key %q", payload.DigestKey)
	}
}

func TestToPayloadDerivesDedupeKeyAndLowCompletionPriority(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	payload := toPayload(NotificationCreateInput{
		UserID: userID, Type: TypeChecklistItemCompleted,
		Title: "Completed", Message: "A checklist item was completed.", EntityID: &itemID,
	})

	if payload.Priority != PriorityLow || payload.Category != "checklist" {
		t.Fatalf("unexpected checklist defaults: priority=%q category=%q", payload.Priority, payload.Category)
	}
	want := TypeChecklistItemCompleted + ":" + itemID.String() + ":recipient:" + userID.String()
	if payload.DedupeKey != want {
		t.Fatalf("unexpected dedupe key %q, want %q", payload.DedupeKey, want)
	}
}
