package aidataset

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type SplitAssignment struct {
	ExampleID        uuid.UUID `json:"exampleId"`
	DuplicateGroupID uuid.UUID `json:"duplicateGroupId"`
	Split            string    `json:"split"`
	SourceGroup      string    `json:"sourceGroup"`
}

type DatasetManifest struct {
	SchemaVersion        string             `json:"schemaVersion"`
	ProjectID            string             `json:"projectId"`
	Version              string             `json:"version"`
	ExampleCount         int                `json:"exampleCount"`
	SplitCounts          map[string]int     `json:"splitCounts"`
	TaskDistribution     map[string]int     `json:"taskDistribution"`
	LanguageDistribution map[string]int     `json:"languageDistribution"`
	SourceDistribution   map[string]int     `json:"sourceDistribution"`
	ConsentCounts        map[string]int     `json:"consentCounts"`
	QualityScore         map[string]float64 `json:"qualityScore"`
	DuplicateGroups      int                `json:"duplicateGroups"`
	SourceGroups         []string           `json:"sourceGroups"`
	ExampleIDs           []string           `json:"exampleIds"`
	Assignments          []SplitAssignment  `json:"assignments"`
	MinimumQualityScore  float64            `json:"minimumQualityScore"`
	GeneratedBy          string             `json:"generatedBy"`
	Notes                []string           `json:"notes"`
}

