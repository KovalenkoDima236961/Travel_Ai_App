package alpha

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxMetadataKeys      = 40
	maxMetadataDepth     = 4
	maxMetadataArrayLen  = 20
	maxMetadataString    = 160
	maxTitleLength       = 160
	maxDescriptionLength = 4000
)

var (
	allowedEvents = map[string]string{
		"signup_completed":             "authentication",
		"login":                        "authentication",
		"logout":                       "authentication",
		"profile_completed":            "profile",
		"preferences_updated":          "profile",
		"trip_created":                 "trips",
		"itinerary_generated":          "trips",
		"itinerary_regenerated":        "trips",
		"itinerary_edited":             "trips",
		"itinerary_archived":           "trips",
		"ai_generation_started":        "ai",
		"ai_generation_completed":      "ai",
		"ai_generation_failed":         "ai",
		"repair_triggered":             "ai",
		"fallback_used":                "ai",
		"itinerary_accepted":           "ai",
		"place_removed":                "ai",
		"place_replaced":               "ai",
		"budget_created":               "budget",
		"budget_edited":                "budget",
		"route_recalculated":           "routes",
		"share_created":                "sharing",
		"share_opened":                 "sharing",
		"notification_opened":          "notifications",
		"feedback_submitted":           "feedback",
		"ai_feedback_submitted":        "feedback",
		"bug_report_submitted":         "feedback",
		"feature_request_submitted":    "feedback",
		"experimental_setting_changed": "settings",
		"trip_reviewed":                "trips",
		"user_returned":                "retention",
		"second_trip_created":          "retention",
		"error_occurred":               "health",
	}
	allowedFeatures = map[string]struct{}{
		"authentication": {}, "profile": {}, "trips": {}, "ai": {}, "budget": {},
		"routes": {}, "sharing": {}, "notifications": {}, "feedback": {},
		"settings": {}, "retention": {}, "health": {}, "onboarding": {},
	}
	feedbackCategories = map[string]struct{}{
		"ai": {}, "ui": {}, "performance": {}, "bug": {}, "security": {},
		"accessibility": {}, "feature_request": {}, "other": {},
	}
	feedbackStatuses = map[string]struct{}{
		"open": {}, "triaged": {}, "in_progress": {}, "resolved": {}, "closed": {}, "duplicate": {},
	}
	feedbackPriorities = map[string]struct{}{
		"low": {}, "normal": {}, "high": {}, "urgent": {},
	}
	testerGroups = map[string]struct{}{
		"internal": {}, "external": {}, "qa": {}, "design_reviewer": {},
	}
	waitlistStatuses = map[string]struct{}{
		"registered": {}, "invited": {}, "accepted": {}, "declined": {}, "removed": {},
	}
	sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|token|secret|authorization|cookie|email|prompt|receipt|ocr|note|private|query|search|address|latitude|longitude|exact|payment|card|cvv|ssn|passport|phone)`)
	unsafeTextPattern   = regexp.MustCompile(`(?i)(bearer\s+[a-z0-9._~+\-/]+=*|sk-[a-z0-9_-]{12,}|password\s*[:=]|token\s*[:=])`)
)

func normalizeTesterGroup(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "design" || normalized == "design_reviewers" {
		normalized = "design_reviewer"
	}
	if normalized == "" {
		normalized = "external"
	}
	if _, ok := testerGroups[normalized]; !ok {
		return "", invalidInput("testerGroup must be internal, external, qa, or design_reviewer")
	}
	return normalized, nil
}

func normalizeWaitlistStatus(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := waitlistStatuses[normalized]; !ok {
		return "", invalidInput("status must be registered, invited, accepted, declined, or removed")
	}
	return normalized, nil
}

func normalizeFeedbackCategory(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "feature" || normalized == "feature_request_submitted" {
		normalized = "feature_request"
	}
	if normalized == "" {
		normalized = "other"
	}
	if _, ok := feedbackCategories[normalized]; !ok {
		return "", invalidInput("category is not supported")
	}
	return normalized, nil
}

func normalizeFeedbackStatus(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := feedbackStatuses[normalized]; !ok {
		return "", invalidInput("status is not supported")
	}
	return normalized, nil
}

func normalizeFeedbackPriority(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "normal"
	}
	if _, ok := feedbackPriorities[normalized]; !ok {
		return "", invalidInput("priority is not supported")
	}
	return normalized, nil
}

func normalizeEventName(value string) (string, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	feature, ok := allowedEvents[normalized]
	if !ok {
		return "", "", invalidInput("eventName is not supported")
	}
	return normalized, feature, nil
}

func normalizeFeature(value, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if _, ok := allowedFeatures[normalized]; ok {
		return normalized
	}
	return fallback
}

func normalizeEmail(value string) (string, string, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return "", "", invalidInput("email must be valid")
	}
	email := strings.ToLower(strings.TrimSpace(parsed.Address))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", "", invalidInput("email must be valid")
	}
	return email, email[at+1:], nil
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func codeHash(code string) string {
	return hashString(strings.ToUpper(strings.TrimSpace(code)))
}

func codeDisplay(code string) (string, string, string) {
	compact := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(code)), "-", "")
	prefix := compact
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}
	suffix := compact
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	display := prefix + "..." + suffix
	return prefix, suffix, display
}

func sanitizeText(value string, maxLen int) string {
	cleaned := strings.TrimSpace(value)
	cleaned = unsafeTextPattern.ReplaceAllString(cleaned, "[redacted]")
	if len(cleaned) > maxLen {
		cleaned = cleaned[:maxLen]
	}
	return cleaned
}

func sanitizeMetadata(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, invalidInput("metadata must be valid JSON")
	}
	sanitized := sanitizeValue(decoded, 0)
	if sanitized == nil {
		sanitized = map[string]any{}
	}
	out, err := json.Marshal(sanitized)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func sanitizeValue(value any, depth int) any {
	if depth > maxMetadataDepth {
		return nil
	}
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, min(len(keys), maxMetadataKeys))
		for _, key := range keys {
			if len(out) >= maxMetadataKeys {
				break
			}
			cleanKey := sanitizeText(key, 80)
			if cleanKey == "" || sensitiveKeyPattern.MatchString(cleanKey) {
				continue
			}
			if sanitized := sanitizeValue(v[key], depth+1); sanitized != nil {
				out[cleanKey] = sanitized
			}
		}
		return out
	case []any:
		limit := min(len(v), maxMetadataArrayLen)
		out := make([]any, 0, limit)
		for i := 0; i < limit; i++ {
			if sanitized := sanitizeValue(v[i], depth+1); sanitized != nil {
				out = append(out, sanitized)
			}
		}
		return out
	case string:
		return sanitizeText(v, maxMetadataString)
	case float64, bool, nil:
		return v
	default:
		return nil
	}
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return sanitizeText(trimmed, 200)
}

func boundedTime(value *time.Time) time.Time {
	if value == nil || value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
