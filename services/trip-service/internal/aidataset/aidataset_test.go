package aidataset

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSanitizeExampleRedactsSensitiveValuesAndKeepsTravelFields(t *testing.T) {
	input := raw(`{
		"destinationName": "Paris",
		"departureDate": "2026-07-10",
		"userEmail": "traveler@example.com",
		"notes": "Call me at 415-555-1212",
		"shareUrl": "https://app.test/share?token=secret-token-123&view=public"
	}`)
	output := raw(`{
		"itinerary": {
			"days": [
				{"name": "Arrival", "items": [{"name": "Louvre Museum"}]}
			]
		}
	}`)

	result, err := SanitizeExample(input, nil, output)
	if err != nil {
		t.Fatalf("sanitize example: %v", err)
	}
	if result.Status != SanitizationFailed {
		t.Fatalf("expected high-risk private data to fail sanitization, got %s", result.Status)
	}
	if !result.PrivateDetected {
		t.Fatal("expected private data detection")
	}
	sanitized := string(result.InputJSON)
	for _, forbidden := range []string{"traveler@example.com", "415-555-1212", "secret-token-123"} {
		if strings.Contains(sanitized, forbidden) {
			t.Fatalf("sanitized input still contains %q: %s", forbidden, sanitized)
		}
	}
	for _, expected := range []string{"Paris", "2026-07-10"} {
		if !strings.Contains(sanitized, expected) {
			t.Fatalf("sanitized input should keep %q: %s", expected, sanitized)
		}
	}
	if containsValue(result.RemovedFields, "input.destinationName") {
		t.Fatalf("destinationName should not be removed: %#v", result.RemovedFields)
	}
}

func TestSanitizeExampleKeepsPlaceNamesAndAllowedGroundingIDs(t *testing.T) {
	input := raw(`{
		"destinationName": "Kyoto",
		"placeId": "places/kiyomizu-dera",
		"name": "Kiyomizu-dera"
	}`)
	output := raw(`{"days":[{"items":[{"name":"Kiyomizu-dera"}]}]}`)

	result, err := SanitizeExample(input, raw(`{"groundingIds":["places/kiyomizu-dera"]}`), output)
	if err != nil {
		t.Fatalf("sanitize example: %v", err)
	}
	if result.Status != SanitizationPassed {
		t.Fatalf("expected safe travel data to pass sanitization, got %s warnings=%v removed=%v", result.Status, result.Warnings, result.RemovedFields)
	}
	sanitized := string(result.InputJSON)
	for _, expected := range []string{"Kyoto", "places/kiyomizu-dera", "Kiyomizu-dera"} {
		if !strings.Contains(sanitized, expected) {
			t.Fatalf("sanitized input should keep %q: %s", expected, sanitized)
		}
	}
}

func TestScoreExampleBlocksConsentAndProviderLicenseIssues(t *testing.T) {
	project := DatasetProject{
		ID:                  uuid.New(),
		Key:                 "itinerary-generation",
		TaskType:            TaskItineraryGeneration,
		SchemaVersion:       SchemaVersion,
		ConsentRequired:     true,
		MinimumQualityScore: 0.85,
	}
	example := cleanExample(project.ID)

	result := ScoreExample(example, project, DefaultConfig())
	if result.Status != QualityPassed {
		t.Fatalf("expected clean example to pass, got status=%s score=%v blockers=%v", result.Status, result.Score, result.HardBlockers)
	}
	if result.Score < 0.85 {
		t.Fatalf("expected score above approval threshold, got %v", result.Score)
	}

	revoked := example
	revoked.ConsentStatus = ConsentRevoked
	result = ScoreExample(revoked, project, DefaultConfig())
	if result.Status != QualityFailed || !containsValue(result.HardBlockers, "consent revoked or prohibited") {
		t.Fatalf("expected revoked consent hard blocker, got status=%s blockers=%v", result.Status, result.HardBlockers)
	}

	blockedLicense := example
	blockedLicense.ProvenanceJSON = raw(`{"licenseStatus":"unknown","copiesProviderText":true}`)
	result = ScoreExample(blockedLicense, project, DefaultConfig())
	if result.Status != QualityFailed || !containsValue(result.HardBlockers, "source license not allowed") {
		t.Fatalf("expected license hard blocker, got status=%s blockers=%v", result.Status, result.HardBlockers)
	}
}

