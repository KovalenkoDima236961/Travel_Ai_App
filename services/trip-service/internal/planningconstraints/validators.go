package planningconstraints

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
)

func ValidatePreviewRequest(req PreviewRequest) error {
	if !req.Source.Valid() {
		return fmt.Errorf("source is invalid")
	}
	if req.TripID != nil && *req.TripID == uuid.Nil {
		return fmt.Errorf("tripId is invalid")
	}
	if req.WorkspaceID != nil && *req.WorkspaceID == uuid.Nil {
		return fmt.Errorf("workspaceId is invalid")
	}
	return nil
}

func IncludePreviousSignals(source Source, explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	switch source {
	case SourceTripDiscovery, SourceTemplateAdaptation, SourceTripGeneration:
		return true
	default:
		return false
	}
}

func IncludeWorkspacePolicy(explicit *bool) bool {
	return explicit == nil || *explicit
}

func IncludeRoute(explicit *bool) bool {
	return explicit == nil || *explicit
}

func IncludeTripState(explicit *bool) bool {
	return explicit == nil || *explicit
}

func AllowsBlockingOverride(source Source) bool {
	return source == SourcePolicyRepair || source == SourceRouteUpdatePreview
}

func cloneUUIDPtr(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return ""
	}
	return value
}

func currencyOrDefault(value string, profile *usercontext.UserProfile) string {
	if currency := normalizeCurrency(value); currency != "" {
		return currency
	}
	if profile != nil {
		if currency := normalizeCurrency(profile.PreferredCurrency); currency != "" {
			return currency
		}
	}
	return defaultCurrency
}

func strictnessOrDefault(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "loose", "target", "strict":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "target"
	}
}

func dateFlexibility(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fixed", "flexible", "weekend", "month", "unknown":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "fixed"
	}
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func cleanModes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		mode := aggregate.NormalizeRouteToken(value)
		if _, ok := aggregate.SupportedTransportModes[mode]; !ok {
			continue
		}
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}

func cleanTokens(values []string, allowed map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		token := normalizeToken(value)
		if _, ok := allowed[token]; !ok {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func allAllowedModes(disallowed []string) []string {
	values := make([]string, 0, len(aggregate.SupportedTransportModes))
	for mode := range aggregate.SupportedTransportModes {
		if mode == aggregate.TransportModeOther || contains(disallowed, mode) {
			continue
		}
		values = append(values, mode)
	}
	return values
}

func appendUnique(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" || contains(values, next) {
		return values
	}
	return append(values, next)
}

func contains(values []string, needle string) bool {
	needle = normalizeToken(needle)
	for _, value := range values {
		if normalizeToken(value) == needle {
			return true
		}
	}
	return false
}

func formatAmount(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func languageName(code string) string {
	switch code {
	case "es":
		return "Spanish"
	case "uk":
		return "Ukrainian"
	case "fr":
		return "French"
	default:
		return "English"
	}
}

func workspaceRuleCount(policy *WorkspacePolicy) int {
	if policy == nil {
		return 0
	}
	return len(policy.BlockingRules) + len(policy.WarningRules)
}

func jsonUnmarshal(raw json.RawMessage, out any) error {
	return json.Unmarshal(raw, out)
}
