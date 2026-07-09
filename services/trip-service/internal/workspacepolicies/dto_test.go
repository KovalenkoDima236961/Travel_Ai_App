package workspacepolicies

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUpsertInputRequiresEnabledForEveryRule(t *testing.T) {
	input := UpsertInput{
		Name:  "Workspace policy",
		Rules: DefaultRules(),
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(
		string(raw),
		`"requireTripBudget":{"enabled":false,`,
		`"requireTripBudget":{`,
		1,
	))
	var decoded UpsertInput
	if err := json.Unmarshal(raw, &decoded); err == nil ||
		!strings.Contains(err.Error(), "requireTripBudget.enabled is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateInputRejectsInvalidMoneyAndTime(t *testing.T) {
	input := UpsertInput{Name: "Policy", Rules: DefaultRules()}
	input.Rules.Rules.MaxTripBudget.Enabled = true
	input.Rules.Rules.MaxTripBudget.Amount = -1
	if err := ValidateInput(&input); err == nil {
		t.Fatal("expected negative amount error")
	}

	input = UpsertInput{Name: "Policy", Rules: DefaultRules()}
	input.Rules.Rules.NoLateActivitiesAfter.Enabled = true
	input.Rules.Rules.NoLateActivitiesAfter.Time = "25:00"
	if err := ValidateInput(&input); err == nil {
		t.Fatal("expected invalid time error")
	}
}
