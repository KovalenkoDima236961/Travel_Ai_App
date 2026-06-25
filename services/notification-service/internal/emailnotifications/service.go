package emailnotifications

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/email"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
)

// UserLookup resolves recipient profiles by id. The concrete users.Client
// satisfies it; tests substitute a fake.
type UserLookup interface {
	LookupByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]users.UserProfile, error)
}

// EmailSendResult summarises the outcome of one batch's email fan-out. It is
// returned to the internal batch handler and surfaced in the response metadata.
//
//	Attempted = messages actually handed to the sender
//	Sent      = attempted that succeeded
//	Failed    = attempted that failed to send
//	Skipped   = candidates not emailed (disabled, not allowlisted, self,
//	            preference-disabled, no recipient email, or no template)
type EmailSendResult struct {
	Attempted           int `json:"attempted"`
	Sent                int `json:"sent"`
	Skipped             int `json:"skipped"`
	SkippedByPreference int `json:"skippedByPreference"`
	Failed              int `json:"failed"`
}

// Config configures the email orchestration.
type Config struct {
	Enabled          bool
	FailOpen         bool
	PublicWebBaseURL string
	Types            []string
}

// Service filters notifications by policy, resolves recipient emails, builds
// messages, and sends them. In-app notification creation always happens first
// and is never rolled back because of an email failure.
type Service struct {
	policy           Policy
	lookup           UserLookup
	sender           email.EmailSender
	publicWebBaseURL string
	failOpen         bool
	log              *zap.Logger
}

// New constructs the email orchestration service.
func New(cfg Config, lookup UserLookup, sender email.EmailSender, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		policy:           NewPolicy(cfg.Enabled, cfg.Types),
		lookup:           lookup,
		sender:           sender,
		publicWebBaseURL: cfg.PublicWebBaseURL,
		failOpen:         cfg.FailOpen,
		log:              log,
	}
}

// SendEmailsForNotifications sends emails for the eligible notifications in a
// created batch and returns a result summary.
//
// Failure behavior:
//   - Recipient lookup error: fail-open logs and skips all eligible (no error);
//     fail-closed returns the error (the handler maps it to 502). In-app rows
//     are already committed and are never rolled back.
//   - Individual send failure: counted as Failed and logged. Fail-closed returns
//     a non-nil error after attempting the rest of the batch; fail-open returns
//     nil so the in-app batch still reports success.
func (s *Service) SendEmailsForNotifications(ctx context.Context, notifications []entity.Notification, gates ...EmailPreferenceGate) (EmailSendResult, error) {
	var result EmailSendResult
	gate := firstGate(gates)

	eligible := make([]entity.Notification, 0, len(notifications))
	idSet := make(map[uuid.UUID]struct{})
	for _, n := range notifications {
		decision := s.policy.Evaluate(n, gate)
		if !decision.Send {
			result.Skipped++
			if decision.SkippedByPreference {
				result.SkippedByPreference++
			}
			continue
		}
		eligible = append(eligible, n)
		idSet[n.UserID] = struct{}{}
	}
	if len(eligible) == 0 {
		return result, nil
	}

	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	profiles, err := s.lookup.LookupByIDs(ctx, ids)
	if err != nil {
		if s.failOpen {
			s.log.Warn("recipient lookup failed; skipping emails (fail-open)",
				zap.Int("eligible", len(eligible)),
				zap.Error(err),
			)
			result.Skipped += len(eligible)
			return result, nil
		}
		result.Failed += len(eligible)
		return result, fmt.Errorf("resolve email recipients: %w", err)
	}

	var firstSendErr error
	for i := range eligible {
		n := eligible[i]
		profile, ok := profiles[n.UserID]
		if !ok || profile.Email == "" {
			result.Skipped++
			s.log.Warn("no recipient email for notification; skipping email",
				zap.String("user_id", n.UserID.String()),
				zap.String("type", n.Type),
			)
			continue
		}

		msg, err := BuildEmailForNotification(BuildEmailInput{
			Notification:     n,
			Recipient:        profile,
			PublicWebBaseURL: s.publicWebBaseURL,
		})
		if err != nil {
			result.Skipped++
			s.log.Warn("could not build email for notification; skipping",
				zap.String("type", n.Type),
				zap.Error(err),
			)
			continue
		}

		result.Attempted++
		if err := s.sender.Send(ctx, msg); err != nil {
			result.Failed++
			s.log.Warn("email send failed",
				zap.String("to", email.MaskEmail(profile.Email)),
				zap.String("type", n.Type),
				zap.Error(err),
			)
			if !s.failOpen && firstSendErr == nil {
				firstSendErr = err
			}
			continue
		}
		result.Sent++
	}

	if firstSendErr != nil {
		return result, fmt.Errorf("send notification emails: %w", firstSendErr)
	}
	return result, nil
}
