package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiobservability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

func (h *Handler) OpsListAIGenerations(w http.ResponseWriter, r *http.Request) {
	if !h.opsAIObservabilityAvailable(w) {
		return
	}
	filters, ok := parseAIGenerationTraceFilters(w, r)
	if !ok {
		return
	}
	result, err := h.aiObservability.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load AI generation traces")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) OpsGetAIGeneration(w http.ResponseWriter, r *http.Request) {
	if !h.opsAIObservabilityAvailable(w) {
		return
	}
	traceID, err := uuid.Parse(chi.URLParam(r, "traceId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid AI generation trace id")
		return
	}
	detail, err := h.aiObservability.Detail(r.Context(), traceID, true)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "trace details unavailable or expired")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load AI generation trace")
		return
	}
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		h.aiObservability.AuditAccess(r.Context(), actor.ID, actor.Email, "ops_ai_generation_trace_viewed", traceID)
		if detail.PromptSnapshot != nil {
			h.aiObservability.AuditAccess(r.Context(), actor.ID, actor.Email, "ops_ai_generation_prompt_snapshot_viewed", traceID)
		}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) opsAIObservabilityAvailable(w http.ResponseWriter) bool {
	if h.aiObservability == nil || !h.aiObservability.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "AI observability is not configured")
		return false
	}
	return true
}

func parseAIGenerationTraceFilters(w http.ResponseWriter, r *http.Request) (aiobservability.ListFilters, bool) {
	q := r.URL.Query()
	filters := aiobservability.ListFilters{
		Status: strings.TrimSpace(q.Get("status")), GenerationType: strings.TrimSpace(q.Get("generationType")), Provider: strings.TrimSpace(q.Get("provider")), Model: strings.TrimSpace(q.Get("model")), QualityStatus: strings.TrimSpace(q.Get("qualityStatus")), Cursor: strings.TrimSpace(q.Get("cursor")), ErrorOnly: strings.EqualFold(strings.TrimSpace(q.Get("errorOnly")), "true"),
	}
	for _, target := range []struct {
		raw, name   string
		destination **uuid.UUID
	}{{q.Get("tripId"), "tripId", &filters.TripID}, {q.Get("jobId"), "jobId", &filters.JobID}, {q.Get("userId"), "userId", &filters.UserID}, {q.Get("workspaceId"), "workspaceId", &filters.WorkspaceID}} {
		if strings.TrimSpace(target.raw) == "" {
			continue
		}
		id, err := uuid.Parse(strings.TrimSpace(target.raw))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+target.name)
			return filters, false
		}
		*target.destination = &id
	}
	for _, target := range []struct {
		raw, name   string
		destination **time.Time
	}{{q.Get("from"), "from", &filters.From}, {q.Get("to"), "to", &filters.To}} {
		if strings.TrimSpace(target.raw) == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(target.raw))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid "+target.name)
			return filters, false
		}
		*target.destination = &parsed
	}
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		var limit int
		if _, err := fmt.Sscanf(raw, "%d", &limit); err != nil || limit < 1 || limit > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return filters, false
		}
		filters.Limit = limit
	}
	return filters, true
}
