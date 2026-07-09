package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ItineraryVersionSource identifies the operation that produced a snapshot.
type ItineraryVersionSource string

const (
	ItineraryVersionSourceGenerated                 ItineraryVersionSource = "GENERATED"
	ItineraryVersionSourceManualEdit                ItineraryVersionSource = "MANUAL_EDIT"
	ItineraryVersionSourceRegenerateDay             ItineraryVersionSource = "REGENERATE_DAY"
	ItineraryVersionSourceRegenerateItem            ItineraryVersionSource = "REGENERATE_ITEM"
	ItineraryVersionSourceBudgetOptimizationApplied ItineraryVersionSource = "BUDGET_OPTIMIZATION_APPLIED"
	ItineraryVersionSourceAIPolicyRepairApplied     ItineraryVersionSource = "AI_POLICY_REPAIR"
	ItineraryVersionSourceCostSplitUpdated          ItineraryVersionSource = "COST_SPLIT_UPDATED"
	ItineraryVersionSourceRestored                  ItineraryVersionSource = "RESTORED"
	ItineraryVersionSourceCreatedFromTemplate       ItineraryVersionSource = "CREATED_FROM_TEMPLATE"
	ItineraryVersionSourceCreatedFromTemplateAI     ItineraryVersionSource = "CREATED_FROM_TEMPLATE_AI"
)

// ItineraryVersion is a full JSONB snapshot of a trip itinerary at one point in
// that trip's edit history. The itinerary is intentionally stored as raw JSON so
// snapshots can preserve the exact API shape accepted by itinerary editing v1.
type ItineraryVersion struct {
	ID              uuid.UUID
	TripID          uuid.UUID
	UserID          uuid.UUID
	CreatedByUserID *uuid.UUID
	VersionNumber   int
	Source          ItineraryVersionSource
	Itinerary       json.RawMessage
	Metadata        map[string]any
	CreatedAt       time.Time
}
