package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var eventNameSanitizer = strings.NewReplacer("\r", " ", "\n", " ")

// WriteSSE writes one well-formed Server-Sent Event block.
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
