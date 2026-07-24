package aidataset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

type Service struct {
	repo *Repository
	cfg  Config
	log  *zap.Logger
}

func NewService(repo *Repository, cfg Config, log *zap.Logger) *Service {
	defaults := DefaultConfig()
	if strings.TrimSpace(cfg.ExportDir) == "" {
		cfg.ExportDir = defaults.ExportDir
	}
	if cfg.ExportRetentionDays <= 0 {
		cfg.ExportRetentionDays = defaults.ExportRetentionDays
	}
	if cfg.MinAutoReviewScore <= 0 {
		cfg.MinAutoReviewScore = defaults.MinAutoReviewScore
	}
	if cfg.MinApprovalScore <= 0 {
		cfg.MinApprovalScore = defaults.MinApprovalScore
	}
	if cfg.MaxDuplicateSimilarity <= 0 {
		cfg.MaxDuplicateSimilarity = defaults.MaxDuplicateSimilarity
	}
	if strings.TrimSpace(cfg.GoldenCasesDir) == "" {
		cfg.GoldenCasesDir = defaults.GoldenCasesDir
	}
	if strings.TrimSpace(cfg.ManualExamplesDir) == "" {
		cfg.ManualExamplesDir = defaults.ManualExamplesDir
	}
	if strings.TrimSpace(cfg.TrainingInstructionText) == "" {
		cfg.TrainingInstructionText = defaults.TrainingInstructionText
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, cfg: cfg, log: log}
}

func (s *Service) Config() Config {
	return s.cfg
}

func (s *Service) ListProjects(ctx context.Context) ([]DatasetProject, error) {
	return s.repo.ListProjects(ctx)
}

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (*DatasetProject, error) {
	user, _ := auth.UserFromContext(ctx)
	if strings.TrimSpace(input.Key) == "" {
		return nil, fmt.Errorf("project key is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if !IsAllowedTaskType(input.TaskType) {
		return nil, fmt.Errorf("unsupported task type")
	}
	if input.SchemaVersion == "" {
		input.SchemaVersion = SchemaVersion
	}
	if input.MinimumQualityScore <= 0 {
		input.MinimumQualityScore = 0.8
	}
	var actor *uuid.UUID
	if user.ID != uuid.Nil {
		actor = &user.ID
	}
	return s.repo.CreateProject(ctx, input, actor)
}

func (s *Service) GetConsent(ctx context.Context, scopeType string, scopeID *string) (*TrainingConsentResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	scopeType = normalizeScopeType(scopeType)
	consent, err := s.repo.LatestConsent(ctx, user.ID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	status := "not_granted"
	granted := false
	if consent != nil {
		status = consent.Status
		granted = consent.Status == ConsentGranted
	}
	return &TrainingConsentResponse{
		Status:         status,
		Granted:        granted,
		ConsentVersion: ConsentVersionV1,
		Record:         consent,
		ExcludedData:   excludedDataList(),
	}, nil
}

func (s *Service) SetConsent(ctx context.Context, scopeType string, scopeID *string, granted bool) (*TrainingConsentResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	scopeType = normalizeScopeType(scopeType)
	now := time.Now().UTC()
	status := ConsentRevoked
	var grantedAt, revokedAt *time.Time
	if granted {
		status = ConsentGranted
		grantedAt = &now
	} else {
		revokedAt = &now
	}
	consent, err := s.repo.CreateConsent(ctx, TrainingConsent{
		ID:             uuid.New(),
		UserID:         user.ID,
		ScopeType:      scopeType,
		ScopeID:        scopeID,
		ConsentVersion: ConsentVersionV1,
		Status:         status,
		GrantedAt:      grantedAt,
		RevokedAt:      revokedAt,
	})
	if err != nil {
		return nil, err
	}
	if !granted {
		count, revokeErr := s.repo.MarkConsentRevoked(ctx, user.ID, scopeType, scopeID)
		if revokeErr != nil {
			return nil, revokeErr
		}
		if count > 0 {
			actor := user.ID
			_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
				ID:          uuid.New(),
				ActorUserID: &actor,
				Action:      ReviewActionConsentRevoked,
				Reason:      stringPtr("training consent revoked"),
				Metadata:    mustJSON(map[string]any{"affectedExamples": count, "scopeType": scopeType, "scopeId": scopeID}),
			})
		}
	}
	return &TrainingConsentResponse{
		Status:         consent.Status,
		Granted:        consent.Status == ConsentGranted,
		ConsentVersion: ConsentVersionV1,
		Record:         consent,
		ExcludedData:   excludedDataList(),
	}, nil
}

