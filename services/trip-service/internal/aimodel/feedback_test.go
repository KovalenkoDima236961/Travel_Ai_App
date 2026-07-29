package aimodel

import (
	"strings"
	"testing"
)

func TestSanitizeFeedbackNoteRedactsAndTruncates(t *testing.T) {
	note := "bad place, email me at person@example.com " + strings.Repeat("x", MaxFeedbackNoteLength+20)
	got := SanitizeFeedbackNote(note)
	if got == nil {
		t.Fatal("expected sanitized note")
	}
	if strings.Contains(*got, "person@example.com") {
		t.Fatalf("expected email to be redacted, got %q", *got)
	}
	if len([]rune(*got)) > MaxFeedbackNoteLength {
		t.Fatalf("expected note at most %d runes, got %d", MaxFeedbackNoteLength, len([]rune(*got)))
	}
}

func TestSanitizeFeedbackNoteEmptyReturnsNil(t *testing.T) {
	if got := SanitizeFeedbackNote("   "); got != nil {
		t.Fatalf("expected nil note, got %q", *got)
	}
}
