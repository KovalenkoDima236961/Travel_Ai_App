package search

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
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

type Handler struct {
	service *Service
	limiter *tripsecurity.RateLimiter
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger, limiter ...*tripsecurity.RateLimiter) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	var rateLimiter *tripsecurity.RateLimiter
	if len(limiter) > 0 {
		rateLimiter = limiter[0]
	}
	return &Handler{service: service, limiter: rateLimiter, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/search", h.Search)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.limiter != nil && !h.limiter.Allow("global_search:"+user.ID.String()) {
		writeError(w, http.StatusTooManyRequests, "search_rate_limited")
		return
	}
	query := r.URL.Query().Get("q")
	scope, ok := ParseScope(r.URL.Query().Get("scope"))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid scope")
		return
	}
	tripID, ok := parseOptionalUUID(w, r.URL.Query().Get("tripId"), "tripId")
	if !ok {
		return
	}
	workspaceID, ok := parseOptionalUUID(w, r.URL.Query().Get("workspaceId"), "workspaceId")
	if !ok {
		return
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	types, ok := parseResultTypes(w, r.URL.Query().Get("types"))
	if !ok {
		return
	}

	response, err := h.service.Search(r.Context(), user.ID, Params{
		Query:           query,
		Scope:           scope,
		TripID:          tripID,
		WorkspaceID:     workspaceID,
		Types:           types,
		Limit:           limit,
		IncludeArchived: strings.EqualFold(r.URL.Query().Get("includeArchived"), "true"),
		IncludeCommands: strings.EqualFold(r.URL.Query().Get("includeCommands"), "true"),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrQueryTooLong):
			writeError(w, http.StatusBadRequest, "search_query_too_long")
			return
		case errors.Is(err, ErrInvalidQuery):
			writeError(w, http.StatusBadRequest, "search_invalid_query")
			return
		case errors.Is(err, ErrInvalidFilter):
			writeError(w, http.StatusBadRequest, "search_invalid_filter")
			return
		}
		h.log.Warn("search request failed",
			zap.String("scope", string(scope)),
			zap.Int("queryLen", len(query)),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func parseOptionalUUID(w http.ResponseWriter, raw string, name string) (*uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return nil, false
	}
	return &id, true
}

func parseResultTypes(w http.ResponseWriter, raw string) ([]ResultType, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}
	parts := strings.Split(raw, ",")
	resultTypes := make([]ResultType, 0, len(parts))
	for _, part := range parts {
		resultType := ResultType(strings.TrimSpace(part))
		if resultType == "" {
			continue
		}
		if !knownResultType(resultType) {
			writeError(w, http.StatusBadRequest, "search_invalid_filter")
			return nil, false
		}
		resultTypes = append(resultTypes, resultType)
	}
	return resultTypes, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