func TestAssignSplitsKeepsSourceGroupsTogetherAndGoldenHoldout(t *testing.T) {
	projectID := uuid.New()
	tripID := uuid.New()
	first := cleanExample(projectID)
	first.ID = uuid.New()
	first.TripID = &tripID
	second := cleanExample(projectID)
	second.ID = uuid.New()
	second.TripID = &tripID

	assignments := AssignSplits([]TrainingExample{first, second})
	if len(assignments) != 2 {
		t.Fatalf("expected two assignments, got %d", len(assignments))
	}
	if assignments[0].SourceGroup != assignments[1].SourceGroup {
		t.Fatalf("expected shared trip source group, got %q and %q", assignments[0].SourceGroup, assignments[1].SourceGroup)
	}
	if assignments[0].Split != assignments[1].Split {
		t.Fatalf("expected shared split for source group, got %q and %q", assignments[0].Split, assignments[1].Split)
	}

	golden := cleanExample(projectID)
	golden.SourceType = "golden_case"
	golden.LabelsJSON = raw(`{"benchmarkSplit":"holdout"}`)
	goldenAssignments := AssignSplits([]TrainingExample{golden})
	if goldenAssignments[0].Split != SplitHoldout {
		t.Fatalf("expected golden benchmark case in holdout, got %s", goldenAssignments[0].Split)
	}

	manifest := BuildManifest(DatasetProject{ID: projectID}, "v1", []TrainingExample{first, second}, assignments, 0.85)
	if len(manifest.Assignments) != 2 || len(manifest.ExampleIDs) != 2 {
		t.Fatalf("manifest should retain assignment snapshot, got assignments=%d exampleIDs=%d", len(manifest.Assignments), len(manifest.ExampleIDs))
	}
	parsed := manifestAssignments(mustTestJSON(manifest))
	if len(parsed) != 2 || parsed[0].ExampleID != first.ID {
		t.Fatalf("manifest assignment parser lost example order: %#v", parsed)
	}
}

func TestConsentRevocationPredicateScopesUpdates(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New().String()
	predicate, args, err := consentRevocationPredicate(userID, ScopeTrip, &tripID)
	if err != nil {
		t.Fatalf("trip predicate: %v", err)
	}
	if predicate != "user_id = $1 AND trip_id = $2" {
		t.Fatalf("unexpected trip predicate: %s", predicate)
	}
	if len(args) != 2 || args[0] != userID {
		t.Fatalf("unexpected trip args: %#v", args)
	}

	versionID := uuid.New().String()
	predicate, args, err = consentRevocationPredicate(userID, ScopeItineraryVersion, &versionID)
	if err != nil {
		t.Fatalf("version predicate: %v", err)
	}
	if !strings.Contains(predicate, "source_entity_id = $2") || !strings.Contains(predicate, "itineraryVersionId") {
		t.Fatalf("unexpected version predicate: %s", predicate)
	}
	if len(args) != 2 || args[1] != versionID {
		t.Fatalf("unexpected version args: %#v", args)
	}
}

