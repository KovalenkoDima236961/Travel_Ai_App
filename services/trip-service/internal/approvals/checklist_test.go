package approvals

import "testing"

func baseReadyInput() ChecklistInput {
	// A trip that passes every check: itinerary present, budgets fine, travelers
	// with valid splits, no bookable gaps, no missing estimates.
	return ChecklistInput{
		ItineraryDayCount:  2,
		ItineraryItemCount: 5,
		HasTripBudget:      true,
		TripBudgetAmount:   1000,
		EstimatedTotal:     800,
		HasWorkspaceBudget: true,
		TravelerCount:      2,
	}
}

func findItem(c Checklist, key string) (ChecklistItem, bool) {
	for _, item := range c.Items {
		if item.Key == key {
			return item, true
		}
	}
	return ChecklistItem{}, false
}

func TestCalculate_AllOK(t *testing.T) {
	c := Calculate(baseReadyInput())
	if c.Status != ChecklistStatusOK {
		t.Fatalf("expected ok status, got %q", c.Status)
	}
	if c.BlockerCount != 0 || c.WarningCount != 0 || c.CriticalCount != 0 {
		t.Fatalf("expected zero counts, got blocker=%d warning=%d critical=%d", c.BlockerCount, c.WarningCount, c.CriticalCount)
	}
	if !c.CanSubmit() {
		t.Fatal("expected CanSubmit true when nothing is blocked")
	}
}

func TestCalculate_NoItineraryBlocks(t *testing.T) {
	in := baseReadyInput()
	in.ItineraryDayCount = 0
	in.ItineraryItemCount = 0
	c := Calculate(in)
	if c.Status != ChecklistStatusBlocked {
		t.Fatalf("expected blocked status, got %q", c.Status)
	}
	if c.BlockerCount != 1 || c.CriticalCount != 1 {
		t.Fatalf("expected one blocker/critical, got blocker=%d critical=%d", c.BlockerCount, c.CriticalCount)
	}
	if c.CanSubmit() {
		t.Fatal("expected CanSubmit false with a missing itinerary")
	}
	item, ok := findItem(c, KeyItineraryExists)
	if !ok || item.Status != ItemStatusBlocked || item.Severity != SeverityBlocker {
		t.Fatalf("itinerary_exists item wrong: %+v ok=%v", item, ok)
	}
}

func TestCalculate_ItineraryWithDaysButNoItemsBlocks(t *testing.T) {
	in := baseReadyInput()
	in.ItineraryDayCount = 3
	in.ItineraryItemCount = 0
	c := Calculate(in)
	if c.CanSubmit() {
		t.Fatal("a day with no items should still block submission")
	}
}

func TestCalculate_MissingBudgetWarns(t *testing.T) {
	in := baseReadyInput()
	in.HasTripBudget = false
	c := Calculate(in)
	item, _ := findItem(c, KeyBudgetExists)
	if item.Status != ItemStatusWarning {
		t.Fatalf("expected budget_exists warning, got %q", item.Status)
	}
	if c.Status != ChecklistStatusWarning {
		t.Fatalf("expected overall warning, got %q", c.Status)
	}
	if !c.CanSubmit() {
		t.Fatal("warnings must not block submission")
	}
}

func TestCalculate_TripBudgetInfoWhenNoBudget(t *testing.T) {
	in := baseReadyInput()
	in.HasTripBudget = false
	c := Calculate(in)
	item, _ := findItem(c, KeyTripBudgetStatus)
	if item.Status != ItemStatusInfo {
		t.Fatalf("expected trip_budget_status info when no budget, got %q", item.Status)
	}
}

func TestCalculate_OverBudgetWarns(t *testing.T) {
	in := baseReadyInput()
	in.EstimatedTotal = 1500 // over the 1000 budget
	c := Calculate(in)
	item, _ := findItem(c, KeyTripBudgetStatus)
	if item.Status != ItemStatusWarning {
		t.Fatalf("expected trip_budget_status warning when over budget, got %q", item.Status)
	}
	if !c.CanSubmit() {
		t.Fatal("over budget is a warning, not a blocker")
	}
}

func TestCalculate_MissingEstimatesWarns(t *testing.T) {
	in := baseReadyInput()
	in.MissingEstimateCount = 3
	c := Calculate(in)
	item, _ := findItem(c, KeyMissingCostEstimates)
	if item.Status != ItemStatusWarning {
		t.Fatalf("expected missing_cost_estimates warning, got %q", item.Status)
	}
}

func TestCalculate_CostSplittingWarnings(t *testing.T) {
	cases := map[string]func(*ChecklistInput){
		"no travelers":     func(in *ChecklistInput) { in.TravelerCount = 0 },
		"invalid splits":   func(in *ChecklistInput) { in.InvalidSplitCount = 2 },
		"unassigned costs": func(in *ChecklistInput) { in.UnassignedCostCount = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := baseReadyInput()
			mutate(&in)
			c := Calculate(in)
			item, _ := findItem(c, KeyCostSplittingConfigured)
			if item.Status != ItemStatusWarning {
				t.Fatalf("expected cost_splitting_configured warning, got %q", item.Status)
			}
			if !c.CanSubmit() {
				t.Fatal("cost splitting problems are warnings, not blockers")
			}
		})
	}
}

