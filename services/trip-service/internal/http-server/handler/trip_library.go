package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) GetTripLibrary(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseTripLibraryFilters(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripLibrary(r.Context(), filters)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripLibrary(result))
}

func (h *Handler) GetTripLibraryInsights(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseOptionalQueryUUID(w, r, "workspaceId")
	if !ok {
		return
	}
	year, ok := parseOptionalQueryInt(w, r, "year")
	if !ok {
		return
	}
	result, err := h.svc.GetTripLibraryInsights(r.Context(), workspaceID, year)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripLibraryInsights(result))
}

func (h *Handler) ArchiveTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.ArchiveTrip
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}
	result, err := h.svc.ArchiveTrip(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripArchiveResult(result))
}

func (h *Handler) RestoreTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.RestoreTrip(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripArchiveResult(result))
}

func parseTripLibraryFilters(w http.ResponseWriter, r *http.Request) (appdto.TripLibraryFilters, bool) {
	workspaceID, ok := parseOptionalQueryUUID(w, r, "workspaceId")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	year, ok := parseOptionalQueryInt(w, r, "year")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	budgetMin, ok := parseOptionalQueryFloat(w, r, "budgetMin")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	budgetMax, ok := parseOptionalQueryFloat(w, r, "budgetMax")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	hasRecap, ok := parseOptionalQueryBool(w, r, "hasRecap")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	hasTemplate, ok := parseOptionalQueryBool(w, r, "hasTemplate")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	hasExpenses, ok := parseOptionalQueryBool(w, r, "hasExpenses")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	archived, ok := parseOptionalQueryBool(w, r, "archived")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return appdto.TripLibraryFilters{}, false
	}
	return appdto.TripLibraryFilters{Query: strings.TrimSpace(r.URL.Query().Get("q")), Lifecycle: strings.TrimSpace(r.URL.Query().Get("lifecycle")), WorkspaceID: workspaceID, Year: year, Destination: strings.TrimSpace(r.URL.Query().Get("destination")), Country: strings.TrimSpace(r.URL.Query().Get("country")), TripType: strings.TrimSpace(r.URL.Query().Get("tripType")), TravelStyle: strings.TrimSpace(r.URL.Query().Get("travelStyle")), TransportMode: strings.TrimSpace(r.URL.Query().Get("transportMode")), BudgetMin: budgetMin, BudgetMax: budgetMax, Currency: strings.TrimSpace(r.URL.Query().Get("currency")), HasRecap: hasRecap, HasTemplate: hasTemplate, HasExpenses: hasExpenses, Archived: archived, Sort: appdto.TripLibrarySort(strings.TrimSpace(r.URL.Query().Get("sort"))), Limit: limit, Cursor: strings.TrimSpace(r.URL.Query().Get("cursor"))}, true
}

func parseOptionalQueryUUID(w http.ResponseWriter, r *http.Request, key string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+key)
		return nil, false
	}
	return &value, true
}
func parseOptionalQueryFloat(w http.ResponseWriter, r *http.Request, key string) (*float64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+key)
		return nil, false
	}
	return &value, true
}
