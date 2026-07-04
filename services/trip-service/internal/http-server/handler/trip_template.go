package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) ListTripTemplates(w http.ResponseWriter, r *http.Request) {
	in, ok := h.parseListTripTemplatesInput(w, r)
	if !ok {
		return
	}
	templates, limit, offset, err := h.svc.ListTripTemplates(r.Context(), in)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripTemplates(templates, limit, offset))
}

func (h *Handler) ListWorkspaceTripTemplates(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	in, ok := h.parseListTripTemplatesInput(w, r)
	if !ok {
		return
	}
	in.Visibility = entity.TripTemplateVisibilityWorkspace
	in.WorkspaceID = &workspaceID
	templates, limit, offset, err := h.svc.ListTripTemplates(r.Context(), in)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripTemplates(templates, limit, offset))
}

func (h *Handler) SaveTripAsTemplate(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.SaveTripAsTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	template, err := h.svc.SaveTripAsTemplate(r.Context(), tripID, input)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripTemplateDetail(*template))
}

func (h *Handler) GetTripTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	template, err := h.svc.GetTripTemplate(r.Context(), templateID)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripTemplateDetail(*template))
}

func (h *Handler) UpdateTripTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	var req request.UpdateTripTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	template, err := h.svc.UpdateTripTemplateMetadata(r.Context(), templateID, req.ToInput())
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripTemplateDetail(*template))
}

func (h *Handler) ArchiveTripTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	var req request.ArchiveTripTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	template, err := h.svc.ArchiveTripTemplate(r.Context(), templateID)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripTemplateDetail(*template))
}

func (h *Handler) DuplicateTripTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	var req request.DuplicateTripTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	template, err := h.svc.DuplicateTripTemplate(r.Context(), templateID, input)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripTemplateDetail(*template))
}

func (h *Handler) CreateTripFromTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, ok := parseUUIDParam(w, r, "templateId", "invalid template id")
	if !ok {
		return
	}
	var req request.CreateTripFromTemplate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	trip, err := h.svc.CreateTripFromTemplate(r.Context(), templateID, input)
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTrip(trip))
}

func (h *Handler) parseListTripTemplatesInput(w http.ResponseWriter, r *http.Request) (appdto.ListTripTemplatesInput, bool) {
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return appdto.ListTripTemplatesInput{}, false
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return appdto.ListTripTemplatesInput{}, false
	}
	var workspaceID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("workspaceId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workspace id")
			return appdto.ListTripTemplatesInput{}, false
		}
		workspaceID = &parsed
	}
	visibility := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("visibility")))
	if visibility == "all" {
		visibility = ""
	}
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	return appdto.ListTripTemplatesInput{
		Limit:       limit,
		Offset:      offset,
		Visibility:  entity.TripTemplateVisibility(visibility),
		Status:      entity.TripTemplateStatus(status),
		WorkspaceID: workspaceID,
		Tag:         r.URL.Query().Get("tag"),
		Query:       r.URL.Query().Get("q"),
	}, true
}

func (h *Handler) writeTemplateServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, domainerrs.ErrNotFound) {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	h.writeServiceError(w, err)
}
