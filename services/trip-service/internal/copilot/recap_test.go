package copilot

import (
	"encoding/json"
	"testing"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
)

func TestSafeRecapExcludesUserNotesAndCandidateMetadata(t *testing.T) {
	context := safeRecap(appdto.RecapJSON{
		Title: "Trip recap", Summary: "Safe summary", UserEditableNotes: "private note",
		LessonsLearned: []string{"Start early"}, FuturePreferences: []appdto.LearningCandidate{{Label: "private preference", Metadata: map[string]any{"secret": "nope"}}},
	})
	encoded, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"private note", "private preference", "nope"} {
		if contains(text, forbidden) {
			t.Fatalf("safe recap leaked %q: %s", forbidden, text)
		}
	}
}
