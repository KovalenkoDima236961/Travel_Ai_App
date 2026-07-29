package alpha

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const (
	inviteColumns       = "id, code_prefix, code_suffix, expires_at, max_activations, current_activations, creator_user_id, notes, tester_group, enabled, created_at, updated_at"
	waitlistColumns     = "id, email, email_domain, status, invited_invite_id, source, notes, created_at, updated_at, invited_at, accepted_at, declined_at, removed_at"
	participantColumns  = "user_id, invite_id, alpha_participant, tester_group, invitation_date, first_login_at, first_trip_at, first_ai_generation_at, last_activity_at, active, created_at, updated_at"
	feedbackColumns     = "id, user_id, category, title, description_sanitized, status, priority, owner_user_id, internal_notes, metadata, app_version, browser_family, os_family, device_type, request_id, correlation_id, provider, model_alias, prompt_version, feature_flags, attachment_count, created_at, updated_at"
	attachmentColumns   = "id, feedback_id, file_name, mime_type, size_bytes, content_sha256, scan_status, storage_status, created_at"
	weeklyReportColumns = "id, week_start, week_end, summary_markdown, metrics, generated_by_user_id, generated_at"
)

var inviteCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{5,63}$`)

type Service struct {
	db  *storage.DB
	log *zap.Logger
	now func() time.Time
}

func New(db *storage.DB, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		db:  db,
		log: log,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Available() bool {
	return s != nil && s.db != nil
}

func (s *Service) CreateInvite(ctx context.Context, in CreateInviteInput) (*Invite, error) {
	if !s.Available() {
		return nil, ErrNotFound
	}
	if in.CreatorUserID == uuid.Nil {
		return nil, invalidInput("creator user id is required")
	}
	group, err := normalizeTesterGroup(in.TesterGroup)
	if err != nil {
		return nil, err
	}
	maxActivations := in.MaxActivations
	if maxActivations <= 0 {
		maxActivations = 1
	}
	code := normalizeInviteCode(in.Code)
	if code == "" {
		code, err = generateInviteCode()
		if err != nil {
			return nil, err
		}
	}
	if !inviteCodePattern.MatchString(code) {
		return nil, invalidInput("code must be 6-64 uppercase letters, digits, or hyphens")
	}
	prefix, suffix, _ := codeDisplay(code)
	id := uuid.New()
	invite, err := scanInvite(s.db.QueryRow(ctx,
		`INSERT INTO alpha_invites (id, code_hash, code_prefix, code_suffix, expires_at, max_activations, creator_user_id, notes, tester_group, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+inviteColumns,
		idArg(id),
		codeHash(code),
		prefix,
		suffix,
		in.ExpiresAt,
		maxActivations,
		idArg(in.CreatorUserID),
		sanitizeText(in.Notes, 500),
		group,
		in.Enabled,
	))
	if err != nil {
		if storage.UniqueConstraintViolation(err) {
			return nil, invalidInput("invite code already exists")
		}
		return nil, err
	}
	invite.Code = code
	return invite, nil
}

