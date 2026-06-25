package presence

import (
	"strings"
	"testing"
)

func TestWriteSSEWritesEventNameAndJSONData(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventPresenceSnapshot, map[string]any{"ok": true}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "event: presence.snapshot\n") {
		t.Fatalf("expected event name, got %q", got)
	}
	if !strings.Contains(got, `data: {"ok":true}`) {
		t.Fatalf("expected JSON data, got %q", got)
	}
}

func TestWriteSSEEndsWithBlankLine(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventPresenceHeartbeat, map[string]string{"ts": "2026-06-25T12:00:00Z"}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}
	if !strings.HasSuffix(out.String(), "\n\n") {
		t.Fatalf("expected blank line terminator, got %q", out.String())
	}
}
