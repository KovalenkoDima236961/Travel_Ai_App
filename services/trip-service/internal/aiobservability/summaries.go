package aiobservability

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// PromptVersionForGenerationType is deliberately stable and shared with the
// AI Planning Service's response metadata. Changing prompt wording requires a
// new version, not an overwrite of a historical version.
func PromptVersionForGenerationType(generationType string) string {
	switch generationType {
	case "full_generation":
		return "itinerary_generation_v1"
	case "day_regeneration", "quality_improvement_day":
		return "day_regeneration_v1"
	case "item_regeneration", "quality_improvement_item":
		return "item_regeneration_v1"
	case "policy_repair":
		return "policy_repair_v1"
	case "budget_optimization_day":
		return "budget_optimization_day_v1"
	case "template_adaptation":
		return "template_adaptation_v1"
	default:
		return "unknown_v1"
	}
}

func BuildJobInputSummary(tripID, jobID, userID uuid.UUID, generationType string, dayNumber, itemIndex *int, hasInstruction, hasPayload bool) json.RawMessage {
	return marshalSafe(map[string]any{
		"tripId": tripID.String(), "jobId": jobID.String(), "generationType": generationType,
		"hasInstruction": hasInstruction, "hasJobPayload": hasPayload,
		"target": map[string]any{"dayNumber": dayNumber, "itemIndex": itemIndex},
	})
}

func DefaultConstraintsSummary() json.RawMessage {
	return marshalSafe(map[string]any{
		"planningConstraintSchemaVersion": "v1", "explicitRequestConstraintsCount": 0,
		"workspacePolicyRuleCount": 0, "blockingPolicyCount": 0, "groupMustHaveCount": 0,
		"groupAvoidCount": 0, "warningsCount": 0, "blockersCount": 0,
	})
}

func DefaultRAGSummary() json.RawMessage {
	return marshalSafe(map[string]any{"retrievalEnabled": false, "retrievedChunkCount": 0, "suspiciousPromptInjectionWarningCount": 0})
}

func DefaultPromptSummary(promptVersion string) json.RawMessage {
	return marshalSafe(map[string]any{"promptVersion": promptVersion, "redactionApplied": true, "promptSnapshotStored": false})
}

func BuildOutputSummary(itinerary json.RawMessage, saved bool, versionID string) json.RawMessage {
	var payload struct {
		Days []struct {
			Items []json.RawMessage `json:"items"`
		} `json:"days"`
	}
	_ = json.Unmarshal(itinerary, &payload)
	items := 0
	for _, day := range payload.Days {
		items += len(day.Items)
	}
	return marshalSafe(map[string]any{"finalItineraryDayCount": len(payload.Days), "itemCount": items, "saved": saved, "versionId": strings.TrimSpace(versionID)})
}

// QualitySummaries intentionally retain only issue identifiers, categories,
// severities, and counts. Titles/descriptions can contain user-owned text.
func QualitySummaries(payload json.RawMessage) (quality string, validation, repair json.RawMessage, repairDurationMS *int) {
	var envelope struct {
		GenerationQuality struct {
			Status             string   `json:"status"`
			ValidatorVersion   string   `json:"validatorVersion"`
			RepairAttempts     int      `json:"repairAttempts"`
			MaxRepairAttempts  int      `json:"maxRepairAttempts"`
			BlockingIssueCount int      `json:"blockingIssueCount"`
			CriticalIssueCount int      `json:"criticalIssueCount"`
			HighIssueCount     int      `json:"highIssueCount"`
			WarningIssueCount  int      `json:"warningIssueCount"`
			Warnings           []string `json:"warnings"`
			RemainingIssues    []struct {
				ID       string `json:"id"`
				Category string `json:"category"`
				Severity string `json:"severity"`
			} `json:"remainingIssues"`
			RepairedIssues []struct {
				ID string `json:"id"`
			} `json:"repairedIssues"`
			RepairAttemptLog []struct {
				Attempt         int      `json:"attempt"`
				TargetIssueIDs  []string `json:"targetIssueIds"`
				IssuesFixed     []string `json:"issuesFixed"`
				IssuesRemaining []string `json:"issuesRemaining"`
				DurationMS      int      `json:"durationMs"`
				RepairScope     struct {
					Type string `json:"type"`
				} `json:"repairScope"`
			} `json:"repairAttemptsLog"`
		} `json:"generationQuality"`
	}
	if len(payload) == 0 || json.Unmarshal(payload, &envelope) != nil || envelope.GenerationQuality.Status == "" {
		return "", nil, nil, nil
	}
	byCategory := map[string]int{}
	bySeverity := map[string]int{}
	blockingIDs := make([]string, 0)
	for _, issue := range envelope.GenerationQuality.RemainingIssues {
		byCategory[issue.Category]++
		bySeverity[issue.Severity]++
		if issue.Severity == "blocking" {
			blockingIDs = append(blockingIDs, issue.ID)
		}
	}
	validation = marshalSafe(map[string]any{
		"validatorVersion": envelope.GenerationQuality.ValidatorVersion, "validationEnabled": true,
		"saveAllowed":           !strings.Contains(envelope.GenerationQuality.Status, "blocked") && envelope.GenerationQuality.Status != "repair_failed" && envelope.GenerationQuality.Status != "schema_invalid" && envelope.GenerationQuality.Status != "ai_output_invalid",
		"issueCountsBySeverity": bySeverity, "issueCountsByCategory": byCategory, "blockingIssueIds": blockingIDs,
		"qualityStatus": envelope.GenerationQuality.Status,
	})
	if len(envelope.GenerationQuality.RepairAttemptLog) > 0 || envelope.GenerationQuality.RepairAttempts > 0 {
		attempts := make([]map[string]any, 0, len(envelope.GenerationQuality.RepairAttemptLog))
		totalDuration := 0
		for _, attempt := range envelope.GenerationQuality.RepairAttemptLog {
			totalDuration += attempt.DurationMS
			attempts = append(attempts, map[string]any{"attempt": attempt.Attempt, "repairScope": attempt.RepairScope.Type, "targetIssueIds": attempt.TargetIssueIDs, "issuesFixed": len(attempt.IssuesFixed), "issuesRemaining": len(attempt.IssuesRemaining), "durationMs": attempt.DurationMS})
		}
		repair = marshalSafe(map[string]any{"repairEnabled": true, "repairAttempts": envelope.GenerationQuality.RepairAttempts, "maxRepairAttempts": envelope.GenerationQuality.MaxRepairAttempts, "repairedIssueCount": len(envelope.GenerationQuality.RepairedIssues), "attempts": attempts, "finalQualityStatus": envelope.GenerationQuality.Status})
		repairDurationMS = intPtr(totalDuration)
	}
	return envelope.GenerationQuality.Status, validation, repair, repairDurationMS
}

func marshalSafe(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}
