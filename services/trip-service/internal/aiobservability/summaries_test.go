package aiobservability

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSafeJSONRemovesSensitiveSummaryContent(t *testing.T) {
	clean := safeJSON(json.RawMessage(`{"email":"traveler@example.com","receiptOcrRawText":"private card data","calendarEventTitle":"private appointment","summary":"Email traveler@example.com"}`))
	text := string(clean)
	for _, forbidden := range []string{"traveler@example.com", "private card data", "private appointment"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary leaked %q: %s", forbidden, text)
		}
	}
}

func TestJobSummaryAndOutputDoNotContainRawInstructions(t *testing.T) {
	tripID, jobID, userID := uuid.New(), uuid.New(), uuid.New()
	summary := BuildJobInputSummary(tripID, jobID, userID, "full_generation", nil, nil, true, true)
	if strings.Contains(string(summary), userID.String()) || strings.Contains(string(summary), "instruction") && strings.Contains(string(summary), "private") {
		t.Fatalf("job summary exposed private content: %s", summary)
	}
	output := BuildOutputSummary(json.RawMessage(`{"days":[{"title":"Private day","items":[{"name":"Secret place"}]}]}`), true, "")
	if strings.Contains(string(output), "Private") || strings.Contains(string(output), "Secret") {
		t.Fatalf("output summary retained itinerary text: %s", output)
	}
}

func TestQualitySummariesKeepCountsAndIDsOnly(t *testing.T) {
	payload := json.RawMessage(`{"generationQuality":{"status":"repaired_with_warnings","validatorVersion":"v1","repairAttempts":1,"maxRepairAttempts":2,"remainingIssues":[{"id":"route_1","category":"route","severity":"warning","title":"Private user request"}],"repairedIssues":[{"id":"policy_1"}],"repairAttemptsLog":[{"attempt":1,"repairScope":{"type":"day"},"targetIssueIds":["policy_1"],"issuesFixed":["policy_1"],"issuesRemaining":["route_1"],"durationMs":42}]}}`)
	quality, validation, repair, duration := QualitySummaries(payload)
	if quality != "repaired_with_warnings" || duration == nil || *duration != 42 {
		t.Fatalf("unexpected quality summary: %q %v", quality, duration)
	}
	if strings.Contains(string(validation), "Private user request") {
		t.Fatalf("validation summary retained issue text: %s", validation)
	}
	if !strings.Contains(string(repair), "policy_1") {
		t.Fatalf("repair issue IDs were not retained: %s", repair)
	}
}

func TestSafeFailureMessageDoesNotRetainProviderErrorText(t *testing.T) {
	message := safeFailureMessage("ai_generation_failed")
	if strings.Contains(message, "private itinerary instruction") || !strings.Contains(message, "provider") {
		t.Fatalf("unsafe failure message: %q", message)
	}
}