func (s *Service) ListExamples(ctx context.Context, filters ExampleFilters) ([]TrainingExample, error) {
	return s.repo.ListExamples(ctx, filters)
}

func (s *Service) GetExample(ctx context.Context, id uuid.UUID) (*TrainingExample, error) {
	return s.repo.GetExample(ctx, id)
}

func (s *Service) CreateCandidate(ctx context.Context, input CreateExampleInput) (*TrainingExample, error) {
	project, err := s.repo.GetProject(ctx, input.DatasetProjectID)
	if err != nil {
		return nil, err
	}
	if input.TaskType == "" {
		input.TaskType = project.TaskType
	}
	if !IsAllowedTaskType(input.TaskType) {
		return nil, fmt.Errorf("unsupported task type")
	}
	if input.ConsentStatus == "" {
		if input.UserID == nil {
			input.ConsentStatus = ConsentNotRequired
		} else {
			input.ConsentStatus = ConsentPending
		}
	}
	example, err := s.repo.CreateTrainingExample(ctx, input)
	if err != nil {
		return nil, err
	}
	actor, _ := auth.UserFromContext(ctx)
	actorID := nullableUserID(actor)
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:                uuid.New(),
		TrainingExampleID: &example.ID,
		ActorUserID:       actorID,
		Action:            ReviewActionCandidateCreated,
		NewStatus:         &example.ReviewStatus,
		Metadata:          mustJSON(map[string]any{"sourceType": example.SourceType, "taskType": example.TaskType}),
	})
	example, err = s.ResanitizeExample(ctx, example.ID)
	if err != nil {
		return nil, err
	}
	return s.RescoreExample(ctx, example.ID)
}

func (s *Service) ResanitizeExample(ctx context.Context, id uuid.UUID) (*TrainingExample, error) {
	example, err := s.repo.GetExample(ctx, id)
	if err != nil {
		return nil, err
	}
	result, err := SanitizeExample(example.InputJSON, example.GroundingJSON, example.ExpectedOutputJSON)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateExampleSanitization(ctx, id, result)
	if err != nil {
		return nil, err
	}
	action := ReviewActionSanitizationPassed
	if result.Status == SanitizationFailed {
		action = ReviewActionSanitizationFailed
	}
	actor, _ := auth.UserFromContext(ctx)
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:                uuid.New(),
		TrainingExampleID: &id,
		ActorUserID:       nullableUserID(actor),
		Action:            action,
		OldStatus:         &example.SanitizationStatus,
		NewStatus:         &updated.SanitizationStatus,
		Metadata:          SanitizationMetadata(result),
	})
	return updated, nil
}

func (s *Service) RescoreExample(ctx context.Context, id uuid.UUID) (*TrainingExample, error) {
	example, err := s.repo.GetExample(ctx, id)
	if err != nil {
		return nil, err
	}
	project, err := s.repo.GetProject(ctx, example.DatasetProjectID)
	if err != nil {
		return nil, err
	}
	groupID := DuplicateGroupID(*example)
	result := ScoreExample(*example, *project, s.cfg)
	updated, err := s.repo.UpdateExampleQuality(ctx, id, result, groupID)
	if err != nil {
		return nil, err
	}
	actor, _ := auth.UserFromContext(ctx)
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:                uuid.New(),
		TrainingExampleID: &id,
		ActorUserID:       nullableUserID(actor),
		Action:            ReviewActionQualityScored,
		OldStatus:         &example.QualityStatus,
		NewStatus:         &updated.QualityStatus,
		Metadata:          mustJSON(result),
	})
	return updated, nil
}

