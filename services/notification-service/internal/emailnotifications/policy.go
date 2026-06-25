// Package emailnotifications orchestrates optional email delivery for selected
// notification types. After in-app notification rows are created, the internal
// batch handler hands the created notifications to this package, which decides
// (policy) which ones warrant an email, resolves recipient emails, builds the
// message (templates), and sends it (email package).
//
// Privacy: templates never include secrets, JWTs, share access tokens, share
// passwords, full itinerary payloads, or full comment bodies. Recipient emails
// are masked in logs.
package emailnotifications

import (
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

// Policy decides whether a single notification should trigger an email. It is
// pure and side-effect free so it is trivially unit-testable.
type Policy struct {
	enabled bool
	allowed map[string]struct{}
}

// NewPolicy builds a policy from the global enabled flag and the allowlist of
// notification types that may trigger email.
func NewPolicy(enabled bool, types []string) Policy {
	allowed := make(map[string]struct{}, len(types))
	for _, t := range types {
		if trimmed := strings.TrimSpace(t); trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	return Policy{enabled: enabled, allowed: allowed}
}

// ShouldSendEmail reports whether the notification warrants an email:
//   - email must be enabled globally;
//   - the notification type must be in the allowlist;
//   - the recipient must be a valid user;
//   - the recipient must not be the actor (never email a user about their own
//     action — guarded here in addition to Trip Service and the create path).
func (p Policy) ShouldSendEmail(notification entity.Notification) bool {
	if !p.enabled {
		return false
	}
	if notification.UserID == uuid.Nil {
		return false
	}
	if notification.ActorUserID != nil && *notification.ActorUserID == notification.UserID {
		return false
	}
	if _, ok := p.allowed[notification.Type]; !ok {
		return false
	}
	return true
}
