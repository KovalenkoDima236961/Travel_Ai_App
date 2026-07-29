// Package alpha owns the closed-alpha operating system: invites, waitlist,
// product analytics, feedback, dashboard aggregates, and weekly reports.
package alpha

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput       = errors.New("invalid alpha input")
	ErrNotFound           = errors.New("alpha record not found")
	ErrInviteUnavailable  = errors.New("invite code is not available")
	ErrAlreadyJoined      = errors.New("waitlist entry already exists")
	ErrAttachmentRejected = errors.New("feedback attachment rejected")
)

type Invite struct {
	ID                 uuid.UUID  `json:"id"`
	Code               string     `json:"code,omitempty"`
	CodeDisplay        string     `json:"codeDisplay"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	MaxActivations     int        `json:"maxActivations"`
	CurrentActivations int        `json:"currentActivations"`
	CreatorUserID      uuid.UUID  `json:"creatorUserId"`
	Notes              string     `json:"notes"`
	TesterGroup        string     `json:"testerGroup"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type CreateInviteInput struct {
	Code           string
	ExpiresAt      *time.Time
	MaxActivations int
	CreatorUserID  uuid.UUID
	Notes          string
	TesterGroup    string
	Enabled        bool
}

type UpdateInviteInput struct {
	ID             uuid.UUID
	ExpiresAt      **time.Time
	MaxActivations *int
	Notes          *string
	TesterGroup    *string
	Enabled        *bool
}

type WaitlistEntry struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	EmailDomain     string     `json:"emailDomain"`
	Status          string     `json:"status"`
	InvitedInviteID *uuid.UUID `json:"invitedInviteId,omitempty"`
	Source          string     `json:"source"`
	Notes           string     `json:"notes"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	InvitedAt       *time.Time `json:"invitedAt,omitempty"`
	AcceptedAt      *time.Time `json:"acceptedAt,omitempty"`
	DeclinedAt      *time.Time `json:"declinedAt,omitempty"`
	RemovedAt       *time.Time `json:"removedAt,omitempty"`
}

type JoinWaitlistInput struct {
	Email  string
	Source string
}

type UpdateWaitlistInput struct {
	ID     uuid.UUID
	Status *string
	Notes  *string
}

type InviteFromWaitlistInput struct {
	WaitlistID     uuid.UUID
	CreatorUserID  uuid.UUID
	TesterGroup    string
	Notes          string
	ExpiresAt      *time.Time
	MaxActivations int
}

type Participant struct {
	UserID              uuid.UUID  `json:"userId"`
	InviteID            *uuid.UUID `json:"inviteId,omitempty"`
	AlphaParticipant    bool       `json:"alphaParticipant"`
	TesterGroup         string     `json:"testerGroup"`
	InvitationDate      *time.Time `json:"invitationDate,omitempty"`
	FirstLoginAt        *time.Time `json:"firstLoginAt,omitempty"`
	FirstTripAt         *time.Time `json:"firstTripAt,omitempty"`
	FirstAIGenerationAt *time.Time `json:"firstAiGenerationAt,omitempty"`
	LastActivityAt      *time.Time `json:"lastActivityAt,omitempty"`
	Active              bool       `json:"active"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type ActivateInviteInput struct {
	UserID    uuid.UUID
	Code      string
	RequestID string
}

type EventInput struct {
	UserID        *uuid.UUID
	SessionID     string
	EventName     string
	Feature       string
	EntityType    string
	EntityID      string
	Metadata      json.RawMessage
	OccurredAt    *time.Time
	RequestID     string
	CorrelationID string
	AppVersion    string
	BrowserFamily string
	OSFamily      string
	DeviceType    string
	Source        string
}

type Event struct {
	ID            uuid.UUID       `json:"id"`
	UserID        *uuid.UUID      `json:"userId,omitempty"`
	EventName     string          `json:"eventName"`
	Feature       string          `json:"feature"`
	EntityType    string          `json:"entityType,omitempty"`
	Metadata      json.RawMessage `json:"metadata"`
	OccurredAt    time.Time       `json:"occurredAt"`
	RequestID     *string         `json:"requestId,omitempty"`
	CorrelationID *string         `json:"correlationId,omitempty"`
	AppVersion    *string         `json:"appVersion,omitempty"`
	BrowserFamily *string         `json:"browserFamily,omitempty"`
	OSFamily      *string         `json:"osFamily,omitempty"`
	DeviceType    *string         `json:"deviceType,omitempty"`
	Source        string          `json:"source"`
}

type FeedbackAttachmentInput struct {
	FileName      string
	MIMEType      string
	SizeBytes     int
	ContentSHA256 string
}

type SubmitFeedbackInput struct {
	UserID        uuid.UUID
	Category      string
	Title         string
	Description   string
	Metadata      json.RawMessage
	AppVersion    string
	BrowserFamily string
	OSFamily      string
	DeviceType    string
	RequestID     string
	CorrelationID string
	Provider      string
	ModelAlias    string
	PromptVersion string
	FeatureFlags  json.RawMessage
	Attachments   []FeedbackAttachmentInput
}

type Feedback struct {
	ID                   uuid.UUID       `json:"id"`
	UserID               uuid.UUID       `json:"userId"`
	Category             string          `json:"category"`
	Title                string          `json:"title"`
	DescriptionSanitized string          `json:"descriptionSanitized"`
	Status               string          `json:"status"`
	Priority             string          `json:"priority"`
	OwnerUserID          *uuid.UUID      `json:"ownerUserId,omitempty"`
	InternalNotes        string          `json:"internalNotes,omitempty"`
	Metadata             json.RawMessage `json:"metadata"`
	AppVersion           *string         `json:"appVersion,omitempty"`
	BrowserFamily        *string         `json:"browserFamily,omitempty"`
	OSFamily             *string         `json:"osFamily,omitempty"`
	DeviceType           *string         `json:"deviceType,omitempty"`
	RequestID            *string         `json:"requestId,omitempty"`
	CorrelationID        *string         `json:"correlationId,omitempty"`
	Provider             *string         `json:"provider,omitempty"`
	ModelAlias           *string         `json:"modelAlias,omitempty"`
	PromptVersion        *string         `json:"promptVersion,omitempty"`
	FeatureFlags         json.RawMessage `json:"featureFlags"`
	AttachmentCount      int             `json:"attachmentCount"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

type FeedbackAttachment struct {
	ID            uuid.UUID `json:"id"`
	FeedbackID    uuid.UUID `json:"feedbackId"`
	FileName      string    `json:"fileName"`
	MIMEType      string    `json:"mimeType"`
	SizeBytes     int       `json:"sizeBytes"`
	ContentSHA256 string    `json:"contentSha256"`
	ScanStatus    string    `json:"scanStatus"`
	StorageStatus string    `json:"storageStatus"`
	CreatedAt     time.Time `json:"createdAt"`
}

type FeedbackDetail struct {
	Feedback    Feedback             `json:"feedback"`
	Attachments []FeedbackAttachment `json:"attachments"`
}

type UpdateFeedbackInput struct {
	ID            uuid.UUID
	Status        *string
	Priority      *string
	OwnerUserID   **uuid.UUID
	InternalNotes *string
}

type Dashboard struct {
	GeneratedAt  time.Time       `json:"generatedAt"`
	Users        UserMetrics     `json:"users"`
	Trips        TripMetrics     `json:"trips"`
	AI           AIMetrics       `json:"ai"`
	Feedback     FeedbackMetrics `json:"feedback"`
	Usage        UsageMetrics    `json:"usage"`
	Costs        CostMetrics     `json:"costs"`
	Health       HealthMetrics   `json:"health"`
	Funnel       []FunnelStage   `json:"funnel"`
	FeatureUsage []FeatureMetric `json:"featureUsage"`
	Alerts       []AlphaAlert    `json:"alerts"`
}

type UserMetrics struct {
	Invited  int `json:"invited"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Retained int `json:"retained"`
}

type TripMetrics struct {
	Created   int `json:"created"`
	Completed int `json:"completed"`
}

type AIMetrics struct {
	Generations      int     `json:"generations"`
	SuccessRate      float64 `json:"successRate"`
	RepairRate       float64 `json:"repairRate"`
	FallbackRate     float64 `json:"fallbackRate"`
	AverageLatencyMS int     `json:"averageLatencyMs"`
	TokenUsage       int     `json:"tokenUsage"`
	Regenerated      int     `json:"regeneratedItineraries"`
	RemovedPlaces    int     `json:"removedPlaces"`
	ReplacedPlaces   int     `json:"replacedPlaces"`
	Accepted         int     `json:"acceptedItineraries"`
	BadPlaceReports  int     `json:"badPlaceReports"`
}

type FeedbackMetrics struct {
	Total           int            `json:"total"`
	BugReports      int            `json:"bugReports"`
	AIReports       int            `json:"aiReports"`
	FeatureRequests int            `json:"featureRequests"`
	ByCategory      map[string]int `json:"byCategory"`
	ByStatus        map[string]int `json:"byStatus"`
}

type UsageMetrics struct {
	DAU int `json:"dau"`
	WAU int `json:"wau"`
	MAU int `json:"mau"`
}

type CostMetrics struct {
	OpenAITokens        int     `json:"openaiTokens"`
	EstimatedOpenAICost float64 `json:"estimatedOpenAiCost"`
}

type HealthMetrics struct {
	Failures  int `json:"failures"`
	Retries   int `json:"retries"`
	Incidents int `json:"incidents"`
}

type FunnelStage struct {
	Name                string  `json:"name"`
	Users               int     `json:"users"`
	Conversion          float64 `json:"conversion"`
	DropoffFromPrevious int     `json:"dropoffFromPrevious"`
}

type FeatureMetric struct {
	Feature     string     `json:"feature"`
	UsageCount  int        `json:"usageCount"`
	UniqueUsers int        `json:"uniqueUsers"`
	FirstUse    *time.Time `json:"firstUse,omitempty"`
	RepeatUse   int        `json:"repeatUse"`
	Unused      bool       `json:"unused"`
}

type AlphaAlert struct {
	Type     string  `json:"type"`
	Severity string  `json:"severity"`
	Message  string  `json:"message"`
	Value    float64 `json:"value"`
}

type WeeklyReport struct {
	ID                uuid.UUID       `json:"id"`
	WeekStart         time.Time       `json:"weekStart"`
	WeekEnd           time.Time       `json:"weekEnd"`
	SummaryMarkdown   string          `json:"summaryMarkdown"`
	Metrics           json.RawMessage `json:"metrics"`
	GeneratedByUserID *uuid.UUID      `json:"generatedByUserId,omitempty"`
	GeneratedAt       time.Time       `json:"generatedAt"`
}
