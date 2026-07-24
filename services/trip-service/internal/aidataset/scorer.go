package aidataset

import (
	"bytes"
	"encoding/json"
	"strings"
)

type QualityResult struct {
	Score        float64            `json:"score"`
	Status       string             `json:"status"`
	Breakdown    map[string]float64 `json:"breakdown"`
	HardBlockers []string           `json:"hardBlockers"`
	Warnings     []string           `json:"warnings"`
}

func ScoreExample(example TrainingExample, project DatasetProject, cfg Config) QualityResult {
	breakdown := map[string]float64{
		"schemaValidity":       schemaValidity(example),
		"groundingQuality":     groundingQuality(example),
		"itineraryValidation":  itineraryValidation(example),
		"placeValidity":        placeValidity(example),
		"schedulePlausibility": schedulePlausibility(example),
		"preferenceMatch":      preferenceMatch(example),
		"budgetPlausibility":   budgetPlausibility(example),
		"userAcceptance":       userAcceptance(example),
		"privacyConfidence":    privacyConfidence(example, project),
	}
	blockers := hardBlockers(example, project)
	score := 0.20*breakdown["schemaValidity"] +
		0.20*breakdown["groundingQuality"] +
		0.15*breakdown["itineraryValidation"] +
		0.10*breakdown["placeValidity"] +
		0.10*breakdown["schedulePlausibility"] +
		0.10*breakdown["preferenceMatch"] +
		0.05*breakdown["budgetPlausibility"] +
		0.05*breakdown["userAcceptance"] +
		0.05*breakdown["privacyConfidence"]

	status := QualityPassed
	if len(blockers) > 0 {
		status = QualityFailed
		score = 0
	} else if score < cfg.MinAutoReviewScore {
		status = QualityFailed
	} else if score < cfg.MinApprovalScore {
		status = QualityNeedsReview
	}
	return QualityResult{
		Score:        roundScore(score),
		Status:       status,
		Breakdown:    breakdown,
		HardBlockers: blockers,
		Warnings:     qualityWarnings(example, score, cfg),
	}
}

func schemaValidity(example TrainingExample) float64 {
	if !isJSONObject(example.InputJSON) || !isJSONObject(example.ExpectedOutputJSON) {
		return 0
	}
	if len(bytes.TrimSpace(example.GroundingJSON)) > 0 && !isJSON(example.GroundingJSON) {
		return 0
	}
	if len(bytes.TrimSpace(example.NegativeOutputJSON)) > 0 && !isJSON(example.NegativeOutputJSON) {
		return 0
	}
	return 1
}

func groundingQuality(example TrainingExample) float64 {
	if hasBoolLabel(example, "ungrounded", true) || hasBoolLabel(example, "knownHallucinatedPlace", true) {
		return 0
	}
	if len(bytes.TrimSpace(example.GroundingJSON)) == 0 {
		if example.SourceType == "manual_curated" || example.SourceType == "golden_case" || example.SourceType == "synthetic" {
			return 0.85
		}
		return 0.65
	}
	obj := jsonMap(example.GroundingJSON)
	if len(obj) == 0 {
		return 0.65
	}
	if hasNonEmptyArray(obj, "groundingIds") || hasNonEmptyArray(obj, "places") || hasNonEmptyArray(obj, "facts") {
		return 1
	}
	if confidence, ok := numericField(obj, "confidence"); ok {
		if confidence < 0 {
			return 0
		}
		if confidence > 1 {
			return 1
		}
		return confidence
	}
	return 0.8
}

func itineraryValidation(example TrainingExample) float64 {
	if example.TaskType != TaskItineraryGeneration &&
		example.TaskType != TaskDayRegeneration &&
		example.TaskType != TaskItemRegeneration &&
		example.TaskType != TaskPolicyRepair &&
		example.TaskType != TaskBudgetOptimization {
		return 1
	}
	obj := jsonMap(example.ExpectedOutputJSON)
	if hasNonEmptyArray(obj, "days") {
		return 1
	}
	if itinerary, ok := obj["itinerary"].(map[string]any); ok && hasNonEmptyArray(itinerary, "days") {
		return 1
	}
	if hasBoolLabel(example, "schemaOnly", true) {
		return 0.7
	}
	return 0.4
}

func placeValidity(example TrainingExample) float64 {
	if hasBoolLabel(example, "hallucinatedPlace", true) || hasBoolLabel(example, "destinationMismatch", true) {
		return 0
	}
	if value, ok := numericLabel(example, "groundedPlaceRate"); ok {
		if value < 0 {
			return 0
		}
		if value > 1 {
			return 1
		}
		return value
	}
	return 0.9
}

func schedulePlausibility(example TrainingExample) float64 {
	if hasBoolLabel(example, "overpackedSchedule", true) {
		return 0.2
	}
	maxItems := maxItemsPerDay(example.ExpectedOutputJSON)
	switch {
	case maxItems == 0:
		return 0.8
	case maxItems <= 6:
		return 1
	case maxItems <= 9:
		return 0.75
	default:
		return 0.35
	}
}

func preferenceMatch(example TrainingExample) float64 {
	if hasBoolLabel(example, "preferenceMismatch", true) {
		return 0.3
	}
	if hasBoolLabel(example, "matchesPreferences", true) {
		return 1
	}
	return 0.8
}

