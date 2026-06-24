package activity

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	// DefaultLimit is the page size used when the caller does not specify one.
	DefaultLimit = 30
	// MaxLimit caps the page size a caller may request.
	MaxLimit = 100

	// maxMetadataKeys caps how many keys are persisted per event so a stray
	// large map can never bloat a row.
	maxMetadataKeys = 24
	// maxMetadataStringLen truncates long string values (defence in depth; the
	// call sites already keep metadata small and free of sensitive data).
	maxMetadataStringLen = 160
)

// ErrInvalidCursor is returned when an opaque pagination cursor cannot be
// decoded. Callers should map it to a 400 response.
var ErrInvalidCursor = errors.New("invalid activity cursor")

// RecordActivityInput is the payload for recording one activity event. Metadata
// must be small and free of secrets; it is sanitized before persistence.
type RecordActivityInput struct {
	TripID      uuid.UUID
	ActorUserID *uuid.UUID
	EventType   string
	EntityType  *string
	EntityID    *uuid.UUID
	Metadata    map[string]any
}

// ListActivityInput selects a page of a trip's activity, newest first. Cursor
// fields are nil for the first page.
type ListActivityInput struct {
	TripID          uuid.UUID
	Limit           int
	CursorCreatedAt *time.Time
	CursorID        *uuid.UUID
}

// ListActivityResult is one page of activity events plus an opaque cursor for
// the next (older) page. NextCursor is empty when no more rows exist.
type ListActivityResult struct {
	Events     []entity.TripActivityEvent
	NextCursor string
}

// cursor is the decoded form of the opaque pagination cursor.
type cursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        uuid.UUID `json:"id"`
}

// EncodeCursor builds the opaque base64url cursor pointing just past the given
// row. The timestamp round-trips through RFC3339Nano so the keyset comparison
// (created_at, id) stays exact.
func EncodeCursor(createdAt time.Time, id uuid.UUID) string {
	payload, err := json.Marshal(cursor{CreatedAt: createdAt.UTC(), ID: id})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

// DecodeCursor parses an opaque cursor produced by EncodeCursor. An empty string
// means "first page" and yields nil values without error.
func DecodeCursor(raw string) (*time.Time, *uuid.UUID, error) {
	if raw == "" {
		return nil, nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, nil, ErrInvalidCursor
	}
	var c cursor
	if err := json.Unmarshal(decoded, &c); err != nil {
		return nil, nil, ErrInvalidCursor
	}
	if c.CreatedAt.IsZero() || c.ID == uuid.Nil {
		return nil, nil, ErrInvalidCursor
	}
	createdAt := c.CreatedAt.UTC()
	id := c.ID
	return &createdAt, &id, nil
}

// NormalizeLimit clamps a requested limit to [1, MaxLimit], applying the default
// when unset (<= 0).
func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// sanitizeMetadata returns a defence-in-depth copy of metadata: nil values are
// dropped, string values are truncated, and the key count is capped. Call sites
// remain responsible for never passing secrets, tokens, passwords, comment
// bodies, or full itinerary payloads.
func sanitizeMetadata(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if len(out) >= maxMetadataKeys {
			break
		}
		if value == nil {
			continue
		}
		if s, ok := value.(string); ok {
			out[key] = truncate(s, maxMetadataStringLen)
			continue
		}
		out[key] = value
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