func (s *Service) ReviewExample(ctx context.Context, id uuid.UUID, input ReviewInput) (*TrainingExample, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(input.Status)
	switch status {
	case ReviewApproved:
		return s.ApproveExample(ctx, id, input.Reason)
	case ReviewRejected, ReviewNeedsChanges, ReviewPending:
	default:
		return nil, ErrInvalidReviewStatus
	}
	current, err := s.repo.GetExample(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateExampleReview(ctx, id, status, user.ID, input.Reason)
	if err != nil {
		return nil, err
	}
	action := ReviewActionRejected
	if status == ReviewPending {
		action = ReviewActionCandidateCreated
	}
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:                uuid.New(),
		TrainingExampleID: &id,
		ActorUserID:       &user.ID,
		Action:            action,
		OldStatus:         &current.ReviewStatus,
		NewStatus:         &updated.ReviewStatus,
		Reason:            stringPtr(input.Reason),
	})
	return updated, nil
}

func (s *Service) ApproveExample(ctx context.Context, id uuid.UUID, reason string) (*TrainingExample, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	example, err := s.repo.GetExample(ctx, id)
	if err != nil {
		return nil, err
	}
	project, err := s.repo.GetProject(ctx, example.DatasetProjectID)
	if err != nil {
		return nil, err
	}
	if example.SanitizationStatus != SanitizationPassed {
		if example.SanitizationStatus == SanitizationFailed {
			return nil, ErrSanitizationFailed
		}
		return nil, ErrPrivateDataDetected
	}
	if project.ConsentRequired && example.ConsentStatus != ConsentNotRequired && example.ConsentStatus != ConsentGranted {
		if example.ConsentStatus == ConsentRevoked {
			return nil, ErrConsentRevoked
		}
		return nil, ErrConsentRequired
	}
	if example.QualityScore == nil || example.QualityStatus == QualityPending {
		example, err = s.RescoreExample(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	if example.QualityStatus == QualityFailed || example.QualityScore == nil || *example.QualityScore < project.MinimumQualityScore || *example.QualityScore < s.cfg.MinApprovalScore {
		return nil, ErrQualityTooLow
	}
	if licenseBlocked(*example) {
		return nil, ErrLicenseNotAllowed
	}
	updated, err := s.repo.UpdateExampleReview(ctx, id, ReviewApproved, user.ID, reason)
	if err != nil {
		return nil, err
	}
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:                uuid.New(),
		TrainingExampleID: &id,
		ActorUserID:       &user.ID,
		Action:            ReviewActionApproved,
		OldStatus:         &example.ReviewStatus,
		NewStatus:         &updated.ReviewStatus,
		Reason:            stringPtr(reason),
	})
	return updated, nil
}

func (s *Service) RejectExample(ctx context.Context, id uuid.UUID, reason string) (*TrainingExample, error) {
	return s.ReviewExample(ctx, id, ReviewInput{Status: ReviewRejected, Reason: reason})
}

func (s *Service) BuildVersion(ctx context.Context, input BuildVersionInput) (*DatasetVersion, error) {
	user, _ := auth.UserFromContext(ctx)
	project, err := s.repo.GetProject(ctx, input.DatasetProjectID)
	if err != nil {
		return nil, err
	}
	minimum := input.MinimumQualityScore
	if minimum <= 0 {
		minimum = project.MinimumQualityScore
	}
	var actor *uuid.UUID
	if user.ID != uuid.Nil {
		actor = &user.ID
	}
	version, err := s.repo.CreateDatasetVersion(ctx, project.ID, input.Version, project.SchemaVersion, actor)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, ErrVersionExists
		}
		return nil, err
	}
	examples, err := s.repo.ListApprovedExamples(ctx, project.ID, minimum, input.Languages, input.TaskTypes)
	if err != nil {
		return nil, err
	}
	examples = keepBestByDuplicateGroup(examples)
	if len(examples) == 0 {
		return nil, ErrNoEligibleExamples
	}
	assignments := AssignSplits(examples)
	if err := s.repo.AssignExampleSplits(ctx, assignments); err != nil {
		return nil, err
	}
	for i := range examples {
		for _, assignment := range assignments {
			if examples[i].ID == assignment.ExampleID {
				examples[i].DuplicateGroupID = &assignment.DuplicateGroupID
				examples[i].Split = &assignment.Split
				break
			}
		}
	}
	manifest := BuildManifest(*project, input.Version, examples, assignments, minimum)
	checksum, err := manifestChecksum(manifest, examples)
	if err != nil {
		return nil, err
	}
	ready, err := s.repo.MarkDatasetVersionReady(ctx, version.ID, manifest, checksum)
	if err != nil {
		return nil, err
	}
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:               uuid.New(),
		DatasetVersionID: &ready.ID,
		ActorUserID:      actor,
		Action:           ReviewActionSplitAssigned,
		NewStatus:        &ready.Status,
		Metadata:         mustJSON(manifest),
	})
	return ready, nil
}