func budgetPlausibility(example TrainingExample) float64 {
	if hasBoolLabel(example, "budgetImplausible", true) {
		return 0.25
	}
	if hasBoolLabel(example, "budgetPlausible", true) {
		return 1
	}
	return 0.85
}

func userAcceptance(example TrainingExample) float64 {
	if hasBoolLabel(example, "userAccepted", true) || hasBoolLabel(example, "postTripPositive", true) || hasBoolLabel(example, "successfulRepair", true) {
		return 1
	}
	if example.SourceType == "manual_curated" || example.SourceType == "golden_case" || example.SourceType == "synthetic" {
		return 0.85
	}
	return 0.65
}

func privacyConfidence(example TrainingExample, project DatasetProject) float64 {
	if example.SanitizationStatus != SanitizationPassed {
		return 0
	}
	if !project.ConsentRequired || example.ConsentStatus == ConsentNotRequired || example.ConsentStatus == ConsentGranted {
		return 1
	}
	return 0
}

func hardBlockers(example TrainingExample, project DatasetProject) []string {
	blockers := make([]string, 0)
	if schemaValidity(example) == 0 {
		blockers = append(blockers, "schema invalid")
	}
	if example.SanitizationStatus == SanitizationFailed {
		blockers = append(blockers, "sanitization failed")
	}
	if project.ConsentRequired && example.ConsentStatus != ConsentNotRequired && example.ConsentStatus != ConsentGranted {
		blockers = append(blockers, "consent not granted")
	}
	if example.ConsentStatus == ConsentRevoked || example.ConsentStatus == ConsentProhibited {
		blockers = append(blockers, "consent revoked or prohibited")
	}
	if hasBoolLabel(example, "privateDataDetected", true) {
		blockers = append(blockers, "private data detected")
	}
	if hasBoolLabel(example, "hallucinatedPlace", true) || hasBoolLabel(example, "knownHallucinatedPlace", true) {
		blockers = append(blockers, "known hallucinated place")
	}
	if hasBoolLabel(example, "hiddenPromptIncluded", true) || rawContainsHiddenPrompt(example.InputJSON) || rawContainsHiddenPrompt(example.ExpectedOutputJSON) {
		blockers = append(blockers, "hidden/system prompt included")
	}
	if licenseBlocked(example) {
		blockers = append(blockers, "source license not allowed")
	}
	return dedupeSorted(blockers)
}

func qualityWarnings(example TrainingExample, score float64, cfg Config) []string {
	warnings := make([]string, 0)
	if score < cfg.MinApprovalScore {
		warnings = append(warnings, "below approval threshold")
	}
	if example.SanitizationStatus == SanitizationNeedsReview {
		warnings = append(warnings, "sanitization needs review")
	}
	if len(bytes.TrimSpace(example.NegativeOutputJSON)) > 0 {
		warnings = append(warnings, "negative output is for evaluation or repair only")
	}
	return dedupeSorted(warnings)
}

func isJSON(raw json.RawMessage) bool {
	var value any
	return json.Unmarshal(raw, &value) == nil
}

func isJSONObject(raw json.RawMessage) bool {
	_, ok := jsonObject(raw)
	return ok
}

func hasNonEmptyArray(obj map[string]any, key string) bool {
	value, ok := obj[key]
	if !ok {
		return false
	}
	array, ok := value.([]any)
	return ok && len(array) > 0
}

func numericField(obj map[string]any, key string) (float64, bool) {
	switch value := obj[key].(type) {
	case json.Number:
		f, err := value.Float64()
		return f, err == nil
	case float64:
		return value, true
	case int:
		return float64(value), true
	default:
		return 0, false
	}
}

func hasBoolLabel(example TrainingExample, key string, want bool) bool {
	labels := jsonMap(example.LabelsJSON)
	value, ok := labels[key].(bool)
	return ok && value == want
}

func numericLabel(example TrainingExample, key string) (float64, bool) {
	return numericField(jsonMap(example.LabelsJSON), key)
}

func licenseBlocked(example TrainingExample) bool {
	provenance := jsonMap(example.ProvenanceJSON)
	status, _ := provenance["licenseStatus"].(string)
	copiesText, _ := provenance["copiesProviderText"].(bool)
	status = strings.ToLower(strings.TrimSpace(status))
	return copiesText && (status == "" || status == "unknown" || status == "incompatible" || status == "not_allowed")
}

func rawContainsHiddenPrompt(raw json.RawMessage) bool {
	lower := strings.ToLower(string(raw))
	return strings.Contains(lower, "system_prompt") ||
		strings.Contains(lower, "systemprompt") ||
		strings.Contains(lower, "hidden prompt") ||
		strings.Contains(lower, "chain_of_thought") ||
		strings.Contains(lower, "chain-of-thought")
}

func maxItemsPerDay(raw json.RawMessage) int {
	obj := jsonMap(raw)
	if itinerary, ok := obj["itinerary"].(map[string]any); ok {
		obj = itinerary
	}
	days, ok := obj["days"].([]any)
	if !ok {
		return 0
	}
	max := 0
	for _, day := range days {
		dayObj, ok := day.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"items", "activities"} {
			if items, ok := dayObj[key].([]any); ok && len(items) > max {
				max = len(items)
			}
		}
	}
	return max
}

func roundScore(value float64) float64 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return float64(int(value*10000+0.5)) / 10000
}
