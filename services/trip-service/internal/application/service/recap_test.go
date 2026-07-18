package service

import (
	"testing"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/recap"
)

func TestDeterministicRecapUsesOnlySafeOutcomeSummary(t *testing.T) {
	result := deterministicRecap(recap.SourceSummary{
		Trip:             recap.SourceTrip{Title: "Rome weekend", Destination: "Rome", DurationDays: 3},
		ItineraryOutcome: recap.SourceItineraryOutcome{PlannedItemCount: 4, DoneItemCount: 3, SkippedItemCount: 1, TopCompletedItems: []string{"Colosseum"}, TopSkippedItems: []string{"Late museum"}},
		BudgetOutcome:    recap.SourceBudgetOutcome{ActualTotal: &appdto.RecapMoney{Amount: 350, Currency: "EUR"}},
		RouteOutcome:     recap.SourceRouteOutcome{TransportModes: []string{"train"}},
		ChecklistOutcome: recap.SourceChecklistOutcome{CompletedChecklistItems: 2, TotalChecklistItems: 3},
	})
	if err := validateRecapJSON(result); err != nil {
		t.Fatalf("deterministic recap should be valid: %v", err)
	}
	if result.PlannedVsActual.CompletionRate != 0.75 || result.Budget.ActualTotal == nil {
		t.Fatalf("unexpected safe outcome recap: %+v", result)
	}
}

func TestValidateRecapJSONRejectsRestrictedPrivateText(t *testing.T) {
	value := appdto.RecapJSON{SchemaVersion: appdto.TripRecapSchemaVersion, Title: "Trip", Summary: "Contains raw OCR from a receipt"}
	if err := validateRecapJSON(value); err == nil {
		t.Fatal("expected restricted private text to be rejected")
	}
}