func (s *Service) ListInvites(ctx context.Context, limit, offset int) ([]Invite, error) {
	limit, offset = normalizeLimitOffset(limit, offset, 100)
	rows, err := s.db.Query(ctx, `SELECT `+inviteColumns+` FROM alpha_invites ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list alpha invites: %w", err)
	}
	defer rows.Close()
	invites := []Invite{}
	for rows.Next() {
		invite, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		invites = append(invites, *invite)
	}
	return invites, rows.Err()
}

func (s *Service) GetInvite(ctx context.Context, id uuid.UUID) (*Invite, error) {
	if id == uuid.Nil {
		return nil, invalidInput("invite id is required")
	}
	return scanInvite(s.db.QueryRow(ctx, `SELECT `+inviteColumns+` FROM alpha_invites WHERE id=$1`, idArg(id)))
}

func (s *Service) UpdateInvite(ctx context.Context, in UpdateInviteInput) (*Invite, error) {
	current, err := s.GetInvite(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	expiresAt := current.ExpiresAt
	if in.ExpiresAt != nil {
		expiresAt = *in.ExpiresAt
	}
	maxActivations := current.MaxActivations
	if in.MaxActivations != nil {
		if *in.MaxActivations < current.CurrentActivations {
			return nil, invalidInput("maxActivations cannot be below currentActivations")
		}
		maxActivations = *in.MaxActivations
	}
	notes := current.Notes
	if in.Notes != nil {
		notes = sanitizeText(*in.Notes, 500)
	}
	group := current.TesterGroup
	if in.TesterGroup != nil {
		group, err = normalizeTesterGroup(*in.TesterGroup)
		if err != nil {
			return nil, err
		}
	}
	enabled := current.Enabled
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	return scanInvite(s.db.QueryRow(ctx,
		`UPDATE alpha_invites
		 SET expires_at=$2, max_activations=$3, notes=$4, tester_group=$5, enabled=$6, updated_at=NOW()
		 WHERE id=$1
		 RETURNING `+inviteColumns,
		idArg(in.ID),
		expiresAt,
		maxActivations,
		notes,
		group,
		enabled,
	))
}

func (s *Service) DisableInvite(ctx context.Context, id uuid.UUID) (*Invite, error) {
	if id == uuid.Nil {
		return nil, invalidInput("invite id is required")
	}
	return scanInvite(s.db.QueryRow(ctx,
		`UPDATE alpha_invites SET enabled=false, updated_at=NOW() WHERE id=$1 RETURNING `+inviteColumns,
		idArg(id),
	))
}

func (s *Service) ActivateInvite(ctx context.Context, in ActivateInviteInput) (*Participant, error) {
	if in.UserID == uuid.Nil {
		return nil, invalidInput("user id is required")
	}
	code := normalizeInviteCode(in.Code)
	if !inviteCodePattern.MatchString(code) {
		return nil, ErrInviteUnavailable
	}
	if participant, err := s.GetParticipant(ctx, in.UserID); err == nil && participant.Active {
		return participant, nil
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin alpha invite activation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	invite, err := scanInvite(tx.QueryRow(ctx,
		`SELECT `+inviteColumns+`
		 FROM alpha_invites
		 WHERE code_hash=$1
		 FOR UPDATE`,
		codeHash(code),
	))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInviteUnavailable
		}
		return nil, err
	}
	if !invite.Enabled || (invite.ExpiresAt != nil && !invite.ExpiresAt.After(s.now())) || invite.CurrentActivations >= invite.MaxActivations {
		return nil, ErrInviteUnavailable
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO alpha_invite_activations (id, invite_id, user_id, request_id)
		 VALUES ($1,$2,$3,$4)`,
		idArg(uuid.New()),
		idArg(invite.ID),
		idArg(in.UserID),
		nullableString(in.RequestID),
	)
	if err != nil {
		if storage.UniqueConstraintViolation(err) {
			participant, getErr := scanParticipant(tx.QueryRow(ctx, `SELECT `+participantColumns+` FROM alpha_participants WHERE user_id=$1`, idArg(in.UserID)))
			if getErr == nil {
				return participant, tx.Commit(ctx)
			}
		}
		return nil, fmt.Errorf("record invite activation: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE alpha_invites SET current_activations=current_activations+1, updated_at=NOW() WHERE id=$1`, idArg(invite.ID))
	if err != nil {
		return nil, fmt.Errorf("increment invite activation count: %w", err)
	}

	now := s.now()
	participant, err := scanParticipant(tx.QueryRow(ctx,
		`INSERT INTO alpha_participants (user_id, invite_id, tester_group, invitation_date, last_activity_at, active)
		 VALUES ($1,$2,$3,$4,$4,true)
		 ON CONFLICT (user_id) DO UPDATE SET
		   invite_id=EXCLUDED.invite_id,
		   tester_group=EXCLUDED.tester_group,
		   invitation_date=COALESCE(alpha_participants.invitation_date, EXCLUDED.invitation_date),
		   last_activity_at=EXCLUDED.last_activity_at,
		   active=true,
		   updated_at=NOW()
		 RETURNING `+participantColumns,
		idArg(in.UserID),
		idArg(invite.ID),
		invite.TesterGroup,
		now,
	))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit alpha invite activation: %w", err)
	}
	return participant, nil
}

func (s *Service) GetParticipant(ctx context.Context, userID uuid.UUID) (*Participant, error) {
	if userID == uuid.Nil {
		return nil, invalidInput("user id is required")
	}
	return scanParticipant(s.db.QueryRow(ctx, `SELECT `+participantColumns+` FROM alpha_participants WHERE user_id=$1`, idArg(userID)))
}

func (s *Service) JoinWaitlist(ctx context.Context, in JoinWaitlistInput) (*WaitlistEntry, error) {
	email, domain, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, err
	}
	source := sanitizeText(in.Source, 80)
	if source == "" {
		source = "web"
	}
	entry, err := scanWaitlist(s.db.QueryRow(ctx,
		`INSERT INTO alpha_waitlist (id, email, email_hash, email_domain, source)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (email) DO UPDATE SET updated_at=NOW()
		 RETURNING `+waitlistColumns,
		idArg(uuid.New()),
		email,
		hashString(email),
		domain,
		source,
	))
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Service) ListWaitlist(ctx context.Context, status string, limit, offset int) ([]WaitlistEntry, error) {
	limit, offset = normalizeLimitOffset(limit, offset, 200)
	args := []any{limit, offset}
	query := `SELECT ` + waitlistColumns + ` FROM alpha_waitlist`
	if strings.TrimSpace(status) != "" {
		normalized, err := normalizeWaitlistStatus(status)
		if err != nil {
			return nil, err
		}
		query += ` WHERE status=$3`
		args = append(args, normalized)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alpha waitlist: %w", err)
	}
	defer rows.Close()
	entries := []WaitlistEntry{}
	for rows.Next() {
		entry, err := scanWaitlist(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

func (s *Service) UpdateWaitlist(ctx context.Context, in UpdateWaitlistInput) (*WaitlistEntry, error) {
	if in.ID == uuid.Nil {
		return nil, invalidInput("waitlist id is required")
	}
	current, err := scanWaitlist(s.db.QueryRow(ctx, `SELECT `+waitlistColumns+` FROM alpha_waitlist WHERE id=$1`, idArg(in.ID)))
	if err != nil {
		return nil, err
	}
	status := current.Status
	if in.Status != nil {
		status, err = normalizeWaitlistStatus(*in.Status)
		if err != nil {
			return nil, err
		}
	}
	notes := current.Notes
	if in.Notes != nil {
		notes = sanitizeText(*in.Notes, 1000)
	}
	return scanWaitlist(s.db.QueryRow(ctx,
		`UPDATE alpha_waitlist
		 SET status=$2,
		     notes=$3,
		     updated_at=NOW(),
		     accepted_at=CASE WHEN $2='accepted' THEN COALESCE(accepted_at, NOW()) ELSE accepted_at END,
		     declined_at=CASE WHEN $2='declined' THEN COALESCE(declined_at, NOW()) ELSE declined_at END,
		     removed_at=CASE WHEN $2='removed' THEN COALESCE(removed_at, NOW()) ELSE removed_at END
		 WHERE id=$1
		 RETURNING `+waitlistColumns,
		idArg(in.ID),
		status,
		notes,
	))
}

func (s *Service) InviteFromWaitlist(ctx context.Context, in InviteFromWaitlistInput) (*Invite, *WaitlistEntry, error) {
	if in.WaitlistID == uuid.Nil || in.CreatorUserID == uuid.Nil {
		return nil, nil, invalidInput("waitlist id and creator user id are required")
	}
	if in.MaxActivations <= 0 {
		in.MaxActivations = 1
	}
	group, err := normalizeTesterGroup(in.TesterGroup)
	if err != nil {
		return nil, nil, err
	}
	entry, err := scanWaitlist(s.db.QueryRow(ctx, `SELECT `+waitlistColumns+` FROM alpha_waitlist WHERE id=$1`, idArg(in.WaitlistID)))
	if err != nil {
		return nil, nil, err
	}
	if entry.Status == "removed" || entry.Status == "declined" {
		return nil, nil, invalidInput("waitlist entry is not eligible for invitation")
	}
	invite, err := s.CreateInvite(ctx, CreateInviteInput{
		CreatorUserID:  in.CreatorUserID,
		TesterGroup:    group,
		Notes:          sanitizeText(firstNonEmpty(in.Notes, "Waitlist invitation for "+entry.EmailDomain), 500),
		ExpiresAt:      in.ExpiresAt,
		MaxActivations: in.MaxActivations,
		Enabled:        true,
	})
	if err != nil {
		return nil, nil, err
	}
	updated, err := scanWaitlist(s.db.QueryRow(ctx,
		`UPDATE alpha_waitlist
		 SET status='invited', invited_invite_id=$2, invited_at=COALESCE(invited_at, NOW()), updated_at=NOW()
		 WHERE id=$1
		 RETURNING `+waitlistColumns,
		idArg(in.WaitlistID),
		idArg(invite.ID),
	))
	if err != nil {
		return nil, nil, err
	}
	return invite, updated, nil
}

func (s *Service) RecordEvent(ctx context.Context, in EventInput) (*Event, error) {
	eventName, defaultFeature, err := normalizeEventName(in.EventName)
	if err != nil {
		alphaAnalyticsRejectedTotal.WithLabelValues("unknown_event").Inc()
		return nil, err
	}
	feature := normalizeFeature(in.Feature, defaultFeature)
	metadata, err := sanitizeMetadata(in.Metadata)
	if err != nil {
		alphaAnalyticsRejectedTotal.WithLabelValues("invalid_metadata").Inc()
		return nil, err
	}
	occurredAt := boundedTime(in.OccurredAt)
	id := uuid.New()
	var sessionHash any
	if strings.TrimSpace(in.SessionID) != "" {
		sessionHash = hashString(in.SessionID)
	}
	var entityHash any
	if strings.TrimSpace(in.EntityID) != "" {
		entityHash = hashString(in.EntityID)
	}
	event := Event{
		ID:         id,
		UserID:     in.UserID,
		EventName:  eventName,
		Feature:    feature,
		EntityType: sanitizeText(in.EntityType, 80),
		Metadata:   metadata,
		OccurredAt: occurredAt,
		Source:     firstNonEmpty(sanitizeText(in.Source, 40), "web"),
	}
	if requestID, ok := optionalClean(in.RequestID, 120); ok {
		event.RequestID = &requestID
	}
	if correlationID, ok := optionalClean(in.CorrelationID, 120); ok {
		event.CorrelationID = &correlationID
	}
	if appVersion, ok := optionalClean(in.AppVersion, 80); ok {
		event.AppVersion = &appVersion
	}
	if browser, ok := optionalClean(in.BrowserFamily, 80); ok {
		event.BrowserFamily = &browser
	}
	if osFamily, ok := optionalClean(in.OSFamily, 80); ok {
		event.OSFamily = &osFamily
	}
	if device, ok := optionalClean(in.DeviceType, 40); ok {
		event.DeviceType = &device
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO product_analytics_events
		 (id, user_id, session_id_hash, event_name, feature, entity_type, entity_id_hash, metadata, occurred_at, request_id, correlation_id, app_version, browser_family, os_family, device_type, source)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		idArg(id),
		nullableIDArg(in.UserID),
		sessionHash,
		eventName,
		feature,
		event.EntityType,
		entityHash,
		[]byte(metadata),
		occurredAt,
		nullableTextPtr(event.RequestID),
		nullableTextPtr(event.CorrelationID),
		nullableTextPtr(event.AppVersion),
		nullableTextPtr(event.BrowserFamily),
		nullableTextPtr(event.OSFamily),
		nullableTextPtr(event.DeviceType),
		event.Source,
	)
	if err != nil {
		return nil, fmt.Errorf("record alpha analytics event: %w", err)
	}
	alphaAnalyticsEventsTotal.WithLabelValues(eventName, feature).Inc()
	recordEventMetrics(eventName)
	if in.UserID != nil {
		if err := s.applyParticipantLifecycle(ctx, *in.UserID, eventName, occurredAt); err != nil && s.log != nil {
			s.log.Warn("alpha participant lifecycle update failed", zap.Error(err), zap.String("event", eventName), zap.String("user_id", in.UserID.String()))
		}
	}
	return &event, nil
}

func (s *Service) SubmitFeedback(ctx context.Context, in SubmitFeedbackInput) (*FeedbackDetail, error) {
	if in.UserID == uuid.Nil {
		return nil, invalidInput("user id is required")
	}
	category, err := normalizeFeedbackCategory(in.Category)
	if err != nil {
		return nil, err
	}
	title := sanitizeText(in.Title, maxTitleLength)
	description := sanitizeText(in.Description, maxDescriptionLength)
	if title == "" {
		return nil, invalidInput("title is required")
	}
	if description == "" {
		return nil, invalidInput("description is required")
	}
	metadata, err := sanitizeMetadata(in.Metadata)
	if err != nil {
		return nil, err
	}
	flags, err := sanitizeMetadata(in.FeatureFlags)
	if err != nil {
		return nil, err
	}
	attachments, err := validateAttachments(in.Attachments)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin alpha feedback submit: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	feedbackID := uuid.New()
	feedback, err := scanFeedback(tx.QueryRow(ctx,
		`INSERT INTO alpha_feedback
		 (id, user_id, category, title, description_sanitized, metadata, app_version, browser_family, os_family, device_type, request_id, correlation_id, provider, model_alias, prompt_version, feature_flags, attachment_count)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		 RETURNING `+feedbackColumns,
		idArg(feedbackID),
		idArg(in.UserID),
		category,
		title,
		description,
		[]byte(metadata),
		nullableString(in.AppVersion),
		nullableString(in.BrowserFamily),
		nullableString(in.OSFamily),
		nullableString(in.DeviceType),
		nullableString(in.RequestID),
		nullableString(in.CorrelationID),
		nullableString(in.Provider),
		nullableString(in.ModelAlias),
		nullableString(in.PromptVersion),
		[]byte(flags),
		len(attachments),
	))
	if err != nil {
		return nil, err
	}
	storedAttachments := make([]FeedbackAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		stored, err := scanAttachment(tx.QueryRow(ctx,
			`INSERT INTO alpha_feedback_attachments
			 (id, feedback_id, file_name, mime_type, size_bytes, content_sha256, scan_status, storage_status)
			 VALUES ($1,$2,$3,$4,$5,$6,'clean','metadata_only')
			 RETURNING `+attachmentColumns,
			idArg(uuid.New()),
			idArg(feedbackID),
			attachment.FileName,
			attachment.MIMEType,
			attachment.SizeBytes,
			attachment.ContentSHA256,
		))
		if err != nil {
			return nil, err
		}
		storedAttachments = append(storedAttachments, *stored)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit alpha feedback submit: %w", err)
	}
	alphaFeedbackTotal.WithLabelValues(category).Inc()
	switch category {
	case "bug":
		alphaBugReportsTotal.Inc()
	case "feature_request":
		alphaFeatureRequestsTotal.Inc()
	}
	_ = s.applyParticipantLifecycle(ctx, in.UserID, "feedback_submitted", s.now())
	return &FeedbackDetail{Feedback: *feedback, Attachments: storedAttachments}, nil
}

func (s *Service) ListFeedback(ctx context.Context, category, status string, limit, offset int) ([]Feedback, error) {
	limit, offset = normalizeLimitOffset(limit, offset, 200)
	args := []any{limit, offset}
	where := []string{}
	if strings.TrimSpace(category) != "" {
		normalized, err := normalizeFeedbackCategory(category)
		if err != nil {
			return nil, err
		}
		args = append(args, normalized)
		where = append(where, fmt.Sprintf("category=$%d", len(args)))
	}
	if strings.TrimSpace(status) != "" {
		normalized, err := normalizeFeedbackStatus(status)
		if err != nil {
			return nil, err
		}
		args = append(args, normalized)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	query := `SELECT ` + feedbackColumns + ` FROM alpha_feedback`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alpha feedback: %w", err)
	}
	defer rows.Close()
	items := []Feedback{}
	for rows.Next() {
		feedback, err := scanFeedback(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *feedback)
	}
	return items, rows.Err()
}

func (s *Service) GetFeedback(ctx context.Context, id uuid.UUID) (*FeedbackDetail, error) {
	if id == uuid.Nil {
		return nil, invalidInput("feedback id is required")
	}
	feedback, err := scanFeedback(s.db.QueryRow(ctx, `SELECT `+feedbackColumns+` FROM alpha_feedback WHERE id=$1`, idArg(id)))
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT `+attachmentColumns+` FROM alpha_feedback_attachments WHERE feedback_id=$1 ORDER BY created_at ASC`, idArg(id))
	if err != nil {
		return nil, fmt.Errorf("list alpha feedback attachments: %w", err)
	}
	defer rows.Close()
	attachments := []FeedbackAttachment{}
	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *attachment)
	}
	return &FeedbackDetail{Feedback: *feedback, Attachments: attachments}, rows.Err()
}

func (s *Service) UpdateFeedback(ctx context.Context, in UpdateFeedbackInput) (*Feedback, error) {
	currentDetail, err := s.GetFeedback(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	current := currentDetail.Feedback
	status := current.Status
	if in.Status != nil {
		status, err = normalizeFeedbackStatus(*in.Status)
		if err != nil {
			return nil, err
		}
	}
	priority := current.Priority
	if in.Priority != nil {
		priority, err = normalizeFeedbackPriority(*in.Priority)
		if err != nil {
			return nil, err
		}
	}
	ownerID := current.OwnerUserID
	if in.OwnerUserID != nil {
		ownerID = *in.OwnerUserID
	}
	notes := current.InternalNotes
	if in.InternalNotes != nil {
		notes = sanitizeText(*in.InternalNotes, 2000)
	}
	return scanFeedback(s.db.QueryRow(ctx,
		`UPDATE alpha_feedback
		 SET status=$2, priority=$3, owner_user_id=$4, internal_notes=$5, updated_at=NOW()
		 WHERE id=$1
		 RETURNING `+feedbackColumns,
		idArg(in.ID),
		status,
		priority,
		nullableIDArg(ownerID),
		notes,
	))
}

func (s *Service) applyParticipantLifecycle(ctx context.Context, userID uuid.UUID, eventName string, at time.Time) error {
	if userID == uuid.Nil {
		return nil
	}
	firstLoginExpr := "first_login_at"
	firstTripExpr := "first_trip_at"
	firstAIExpr := "first_ai_generation_at"
	switch eventName {
	case "login", "signup_completed":
		firstLoginExpr = "COALESCE(first_login_at, $2)"
	case "trip_created", "second_trip_created":
		firstTripExpr = "COALESCE(first_trip_at, $2)"
	case "itinerary_generated", "ai_generation_completed":
		firstAIExpr = "COALESCE(first_ai_generation_at, $2)"
	}
	_, err := s.db.Exec(ctx,
		`UPDATE alpha_participants
		 SET first_login_at=`+firstLoginExpr+`,
		     first_trip_at=`+firstTripExpr+`,
		     first_ai_generation_at=`+firstAIExpr+`,
		     last_activity_at=$2,
		     updated_at=NOW()
		 WHERE user_id=$1`,
		idArg(userID),
		at,
	)
	return err
}

func normalizeInviteCode(value string) string {
	code := strings.ToUpper(strings.TrimSpace(value))
	code = strings.ReplaceAll(code, " ", "")
	return code
}

func generateInviteCode() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	if len(encoded) > 20 {
		encoded = encoded[:20]
	}
	parts := []string{}
	for len(encoded) > 0 {
		n := 4
		if len(encoded) < n {
			n = len(encoded)
		}
		parts = append(parts, encoded[:n])
		encoded = encoded[n:]
	}
	return strings.Join(parts, "-"), nil
}

func validateAttachments(in []FeedbackAttachmentInput) ([]FeedbackAttachmentInput, error) {
	if len(in) > 3 {
		return nil, invalidInput("at most 3 attachments are allowed")
	}
	out := make([]FeedbackAttachmentInput, 0, len(in))
	for _, attachment := range in {
		mimeType := strings.ToLower(strings.TrimSpace(attachment.MIMEType))
		if mimeType != "image/png" && mimeType != "image/jpeg" && mimeType != "image/webp" {
			return nil, ErrAttachmentRejected
		}
		if attachment.SizeBytes <= 0 || attachment.SizeBytes > 5*1024*1024 {
			return nil, ErrAttachmentRejected
		}
		digest := strings.ToLower(strings.TrimSpace(attachment.ContentSHA256))
		if digest == "" {
			digest = hashString(fmt.Sprintf("%s:%s:%d", attachment.FileName, mimeType, attachment.SizeBytes))
		}
		if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
			return nil, ErrAttachmentRejected
		}
		out = append(out, FeedbackAttachmentInput{
			FileName:      sanitizeText(firstNonEmpty(attachment.FileName, "screenshot"), 120),
			MIMEType:      mimeType,
			SizeBytes:     attachment.SizeBytes,
			ContentSHA256: digest,
		})
	}
	return out, nil
}

func recordEventMetrics(eventName string) {
	switch eventName {
	case "itinerary_generated", "itinerary_regenerated", "ai_generation_started":
		alphaTripGenerationTotal.WithLabelValues(eventName).Inc()
	case "ai_generation_completed":
		alphaAISuccessTotal.WithLabelValues("success").Inc()
	case "ai_generation_failed":
		alphaAISuccessTotal.WithLabelValues("failed").Inc()
	}
}

func normalizeLimitOffset(limit, offset, maxLimit int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if maxLimit <= 0 {
		maxLimit = 100
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func optionalClean(value string, maxLen int) (string, bool) {
	clean := sanitizeText(value, maxLen)
	return clean, clean != ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