func DuplicateGroupID(example TrainingExample) uuid.UUID {
	parts := []string{example.TaskType, strings.ToLower(example.Language), example.SourceType}
	if example.TripID != nil {
		parts = append(parts, "trip:"+example.TripID.String())
	}
	if example.SourceEntityType != nil {
		parts = append(parts, "entityType:"+*example.SourceEntityType)
	}
	if example.SourceEntityID != nil {
		parts = append(parts, "entityID:"+*example.SourceEntityID)
	}
	if example.TripID == nil && example.SourceEntityID == nil {
		input, _ := normalizeJSON(example.InputJSON)
		output, _ := normalizeJSON(example.ExpectedOutputJSON)
		parts = append(parts, string(input), string(output))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return uuidFromSHA(sum)
}

func AssignSplits(examples []TrainingExample) []SplitAssignment {
	groupSplit := map[string]string{}
	assignments := make([]SplitAssignment, 0, len(examples))
	for _, example := range examples {
		groupID := DuplicateGroupID(example)
		if example.DuplicateGroupID != nil {
			groupID = *example.DuplicateGroupID
		}
		groupKey := sourceGroupKey(example, groupID)
		split, ok := groupSplit[groupKey]
		if !ok {
			split = splitForExample(example, groupKey)
			groupSplit[groupKey] = split
		}
		assignments = append(assignments, SplitAssignment{
			ExampleID:        example.ID,
			DuplicateGroupID: groupID,
			Split:            split,
			SourceGroup:      groupKey,
		})
	}
	return assignments
}

func BuildManifest(project DatasetProject, version string, examples []TrainingExample, assignments []SplitAssignment, minimumQualityScore float64) DatasetManifest {
	splitByExample := map[uuid.UUID]string{}
	groups := map[string]struct{}{}
	for _, assignment := range assignments {
		splitByExample[assignment.ExampleID] = assignment.Split
		groups[assignment.SourceGroup] = struct{}{}
	}
	manifest := DatasetManifest{
		SchemaVersion:        SchemaVersion,
		ProjectID:            project.ID.String(),
		Version:              version,
		SplitCounts:          map[string]int{SplitTrain: 0, SplitValidation: 0, SplitTest: 0, SplitHoldout: 0},
		TaskDistribution:     map[string]int{},
		LanguageDistribution: map[string]int{},
		SourceDistribution:   map[string]int{},
		ConsentCounts:        map[string]int{},
		QualityScore:         map[string]float64{},
		ExampleIDs:           make([]string, 0, len(examples)),
		Assignments:          assignments,
		MinimumQualityScore:  minimumQualityScore,
		GeneratedBy:          "trip-service-ai-dataset-v1",
		Notes: []string{
			"Only reviewed, sanitized, consent-valid examples are eligible.",
			"RAG/provider IDs remain factual references; provider text is not copied unless license permits it.",
			"Holdout examples are excluded from future training use.",
		},
	}
	var sum float64
	var min, max float64
	for i, example := range examples {
		split := splitByExample[example.ID]
		if split == "" {
			split = SplitTrain
		}
		manifest.ExampleIDs = append(manifest.ExampleIDs, example.ID.String())
		manifest.SplitCounts[split]++
		manifest.TaskDistribution[example.TaskType]++
		manifest.LanguageDistribution[example.Language]++
		manifest.SourceDistribution[example.SourceType]++
		manifest.ConsentCounts[example.ConsentStatus]++
		if example.QualityScore != nil {
			score := *example.QualityScore
			sum += score
			if i == 0 || score < min {
				min = score
			}
			if i == 0 || score > max {
				max = score
			}
		}
	}
	manifest.ExampleCount = len(examples)
	manifest.DuplicateGroups = len(groups)
	manifest.SourceGroups = mapKeys(groups)
	sort.Strings(manifest.SourceGroups)
	if len(examples) > 0 {
		manifest.QualityScore["average"] = roundScore(sum / float64(len(examples)))
		manifest.QualityScore["min"] = roundScore(min)
		manifest.QualityScore["max"] = roundScore(max)
	}
	return manifest
}

func splitForExample(example TrainingExample, groupKey string) string {
	if example.SourceType == "golden_case" && labelString(example, "benchmarkSplit") == SplitHoldout {
		return SplitHoldout
	}
	if labelString(example, "split") == SplitHoldout {
		return SplitHoldout
	}
	sum := sha256.Sum256([]byte(groupKey))
	bucket := int(binary.BigEndian.Uint64(sum[:8]) % 100)
	switch {
	case bucket < 60:
		return SplitTrain
	case bucket < 80:
		return SplitValidation
	case bucket < 90:
		return SplitTest
	default:
		return SplitHoldout
	}
}

func sourceGroupKey(example TrainingExample, groupID uuid.UUID) string {
	if example.TripID != nil {
		return "trip:" + example.TripID.String()
	}
	if example.SourceEntityType != nil && example.SourceEntityID != nil {
		return *example.SourceEntityType + ":" + *example.SourceEntityID
	}
	return "duplicate:" + groupID.String()
}

func uuidFromSHA(sum [32]byte) uuid.UUID {
	var raw [16]byte
	copy(raw[:], sum[:16])
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return uuid.UUID(raw)
}

func labelString(example TrainingExample, key string) string {
	value, _ := jsonMap(example.LabelsJSON)[key].(string)
	return strings.TrimSpace(value)
}

func mapKeys(in map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	return out
}

func manifestChecksum(manifest DatasetManifest, examples []TrainingExample) (string, error) {
	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write(manifestRaw)
	for _, example := range examples {
		normalizedInput, err := normalizeJSON(example.InputJSON)
		if err != nil {
			return "", err
		}
		normalizedOutput, err := normalizeJSON(example.ExpectedOutputJSON)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(example.ID.String()))
		_, _ = hash.Write(normalizedInput)
		_, _ = hash.Write(normalizedOutput)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func manifestAssignments(raw json.RawMessage) []SplitAssignment {
	var manifest DatasetManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil
	}
	if len(manifest.Assignments) > 0 {
		return manifest.Assignments
	}
	assignments := make([]SplitAssignment, 0, len(manifest.ExampleIDs))
	for _, rawID := range manifest.ExampleIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil
		}
		assignments = append(assignments, SplitAssignment{ExampleID: id, Split: SplitTrain})
	}
	return assignments
}
