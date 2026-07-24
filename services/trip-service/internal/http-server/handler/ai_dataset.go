package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aidataset"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

type aiTrainingConsentRequest struct {
	Granted bool `json:"granted"`
}

type aiDatasetReasonRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) GetAITrainingConsent(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	result, err := h.aiDatasets.GetConsent(r.Context(), aidataset.ScopeGlobalFutureExamples, nil)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateAITrainingConsent(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	var req aiTrainingConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.aiDatasets.SetConsent(r.Context(), aidataset.ScopeGlobalFutureExamples, nil, req.Granted)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GrantTripAITrainingConsent(w http.ResponseWriter, r *http.Request) {
	h.setTripAITrainingConsent(w, r, true)
}

func (h *Handler) RevokeTripAITrainingConsent(w http.ResponseWriter, r *http.Request) {
	h.setTripAITrainingConsent(w, r, false)
}

func (h *Handler) setTripAITrainingConsent(w http.ResponseWriter, r *http.Request, granted bool) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	scopeID := tripID.String()
	result, err := h.aiDatasets.SetConsent(r.Context(), aidataset.ScopeTrip, &scopeID, granted)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GrantItineraryVersionAITrainingConsent(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid itinerary version id")
	if !ok {
		return
	}
	scopeID := versionID.String()
	result, err := h.aiDatasets.SetConsent(r.Context(), aidataset.ScopeItineraryVersion, &scopeID, true)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) registerOpsAIDatasetRoutes(r chi.Router) {
	r.Get("/ai/fine-tuning/readiness", h.OpsAIFineTuningReadiness)
	r.Get("/ai/datasets/projects", h.OpsListAIDatasetProjects)
	r.Post("/ai/datasets/projects", h.OpsCreateAIDatasetProject)
	r.Post("/ai/datasets/projects/{projectId}/extract-golden", h.OpsExtractGoldenAIDatasetExamples)
	r.Post("/ai/datasets/projects/{projectId}/extract-manual", h.OpsExtractManualAIDatasetExamples)
	r.Post("/ai/datasets/projects/{projectId}/versions", h.OpsBuildAIDatasetVersion)
	r.Get("/ai/datasets/examples", h.OpsListAIDatasetExamples)
	r.Get("/ai/datasets/examples/{exampleId}", h.OpsGetAIDatasetExample)
	r.Patch("/ai/datasets/examples/{exampleId}/review", h.OpsReviewAIDatasetExample)
	r.Post("/ai/datasets/examples/{exampleId}/approve", h.OpsApproveAIDatasetExample)
	r.Post("/ai/datasets/examples/{exampleId}/reject", h.OpsRejectAIDatasetExample)
	r.Post("/ai/datasets/examples/{exampleId}/resanitize", h.OpsResanitizeAIDatasetExample)
	r.Post("/ai/datasets/examples/{exampleId}/rescore", h.OpsRescoreAIDatasetExample)
	r.Get("/ai/datasets/duplicates", h.OpsListAIDatasetDuplicates)
	r.Post("/ai/datasets/duplicates/{groupId}/resolve", h.OpsResolveAIDatasetDuplicateGroup)
	r.Post("/ai/datasets/versions/{versionId}/export", h.OpsExportAIDatasetVersion)
	r.Get("/ai/datasets/versions/{versionId}/export/status", h.OpsGetAIDatasetExportStatus)
	r.Get("/ai/datasets/versions/{versionId}/download", h.OpsDownloadAIDatasetVersion)
}

func (h *Handler) OpsAIFineTuningReadiness(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	result, err := h.aiDatasets.Readiness(r.Context())
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) OpsListAIDatasetProjects(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	projects, err := h.aiDatasets.ListProjects(r.Context())
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) OpsCreateAIDatasetProject(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	var input aidataset.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	project, err := h.aiDatasets.CreateProject(r.Context(), input)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) OpsExtractGoldenAIDatasetExamples(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseUUIDParam(w, r, "projectId", "invalid dataset project id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	count, err := h.aiDatasets.ImportGoldenCases(r.Context(), projectID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"created": count})
}

func (h *Handler) OpsExtractManualAIDatasetExamples(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseUUIDParam(w, r, "projectId", "invalid dataset project id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	count, err := h.aiDatasets.ImportManualExamples(r.Context(), projectID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"created": count})
}

func (h *Handler) OpsBuildAIDatasetVersion(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseUUIDParam(w, r, "projectId", "invalid dataset project id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	var input aidataset.BuildVersionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.DatasetProjectID = projectID
	version, err := h.aiDatasets.BuildVersion(r.Context(), input)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, version)
}

func (h *Handler) OpsListAIDatasetExamples(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	filters, ok := parseAIDatasetExampleFilters(w, r)
	if !ok {
		return
	}
	examples, err := h.aiDatasets.ListExamples(r.Context(), filters)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"examples": examples})
}

func (h *Handler) OpsGetAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	example, err := h.aiDatasets.GetExample(r.Context(), exampleID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsReviewAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	var input aidataset.ReviewInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	example, err := h.aiDatasets.ReviewExample(r.Context(), exampleID, input)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsApproveAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	req := decodeAIDatasetReason(r)
	example, err := h.aiDatasets.ApproveExample(r.Context(), exampleID, req.Reason)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsRejectAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	req := decodeAIDatasetReason(r)
	example, err := h.aiDatasets.RejectExample(r.Context(), exampleID, req.Reason)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsResanitizeAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	example, err := h.aiDatasets.ResanitizeExample(r.Context(), exampleID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsRescoreAIDatasetExample(w http.ResponseWriter, r *http.Request) {
	exampleID, ok := parseUUIDParam(w, r, "exampleId", "invalid dataset example id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	example, err := h.aiDatasets.RescoreExample(r.Context(), exampleID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, example)
}

func (h *Handler) OpsListAIDatasetDuplicates(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	examples, err := h.aiDatasets.ListExamples(r.Context(), aidataset.ExampleFilters{Limit: 200})
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	groups := map[string][]aidataset.TrainingExample{}
	for _, example := range examples {
		if example.DuplicateGroupID != nil {
			groups[example.DuplicateGroupID.String()] = append(groups[example.DuplicateGroupID.String()], example)
		}
	}
	for key, group := range groups {
		if len(group) < 2 {
			delete(groups, key)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups})
}

func (h *Handler) OpsResolveAIDatasetDuplicateGroup(w http.ResponseWriter, r *http.Request) {
	if !h.aiDatasetsAvailable(w) {
		return
	}
	groupID := strings.TrimSpace(chi.URLParam(r, "groupId"))
	if _, err := uuid.Parse(groupID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid duplicate group id")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"resolved": false,
		"groupId":  groupID,
		"message":  "Resolve duplicates by approving the best example and rejecting or changing the others.",
	})
}

func (h *Handler) OpsExportAIDatasetVersion(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid dataset version id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	version, err := h.aiDatasets.ExportVersion(r.Context(), versionID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, version)
}

func (h *Handler) OpsGetAIDatasetExportStatus(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid dataset version id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	status, err := h.aiDatasets.GetExportStatus(r.Context(), versionID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) OpsDownloadAIDatasetVersion(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid dataset version id")
	if !ok || !h.aiDatasetsAvailable(w) {
		return
	}
	reader, filename, err := h.aiDatasets.OpenExport(r.Context(), versionID)
	if err != nil {
		h.writeAIDatasetError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, reader); err != nil {
		h.log.Warn("stream AI dataset export failed")
	}
}

func parseAIDatasetExampleFilters(w http.ResponseWriter, r *http.Request) (aidataset.ExampleFilters, bool) {
	q := r.URL.Query()
	filters := aidataset.ExampleFilters{
		ReviewStatus:       strings.TrimSpace(q.Get("reviewStatus")),
		SanitizationStatus: strings.TrimSpace(q.Get("sanitizationStatus")),
		QualityStatus:      strings.TrimSpace(q.Get("qualityStatus")),
		ConsentStatus:      strings.TrimSpace(q.Get("consentStatus")),
		Language:           strings.TrimSpace(q.Get("language")),
		TaskType:           strings.TrimSpace(q.Get("taskType")),
		Split:              strings.TrimSpace(q.Get("split")),
		SourceType:         strings.TrimSpace(q.Get("sourceType")),
	}
	if raw := strings.TrimSpace(q.Get("datasetProjectId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid datasetProjectId")
			return filters, false
		}
		filters.DatasetProjectID = &id
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return filters, false
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return filters, false
	}
	filters.Limit = limit
	filters.Offset = offset
	return filters, true
}

func decodeAIDatasetReason(r *http.Request) aiDatasetReasonRequest {
	var req aiDatasetReasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return aiDatasetReasonRequest{}
	}
	req.Reason = strings.TrimSpace(req.Reason)
	return req
}

func (h *Handler) aiDatasetsAvailable(w http.ResponseWriter) bool {
	if h.aiDatasets == nil {
		writeError(w, http.StatusServiceUnavailable, "AI dataset curation is not configured")
		return false
	}
	return true
}

func (h *Handler) writeAIDatasetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, aidataset.ErrConsentRequired):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_consent_required", "Training consent is required before this example can be approved or exported.")
	case errors.Is(err, aidataset.ErrConsentRevoked):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_consent_revoked", "Training consent was revoked for this example.")
	case errors.Is(err, aidataset.ErrSanitizationFailed):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_sanitization_failed", "Sanitization failed for this example.")
	case errors.Is(err, aidataset.ErrQualityTooLow):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_quality_too_low", "The example quality score is below the approval threshold.")
	case errors.Is(err, aidataset.ErrDatasetDuplicate):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_duplicate", "This example belongs to a duplicate group.")
	case errors.Is(err, aidataset.ErrVersionExists):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_version_exists", "This dataset version already exists.")
	case errors.Is(err, aidataset.ErrVersionNotReady):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_version_not_ready", "The dataset version is not ready for this action.")
	case errors.Is(err, aidataset.ErrExportDisabled):
		writeAIDatasetCode(w, http.StatusServiceUnavailable, "ai_dataset_export_disabled", "AI dataset export is disabled.")
	case errors.Is(err, aidataset.ErrExportFailed):
		writeAIDatasetCode(w, http.StatusInternalServerError, "ai_dataset_export_failed", "AI dataset export failed.")
	case errors.Is(err, aidataset.ErrPrivateDataDetected):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_private_data_detected", "Private or sensitive data was detected.")
	case errors.Is(err, aidataset.ErrLicenseNotAllowed):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_license_not_allowed", "The example source license does not allow export.")
	case errors.Is(err, aidataset.ErrInvalidReviewStatus):
		writeAIDatasetCode(w, http.StatusBadRequest, "invalid_review_status", "Invalid dataset review status.")
	case errors.Is(err, aidataset.ErrNoEligibleExamples):
		writeAIDatasetCode(w, http.StatusConflict, "ai_dataset_no_eligible_examples", "No approved, sanitized, consent-valid examples are eligible.")
	case errors.Is(err, domainerrs.ErrNotFound):
		writeAIDatasetCode(w, http.StatusNotFound, "ai_dataset_not_found", "AI dataset resource not found.")
	default:
		writeAIDatasetCode(w, http.StatusInternalServerError, "ai_dataset_internal_error", "AI dataset request failed.")
	}
}

func writeAIDatasetCode(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error":   code,
		"message": message,
	})
}
