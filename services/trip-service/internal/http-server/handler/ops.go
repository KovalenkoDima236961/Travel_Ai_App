package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/featureflags"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
)

type opsActionRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) RegisterOpsRoutes(r chi.Router, staleThreshold time.Duration) {
	r.Route("/ops", func(r chi.Router) {
		r.Use(h.featureMiddleware(featureflags.OpsDashboardEnabled))
		r.Get("/feature-flags", h.OpsListFeatureFlags)
		r.Get("/feature-flags/{key}", h.OpsGetFeatureFlag)
		r.Patch("/feature-flags/{key}", h.OpsUpdateFeatureFlag)
		r.Post("/feature-flags/{key}/reset", h.OpsResetFeatureFlag)
		r.Get("/feature-flags/{key}/audit", h.OpsListFeatureFlagAudit)
		r.Get("/jobs", h.OpsListJobs)
		r.Get("/jobs/summary", h.OpsJobSummary(staleThreshold))
		r.Get("/jobs/{jobId}", h.OpsGetJob(staleThreshold))
		r.Post("/jobs/{jobId}/retry", h.OpsRetryJob)
		r.Post("/jobs/{jobId}/cancel", h.OpsCancelJob)
		r.Post("/jobs/{jobId}/mark-failed", h.OpsMarkJobFailed(staleThreshold))
		r.Get("/ai-generations", h.OpsListAIGenerations)
		r.Get("/ai-generations/{traceId}", h.OpsGetAIGeneration)
	})
}

func (h *Handler) OpsListJobs(w http.ResponseWriter, r *http.Request) {
	if !h.opsGenerationJobsAvailable(w) {
		return
	}
	filters, ok := parseOpsJobFilters(w, r)
	if !ok {
		return
	}
	result, err := h.generationJobs.OpsList(r.Context(), filters)
	if err != nil {
		h.writeOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) OpsGetJob(staleThreshold time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.opsGenerationJobsAvailable(w) {
			return
		}
		jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
		if !ok {
			return
		}
		result, err := h.generationJobs.OpsGet(r.Context(), jobID, staleThreshold)
		if err != nil {
			h.writeOpsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (h *Handler) OpsJobSummary(staleThreshold time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.opsGenerationJobsAvailable(w) {
			return
		}
		result, err := h.generationJobs.OpsSummary(r.Context(), staleThreshold)
		if err != nil {
			h.writeOpsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (h *Handler) OpsRetryJob(w http.ResponseWriter, r *http.Request) {
	if !h.opsGenerationJobsAvailable(w) {
		return
	}
	jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
	if !ok {
		return
	}
	req, ok := decodeOpsActionRequest(w, r)
	if !ok {
		return
	}
	result, err := h.generationJobs.OpsRetry(r.Context(), jobID, req.Reason)
	if err != nil {
		h.writeOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) OpsCancelJob(w http.ResponseWriter, r *http.Request) {
	if !h.opsGenerationJobsAvailable(w) {
		return
	}
	jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
	if !ok {
		return
	}
	req, ok := decodeOpsActionRequest(w, r)
	if !ok {
		return
	}
	result, err := h.generationJobs.OpsCancel(r.Context(), jobID, req.Reason)
	if err != nil {
		h.writeOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) OpsMarkJobFailed(staleThreshold time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.opsGenerationJobsAvailable(w) {
			return
		}
		jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
		if !ok {
			return
		}
		req, ok := decodeOpsActionRequest(w, r)
		if !ok {
			return
		}
		result, err := h.generationJobs.OpsMarkFailed(r.Context(), jobID, req.Reason, staleThreshold)
		if err != nil {
			h.writeOpsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func parseOpsJobFilters(w http.ResponseWriter, r *http.Request) (generationjobs.OpsJobListFilters, bool) {
	q := r.URL.Query()
	var filters generationjobs.OpsJobListFilters
	if raw := strings.TrimSpace(q.Get("status")); raw != "" {
		status := entity.GenerationJobStatus(raw)
		switch status {
		case entity.GenerationJobStatusQueued,
			entity.GenerationJobStatusRunning,
			entity.GenerationJobStatusCompleted,
			entity.GenerationJobStatusFailed,
			entity.GenerationJobStatusCancelled:
			filters.Status = &status
		default:
			writeError(w, http.StatusBadRequest, "invalid status")
			return filters, false
		}
	}
	if raw := strings.TrimSpace(q.Get("jobType")); raw != "" {
		jobType := entity.GenerationJobType(raw)
		if !generationjobs.IsSupportedJobType(jobType) {
			writeError(w, http.StatusBadRequest, "invalid jobType")
			return filters, false
		}
		filters.JobType = &jobType
	}
	if raw := strings.TrimSpace(q.Get("tripId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid tripId")
			return filters, false
		}
		filters.TripID = &id
	}
	if raw := strings.TrimSpace(q.Get("userId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid userId")
			return filters, false
		}
		filters.UserID = &id
	}
	filters.ErrorCode = strings.TrimSpace(q.Get("errorCode"))
	if raw := strings.TrimSpace(q.Get("createdAfter")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid createdAfter")
			return filters, false
		}
		filters.CreatedAfter = &parsed
	}
	if raw := strings.TrimSpace(q.Get("createdBefore")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid createdBefore")
			return filters, false
		}
		filters.CreatedBefore = &parsed
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

func decodeOpsActionRequest(w http.ResponseWriter, r *http.Request) (opsActionRequest, bool) {
	var req opsActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return opsActionRequest{}, false
	}
	return req, true
}

func (h *Handler) opsGenerationJobsAvailable(w http.ResponseWriter) bool {
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return false
	}
	return true
}

func (h *Handler) writeOpsError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, generationjobs.ErrNotCancellable):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "generation_job_not_cancellable",
			"message": "Only queued generation jobs can be cancelled.",
		})
	case errors.Is(err, generationjobs.ErrOpsJobNotStale):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "generation_job_not_stale",
			"message": "Only stale running generation jobs can be marked failed.",
		})
	case errors.Is(err, generationjobs.ErrOpsInvalidAction):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "ops_action_not_allowed",
			"message": "This ops action is not allowed for the current job state.",
		})
	default:
		h.writeGenerationJobError(w, err)
	}
}
