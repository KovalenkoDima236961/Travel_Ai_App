package notifications

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

const (
	// DefaultLimit is the page size used when the caller does not specify one.
	DefaultLimit = 30
	// MaxLimit caps the page size a caller may request.
	MaxLimit = 100

	// MaxBatchSize caps how many notifications a single internal batch request
	// may create.
	MaxBatchSize = 100

	// MaxTitleLength and MaxMessageLength bound the user-visible text. They match
	// the database CHECK constraints so an over-long value is rejected before it
	// reaches Postgres.
	MaxTitleLength       = 200
	MaxMessageLength     = 1000
	MaxGroupingKeyLength = 500

	// maxMetadataKeys caps how many keys are persisted per notification so a
	// stray large map can never bloat a row.
	maxMetadataKeys = 24
	// maxMetadataStringLen truncates long string values (defence in depth; the
	// call sites already keep metadata small and free of sensitive data).
	maxMetadataStringLen = 200
)

// ErrInvalidCursor is returned when an opaque pagination cursor cannot be
// decoded. Callers should map it to a 400 response.
var ErrInvalidCursor = errors.New("invalid notification cursor")

// CreateInput is the payload for creating one notification. It is validated and
// sanitized before persistence. Metadata must be small and free of secrets.
type CreateInput struct {
	UserID      uuid.UUID
	TripID      *uuid.UUID
	ActorUserID *uuid.UUID
	Type        string
	Title       string
	Message     string
	EntityType  *string
	EntityID    *uuid.UUID
	Metadata    map[string]any
	Priority    string
	Category    string
	DigestKey   *string
	DedupeKey   *string
}

// InAppPreferenceGate reports whether an in-app row may be created for a
// recipient/type pair. It is implemented by the preferences EffectiveSet without
// importing that package here.
type InAppPreferenceGate interface {
	AllowInApp(userID uuid.UUID, notificationType string) bool
}

type InAppNotificationGate interface {
	AllowInAppNotification(notification entity.Notification) bool
}

// InAppDeliveryResolver annotates persisted in-app rows with the deterministic
// mode/status chosen by the richer delivery policy. Simpler legacy gates do not
// need to implement it.
type InAppDeliveryResolver interface {
	InAppDelivery(notification entity.Notification) (mode, status string)
}

// BatchCreateResult reports how an internal batch was handled. Created contains
// only persisted in-app rows. EmailCandidates contains all non-self,
// successfully validated notification objects so email can remain independent
// from in-app preferences.
type BatchCreateResult struct {
	Requested           int
	Created             []entity.Notification
	GroupedInApp        []entity.Notification
	EmailCandidates     []entity.Notification
	Skipped             int
	SkippedByPreference int
	Grouped             int
	DuplicatesDropped   int
}

// ListInput selects a page of a user's notifications, newest first. Cursor
// fields are nil for the first page.
type ListInput struct {
	UserID          uuid.UUID
	Limit           int
	CursorCreatedAt *time.Time
	CursorID        *uuid.UUID
}

// ListResult is one page of notifications plus an opaque cursor for the next
// (older) page. NextCursor is empty when no more rows exist.
type ListResult struct {
	Notifications []entity.Notification
	NextCursor    string
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
		if value == nil || unsafeMetadataKey(key) {
			continue
		}
		if sanitized, ok := sanitizeMetadataValue(value, 0); ok {
			out[key] = sanitized
		}
	}
	return out
}

func sanitizeMetadataValue(value any, depth int) (any, bool) {
	if value == nil || depth > 2 {
		return nil, false
	}
	switch typed := value.(type) {
	case string:
		return truncate(typed, maxMetadataStringLen), true
	case bool, float64, float32, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return typed, true
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if len(out) >= maxMetadataKeys || unsafeMetadataKey(key) {
				continue
			}
			if sanitized, ok := sanitizeMetadataValue(nested, depth+1); ok {
				out[key] = sanitized
			}
		}
		return out, true
	case []any:
		limit := len(typed)
		if limit > maxMetadataKeys {
			limit = maxMetadataKeys
		}
		out := make([]any, 0, limit)
		for _, nested := range typed[:limit] {
			if sanitized, ok := sanitizeMetadataValue(nested, depth+1); ok {
				out = append(out, sanitized)
			}
		}
		return out, true
	default:
		raw, err := json.Marshal(value)
		if err != nil || len(raw) > maxMetadataStringLen*maxMetadataKeys {
			return nil, false
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, false
		}
		return sanitizeMetadataValue(decoded, depth+1)
	}
}

func unsafeMetadataKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, forbidden := range []string{
		"password", "token", "secret", "authorization", "credential",
		"ocrtext", "receiptocr", "aiprompt", "prompttext", "commentbody",
		"privatenote", "expensenote", "calendareventdetails", "p256dh",
	} {
		if strings.Contains(normalized, forbidden) {
			return true
		}
	}
	return normalized == "auth"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
