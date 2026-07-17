package search

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{service: service, log: log}
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
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}
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

	response, err := h.service.Search(r.Context(), user.ID, Params{
		Query:           query,
		Scope:           scope,
		TripID:          tripID,
		WorkspaceID:     workspaceID,
		Limit:           limit,
		IncludeCommands: strings.EqualFold(r.URL.Query().Get("includeCommands"), "true"),
	})
	if err != nil {
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
