package groupreadiness

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type Level string

const (
	LevelReady          Level = "ready"
	LevelAlmostReady    Level = "almost_ready"
	LevelNeedsAttention Level = "needs_attention"
	LevelNotReady       Level = "not_ready"
)

type Category string

const (
	CategoryAvailability Category = "availability"
	CategoryCalendar     Category = "calendar"
	CategoryPolls        Category = "polls"
	CategoryChecklist    Category = "checklist"
	CategoryReminders    Category = "reminders"
	CategoryExpenses     Category = "expenses"
	CategorySettlements  Category = "settlements"
	CategoryComments     Category = "comments"
	CategoryActivity     Category = "activity"
	CategoryApproval     Category = "approval"
	CategoryOfflineSync  Category = "offline_sync"
	CategoryProfile      Category = "profile"
	CategoryOther        Category = "other"
)

type ItemStatus string

const (
	ItemStatusComplete      ItemStatus = "complete"
	ItemStatusIncomplete    ItemStatus = "incomplete"
	ItemStatusMissing       ItemStatus = "missing"
	ItemStatusOverdue       ItemStatus = "overdue"
	ItemStatusPending       ItemStatus = "pending"
	ItemStatusNotApplicable ItemStatus = "not_applicable"
	ItemStatusUnknown       ItemStatus = "unknown"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Item struct {
	ID          string     `json:"id"`
	Category    Category   `json:"category"`
	Status      ItemStatus `json:"status"`
	Severity    Severity   `json:"severity"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Action      *Action    `json:"action,omitempty"`
}

type CompletedItem struct {
	Category Category `json:"category"`
	Title    string   `json:"title"`
}

type CollaboratorReadiness struct {
	UserID         uuid.UUID       `json:"userId"`
	DisplayName    string          `json:"displayName"`
	Role           string          `json:"role"`
	Status         string          `json:"status"`
	Score          int             `json:"score"`
	Level          Level           `json:"level"`
	IsCurrentUser  bool            `json:"isCurrentUser"`
	Items          []Item          `json:"items"`
	CompletedItems []CompletedItem `json:"completedItems"`
	NextAction     *Action         `json:"nextAction,omitempty"`
}

type CategorySummary struct {
	Category        Category `json:"category"`
	ReadyCount      int      `json:"readyCount"`
	TotalCount      int      `json:"totalCount"`
	OpenIssueCount  int      `json:"openIssueCount"`
	HighestSeverity Severity `json:"highestSeverity"`
}

type TopAction struct {
	ID           string     `json:"id"`
	Label        string     `json:"label"`
	Description  string     `json:"description"`
	Href         string     `json:"href"`
	ActionType   string     `json:"actionType"`
	TargetUserID *uuid.UUID `json:"targetUserId,omitempty"`
}

type Response struct {
	TripID          uuid.UUID               `json:"tripId"`
	Score           int                     `json:"score"`
	Level           Level                   `json:"level"`
	Summary         string                  `json:"summary"`
	GeneratedAt     time.Time               `json:"generatedAt"`
	Members         []CollaboratorReadiness `json:"members"`
	CategorySummary []CategorySummary       `json:"categorySummary"`
	TopActions      []TopAction             `json:"topActions"`
	Debug           map[string]any          `json:"debug,omitempty"`
}

type Member struct {
	UserID        uuid.UUID
	DisplayName   string
	Role          string
	IsCurrentUser bool
}

type PollSnapshot struct {
	Poll  entity.TripPoll
	Votes []entity.TripPollVote
}

type Snapshot struct {
	Trip                  *entity.Trip
	Members               []Member
	AvailabilityResponses []entity.TripAvailabilityResponse
	Polls                 []PollSnapshot
	Checklist             *entity.TripChecklist
	Reminders             []entity.TripReminder
	Settlements           []entity.TripSettlement
	Approval              *entity.TripApprovalFields
	SubsystemFailures     []string
	Now                   time.Time
}

type Options struct {
	IncludeDetails bool
	IncludeDebug   bool
	CanNudge       bool
}

type NudgeRequest struct {
	TargetUserIDs     []uuid.UUID `json:"targetUserIds"`
	Categories        []Category  `json:"categories"`
	Message           string      `json:"message"`
	DedupeWindowHours int         `json:"dedupeWindowHours"`
}

type NudgeResponse struct {
	SentCount         int         `json:"sentCount"`
	SkippedCount      int         `json:"skippedCount"`
	DedupedCount      int         `json:"dedupedCount"`
	TargetUserIDs     []uuid.UUID `json:"targetUserIds"`
	Categories        []Category  `json:"categories"`
	DedupeWindowHours int         `json:"dedupeWindowHours"`
}