func (s *Service) ExportVersion(ctx context.Context, versionID uuid.UUID) (*DatasetVersion, error) {
	if !s.cfg.ExportEnabled {
		return nil, ErrExportDisabled
	}
	user, _ := auth.UserFromContext(ctx)
	version, err := s.repo.GetDatasetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if version.Status != VersionStatusReady && version.Status != VersionStatusExported {
		return nil, ErrVersionNotReady
	}
	project, err := s.repo.GetProject(ctx, version.DatasetProjectID)
	if err != nil {
		return nil, err
	}
	minimum := project.MinimumQualityScore
	assignments := manifestAssignments(version.ManifestJSON)
	examples, err := s.examplesForVersion(ctx, *version, assignments, minimum)
	if err != nil {
		return nil, err
	}
	if len(examples) == 0 {
		return nil, ErrNoEligibleExamples
	}
	pkg, err := ExportJSONL(s.cfg.ExportDir, *project, *version, examples, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExportFailed, err)
	}
	exported, err := s.repo.MarkDatasetVersionExported(ctx, version.ID, pkg.Path, pkg.Checksum)
	if err != nil {
		return nil, err
	}
	actor := nullableUserID(user)
	_ = s.repo.InsertReviewEvent(ctx, ReviewEvent{
		ID:               uuid.New(),
		DatasetVersionID: &version.ID,
		ActorUserID:      actor,
		Action:           ReviewActionExported,
		OldStatus:        &version.Status,
		NewStatus:        &exported.Status,
		Metadata:         mustJSON(pkg),
	})
	return exported, nil
}

func (s *Service) examplesForVersion(ctx context.Context, version DatasetVersion, assignments []SplitAssignment, minimum float64) ([]TrainingExample, error) {
	if len(assignments) == 0 {
		examples, err := s.repo.ListApprovedExamples(ctx, version.DatasetProjectID, minimum, nil, nil)
		if err != nil {
			return nil, err
		}
		return keepBestByDuplicateGroup(examples), nil
	}
	ids := make([]uuid.UUID, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ExampleID)
	}
	examples, err := s.repo.ListExamplesByIDs(ctx, version.DatasetProjectID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]TrainingExample, len(examples))
	for _, example := range examples {
		byID[example.ID] = example
	}
	out := make([]TrainingExample, 0, len(assignments))
	for _, assignment := range assignments {
		example, ok := byID[assignment.ExampleID]
		if !ok {
			return nil, ErrVersionNotReady
		}
		if err := ensureExportEligible(example, minimum); err != nil {
			return nil, err
		}
		example.DuplicateGroupID = &assignment.DuplicateGroupID
		example.Split = &assignment.Split
		out = append(out, example)
	}
	return out, nil
}

func ensureExportEligible(example TrainingExample, minimum float64) error {
	if example.ReviewStatus != ReviewApproved || example.QualityStatus == QualityFailed || example.QualityScore == nil || *example.QualityScore < minimum {
		return ErrVersionNotReady
	}
	if example.SanitizationStatus != SanitizationPassed {
		return ErrSanitizationFailed
	}
	switch example.ConsentStatus {
	case ConsentNotRequired, ConsentGranted:
	default:
		if example.ConsentStatus == ConsentRevoked || example.ConsentStatus == ConsentProhibited {
			return ErrConsentRevoked
		}
		return ErrConsentRequired
	}
	if licenseBlocked(example) {
		return ErrLicenseNotAllowed
	}
	return nil
}

func (s *Service) GetExportStatus(ctx context.Context, versionID uuid.UUID) (*ExportStatus, error) {
	version, err := s.repo.GetDatasetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	return &ExportStatus{VersionID: version.ID, Status: version.Status, ExportPath: version.ExportPath, Checksum: version.Checksum}, nil
}

