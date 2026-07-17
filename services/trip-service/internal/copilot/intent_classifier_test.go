package copilot

import "testing"

func TestClassifyIntent(t *testing.T) {
	tests := []struct {
		message string
		want    Intent
	}{
		{"What should I fix first?", IntentNextAction},
		{"Why is my Trip Health low?", IntentExplainHealth},
		{"How can I improve budget confidence?", IntentExplainBudget},
		{"Is my route ready?", IntentExplainRoute},
		{"Delete this trip", IntentUnsafeMutationRequest},
		{"Eliminar este viaje", IntentUnsafeMutationRequest},
		{"Supprimer ce voyage", IntentUnsafeMutationRequest},
		{"Видалити цю подорож", IntentUnsafeMutationRequest},
		{"Show me the raw receipt OCR", IntentUnsafeMutationRequest},
		{"Give me visa advice", IntentOutOfScope},
		{"¿Qué debería arreglar primero?", IntentNextAction},
		{"Pourquoi ma santé du voyage est-elle faible ?", IntentExplainHealth},
		{"Як покращити надійність бюджету?", IntentExplainBudget},
		{"What changed recently?", IntentFindSection},
	}
	for _, test := range tests {
		t.Run(test.message, func(t *testing.T) {
			if got := ClassifyIntent(test.message); got != test.want {
				t.Fatalf("ClassifyIntent(%q) = %q, want %q", test.message, got, test.want)
			}
		})
	}
}