func TestExportJSONLWritesPrivatePackageAndZip(t *testing.T) {
	tempDir := t.TempDir()
	project := DatasetProject{
		ID:                  uuid.New(),
		Key:                 "itinerary/generation",
		TaskType:            TaskItineraryGeneration,
		SchemaVersion:       SchemaVersion,
		MinimumQualityScore: 0.85,
	}
	example := cleanExample(project.ID)
	train := SplitTrain
	example.Split = &train
	score := 0.97
	example.QualityScore = &score
	manifest := mustTestJSON(BuildManifest(project, "v1", []TrainingExample{example}, AssignSplits([]TrainingExample{example}), 0.85))
	version := DatasetVersion{
		ID:               uuid.New(),
		DatasetProjectID: project.ID,
		Version:          "v1",
		Status:           VersionStatusReady,
		SchemaVersion:    SchemaVersion,
		ManifestJSON:     manifest,
		CreatedAt:        time.Now().UTC(),
	}

	pkg, err := ExportJSONL(tempDir, project, version, []TrainingExample{example}, Config{ExportDir: tempDir})
	if err != nil {
		t.Fatalf("export jsonl: %v", err)
	}
	if strings.Contains(pkg.Path, "/") {
		t.Fatalf("export path should be a safe segment, got %q", pkg.Path)
	}
	exportDir := filepath.Join(tempDir, pkg.Path)
	for _, name := range []string{"train.jsonl", "validation.jsonl", "test.jsonl", "holdout.jsonl", "manifest.json", "checksums.txt", "README.md"} {
		if _, err := os.Stat(filepath.Join(exportDir, name)); err != nil {
			t.Fatalf("expected export file %s: %v", name, err)
		}
	}
	trainFile, err := os.ReadFile(filepath.Join(exportDir, "train.jsonl"))
	if err != nil {
		t.Fatalf("read train jsonl: %v", err)
	}
	if bytes.Contains(trainFile, []byte("traveler@example.com")) || bytes.Contains(trainFile, []byte("tripId")) {
		t.Fatalf("export contains disallowed private fields: %s", trainFile)
	}

	reader, filename, err := OpenExportZip(tempDir, pkg.Path)
	if err != nil {
		t.Fatalf("open export zip: %v", err)
	}
	defer reader.Close()
	if filename != pkg.FileName {
		t.Fatalf("expected zip filename %q, got %q", pkg.FileName, filename)
	}
	zipBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	seen := map[string]bool{}
	for _, file := range zipReader.File {
		seen[file.Name] = true
	}
	for _, name := range []string{"train.jsonl", "validation.jsonl", "test.jsonl", "holdout.jsonl", "manifest.json", "checksums.txt", "README.md"} {
		if !seen[name] {
			t.Fatalf("zip missing %s; entries=%v", name, seen)
		}
	}
}

func cleanExample(projectID uuid.UUID) TrainingExample {
	return TrainingExample{
		ID:                 uuid.New(),
		DatasetProjectID:   projectID,
		SourceType:         "manual_curated",
		TaskType:           TaskItineraryGeneration,
		Language:           "en",
		SchemaVersion:      SchemaVersion,
		InputJSON:          raw(`{"destinationName":"Lisbon","preferences":["food","walkable neighborhoods"]}`),
		GroundingJSON:      raw(`{"places":[{"name":"Time Out Market","groundingId":"places/time-out-market"}],"confidence":0.95}`),
		ExpectedOutputJSON: raw(`{"days":[{"items":[{"name":"Time Out Market"},{"name":"Alfama walk"}]}]}`),
		LabelsJSON:         raw(`{"matchesPreferences":true,"budgetPlausible":true,"userAccepted":true,"groundedPlaceRate":1}`),
		ProvenanceJSON:     raw(`{"licenseStatus":"allowed","copiesProviderText":false}`),
		ConsentStatus:      ConsentGranted,
		SanitizationStatus: SanitizationPassed,
		QualityStatus:      QualityPassed,
		ReviewStatus:       ReviewApproved,
		ExportStatus:       ExportNotExported,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

func raw(value string) json.RawMessage {
	return json.RawMessage([]byte(value))
}

func mustTestJSON(value any) json.RawMessage {
	out, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return out
}

func containsValue(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
