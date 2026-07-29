package aimodel

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiprivacy"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const MaxFeedbackNoteLength = 500

var allowedFeedbackCategories = map[string]bool{
	"better_than_standard": true,
	"worse_than_standard":  true,
	"bad_places":           true,
	"bad_schedule":         true,
	"too_slow":             true,
	"wrong_language":       true,
	"formatting_problem":   true,
	"other":                true,
}

type FeedbackInput struct {
	TripID              uuid.UUID
	GenerationJobID     *uuid.UUID
	ItineraryVersionID  *uuid.UUID
	RequestAssignmentID *uuid.UUID
	DeploymentID        *uuid.UUID
	UserID              uuid.UUID
	Feedback            string
	Note                string
}

type FeedbackRecord struct {
	ID                  uuid.UUID  `json:"id"`
	TripID              uuid.UUID  `json:"tripId"`
	GenerationJobID     *uuid.UUID `json:"generationJobId,omitempty"`
	ItineraryVersionID  *uuid.UUID `json:"itineraryVersionId,omitempty"`
	RequestAssignmentID *uuid.UUID `json:"requestAssignmentId,omitempty"`
	DeploymentID        *uuid.UUID `json:"deploymentId,omitempty"`
	UserID              uuid.UUID  `json:"userId"`
	Feedback            string     `json:"feedback"`
	NoteSanitized       *string    `json:"noteSanitized,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type FeedbackService struct {
	db *storage.DB
}

func NewFeedbackService(db *storage.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

func (s *FeedbackService) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *FeedbackService) RecordFeedback(ctx context.Context, in FeedbackInput) (*FeedbackRecord, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ai model feedback is not configured")
	}
	category := strings.TrimSpace(in.Feedback)
	if !allowedFeedbackCategories[category] {
		return nil, fmt.Errorf("invalid ai model feedback category")
	}
	note := SanitizeFeedbackNote(in.Note)
	record := FeedbackRecord{
		ID:                  uuid.New(),
		TripID:              in.TripID,
		GenerationJobID:     in.GenerationJobID,
		ItineraryVersionID:  in.ItineraryVersionID,
		RequestAssignmentID: in.RequestAssignmentID,
		DeploymentID:        in.DeploymentID,
		UserID:              in.UserID,
		Feedback:            category,
		NoteSanitized:       note,
	}

	err := s.db.QueryRow(ctx, `
INSERT INTO ai_model_feedback (
    id,
    trip_id,
    generation_job_id,
    itinerary_version_id,
    request_assignment_id,
    deployment_id,
    user_id,
    feedback,
    note_sanitized
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING created_at`,
		idArg(record.ID),
		idArg(record.TripID),
		nullableIDArg(record.GenerationJobID),
		nullableIDArg(record.ItineraryVersionID),
		nullableIDArg(record.RequestAssignmentID),
		nullableIDArg(record.DeploymentID),
		idArg(record.UserID),
		record.Feedback,
		nullableText(record.NoteSanitized),
	).Scan(&record.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record ai model feedback: %w", err)
	}
	return &record, nil
}

func SanitizeFeedbackNote(note string) *string {
	trimmed := strings.TrimSpace(note)
	if trimmed == "" {
		return nil
	}
	redacted, _ := aiprivacy.RedactText(trimmed)
	if utf8.RuneCountInString(redacted) > MaxFeedbackNoteLength {
		runes := []rune(redacted)
		redacted = string(runes[:MaxFeedbackNoteLength])
	}
	return &redacted
}

func idArg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func nullableIDArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return idArg(*id)
}

func nullableText(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}
