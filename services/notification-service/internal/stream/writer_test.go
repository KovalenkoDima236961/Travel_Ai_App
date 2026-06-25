package stream

import (
	"strings"
	"testing"
)

func TestWriteSSEWritesEventNameAndJSONData(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventNotificationCreated, map[string]any{"ok": true}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "event: notification.created\n") {
		t.Fatalf("expected event name, got %q", got)
	}
	if !strings.Contains(got, `data: {"ok":true}`) {
		t.Fatalf("expected JSON data, got %q", got)
	}
}

func TestWriteSSEEndsWithBlankLine(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventHeartbeat, map[string]string{"ts": "2026-06-25T12:00:00Z"}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}
	if !strings.HasSuffix(out.String(), "\n\n") {
		t.Fatalf("expected blank line terminator, got %q", out.String())
	}
}

func TestWriteSSEEscapesSpecialCharactersAsJSON(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventNotificationCreated, map[string]string{"message": "line 1\nline 2"}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "line 1\nline 2") {
		t.Fatalf("expected newline escaped inside JSON, got %q", got)
	}
	if !strings.Contains(got, `line 1\nline 2`) {
		t.Fatalf("expected escaped newline sequence, got %q", got)
	}
}
