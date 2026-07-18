// Package copilot implements the private, advisory Trip Copilot boundary.
// It only consumes sanitized summaries and can only suggest existing UI links.
package copilot

import "time"

type Config struct {
	Enabled              bool
	Mode                 string
	FailOpen             bool
	MaxMessageChars      int
	MaxContextChars      int
	Timeout              time.Duration
	StoreHistory         bool
	HistoryRetentionDays int
	PublicShareEnabled   bool
	RateLimitPerMinute   int
}

func NormalizeConfig(cfg Config) Config {
	if cfg.Mode != "ai" {
		cfg.Mode = "mock"
	}
	if cfg.MaxMessageChars <= 0 {
		cfg.MaxMessageChars = 2000
	}
	if cfg.MaxContextChars <= 0 {
		cfg.MaxContextChars = 12000
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.HistoryRetentionDays <= 0 {
		cfg.HistoryRetentionDays = 7
	}
	if cfg.RateLimitPerMinute <= 0 {
		cfg.RateLimitPerMinute = 20
	}
	return cfg
}

type Intent string

const (
	IntentNextAction            Intent = "next_action"
	IntentExplainHealth         Intent = "explain_health"
	IntentExplainVerification   Intent = "explain_verification"
	IntentExplainBudget         Intent = "explain_budget"
	IntentExplainRoute          Intent = "explain_route"
	IntentExplainGroupReadiness Intent = "explain_group_readiness"
	IntentExplainChecklist      Intent = "explain_checklist"
	IntentExplainExpenses       Intent = "explain_expenses"
	IntentExplainApproval       Intent = "explain_approval"
	IntentExplainRecap          Intent = "explain_recap"
	IntentExplainFeature        Intent = "explain_feature"
	IntentHowTo                 Intent = "how_to"
	IntentFindSection           Intent = "find_section"
	IntentUnsafeMutationRequest Intent = "unsafe_mutation_request"
	IntentOutOfScope            Intent = "out_of_scope"
	IntentGeneralTripQuestion   Intent = "general_trip_question"
)

type ClientContext struct {
	CurrentTab         string `json:"currentTab,omitempty"`
	CurrentPath        string `json:"currentPath,omitempty"`
	SelectedIssueID    string `json:"selectedIssueId,omitempty"`
	SelectedDayNumber  *int   `json:"selectedDayNumber,omitempty"`
	SelectedRouteLegID string `json:"selectedRouteLegId,omitempty"`
	Date               string `json:"date,omitempty"`
	DayNumber          *int   `json:"dayNumber,omitempty"`
	CurrentItemID      string `json:"currentItemId,omitempty"`
	NextItemID         string `json:"nextItemId,omitempty"`
}

type ChatRequest struct {
	ConversationID string        `json:"conversationId,omitempty"`
	Message        string        `json:"message"`
	ClientContext  ClientContext `json:"clientContext,omitempty"`
}

type ActionStyle string

const (
	ActionStylePrimary   ActionStyle = "primary"
	ActionStyleSecondary ActionStyle = "secondary"
)

type ActionRisk string

const (
	RiskSafeNavigation ActionRisk = "safe_navigation"
	RiskLowRiskPrepare ActionRisk = "low_risk_prepare"
	RiskMediumReview   ActionRisk = "medium_risk_review"
	RiskHighMutation   ActionRisk = "high_risk_mutation"
)

type Action struct {
	Type  string      `json:"type"`
	Label string      `json:"label"`
	Href  string      `json:"href"`
	Style ActionStyle `json:"style"`
}

type Source struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

type PermissionSummary struct {
	Role             string `json:"role"`
	CanEditItinerary bool   `json:"canEditItinerary"`
	CanEditRoute     bool   `json:"canEditRoute"`
	CanManageShare   bool   `json:"canManageShare"`
	CanUploadReceipt bool   `json:"canUploadReceipt"`
	CanComment       bool   `json:"canComment"`
	CanVote          bool   `json:"canVote"`
}

type ResponseMetadata struct {
	Mode            string   `json:"mode"`
	Intent          Intent   `json:"intent"`
	SafeContextUsed []string `json:"safeContextUsed"`
}

type ChatResponse struct {
	ConversationID  string           `json:"conversationId"`
	MessageID       string           `json:"messageId"`
	Answer          string           `json:"answer"`
	Actions         []Action         `json:"actions"`
	Sources         []Source         `json:"sources"`
	Warnings        []string         `json:"warnings"`
	PermissionNotes []string         `json:"permissionNotes"`
	Metadata        ResponseMetadata `json:"metadata"`
}

type AIResponse struct {
	Answer      string   `json:"answer"`
	Actions     []Action `json:"actions"`
	SourceTypes []string `json:"sourceTypes"`
	Warnings    []string `json:"warnings"`
}

type SafeContext struct {
	Trip            map[string]any `json:"trip"`
	CommandCenter   map[string]any `json:"commandCenter,omitempty"`
	Health          map[string]any `json:"health,omitempty"`
	Verification    map[string]any `json:"verification,omitempty"`
	Budget          map[string]any `json:"budget,omitempty"`
	Group           map[string]any `json:"groupReadiness,omitempty"`
	Route           map[string]any `json:"route,omitempty"`
	Itinerary       map[string]any `json:"itinerary,omitempty"`
	TravelDay       map[string]any `json:"travelDay,omitempty"`
	Checklist       map[string]any `json:"checklist,omitempty"`
	Reminders       map[string]any `json:"reminders,omitempty"`
	Expenses        map[string]any `json:"expenses,omitempty"`
	Approval        map[string]any `json:"approval,omitempty"`
	Policy          map[string]any `json:"policy,omitempty"`
	Generation      map[string]any `json:"generationQuality,omitempty"`
	Personalization map[string]any `json:"personalization,omitempty"`
	Recap           map[string]any `json:"recap,omitempty"`
	Unavailable     []string       `json:"unavailable,omitempty"`
}
