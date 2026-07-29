package alpha

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSanitizeMetadataDropsSensitiveKeys(t *testing.T) {
	raw := json.RawMessage(`{
		"feature":"trips",
		"email":"person@example.com",
		"prompt":"make me a trip",
		"token":"abc",
		"safeNested":{"placeCategory":"museum","latitude":48.2}
	}`)

	sanitized, err := sanitizeMetadata(raw)
	if err != nil {
		t.Fatalf("sanitizeMetadata() error = %v", err)
	}
	text := string(sanitized)
	for _, forbidden := range []string{"email", "person@example.com", "prompt", "token", "latitude"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("sanitized metadata leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "museum") {
		t.Fatalf("sanitized metadata removed safe value: %s", text)
	}
}

func TestNormalizeEventNameRejectsUnknownEvents(t *testing.T) {
	if _, _, err := normalizeEventName("password_entered"); err == nil {
		t.Fatal("normalizeEventName accepted an unknown event")
	}
}

func TestNormalizeEventNameAcceptsGenericFeedback(t *testing.T) {
	eventName, feature, err := normalizeEventName("feedback_submitted")
	if err != nil {
		t.Fatalf("normalizeEventName() error = %v", err)
	}
	if eventName != "feedback_submitted" || feature != "feedback" {
		t.Fatalf("normalizeEventName() = %q/%q", eventName, feature)
	}
}

func TestValidateAttachmentsRejectsUnsafeInputs(t *testing.T) {
	validDigest := strings.Repeat("a", 64)
	tests := []struct {
		name       string
		attachment FeedbackAttachmentInput
	}{
		{
			name: "too large",
			attachment: FeedbackAttachmentInput{
				FileName: "screen.png", MIMEType: "image/png", SizeBytes: 5*1024*1024 + 1, ContentSHA256: validDigest,
			},
		},
		{
			name: "unsupported type",
			attachment: FeedbackAttachmentInput{
				FileName: "screen.svg", MIMEType: "image/svg+xml", SizeBytes: 100, ContentSHA256: validDigest,
			},
		},
		{
			name: "invalid digest",
			attachment: FeedbackAttachmentInput{
				FileName: "screen.png", MIMEType: "image/png", SizeBytes: 100, ContentSHA256: "not-a-digest",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateAttachments([]FeedbackAttachmentInput{test.attachment})
			if !errors.Is(err, ErrAttachmentRejected) {
				t.Fatalf("validateAttachments() error = %v, want ErrAttachmentRejected", err)
			}
		})
	}
}