func (s *Service) OpenExport(ctx context.Context, versionID uuid.UUID) (io.ReadCloser, string, error) {
	version, err := s.repo.GetDatasetVersion(ctx, versionID)
	if err != nil {
		return nil, "", err
	}
	if version.Status != VersionStatusExported || version.ExportPath == nil {
		return nil, "", ErrVersionNotReady
	}
	return OpenExportZip(s.cfg.ExportDir, *version.ExportPath)
}

func (s *Service) Readiness(ctx context.Context) (FineTuningReadiness, error) {
	data, err := s.repo.ReadinessData(ctx)
	if err != nil {
		return FineTuningReadiness{}, err
	}
	blockers := make([]string, 0)
	if data.ApprovedCount < 500 {
		blockers = append(blockers, "fewer than 500 approved high-quality examples")
	}
	if data.SanitizationFailureCount > 0 {
		blockers = append(blockers, "unresolved sanitization failures")
	}
	if data.HoldoutCount == 0 {
		blockers = append(blockers, "no frozen holdout examples")
	}
	if data.DuplicateCount > 0 {
		blockers = append(blockers, "duplicate groups require review")
	}
	recommendation := "Keep curating sanitized, reviewed examples and run the stable benchmark before any training."
	if len(blockers) == 0 {
		recommendation = "Dataset lifecycle is ready for a narrow controlled experiment, after a fresh baseline evaluation and rollback plan review."
	}
	return FineTuningReadiness{
		Ready:                    len(blockers) == 0,
		Blockers:                 blockers,
		ApprovedExampleCount:     data.ApprovedCount,
		TaskDistribution:         data.TaskDistribution,
		LanguageDistribution:     data.LanguageDistribution,
		ConsentCoverage:          data.ConsentCoverage,
		SanitizationFailureCount: data.SanitizationFailureCount,
		DuplicateCount:           data.DuplicateCount,
		HoldoutCount:             data.HoldoutCount,
		BaselineEvalStatus:       "required_before_training",
		Recommendation:           recommendation,
	}, nil
}

