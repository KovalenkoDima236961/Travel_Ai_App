package copilot

import (
	"regexp"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiprivacy"
)

var (
	unsafeClaimPattern = regexp.MustCompile(`(?i)\b(i|we|copilot)\s+(have\s+)?(booked|paid|sent|deleted|updated|changed|applied|restored|removed)\b`)
	pathPattern        = regexp.MustCompile(`(?i)(?:^|\s)(?:/Users/|/tmp/|C:\\Users\\|file://)`)
)

func validateAIResponse(response AIResponse, available []Action) (AIResponse, error) {
	answer := strings.TrimSpace(response.Answer)
	if answer == "" || len(answer) > 2400 || unsafeClaimPattern.MatchString(answer) || pathPattern.MatchString(answer) {
		return AIResponse{}, ErrResponseInvalid
	}
	redacted, redactionCount := aiprivacy.RedactText(answer)
	if redactionCount > 0 || redacted != answer {
		return AIResponse{}, ErrResponseInvalid
	}

	actions := make([]Action, 0, 2)
	seenActions := map[string]struct{}{}
	for _, requested := range response.Actions {
		if _, seen := seenActions[requested.Type]; seen || actionRisk(requested.Type) == RiskHighMutation {
			continue
		}
		trusted, ok := actionByType(available, requested.Type)
		if !ok {
			continue
		}
		if len(actions) == 0 {
			trusted.Style = ActionStylePrimary
		}
		actions = append(actions, trusted)
		seenActions[requested.Type] = struct{}{}
		if len(actions) == 2 {
			break
		}
	}

	sources := make([]string, 0, 4)
	seenSources := map[string]struct{}{}
	for _, sourceType := range response.SourceTypes {
		if _, ok := sourceDefinition(sourceType); !ok {
			continue
		}
		if _, seen := seenSources[sourceType]; seen {
			continue
		}
		sources = append(sources, sourceType)
		seenSources[sourceType] = struct{}{}
		if len(sources) == 4 {
			break
		}
	}
	response.Answer = answer
	response.Actions = actions
	response.SourceTypes = sources
	response.Warnings = sanitizeWarnings(response.Warnings)
	return response, nil
}

func sanitizeWarnings(values []string) []string {
	out := make([]string, 0, min(3, len(values)))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 240 || pathPattern.MatchString(value) {
			continue
		}
		redacted, count := aiprivacy.RedactText(value)
		if count > 0 || redacted != value {
			continue
		}
		out = append(out, value)
		if len(out) == 3 {
			break
		}
	}
	return out
}

type sourceInfo struct {
	label string
	tab   string
}

var sourceDefinitions = map[string]sourceInfo{
	"command_center":       {"Command Center", "overview"},
	"trip_health":          {"Trip Health", "health"},
	"budget_confidence":    {"Budget Confidence", "budget"},
	"group_readiness":      {"Group Readiness", "group-readiness"},
	"route_summary":        {"Route & Transport", "route"},
	"itinerary_summary":    {"Itinerary", "itinerary"},
	"checklist_summary":    {"Checklist", "checklist"},
	"reminders_summary":    {"Reminders", "reminders"},
	"expenses_summary":     {"Expenses", "expenses"},
	"approval_status":      {"Approval", "approval"},
	"policy_evaluation":    {"Policy", "policy"},
	"generation_quality":   {"Generation Quality", "itinerary"},
	"personalization":      {"Trip settings", "settings"},
	"notification_summary": {"Notifications", "settings"},
	"app_help":             {"App help", "overview"},
	"unknown":              {"Trip summary", "overview"},
}

func sourceDefinition(sourceType string) (sourceInfo, bool) {
	value, ok := sourceDefinitions[sourceType]
	return value, ok
}
