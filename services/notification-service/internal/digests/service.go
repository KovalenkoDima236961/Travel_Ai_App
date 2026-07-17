package digests

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/email"
)

type Repository interface {
	QueueDigestItem(ctx context.Context, input QueueInput) (*entity.NotificationDigestBatch, bool, bool, error)
	ClaimDueDigestBatch(ctx context.Context, now time.Time) (*entity.NotificationDigestBatch, error)
	GetDigestBatchByID(ctx context.Context, id uuid.UUID) (*entity.NotificationDigestBatch, error)
	GetDigestBatchByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*entity.NotificationDigestBatch, error)
	ListDigestBatchesByUser(ctx context.Context, input ListInput) ([]entity.NotificationDigestBatch, error)
	MarkDigestBatchSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	MarkDigestBatchFailed(ctx context.Context, id uuid.UUID, retry bool, nextAttempt *time.Time, code, safeMessage string) error
}

type UserLookup interface {
	LookupByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]users.UserProfile, error)
}
type PushDispatcher interface {
	SendDigest(ctx context.Context, userID uuid.UUID, title, body string) error
}

type Service struct {
	repo   Repository
	lookup UserLookup
	sender email.EmailSender
	push   PushDispatcher
	cfg    Config
	log    *zap.Logger
}

func New(repo Repository, lookup UserLookup, sender email.EmailSender, push PushDispatcher, cfg Config, log *zap.Logger) *Service {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 5 * time.Minute
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, lookup: lookup, sender: sender, push: push, cfg: cfg, log: log}
}

func (s *Service) Queue(ctx context.Context, input QueueInput) (bool, error) {
	if input.ScheduledFor.IsZero() {
		input.ScheduledFor = time.Now().UTC().Add(time.Hour)
	}
	_, grouped, batchCreated, err := s.repo.QueueDigestItem(ctx, input)
	if err == nil {
		recordDigestQueued(input.Channel, input.Mode, grouped, batchCreated)
	}
	return grouped, err
}

func (s *Service) ProcessDue(ctx context.Context, input ProcessInput) (*ProcessResult, error) {
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if input.Limit <= 0 {
		input.Limit = 100
	}
	if input.Limit > 500 {
		input.Limit = 500
	}
	result := &ProcessResult{}
	for result.Processed < input.Limit {
		batch, err := s.repo.ClaimDueDigestBatch(ctx, input.Now)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			break
		}
		result.Processed++
		full, err := s.repo.GetDigestBatchByID(ctx, batch.ID)
		if err == nil && len(full.Items) > 0 {
			err = s.deliver(ctx, full)
		}
		if err == nil {
			if err = s.repo.MarkDigestBatchSent(ctx, batch.ID, input.Now); err != nil {
				return nil, err
			}
			result.Sent++
			recordDigestSent(batch.Channel, batch.Mode)
			s.log.Info("notification digest batch sent",
				zap.String("digest_id", batch.ID.String()),
				zap.String("channel", batch.Channel),
				zap.String("mode", batch.Mode),
				zap.Int("attempt", batch.Attempts),
			)
			continue
		}
		retry := batch.Attempts < s.cfg.MaxAttempts
		var next *time.Time
		if retry {
			value := input.Now.Add(s.cfg.RetryDelay)
			next = &value
			result.Retrying++
		} else {
			result.Failed++
		}
		if markErr := s.repo.MarkDigestBatchFailed(ctx, batch.ID, retry, next, "delivery_failed", "Digest delivery is temporarily unavailable."); markErr != nil {
			return nil, markErr
		}
		recordDigestFailed(batch.Channel, batch.Mode)
		s.log.Warn("notification digest delivery failed", zap.String("digest_id", batch.ID.String()), zap.String("channel", batch.Channel), zap.Int("attempt", batch.Attempts), zap.Bool("retry", retry), zap.Error(err))
	}
	return result, nil
}

func (s *Service) ListPending(ctx context.Context, userID uuid.UUID, limit int) ([]entity.NotificationDigestBatch, error) {
	return s.listWithItems(ctx, ListInput{UserID: userID, Status: StatusPending, Limit: limit})
}
func (s *Service) ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]entity.NotificationDigestBatch, error) {
	return s.listWithItems(ctx, ListInput{UserID: userID, Status: "history", Limit: limit})
}
func (s *Service) Get(ctx context.Context, id, userID uuid.UUID) (*entity.NotificationDigestBatch, error) {
	return s.repo.GetDigestBatchByIDAndUser(ctx, id, userID)
}

func (s *Service) listWithItems(ctx context.Context, input ListInput) ([]entity.NotificationDigestBatch, error) {
	batches, err := s.repo.ListDigestBatchesByUser(ctx, input)
	if err != nil {
		return nil, err
	}
	for i := range batches {
		full, err := s.repo.GetDigestBatchByIDAndUser(ctx, batches[i].ID, input.UserID)
		if err != nil {
			return nil, err
		}
		batches[i] = *full
	}
	return batches, nil
}

