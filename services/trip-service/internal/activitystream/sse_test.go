package activitystream

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteSSEWritesEventNameAndJSONData(t *testing.T) {
	var out strings.Builder
	if err := WriteSSE(&out, EventActivityCreated, map[string]any{"ok": true}); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "event: activity.created\n") {
		t.Fatalf("expected event name, got %q", got)
	}
	if !strings.Contains(got, `data: {"ok":true}`) {
		t.Fatalf("expected JSON data, got %q", got)
	}
	if !strings.HasSuffix(got, "\n\n") {
		t.Fatalf("expected blank line terminator, got %q", got)
	}
}

func TestWriteSSEHeartbeatIsValidJSON(t *testing.T) {
	var out strings.Builder
	event := HeartbeatEvent()
	if err := WriteSSE(&out, event.Name, event.Data); err != nil {
		t.Fatalf("WriteSSE: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "event: activity.heartbeat\n") {
		t.Fatalf("expected heartbeat event, got %q", got)
	}
	raw := strings.TrimPrefix(strings.Split(strings.TrimSpace(got), "\n")[1], "data: ")
	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("expected valid heartbeat JSON, got %q: %v", raw, err)
	}
	if payload["ts"] == "" {
		t.Fatalf("expected heartbeat timestamp, got %+v", payload)
	}
}
