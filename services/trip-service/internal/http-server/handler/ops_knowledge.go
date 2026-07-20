package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
)

// Ops knowledge endpoints expose the review surface for provider-backed travel
// data. They are registered under /ops, which already requires the ops admin
// check and the Ops Dashboard feature flag.
//
// These endpoints return knowledge records and provenance only. They never
// return provider credentials, raw provider payloads, private user feedback, or
// trip content.

type knowledgeReviewRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type knowledgeMergeRequest struct {
	CanonicalPlaceID string `json:"canonicalPlaceId"`
	Reason           string `json:"reason"`
}

type knowledgeIngestionRunRequest struct {
	DestinationID   string   `json:"destinationId,omitempty"`
	DestinationName string   `json:"destinationName"`
	CountryCode     string   `json:"countryCode,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	DryRun          bool     `json:"dryRun,omitempty"`
}

// registerOpsKnowledgeRoutes mounts the AI knowledge quality endpoints. They
// are skipped entirely when the knowledge store is not configured, so a
// deployment without grounding data does not advertise unusable endpoints.
func (h *Handler) registerOpsKnowledgeRoutes(r chi.Router) {
	if h.knowledge == nil {
		return
	}
	r.Route("/ai/knowledge", func(r chi.Router) {
		r.Get("/provider-ingestion/status", h.OpsKnowledgeIngestionStatus)
		r.Post("/provider-ingestion/run", h.OpsKnowledgeIngestionRun)
		r.Get("/places", h.OpsListKnowledgePlaces)
		r.Get("/places/{placeId}", h.OpsGetKnowledgePlace)
		r.Patch("/places/{placeId}/review", h.OpsReviewKnowledgePlace)
		r.Post("/places/{placeId}/refresh", h.OpsRefreshKnowledgePlace)
		r.Get("/duplicates", h.OpsListKnowledgeDuplicates)
		r.Post("/duplicates/{groupId}/merge", h.OpsMergeKnowledgeDuplicates)
		r.Post("/duplicates/{groupId}/reject", h.OpsRejectKnowledgeDuplicates)
		r.Get("/quality-summary", h.OpsKnowledgeQualitySummary)
		r.Get("/provider-observations", h.OpsListKnowledgeObservations)
	})
}

func (h *Handler) OpsKnowledgeQualitySummary(w http.ResponseWriter, r *http.Request) {
	thresholds := knowledge.DefaultThresholds()
	summary, err := h.knowledge.QualitySummary(r.Context(), thresholds.StrongMinQuality, thresholds.StaleAfterDays)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) OpsKnowledgeIngestionStatus(w http.ResponseWriter, r *http.Request) {
	thresholds := knowledge.DefaultThresholds()
	summary, err := h.knowledge.QualitySummary(r.Context(), thresholds.StrongMinQuality, thresholds.StaleAfterDays)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": summary,
		"thresholds": map[string]float64{
			"strongMinQuality": thresholds.StrongMinQuality,
			"weakMinQuality":   thresholds.WeakMinQuality,
			"needsReviewBelow": thresholds.NeedsReviewBelow,
			"rejectBelow":      thresholds.RejectBelow,
		},
		"staleAfterDays": thresholds.StaleAfterDays,
	})
}

// OpsKnowledgeIngestionRun triggers ingestion. When no ingestor is configured
// the request is rejected rather than silently accepted, so an operator is
// never told a run started when nothing will happen.
func (h *Handler) OpsKnowledgeIngestionRun(w http.ResponseWriter, r *http.Request) {
	if h.knowledgeIngestor == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "knowledge_provider_unavailable",
			"message": "No knowledge provider is configured for this deployment.",
		})
		return
	}
	var request knowledgeIngestionRunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(request.DestinationName) == "" {
		writeError(w, http.StatusBadRequest, "destinationName is required")
		return
	}

	ingestRequest := knowledge.IngestRequest{
		DestinationName: request.DestinationName,
		CountryCode:     request.CountryCode,
		Categories:      request.Categories,
		Limit:           request.Limit,
		DryRun:          request.DryRun,
	}
	if strings.TrimSpace(request.DestinationID) != "" {
		destinationID, err := uuid.Parse(request.DestinationID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid destinationId")
			return
		}
		ingestRequest.DestinationID = &destinationID
	}

	result, err := h.knowledgeIngestor.IngestDestination(r.Context(), ingestRequest)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) OpsListKnowledgePlaces(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseKnowledgePlaceFilters(w, r)
	if !ok {
		return
	}
	places, err := h.knowledge.ListPlacesForReview(r.Context(), filters)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": places, "count": len(places)})
}

func (h *Handler) OpsGetKnowledgePlace(w http.ResponseWriter, r *http.Request) {
	placeID, ok := parseUUIDParam(w, r, "placeId", "invalid place id")
	if !ok {
		return
	}
	detail, err := h.knowledge.GetPlaceDetail(r.Context(), placeID)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) OpsReviewKnowledgePlace(w http.ResponseWriter, r *http.Request) {
	placeID, ok := parseUUIDParam(w, r, "placeId", "invalid place id")
	if !ok {
		return
	}
	var request knowledgeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	switch request.Action {
	case "approved", "rejected", "needs_review":
	default:
		writeError(w, http.StatusBadRequest, "action must be approved, rejected, or needs_review")
		return
	}
	// A rejection without a reason is not auditable, so it is refused.
	if request.Action == "rejected" && strings.TrimSpace(request.Reason) == "" {
		writeError(w, http.StatusBadRequest, "reason is required when rejecting a record")
		return
	}

	actorID := opsActorID(r)
	if err := h.knowledge.ReviewAction(r.Context(), placeID, actorID, request.Action, request.Reason); err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	detail, err := h.knowledge.GetPlaceDetail(r.Context(), placeID)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) OpsRefreshKnowledgePlace(w http.ResponseWriter, r *http.Request) {
	if h.knowledgeIngestor == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "knowledge_provider_unavailable",
			"message": "No knowledge provider is configured for this deployment.",
		})
		return
	}
	placeID, ok := parseUUIDParam(w, r, "placeId", "invalid place id")
	if !ok {
		return
	}
	detail, err := h.knowledge.GetPlaceDetail(r.Context(), placeID)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	result, err := h.knowledgeIngestor.IngestDestination(r.Context(), knowledge.IngestRequest{
		DestinationID:   &detail.DestinationID,
		DestinationName: detail.DestinationName,
	})
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) OpsListKnowledgeDuplicates(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := parseOptionalUUIDQuery(w, r, "destinationId")
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := parseOptionalIntQuery(r, "limit", 50)
	groups, err := h.knowledge.ListDuplicateGroups(r.Context(), destinationID, status, limit)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": groups, "count": len(groups)})
}

func (h *Handler) OpsMergeKnowledgeDuplicates(w http.ResponseWriter, r *http.Request) {
	groupID, ok := parseUUIDParam(w, r, "groupId", "invalid duplicate group id")
	if !ok {
		return
	}
	var request knowledgeMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	canonicalPlaceID, err := uuid.Parse(strings.TrimSpace(request.CanonicalPlaceID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "canonicalPlaceId must be a valid place id")
		return
	}
	resolution, err := h.knowledge.MergeDuplicateGroup(r.Context(), groupID, canonicalPlaceID, opsActorID(r), request.Reason)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resolution)
}

func (h *Handler) OpsRejectKnowledgeDuplicates(w http.ResponseWriter, r *http.Request) {
	groupID, ok := parseUUIDParam(w, r, "groupId", "invalid duplicate group id")
	if !ok {
		return
	}
	request, ok := decodeOpsActionRequest(w, r)
	if !ok {
		return
	}
	if err := h.knowledge.RejectDuplicateGroup(r.Context(), groupID, opsActorID(r), request.Reason); err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groupId": groupID, "status": knowledge.DuplicateGroupRejected})
}

func (h *Handler) OpsListKnowledgeObservations(w http.ResponseWriter, r *http.Request) {
	destinationID, ok := parseOptionalUUIDQuery(w, r, "destinationId")
	if !ok {
		return
	}
	matchStatus := strings.TrimSpace(r.URL.Query().Get("matchStatus"))
	limit := parseOptionalIntQuery(r, "limit", 100)
	observations, err := h.knowledge.ListObservations(r.Context(), destinationID, matchStatus, limit)
	if err != nil {
		h.writeKnowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": observations, "count": len(observations)})
}

func parseKnowledgePlaceFilters(w http.ResponseWriter, r *http.Request) (knowledge.PlaceReviewFilters, bool) {
	var filters knowledge.PlaceReviewFilters
	destinationID, ok := parseOptionalUUIDQuery(w, r, "destinationId")
	if !ok {
		return filters, false
	}
	filters.DestinationID = destinationID

	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("reviewStatus")); raw != "" {
		switch raw {
		case knowledge.ReviewStatusAuto, knowledge.ReviewStatusApproved, knowledge.ReviewStatusRejected,
			knowledge.ReviewStatusNeedsReview, knowledge.ReviewStatusMerged:
			filters.ReviewStatus = raw
		default:
			writeError(w, http.StatusBadRequest, "invalid reviewStatus")
			return filters, false
		}
	}
	filters.Filter = strings.TrimSpace(query.Get("filter"))
	switch filters.Filter {
	case "", "low_quality", "stale", "missing_coordinates", "needs_review", "duplicates", "rejected":
	default:
		writeError(w, http.StatusBadRequest, "invalid filter")
		return filters, false
	}
	filters.Limit = parseOptionalIntQuery(r, "limit", 100)
	return filters, true
}

func parseOptionalUUIDQuery(w http.ResponseWriter, r *http.Request, key string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+key)
		return nil, false
	}
	return &parsed, true
}

func parseOptionalIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// opsActorID resolves the acting ops admin for the audit trail. Audit events
// tolerate a nil actor rather than failing the action, since losing the review
// decision would be worse than an unattributed entry.
func opsActorID(r *http.Request) *uuid.UUID {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		return nil
	}
	return &user.ID
}

func (h *Handler) writeKnowledgeError(w http.ResponseWriter, err error) {
	if code := knowledge.ErrorCode(err); code != "" {
		status := http.StatusConflict
		switch {
		case errors.Is(err, knowledge.ErrDuplicateGroupNotFound):
			status = http.StatusNotFound
		case errors.Is(err, knowledge.ErrProviderUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, knowledge.ErrProviderRateLimited):
			status = http.StatusTooManyRequests
		case errors.Is(err, knowledge.ErrObservationInvalid), errors.Is(err, knowledge.ErrLicenseMissing):
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, map[string]any{"error": code, "message": err.Error()})
		return
	}
	h.log.Error("ops knowledge request failed", zap.Error(err))
	writeError(w, http.StatusInternalServerError, "knowledge request failed")
}