func (s *Service) deliver(ctx context.Context, batch *entity.NotificationDigestBatch) error {
	count := digestEventCount(batch.Items)
	switch batch.Channel {
	case "in_app":
		return nil
	case "push":
		if s.push == nil {
			return nil
		}
		return s.push.SendDigest(ctx, batch.UserID, "Trip update digest", fmt.Sprintf("%s across %d groups", pluralUpdates(count), len(batch.Items)))
	case "email":
		if s.lookup == nil || s.sender == nil {
			return fmt.Errorf("email digest delivery is not configured")
		}
		profiles, err := s.lookup.LookupByIDs(ctx, []uuid.UUID{batch.UserID})
		if err != nil {
			return err
		}
		profile, ok := profiles[batch.UserID]
		if !ok || strings.TrimSpace(profile.Email) == "" {
			return fmt.Errorf("digest recipient email is unavailable")
		}
		return s.sender.Send(ctx, buildDigestEmail(profile, batch, s.cfg.PublicWebBaseURL))
	default:
		return fmt.Errorf("unsupported digest channel %q", batch.Channel)
	}
}

func buildDigestEmail(profile users.UserProfile, batch *entity.NotificationDigestBatch, baseURL string) email.EmailMessage {
	groups := groupedItems(batch.Items)
	subject := "Your trip updates"
	if batch.Mode == "daily_digest" {
		subject = "Your daily trip updates"
	} else if batch.Mode == "weekly_digest" {
		subject = "Your weekly trip updates"
	} else if batch.Mode == "hourly_digest" {
		subject = "Your hourly trip updates"
	}
	var textBody, htmlBody strings.Builder
	name := strings.TrimSpace(profile.DisplayName)
	if name == "" {
		name = "there"
	}
	textBody.WriteString("Hi " + name + ",\n\n")
	htmlBody.WriteString("<div><p>Hi " + html.EscapeString(name) + ",</p>")
	for _, group := range groups {
		textBody.WriteString(group.name + "\n")
		htmlBody.WriteString("<h3>" + html.EscapeString(group.name) + "</h3><ul>")
		for _, item := range group.items {
			line := safeDigestLine(item)
			textBody.WriteString("- " + line + "\n")
			htmlBody.WriteString("<li>" + html.EscapeString(line) + "</li>")
		}
		textBody.WriteString("\n")
		htmlBody.WriteString("</ul>")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	textBody.WriteString("Open notifications: " + baseURL + "/notifications\nNotification settings: " + baseURL + "/settings\n")
	htmlBody.WriteString(`<p><a href="` + html.EscapeString(baseURL+"/notifications") + `">Open notifications</a> · <a href="` + html.EscapeString(baseURL+"/settings") + `">Notification settings</a></p></div>`)
	return email.EmailMessage{ToEmail: profile.Email, ToName: profile.DisplayName, Subject: subject, TextBody: textBody.String(), HTMLBody: htmlBody.String()}
}

func safeDigestLine(item entity.NotificationDigestItem) string {
	label := map[string]string{
		"collaboration": "collaboration update", "comments": "comment update",
		"trip_updates": "trip update", "role_changes": "role change",
		"checklist": "checklist update", "checklist_reminders": "checklist reminder",
		"reminders": "trip reminder", "pre_trip_reminders": "pre-trip reminder",
		"expenses": "expense update", "settlements": "settlement update",
		"approval": "approval update", "budget": "budget update",
		"health": "Trip Health update", "offline_sync": "offline sync update",
		"calendar": "calendar sync update", "ai_generation": "AI generation update",
		"security": "security update", "system": "system update",
	}[item.Category]
	if label == "" {
		label = "notification update"
	}
	if item.EventCount == 1 {
		return "1 " + label
	}
	return fmt.Sprintf("%d %ss", item.EventCount, label)
}

type digestGroup struct {
	name  string
	items []entity.NotificationDigestItem
}

func groupedItems(items []entity.NotificationDigestItem) []digestGroup {
	byTrip := make(map[string][]entity.NotificationDigestItem)
	for _, item := range items {
		key := "Other updates"
		if name, ok := item.Metadata["tripName"].(string); ok && strings.TrimSpace(name) != "" {
			key = name
		} else if destination, ok := item.Metadata["destination"].(string); ok && strings.TrimSpace(destination) != "" {
			key = destination
		} else if item.TripID != nil {
			key = "Trip " + item.TripID.String()[:8]
		}
		byTrip[key] = append(byTrip[key], item)
	}
	keys := make([]string, 0, len(byTrip))
	for key := range byTrip {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]digestGroup, 0, len(keys))
	for _, key := range keys {
		sort.SliceStable(byTrip[key], func(i, j int) bool { return byTrip[key][i].Category < byTrip[key][j].Category })
		out = append(out, digestGroup{name: key, items: byTrip[key]})
	}
	return out
}
func digestEventCount(items []entity.NotificationDigestItem) int {
	count := 0
	for _, item := range items {
		count += item.EventCount
	}
	return count
}
func pluralUpdates(count int) string {
	if count == 1 {
		return "1 update"
	}
	return fmt.Sprintf("%d updates", count)
}
