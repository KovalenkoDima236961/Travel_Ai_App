package activitystream

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

var eventNameSanitizer = strings.NewReplacer("\r", " ", "\n", " ")

func WriteSSE(w io.Writer, eventName string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	eventName = strings.TrimSpace(eventNameSanitizer.Replace(eventName))
	if eventName != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", eventName); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}

func HeartbeatEvent() ActivityStreamEvent {
	return ActivityStreamEvent{
		Name: EventActivityHeartbeat,
		Data: heartbeatPayload{Timestamp: time.Now().UTC().Format(time.RFC3339Nano)},
	}
}
