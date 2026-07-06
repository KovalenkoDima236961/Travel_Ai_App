package approvals

// ItemStatus is the per-check outcome shown in the checklist UI.
type ItemStatus string

const (
	ItemStatusOK      ItemStatus = "ok"
	ItemStatusWarning ItemStatus = "warning"
	ItemStatusBlocked ItemStatus = "blocked"
	ItemStatusInfo    ItemStatus = "info"
)

// Severity describes how strongly a failing check affects submission. Only a
// failing blocker prevents submission; warnings can be acknowledged.
type Severity string

const (
	SeverityBlocker Severity = "blocker"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ChecklistStatus is the rolled-up status of the whole checklist.
type ChecklistStatus string

const (
	ChecklistStatusOK      ChecklistStatus = "ok"
	ChecklistStatusWarning ChecklistStatus = "warning"
	ChecklistStatusBlocked ChecklistStatus = "blocked"
)

// Checklist item keys. These are stable identifiers reused by the API, the UI,
// and acknowledged-warning payloads, so they must not change casually.
const (
	KeyItineraryExists         = "itinerary_exists"
	KeyBudgetExists            = "budget_exists"
	KeyWorkspaceBudgetStatus   = "workspace_budget_status"
	KeyTripBudgetStatus        = "trip_budget_status"
	KeyCostSplittingConfigured = "cost_splitting_configured"
	KeyAvailabilityChecked     = "availability_checked"
	KeyMissingCostEstimates    = "missing_cost_estimates"
)

// ChecklistItem is one evaluated check.
type ChecklistItem struct {
	Key      string     `json:"key"`
	Status   ItemStatus `json:"status"`
	Severity Severity   `json:"severity"`
	Title    string     `json:"title"`
	Message  string     `json:"message"`
}

// Checklist is the full evaluated submission readiness report.
type Checklist struct {
	Status        ChecklistStatus `json:"status"`
	Items         []ChecklistItem `json:"items"`
	WarningCount  int             `json:"warningCount"`
	CriticalCount int             `json:"criticalCount"`
	BlockerCount  int             `json:"blockerCount"`
}

// CanSubmit reports whether the checklist permits submission. Warnings never
// block; only a failed blocker (a missing itinerary in v1) does.
func (c Checklist) CanSubmit() bool { return c.BlockerCount == 0 }

// ChecklistInput carries the already-gathered signals the calculator needs. The
// service populates it from the trip itinerary, cost-splitting summary, trip and
// workspace budgets, and enrichment metadata; the calculator itself does no I/O.
type ChecklistInput struct {
	// Itinerary.
	ItineraryDayCount  int
	ItineraryItemCount int

	// Trip budget.
	HasTripBudget    bool
	TripBudgetAmount float64
	EstimatedTotal   float64

	// Workspace shared budget.
	HasWorkspaceBudget       bool
	WorkspaceBudgetExceeded  bool
	WorkspaceBudgetNearLimit bool

	// Cost splitting.
	TravelerCount        int
	UnassignedCostCount  int
	InvalidSplitCount    int
	MissingEstimateCount int
	DefaultSplitCount    int

	// Availability / bookable items.
	BookableItemCount          int
	AvailabilityUncheckedCount int
}

// Calculate evaluates every check and rolls up the overall status and counts.
// It is deterministic and side-effect free so it can be snapshotted into the
// approval history and unit-tested exhaustively.
func Calculate(in ChecklistInput) Checklist {
	items := []ChecklistItem{
		itineraryItem(in),
		budgetExistsItem(in),
		workspaceBudgetItem(in),
		tripBudgetItem(in),
		costSplittingItem(in),
		availabilityItem(in),
		missingEstimatesItem(in),
	}

	checklist := Checklist{Items: items}
	for _, item := range items {
		switch item.Status {
		case ItemStatusBlocked:
			checklist.BlockerCount++
			if item.Severity == SeverityBlocker {
				checklist.CriticalCount++
			}
		case ItemStatusWarning:
			checklist.WarningCount++
		}
	}

	switch {
	case checklist.BlockerCount > 0:
		checklist.Status = ChecklistStatusBlocked
	case checklist.WarningCount > 0:
		checklist.Status = ChecklistStatusWarning
	default:
		checklist.Status = ChecklistStatusOK
	}
	return checklist
}

func itineraryItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyItineraryExists,
		Severity: SeverityBlocker,
		Title:    "Itinerary exists",
	}
	if in.ItineraryDayCount >= 1 && in.ItineraryItemCount >= 1 {
		item.Status = ItemStatusOK
		item.Message = "Trip has a generated itinerary."
	} else {
		item.Status = ItemStatusBlocked
		item.Message = "Add at least one itinerary day with one activity before submitting for approval."
	}
	return item
}

func budgetExistsItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyBudgetExists,
		Severity: SeverityWarning,
		Title:    "Trip budget set",
	}
	if in.HasTripBudget {
		item.Status = ItemStatusOK
		item.Message = "Trip has a budget."
	} else {
		item.Status = ItemStatusWarning
		item.Message = "No trip budget is set. Reviewers will not see a spending target."
	}
	return item
}

func workspaceBudgetItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyWorkspaceBudgetStatus,
		Severity: SeverityWarning,
		Title:    "Workspace budget",
	}
	switch {
	case !in.HasWorkspaceBudget:
		item.Status = ItemStatusWarning
		item.Message = "No workspace budget is configured for this workspace."
	case in.WorkspaceBudgetExceeded:
		item.Status = ItemStatusWarning
		item.Message = "The workspace budget is over its limit."
	case in.WorkspaceBudgetNearLimit:
		item.Status = ItemStatusWarning
		item.Message = "The workspace budget is close to its limit."
	default:
		item.Status = ItemStatusOK
		item.Message = "The workspace budget is within limits."
	}
	return item
}

func tripBudgetItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyTripBudgetStatus,
		Severity: SeverityWarning,
		Title:    "Estimated cost vs trip budget",
	}
	switch {
	case !in.HasTripBudget:
		item.Status = ItemStatusInfo
		item.Message = "No trip budget to compare the estimated cost against."
	case in.EstimatedTotal > in.TripBudgetAmount:
		item.Status = ItemStatusWarning
		item.Message = "The estimated cost is over the trip budget."
	default:
		item.Status = ItemStatusOK
		item.Message = "The estimated cost is within the trip budget."
	}
	return item
}

func costSplittingItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyCostSplittingConfigured,
		Severity: SeverityWarning,
		Title:    "Cost splitting configured",
	}
	switch {
	case in.TravelerCount == 0:
		item.Status = ItemStatusWarning
		item.Message = "No travelers are added, so costs cannot be split."
	case in.InvalidSplitCount > 0:
		item.Status = ItemStatusWarning
		item.Message = "Some costs have invalid split rules."
	case in.UnassignedCostCount > 0:
		item.Status = ItemStatusWarning
		item.Message = "Some costs are not assigned to any traveler."
	default:
		item.Status = ItemStatusOK
		item.Message = "Costs are split across travelers."
	}
	return item
}

func availabilityItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyAvailabilityChecked,
		Severity: SeverityWarning,
		Title:    "Availability checked",
	}
	switch {
	case in.BookableItemCount == 0:
		item.Status = ItemStatusOK
		item.Message = "No bookable items require an availability check."
	case in.AvailabilityUncheckedCount > 0:
		item.Status = ItemStatusWarning
		item.Message = "Some bookable items have not had their availability or price checked."
	default:
		item.Status = ItemStatusOK
		item.Message = "Bookable items have availability information."
	}
	return item
}

func missingEstimatesItem(in ChecklistInput) ChecklistItem {
	item := ChecklistItem{
		Key:      KeyMissingCostEstimates,
		Severity: SeverityWarning,
		Title:    "Cost estimates complete",
	}
	if in.MissingEstimateCount == 0 {
		item.Status = ItemStatusOK
		item.Message = "All items have a cost estimate."
	} else {
		item.Status = ItemStatusWarning
		item.Message = "Some items are missing a cost estimate."
	}
	return item
}
