package copilot

import (
	"strings"
	"testing"
)

func TestValidateAIResponseKeepsOnlyAvailableTrustedActions(t *testing.T) {
	available := []Action{{Type: "open_trip_health", Label: "Open Trip Health", Href: "/trips/a?tab=health", Style: ActionStyleSecondary}}
	response, err := validateAIResponse(AIResponse{
		Answer: "Review Trip Health first.",
		Actions: []Action{
			{Type: "open_trip_health", Label: "Injected", Href: "https://example.com", Style: ActionStylePrimary},
			{Type: "add_expense", Label: "Add", Href: "/trips/a?tab=expenses", Style: ActionStylePrimary},
		},
		SourceTypes: []string{"trip_health", "not_a_source"},
	}, available)
	if err != nil {
		t.Fatalf("validateAIResponse returned error: %v", err)
	}
	if len(response.Actions) != 1 || response.Actions[0].Label != "Open Trip Health" {
		t.Fatalf("expected trusted available action, got %#v", response.Actions)
	}
	if len(response.SourceTypes) != 1 || response.SourceTypes[0] != "trip_health" {
		t.Fatalf("unexpected source types: %#v", response.SourceTypes)
	}
}

func TestValidateAIResponseRejectsUnsafeClaims(t *testing.T) {
	_, err := validateAIResponse(AIResponse{Answer: "I have deleted the trip."}, nil)
	if err != ErrResponseInvalid {
		t.Fatalf("expected ErrResponseInvalid, got %v", err)
	}
}

func TestValidateAIResponseRejectsSensitiveOrOversizedAnswers(t *testing.T) {
	for _, answer := range []string{
		"Bearer abcdefghijklmnopqrstuvwxyz",
		strings.Repeat("safe ", 500),
	} {
		if _, err := validateAIResponse(AIResponse{Answer: answer}, nil); err != ErrResponseInvalid {
			t.Fatalf("expected invalid response for %q, got %v", answer[:min(24, len(answer))], err)
		}
	}
}
