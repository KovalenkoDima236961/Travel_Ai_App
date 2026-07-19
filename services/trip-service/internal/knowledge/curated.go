package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CuratedDataset is the validation and idempotent-ingestion boundary for the
// repository-owned JSON/Markdown corpus.
type CuratedDataset struct {
	Sources      map[string]Source
	Destinations []DestinationKnowledge
	Documents    map[string]KnowledgeDocument
}

// LoadCurated reads every supported curated file, validates it, and derives
// stable document checksums. It has no database or network side effects, so a
// worker can use it safely for dry-run and CI validation.
func LoadCurated(dataDir, destinationFilter string) (*CuratedDataset, error) {
	dataDir = strings.TrimSpace(dataDir)
	if dataDir == "" {
		return nil, fmt.Errorf("knowledge data directory is required")
	}
	sources, err := loadSources(filepath.Join(dataDir, "sources.json"))
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(dataDir, "destinations", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list destination files: %w", err)
	}
	sort.Strings(paths)
	dataset := &CuratedDataset{Sources: sources, Documents: map[string]KnowledgeDocument{}}
	filter := NormalizeName(destinationFilter)
	for _, path := range paths {
		var destination DestinationKnowledge
		if err := decodeJSONFile(path, &destination); err != nil {
			return nil, err
		}
		if filter != "" && NormalizeName(destination.CanonicalName) != filter {
			continue
		}
		if err := destination.NormalizeAndValidate(sources); err != nil {
			return nil, fmt.Errorf("validate %s: %w", filepath.Base(path), err)
		}
		dataset.Destinations = append(dataset.Destinations, destination)
		document, err := loadDestinationDocument(filepath.Join(dataDir, "documents"), destination)
		if err != nil {
			return nil, err
		}
		dataset.Documents[NormalizeName(destination.CanonicalName)] = document
	}
	if len(dataset.Destinations) == 0 {
		return nil, fmt.Errorf("no curated destinations matched %q", destinationFilter)
	}
	return dataset, nil
}

func loadSources(path string) (map[string]Source, error) {
	var sources []Source
	if err := decodeJSONFile(path, &sources); err != nil {
		return nil, err
	}
	result := make(map[string]Source, len(sources))
	for _, source := range sources {
		source.SourceKey = strings.TrimSpace(source.SourceKey)
		source.SourceType = strings.TrimSpace(source.SourceType)
		source.DisplayName = strings.TrimSpace(source.DisplayName)
		source.TrustLevel = strings.TrimSpace(source.TrustLevel)
		if source.SourceKey == "" || source.SourceType == "" || source.DisplayName == "" || source.TrustLevel == "" {
			return nil, fmt.Errorf("source metadata is incomplete")
		}
		if _, exists := result[source.SourceKey]; exists {
			return nil, fmt.Errorf("duplicate source key %q", source.SourceKey)
		}
		result[source.SourceKey] = source
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one knowledge source is required")
	}
	return result, nil
}

func loadDestinationDocument(directory string, destination DestinationKnowledge) (KnowledgeDocument, error) {
	name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(destination.CanonicalName), " ", "-"))
	path := filepath.Join(directory, name+".en.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return KnowledgeDocument{}, fmt.Errorf("read %s: %w", path, err)
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return KnowledgeDocument{}, fmt.Errorf("document %s is empty", path)
	}
	return KnowledgeDocument{
		Title:       destination.CanonicalName + " planning notes",
		Content:     trimmed,
		ContentType: "markdown",
		Language:    "en",
		SourceKey:   SourceTypeManualCurated,
		Checksum:    Checksum(trimmed),
	}, nil
}

func decodeJSONFile(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
