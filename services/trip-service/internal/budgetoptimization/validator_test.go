package budgetoptimization

import (
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func TestNormalizeProposalContent_AcceptsValidProposal(t *testing.T) {
	content := validProposalContent()

	if err := NormalizeProposalContent(&content, 2, "EUR"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content.ProposedDay.Day != 2 {
		t.Fatalf("expected proposed day to normalize to day 2, got %d", content.ProposedDay.Day)
	}
}

func TestNormalizeProposalContent_RejectsWrongDay(t *testing.T) {
	content := validProposalContent()
	content.DayNumber = 3

	if err := NormalizeProposalContent(&content, 2, "EUR"); err == nil {
		t.Fatal("expected wrong day number to be rejected")
	}
}

func TestNormalizeProposalContent_RejectsNoSavings(t *testing.T) {
	content := validProposalContent()
	content.EstimatedSavingsAmount = 0

	err := NormalizeProposalContent(&content, 2, "EUR")
	if err == nil || !strings.Contains(err.Error(), "no_optimization_found") {
		t.Fatalf("expected no_optimization_found error, got %v", err)
	}
}

func TestNormalizeProposalContent_RejectsNegativeItemCost(t *testing.T) {
	content := validProposalContent()
	negative := -1.0
	content.ProposedDay.Items[0].EstimatedCost = &aggregate.EstimatedCost{
		Amount:   &negative,
		Currency: "EUR",
	}

	if err := NormalizeProposalContent(&content, 2, "EUR"); err == nil {
		t.Fatal("expected negative item cost to be rejected")
	}
}

func validProposalContent() ProposalContent {
	oldIndex := 0
	savings := 40.0
	amount := 60.0
	return ProposalContent{
		Summary:                   "Replace a paid tour with a cheaper self-guided visit.",
		Scope:                     ScopeDay,
		DayNumber:                 2,
		Currency:                  "EUR",
		BaseDayEstimatedTotal:     120,
		ProposedDayEstimatedTotal: 80,
		EstimatedSavingsAmount:    savings,
		Confidence:                ConfidenceMedium,
		Changes: []ProposalChange{
			{
				Type:                   ChangeReplaceItem,
				OldItemIndex:           &oldIndex,
				OldItemName:            "Paid tour",
				NewItemName:            "Self-guided visit",
				EstimatedSavingsAmount: &savings,
				Currency:               "EUR",
			},
		},
		ProposedDay: aggregate.ItineraryDay{
			Day:   2,
			Title: "Budget Day",
			Items: []aggregate.ItineraryItem{
				{
					Time: "10:00",
					Type: "activity",
					Name: "Self-guided visit",
					EstimatedCost: &aggregate.EstimatedCost{
						Amount:   &amount,
						Currency: "EUR",
					},
				},
			},
		},
	}
}