func TestCalculate_AvailabilityUncheckedWarns(t *testing.T) {
	in := baseReadyInput()
	in.BookableItemCount = 3
	in.AvailabilityUncheckedCount = 2
	c := Calculate(in)
	item, _ := findItem(c, KeyAvailabilityChecked)
	if item.Status != ItemStatusWarning {
		t.Fatalf("expected availability_checked warning, got %q", item.Status)
	}
}

func TestCalculate_AvailabilityOKWhenNoBookableItems(t *testing.T) {
	in := baseReadyInput()
	in.BookableItemCount = 0
	in.AvailabilityUncheckedCount = 0
	c := Calculate(in)
	item, _ := findItem(c, KeyAvailabilityChecked)
	if item.Status != ItemStatusOK {
		t.Fatalf("expected availability_checked ok with no bookable items, got %q", item.Status)
	}
}

func TestCalculate_NoAvailabilitySignalItemsWhenAbsent(t *testing.T) {
	c := Calculate(baseReadyInput())
	for _, key := range []string{
		KeyAvailabilityLowConfidence,
		KeyAvailabilityUnavailable,
		KeyAvailabilityPriceChanged,
		KeyAvailabilityFallback,
	} {
		if _, ok := findItem(c, key); ok {
			t.Fatalf("expected %q to be absent when no signal present", key)
		}
	}
}

func TestCalculate_AvailabilityLowConfidenceWarns(t *testing.T) {
	in := baseReadyInput()
	in.AvailabilityLowConfidenceCount = 1
	c := Calculate(in)
	item, ok := findItem(c, KeyAvailabilityLowConfidence)
	if !ok || item.Status != ItemStatusWarning || item.Severity != SeverityWarning {
		t.Fatalf("expected low-confidence warning, got %+v ok=%v", item, ok)
	}
	if c.Status != ChecklistStatusWarning || !c.CanSubmit() {
		t.Fatalf("low-confidence should warn but not block, got status=%q canSubmit=%v", c.Status, c.CanSubmit())
	}
}

func TestCalculate_AvailabilityUnavailableWarns(t *testing.T) {
	in := baseReadyInput()
	in.AvailabilityUnavailableCount = 2
	c := Calculate(in)
	item, ok := findItem(c, KeyAvailabilityUnavailable)
	if !ok || item.Status != ItemStatusWarning {
		t.Fatalf("expected unavailable warning, got %+v ok=%v", item, ok)
	}
	if !c.CanSubmit() {
		t.Fatal("unavailable items must not block submission in v1")
	}
}

func TestCalculate_AvailabilityPriceChangedWarns(t *testing.T) {
	in := baseReadyInput()
	in.AvailabilityPriceChangedCount = 1
	c := Calculate(in)
	item, ok := findItem(c, KeyAvailabilityPriceChanged)
	if !ok || item.Status != ItemStatusWarning {
		t.Fatalf("expected price-changed warning, got %+v ok=%v", item, ok)
	}
}

func TestCalculate_AvailabilityFallbackIsInfoAndDoesNotBlockOrWarn(t *testing.T) {
	in := baseReadyInput()
	in.AvailabilityFallbackCount = 1
	c := Calculate(in)
	item, ok := findItem(c, KeyAvailabilityFallback)
	if !ok || item.Status != ItemStatusInfo || item.Severity != SeverityInfo {
		t.Fatalf("expected fallback info item, got %+v ok=%v", item, ok)
	}
	// Info must not inflate the warning count or change the OK roll-up.
	if c.Status != ChecklistStatusOK || c.WarningCount != 0 {
		t.Fatalf("fallback info should keep status ok, got status=%q warnings=%d", c.Status, c.WarningCount)
	}
	if !c.CanSubmit() {
		t.Fatal("fallback info must not block submission")
	}
}

func TestCalculate_WorkspaceBudgetMissingWarns(t *testing.T) {
	in := baseReadyInput()
	in.HasWorkspaceBudget = false
	c := Calculate(in)
	item, _ := findItem(c, KeyWorkspaceBudgetStatus)
	if item.Status != ItemStatusWarning {
		t.Fatalf("expected workspace_budget_status warning when none configured, got %q", item.Status)
	}
}

func TestTransitions(t *testing.T) {
	if !CanSubmitFrom(StatusDraft) || !CanSubmitFrom(StatusChangesRequested) || !CanSubmitFrom(StatusCancelled) {
		t.Fatal("submit should be allowed from draft/changes_requested/cancelled")
	}
	if CanSubmitFrom(StatusPendingApproval) || CanSubmitFrom(StatusApproved) || CanSubmitFrom(StatusNotRequired) {
		t.Fatal("submit must be rejected from pending/approved/not_required")
	}
	if !CanApproveFrom(StatusPendingApproval) || CanApproveFrom(StatusDraft) || CanApproveFrom(StatusApproved) {
		t.Fatal("approve only from pending")
	}
	if !CanRequestChangesFrom(StatusPendingApproval) || CanRequestChangesFrom(StatusApproved) {
		t.Fatal("request-changes only from pending")
	}
	if !CanCancelFrom(StatusPendingApproval) || CanCancelFrom(StatusDraft) {
		t.Fatal("cancel only from pending")
	}
}