func (s *Service) ImportGoldenCases(ctx context.Context, projectID uuid.UUID) (int, error) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(s.cfg.GoldenCasesDir)
	if err != nil {
		return 0, fmt.Errorf("read golden cases: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.cfg.GoldenCasesDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return count, err
		}
		var payload struct {
			ID                string          `json:"id"`
			Input             json.RawMessage `json:"input"`
			Constraints       json.RawMessage `json:"constraints"`
			ExpectedQualities json.RawMessage `json:"expectedQualities"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return count, fmt.Errorf("decode golden case %s: %w", entry.Name(), err)
		}
		sourceID := payload.ID
		if sourceID == "" {
			sourceID = strings.TrimSuffix(entry.Name(), ".json")
		}
		existing, err := s.repo.FindExampleBySource(ctx, project.ID, "golden_case", sourceID)
		if err != nil {
			return count, err
		}
		if existing != nil {
			continue
		}
		input := mustJSON(map[string]any{
			"request":     rawJSONValue(payload.Input),
			"constraints": rawJSONValue(payload.Constraints),
		})
		expected := mustJSON(map[string]any{
			"expectedQualities": rawJSONValue(payload.ExpectedQualities),
		})
		entityType := "ai_itinerary_eval_case"
		labels := mustJSON(map[string]any{
			"benchmarkSplit":     SplitHoldout,
			"matchesPreferences": true,
			"budgetPlausible":    true,
			"groundedPlaceRate":  1,
			"schemaOnly":         true,
		})
		provenance := mustJSON(map[string]any{
			"source":        "golden_case",
			"sourcePath":    path,
			"licenseStatus": "project_owned",
			"reviewed":      true,
		})
		grounding := mustJSON(map[string]any{
			"facts": []map[string]any{{"source": "golden_case", "caseId": sourceID}},
		})
		_, err = s.CreateCandidate(ctx, CreateExampleInput{
			DatasetProjectID:   project.ID,
			SourceType:         "golden_case",
			SourceEntityType:   &entityType,
			SourceEntityID:     &sourceID,
			TaskType:           project.TaskType,
			Language:           "en",
			SchemaVersion:      SchemaVersion,
			InputJSON:          input,
			GroundingJSON:      grounding,
			ExpectedOutputJSON: expected,
			LabelsJSON:         labels,
			ProvenanceJSON:     provenance,
			ConsentStatus:      ConsentNotRequired,
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Service) ImportManualExamples(ctx context.Context, projectID uuid.UUID) (int, error) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	count := 0
	root := filepath.Join(s.cfg.ManualExamplesDir, strings.ReplaceAll(project.TaskType, "_", "-"))
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var payload struct {
			ID             string          `json:"id"`
			TaskType       string          `json:"taskType"`
			Language       string          `json:"language"`
			Input          json.RawMessage `json:"input"`
			Grounding      json.RawMessage `json:"grounding"`
			Output         json.RawMessage `json:"output"`
			NegativeOutput json.RawMessage `json:"negativeOutput"`
			Labels         json.RawMessage `json:"labels"`
			Provenance     json.RawMessage `json:"provenance"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("decode manual example %s: %w", path, err)
		}
		sourceID := payload.ID
		if sourceID == "" {
			sourceID = strings.TrimSuffix(entry.Name(), ".json")
		}
		existing, err := s.repo.FindExampleBySource(ctx, project.ID, "manual_curated", sourceID)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}
		taskType := payload.TaskType
		if taskType == "" {
			taskType = project.TaskType
		}
		language := payload.Language
		if language == "" {
			language = "en"
		}
		entityType := "manual_example"
		provenance := mergeJSONObjects(payload.Provenance, map[string]any{
			"source":        "manual_curated",
			"sourcePath":    path,
			"licenseStatus": "project_owned",
		})
		_, err = s.CreateCandidate(ctx, CreateExampleInput{
			DatasetProjectID:   project.ID,
			SourceType:         "manual_curated",
			SourceEntityType:   &entityType,
			SourceEntityID:     &sourceID,
			TaskType:           taskType,
			Language:           language,
			SchemaVersion:      SchemaVersion,
			InputJSON:          payload.Input,
			GroundingJSON:      payload.Grounding,
			ExpectedOutputJSON: payload.Output,
			NegativeOutputJSON: payload.NegativeOutput,
			LabelsJSON:         rawOrEmptyObject(payload.Labels),
			ProvenanceJSON:     provenance,
			ConsentStatus:      ConsentNotRequired,
		})
		if err != nil {
			return err
		}
		count++
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return count, err
}

func keepBestByDuplicateGroup(examples []TrainingExample) []TrainingExample {
	best := map[uuid.UUID]TrainingExample{}
	order := make([]uuid.UUID, 0)
	for _, example := range examples {
		groupID := DuplicateGroupID(example)
		if example.DuplicateGroupID != nil {
			groupID = *example.DuplicateGroupID
		}
		current, ok := best[groupID]
		if !ok {
			best[groupID] = example
			order = append(order, groupID)
			continue
		}
		if scoreValue(example.QualityScore) > scoreValue(current.QualityScore) {
			best[groupID] = example
		}
	}
	out := make([]TrainingExample, 0, len(best))
	for _, id := range order {
		out = append(out, best[id])
	}
	return out
}

func scoreValue(score *float64) float64 {
	if score == nil {
		return 0
	}
	return *score
}

func normalizeScopeType(scopeType string) string {
	scopeType = strings.TrimSpace(scopeType)
	if scopeType == "" {
		return ScopeGlobalFutureExamples
	}
	return scopeType
}

func excludedDataList() []string {
	return []string{
		"receipts and OCR text",
		"calendar events and free/busy details",
		"comments, collaboration messages, and private notes",
		"user names, emails, phone numbers, and exact home addresses",
		"tokens, passwords, API keys, and public share tokens",
		"raw provider payloads and unlicensed copyrighted text",
		"raw prompts, hidden system instructions, and internal logs",
	}
}

func nullableUserID(user auth.AuthenticatedUser) *uuid.UUID {
	if user.ID == uuid.Nil {
		return nil
	}
	return &user.ID
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func IsNotFound(err error) bool {
	return errors.Is(err, domainerrs.ErrNotFound)
}
