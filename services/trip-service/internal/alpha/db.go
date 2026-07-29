package alpha

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

func idArg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}

func nullableIDArg(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return idArg(*id)
}

func uuidFromPg(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func uuidValueFromPg(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return uuid.UUID(value.Bytes)
}

func timeFromPg(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func dateFromPg(value pgtype.Date) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func nullableTextPtr(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}

func jsonOrEmpty(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func scanInvite(row pgx.Row) (*Invite, error) {
	var (
		invite               Invite
		id, creator          pgtype.UUID
		expiresAt            pgtype.Timestamptz
		createdAt, updatedAt pgtype.Timestamptz
		prefix, suffix       string
	)
	err := row.Scan(
		&id,
		&prefix,
		&suffix,
		&expiresAt,
		&invite.MaxActivations,
		&invite.CurrentActivations,
		&creator,
		&invite.Notes,
		&invite.TesterGroup,
		&invite.Enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan alpha invite: %w", err)
	}
	invite.ID = uuidValueFromPg(id)
	invite.CreatorUserID = uuidValueFromPg(creator)
	invite.ExpiresAt = timeFromPg(expiresAt)
	invite.CreatedAt = createdAt.Time.UTC()
	invite.UpdatedAt = updatedAt.Time.UTC()
	invite.CodeDisplay = prefix + "..." + suffix
	return &invite, nil
}

func scanWaitlist(row pgx.Row) (*WaitlistEntry, error) {
	var (
		entry                                        WaitlistEntry
		id, inviteID                                 pgtype.UUID
		createdAt, updatedAt                         pgtype.Timestamptz
		invitedAt, acceptedAt, declinedAt, removedAt pgtype.Timestamptz
	)
	err := row.Scan(
		&id,
		&entry.Email,
		&entry.EmailDomain,
		&entry.Status,
		&inviteID,
		&entry.Source,
		&entry.Notes,
		&createdAt,
		&updatedAt,
		&invitedAt,
		&acceptedAt,
		&declinedAt,
		&removedAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan alpha waitlist entry: %w", err)
	}
	entry.ID = uuidValueFromPg(id)
	entry.InvitedInviteID = uuidFromPg(inviteID)
	entry.CreatedAt = createdAt.Time.UTC()
	entry.UpdatedAt = updatedAt.Time.UTC()
	entry.InvitedAt = timeFromPg(invitedAt)
	entry.AcceptedAt = timeFromPg(acceptedAt)
	entry.DeclinedAt = timeFromPg(declinedAt)
	entry.RemovedAt = timeFromPg(removedAt)
	return &entry, nil
}

func scanParticipant(row pgx.Row) (*Participant, error) {
	var (
		participant                                    Participant
		userID, inviteID                               pgtype.UUID
		invitationDate, firstLogin, firstTrip, firstAI pgtype.Timestamptz
		lastActivity, createdAt, updatedAt             pgtype.Timestamptz
	)
	err := row.Scan(
		&userID,
		&inviteID,
		&participant.AlphaParticipant,
		&participant.TesterGroup,
		&invitationDate,
		&firstLogin,
		&firstTrip,
		&firstAI,
		&lastActivity,
		&participant.Active,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan alpha participant: %w", err)
	}
	participant.UserID = uuidValueFromPg(userID)
	participant.InviteID = uuidFromPg(inviteID)
	participant.InvitationDate = timeFromPg(invitationDate)
	participant.FirstLoginAt = timeFromPg(firstLogin)
	participant.FirstTripAt = timeFromPg(firstTrip)
	participant.FirstAIGenerationAt = timeFromPg(firstAI)
	participant.LastActivityAt = timeFromPg(lastActivity)
	participant.CreatedAt = createdAt.Time.UTC()
	participant.UpdatedAt = updatedAt.Time.UTC()
	return &participant, nil
}

func scanFeedback(row pgx.Row) (*Feedback, error) {
	var (
		feedback                        Feedback
		id, userID, ownerID             pgtype.UUID
		appVersion, browser, os, device pgtype.Text
		requestID, correlationID        pgtype.Text
		provider, modelAlias, prompt    pgtype.Text
		metadataRaw, flagsRaw           []byte
		createdAt, updatedAt            pgtype.Timestamptz
	)
	err := row.Scan(
		&id,
		&userID,
		&feedback.Category,
		&feedback.Title,
		&feedback.DescriptionSanitized,
		&feedback.Status,
		&feedback.Priority,
		&ownerID,
		&feedback.InternalNotes,
		&metadataRaw,
		&appVersion,
		&browser,
		&os,
		&device,
		&requestID,
		&correlationID,
		&provider,
		&modelAlias,
		&prompt,
		&flagsRaw,
		&feedback.AttachmentCount,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan alpha feedback: %w", err)
	}
	feedback.ID = uuidValueFromPg(id)
	feedback.UserID = uuidValueFromPg(userID)
	feedback.OwnerUserID = uuidFromPg(ownerID)
	feedback.Metadata = jsonOrEmpty(metadataRaw)
	feedback.FeatureFlags = jsonOrEmpty(flagsRaw)
	feedback.AppVersion = textPtr(appVersion)
	feedback.BrowserFamily = textPtr(browser)
	feedback.OSFamily = textPtr(os)
	feedback.DeviceType = textPtr(device)
	feedback.RequestID = textPtr(requestID)
	feedback.CorrelationID = textPtr(correlationID)
	feedback.Provider = textPtr(provider)
	feedback.ModelAlias = textPtr(modelAlias)
	feedback.PromptVersion = textPtr(prompt)
	feedback.CreatedAt = createdAt.Time.UTC()
	feedback.UpdatedAt = updatedAt.Time.UTC()
	return &feedback, nil
}

func scanAttachment(row pgx.Row) (*FeedbackAttachment, error) {
	var (
		attachment     FeedbackAttachment
		id, feedbackID pgtype.UUID
		createdAt      pgtype.Timestamptz
	)
	err := row.Scan(
		&id,
		&feedbackID,
		&attachment.FileName,
		&attachment.MIMEType,
		&attachment.SizeBytes,
		&attachment.ContentSHA256,
		&attachment.ScanStatus,
		&attachment.StorageStatus,
		&createdAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan alpha feedback attachment: %w", err)
	}
	attachment.ID = uuidValueFromPg(id)
	attachment.FeedbackID = uuidValueFromPg(feedbackID)
	attachment.CreatedAt = createdAt.Time.UTC()
	return &attachment, nil
}

func scanWeeklyReport(row pgx.Row) (*WeeklyReport, error) {
	var (
		report             WeeklyReport
		id, generatedBy    pgtype.UUID
		weekStart, weekEnd pgtype.Date
		metricsRaw         []byte
		generatedAt        pgtype.Timestamptz
	)
	err := row.Scan(
		&id,
		&weekStart,
		&weekEnd,
		&report.SummaryMarkdown,
		&metricsRaw,
		&generatedBy,
		&generatedAt,
	)
	if err != nil {
		if storage.NoRowsFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan weekly alpha report: %w", err)
	}
	report.ID = uuidValueFromPg(id)
	report.WeekStart = dateFromPg(weekStart)
	report.WeekEnd = dateFromPg(weekEnd)
	report.Metrics = jsonOrEmpty(metricsRaw)
	report.GeneratedByUserID = uuidFromPg(generatedBy)
	report.GeneratedAt = generatedAt.Time.UTC()
	return &report, nil
}
