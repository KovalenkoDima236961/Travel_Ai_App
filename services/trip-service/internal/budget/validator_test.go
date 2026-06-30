package budget

import (
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func TestNormalizeEstimatedCost_RepairsSoftIssues(t *testing.T) {
	amount := 12.5
	c := &aggregate.EstimatedCost{
		Amount:     &amount,
		Currency:   " eur ",
		Category:   "MYSTERY",
		Confidence: "maybe",
		Source:     "",
		Note:       "  ok  ",
	}
	if err := NormalizeEstimatedCost(c, SourceManual); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Currency != "EUR" {
		t.Fatalf("expected currency normalized to EUR, got %q", c.Currency)
	}
	if c.Category != CategoryOther {
		t.Fatalf("expected unknown category to become other, got %q", c.Category)
	}
	if c.Confidence != ConfidenceLow {
		t.Fatalf("expected unknown confidence to become low, got %q", c.Confidence)
	}
	if c.Source != SourceManual {
		t.Fatalf("expected source defaulted to manual, got %q", c.Source)
	}
	if c.Note != "ok" {
		t.Fatalf("expected note trimmed, got %q", c.Note)
	}
}

func TestNormalizeEstimatedCost_RejectsNegativeAmount(t *testing.T) {
	amount := -1.0
	err := NormalizeEstimatedCost(&aggregate.EstimatedCost{Amount: &amount}, SourceManual)
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestNormalizeEstimatedCost_RejectsBadCurrency(t *testing.T) {
	amount := 1.0
	err := NormalizeEstimatedCost(&aggregate.EstimatedCost{Amount: &amount, Currency: "EU"}, SourceManual)
	if err == nil {
		t.Fatal("expected error for malformed currency")
	}
}

func TestNormalizeEstimatedCost_TruncatesLongNote(t *testing.T) {
	c := &aggregate.EstimatedCost{Note: strings.Repeat("x", 400)}
	if err := NormalizeEstimatedCost(c, SourceAI); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len([]rune(c.Note)) != maxNoteLength {
		t.Fatalf("expected note truncated to %d, got %d", maxNoteLength, len([]rune(c.Note)))
	}
}

func TestNormalizeEstimatedCost_DefaultsSourceAI(t *testing.T) {
	amount := 5.0
	c := &aggregate.EstimatedCost{Amount: &amount}
	if err := NormalizeEstimatedCost(c, SourceAI); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Source != SourceAI {
		t.Fatalf("expected source ai, got %q", c.Source)
	}
	if c.Confidence != ConfidenceLow {
		t.Fatalf("expected confidence defaulted to low when amount present, got %q", c.Confidence)
	}
}

func TestNormalizeBudgetInput(t *testing.T) {
	amount := 700.0
	gotAmount, gotCurrency, err := NormalizeBudgetInput(&amount, "", "GBP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAmount == nil || *gotAmount != 700 || gotCurrency != "GBP" {
		t.Fatalf("expected fallback currency GBP, got amount=%v currency=%q", gotAmount, gotCurrency)
	}

	// Nil amount clears the budget.
	clearedAmount, clearedCurrency, err := NormalizeBudgetInput(nil, "EUR", "GBP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clearedAmount != nil || clearedCurrency != "" {
		t.Fatalf("expected cleared budget, got amount=%v currency=%q", clearedAmount, clearedCurrency)
	}

	// Negative amount rejected.
	negative := -1.0
	if _, _, err := NormalizeBudgetInput(&negative, "EUR", ""); err == nil {
		t.Fatal("expected error for negative budget amount")
	}

	// Bad currency rejected.
	if _, _, err := NormalizeBudgetInput(&amount, "EU", ""); err == nil {
		t.Fatal("expected error for malformed budget currency")
	}

	// No currency anywhere defaults to EUR.
	_, defaultCurrency, err := NormalizeBudgetInput(&amount, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultCurrency != DefaultCurrency {
		t.Fatalf("expected default currency, got %q", defaultCurrency)
	}
}
