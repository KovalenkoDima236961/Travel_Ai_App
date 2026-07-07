package email

import (
	"fmt"
	"strings"
)

// EmailMessage is a single email to deliver. TextBody is required; HTMLBody is
// optional and, when present, is sent as the richer alternative part.
//
// Callers must never place secrets in any field: no JWTs, refresh tokens, share
// access tokens, share passwords, API keys, or full private itinerary payloads.
type EmailMessage struct {
	ToEmail  string
	ToName   string
	Subject  string
	TextBody string
	HTMLBody string
}

// Validate enforces the minimum shape of a sendable message: a recipient
// address, a subject, and a plain-text body. An empty recipient or subject is
// rejected so a malformed template never reaches the wire.
func (m EmailMessage) Validate() error {
	if strings.TrimSpace(m.ToEmail) == "" {
		return fmt.Errorf("email message: recipient address is required")
	}
	if strings.TrimSpace(m.Subject) == "" {
		return fmt.Errorf("email message: subject is required")
	}
	if strings.TrimSpace(m.TextBody) == "" {
		return fmt.Errorf("email message: text body is required")
	}
	return nil
}

// MaskEmail partially masks an email address for safe logging, e.g.
// "anna@example.com" -> "an***@example.com". Invalid or very short addresses are
// fully masked so a malformed value never leaks in logs.
func MaskEmail(email string) string {
	trimmed := strings.TrimSpace(email)
	at := strings.LastIndex(trimmed, "@")
	if at <= 0 || at == len(trimmed)-1 {
		return "***"
	}
	local := trimmed[:at]
	domain := trimmed[at:]
	if len(local) <= 2 {
		return "***" + domain
	}
	return local[:2] + "***" + domain
}
