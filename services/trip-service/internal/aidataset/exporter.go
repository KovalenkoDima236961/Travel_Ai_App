package aidataset

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var safeSegmentPattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type ExportPackage struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	SizeBytes int64  `json:"sizeBytes"`
	FileName  string `json:"fileName"`
}

func ExportJSONL(baseDir string, project DatasetProject, version DatasetVersion, examples []TrainingExample, cfg Config) (ExportPackage, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = cfg.ExportDir
	}
	if strings.TrimSpace(baseDir) == "" {
		baseDir = DefaultConfig().ExportDir
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return ExportPackage{}, fmt.Errorf("resolve dataset export dir: %w", err)
	}
	dirName := sanitizeSegment(project.Key) + "-" + sanitizeSegment(version.Version) + "-" + version.ID.String()
	exportDir := filepath.Join(absBase, dirName)
	if err := ensureUnderBase(absBase, exportDir); err != nil {
		return ExportPackage{}, err
	}
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return ExportPackage{}, fmt.Errorf("create dataset export dir: %w", err)
	}

	bySplit := map[string][]TrainingExample{
		SplitTrain:      {},
		SplitValidation: {},
		SplitTest:       {},
		SplitHoldout:    {},
	}
	for _, example := range examples {
		split := SplitTrain
		if example.Split != nil && *example.Split != "" {
			split = *example.Split
		}
		bySplit[split] = append(bySplit[split], example)
	}

	checksums := make(map[string]string)
	for _, split := range []string{SplitTrain, SplitValidation, SplitTest, SplitHoldout} {
		name := split + ".jsonl"
		sum, err := writeJSONL(filepath.Join(exportDir, name), bySplit[split])
		if err != nil {
			return ExportPackage{}, err
		}
		checksums[name] = sum
	}
	manifestName := "manifest.json"
	manifestChecksum, err := writePrettyJSON(filepath.Join(exportDir, manifestName), version.ManifestJSON)
	if err != nil {
		return ExportPackage{}, err
	}
	checksums[manifestName] = manifestChecksum
	readmeChecksum, err := writeReadme(filepath.Join(exportDir, "README.md"), project, version)
	if err != nil {
		return ExportPackage{}, err
	}
	checksums["README.md"] = readmeChecksum
	checksumFileChecksum, err := writeChecksums(filepath.Join(exportDir, "checksums.txt"), checksums)
	if err != nil {
		return ExportPackage{}, err
	}
	checksums["checksums.txt"] = checksumFileChecksum

	overall := sha256.New()
	names := make([]string, 0, len(checksums))
	for name := range checksums {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		_, _ = overall.Write([]byte(name + " " + checksums[name] + "\n"))
	}
	size, err := dirSize(exportDir)
	if err != nil {
		return ExportPackage{}, err
	}
	return ExportPackage{
		Path:      dirName,
		Checksum:  hex.EncodeToString(overall.Sum(nil)),
		SizeBytes: size,
		FileName:  dirName + ".zip",
	}, nil
}

func OpenExportZip(baseDir, exportPath string) (io.ReadCloser, string, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve dataset export dir: %w", err)
	}
	exportDir := filepath.Join(absBase, filepath.Clean(filepath.FromSlash(exportPath)))
	if err := ensureUnderBase(absBase, exportDir); err != nil {
		return nil, "", err
	}
	info, err := os.Stat(exportDir)
	if err != nil {
		return nil, "", fmt.Errorf("open dataset export: %w", err)
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("dataset export is not a directory")
	}
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	files := []string{"train.jsonl", "validation.jsonl", "test.jsonl", "holdout.jsonl", "manifest.json", "checksums.txt", "README.md"}
	for _, name := range files {
		path := filepath.Join(exportDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read dataset export file: %w", err)
		}
		file, err := zipWriter.Create(name)
		if err != nil {
			return nil, "", fmt.Errorf("create dataset zip entry: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return nil, "", fmt.Errorf("write dataset zip entry: %w", err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, "", fmt.Errorf("finish dataset zip: %w", err)
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes())), filepath.Base(exportDir) + ".zip", nil
}

func writeJSONL(path string, examples []TrainingExample) (string, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("create jsonl export: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	writer := bufio.NewWriter(io.MultiWriter(file, hash))
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	for _, example := range examples {
		line := map[string]any{
			"id":            example.ID.String(),
			"task":          example.TaskType,
			"language":      example.Language,
			"schemaVersion": example.SchemaVersion,
			"input":         rawJSONValue(example.InputJSON),
			"grounding":     rawJSONValue(example.GroundingJSON),
			"output":        rawJSONValue(example.ExpectedOutputJSON),
			"labels":        rawJSONValue(example.LabelsJSON),
			"metadata": map[string]any{
				"sourceType":   example.SourceType,
				"qualityScore": example.QualityScore,
			},
		}
		if err := encoder.Encode(line); err != nil {
			return "", fmt.Errorf("write jsonl line: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("flush jsonl export: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writePrettyJSON(path string, raw json.RawMessage) (string, error) {
	var value any
	if len(bytes.TrimSpace(raw)) == 0 {
		value = map[string]any{}
	} else if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode manifest json: %w", err)
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode manifest json: %w", err)
	}
	content = append(content, '\n')
	return writeFileWithChecksum(path, content)
}

func writeReadme(path string, project DatasetProject, version DatasetVersion) (string, error) {
	content := fmt.Sprintf(`# AI Dataset Export

Project: %s
Version: %s
Created: %s

This private export contains only reviewed, sanitized, consent-valid examples.
It does not include raw prompts, hidden system instructions, user identifiers,
trip identifiers, comments, receipts/OCR, calendar details, private notes,
tokens, or raw provider payloads.

RAG/provider verification remains the factual source for travel knowledge.
Provider facts should be referenced through grounding IDs unless training use of
copied provider text is explicitly licensed.
`, project.Key, version.Version, time.Now().UTC().Format(time.RFC3339))
	return writeFileWithChecksum(path, []byte(content))
}

func writeChecksums(path string, checksums map[string]string) (string, error) {
	names := make([]string, 0, len(checksums))
	for name := range checksums {
		names = append(names, name)
	}
	sort.Strings(names)
	var builder strings.Builder
	for _, name := range names {
		builder.WriteString(checksums[name])
		builder.WriteString("  ")
		builder.WriteString(name)
		builder.WriteByte('\n')
	}
	return writeFileWithChecksum(path, []byte(builder.String()))
}

func writeFileWithChecksum(path string, content []byte) (string, error) {
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write dataset export file: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func rawJSONValue(raw json.RawMessage) any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	return value
}

func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func sanitizeSegment(value string) string {
	value = strings.Trim(safeSegmentPattern.ReplaceAllString(strings.TrimSpace(value), "-"), "-")
	if value == "" {
		return "dataset"
	}
	if len(value) > 80 {
		return value[:80]
	}
	return value
}

func ensureUnderBase(base, path string) error {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return fmt.Errorf("resolve dataset export path: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("invalid dataset export path")
	}
	return nil
}
