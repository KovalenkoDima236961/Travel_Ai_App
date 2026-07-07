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

// EmailPreferenceGate reports whether email is enabled for the notification's
// mapped category for the recipient. The preferences EffectiveSet implements it.
type EmailPreferenceGate interface {
	AllowEmail(userID uuid.UUID, notificationType string) bool
}

// EmailDecision is the policy result for one notification. SkippedByPreference
// is true only when the notification passed global/self/allowlist checks but the
// recipient disabled email for the mapped category.
type EmailDecision struct {
	Send                bool
	SkippedByPreference bool
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
//   - the recipient's email preference must allow the mapped category when a
//     preference gate is supplied;
//   - the recipient must be a valid user;
//   - the recipient must not be the actor (never email a user about their own
//     action — guarded here in addition to Trip Service and the create path).
func (p Policy) ShouldSendEmail(notification entity.Notification, gates ...EmailPreferenceGate) bool {
	return p.Evaluate(notification, firstGate(gates)).Send
}

// Evaluate returns the full policy decision, including whether a skip was caused
// specifically by recipient email preferences.
func (p Policy) Evaluate(notification entity.Notification, gate EmailPreferenceGate) EmailDecision {
	if !p.enabled {
		return EmailDecision{}
	}
	if notification.UserID == uuid.Nil {
		return EmailDecision{}
	}
	if notification.ActorUserID != nil && *notification.ActorUserID == notification.UserID {
		return EmailDecision{}
	}
	if _, ok := p.allowed[notification.Type]; !ok {
		return EmailDecision{}
	}
	if gate != nil && !gate.AllowEmail(notification.UserID, notification.Type) {
		return EmailDecision{SkippedByPreference: true}
	}
	return EmailDecision{Send: true}
}

func firstGate(gates []EmailPreferenceGate) EmailPreferenceGate {
	if len(gates) == 0 {
		return nil
	}
	return gates[0]
}
