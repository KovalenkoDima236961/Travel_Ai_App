package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activitystream"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/editlocks"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/validation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/presence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

// Handler wires the trip use case to HTTP.
type Handler struct {
	svc               *service.Service
	validator         validation.Validator
	log               *zap.Logger
	presence          presence.Manager
	presenceCfg       presence.Config
	activityStream    activitystream.Manager
	activityStreamCfg activitystream.Config
	editLocks         editlocks.Manager
	editLockCfg       editlocks.Config
	generationJobs    *generationjobs.Service
	workspacePolicies *workspacepolicies.Service
}

// New constructs the trip HTTP handler.
func New(svc *service.Service, validator validation.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: validator, log: log}
}

// EnablePresence wires optional trip presence endpoints onto the handler.
func (h *Handler) EnablePresence(manager presence.Manager, cfg presence.Config) *Handler {
	h.presence = manager
	h.presenceCfg = presence.Normalize(cfg)
	return h
}

// EnableActivityStream wires optional trip activity SSE endpoints onto the handler.
func (h *Handler) EnableActivityStream(manager activitystream.Manager, cfg activitystream.Config) *Handler {
	h.activityStream = manager
	h.activityStreamCfg = activitystream.Normalize(cfg)
	return h
}

// EnableEditLocks wires optional advisory edit-lock endpoints onto the handler.
func (h *Handler) EnableEditLocks(manager editlocks.Manager, cfg editlocks.Config) *Handler {
	h.editLocks = manager
	h.editLockCfg = editlocks.Normalize(cfg)
	return h
}

func (h *Handler) EnableGenerationJobs(svc *generationjobs.Service) *Handler {
	h.generationJobs = svc
	return h
}

func (h *Handler) EnableWorkspacePolicies(svc *workspacepolicies.Service) *Handler {
	h.workspacePolicies = svc
	return h
}

// RegisterRoutes mounts the trip routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/collaboration/invitations", h.ListCollaborationInvitations)
	r.Post("/planning-constraints/preview", h.PreviewPlanningConstraints)
	r.Route("/trips", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/shared-with-me", h.ListSharedTrips)
		r.Get("/{id}", h.Get)
		r.Post("/{id}/templates", h.SaveTripAsTemplate)
		r.Get("/{id}/accommodation", h.GetAccommodation)
		r.Put("/{id}/accommodation", h.UpdateAccommodation)
		r.Delete("/{id}/accommodation", h.DeleteAccommodation)
		r.Get("/{id}/route", h.GetRoute)
		r.Put("/{id}/route", h.UpdateRoute)
		r.Patch("/{id}/accommodation/cost-split", h.UpdateAccommodationCostSplit)
		r.Get("/{id}/budget-summary", h.GetBudgetSummary)
		r.Get("/{id}/analytics/costs", h.GetTripCostAnalytics)
		r.Get("/{id}/cost-splitting/summary", h.GetCostSplittingSummary)
		r.Put("/{id}/budget", h.UpdateTripBudget)
		r.Get("/{id}/share", h.GetShare)
		r.Post("/{id}/share", h.CreateShare)
		r.Patch("/{id}/share", h.UpdateShare)
		r.Delete("/{id}/share", h.DisableShare)
		r.Get("/{id}/presence", h.GetPresenceSnapshot)
		r.Get("/{id}/presence/stream", h.StreamPresence)
		r.Post("/{id}/presence/state", h.UpdatePresenceState)
		r.Get("/{id}/edit-lock", h.GetEditLock)
		r.Post("/{id}/edit-lock", h.AcquireEditLock)
		r.Delete("/{id}/edit-lock", h.ReleaseEditLock)
		r.Get("/{id}/calendar-sync/google/status", h.GetGoogleCalendarSyncStatus)
		r.Post("/{id}/calendar-sync/google/sync", h.SyncGoogleCalendar)
		r.Delete("/{id}/calendar-sync/google", h.RemoveGoogleCalendarSync)
		r.Post("/{id}/collaborators", h.InviteTripCollaborator)
		r.Get("/{id}/collaborators", h.ListTripCollaborators)
		r.Patch("/{id}/collaborators/{collaboratorId}", h.UpdateTripCollaborator)
		r.Delete("/{id}/collaborators/{collaboratorId}", h.RemoveTripCollaborator)
		r.Get("/{id}/travelers", h.ListTripTravelers)
		r.Post("/{id}/travelers", h.CreateTripTraveler)
		r.Patch("/{id}/travelers/{travelerId}", h.UpdateTripTraveler)
		r.Delete("/{id}/travelers/{travelerId}", h.RemoveTripTraveler)
		r.Post("/{id}/collaborators/{collaboratorId}/accept", h.AcceptTripCollaborator)
		r.Post("/{id}/collaborators/{collaboratorId}/decline", h.DeclineTripCollaborator)
		r.Post("/{id}/generate", h.Generate)
		r.Post("/{id}/generation-jobs", h.CreateGenerationJob)
		r.Get("/{id}/generation-jobs", h.ListGenerationJobs)
		r.Get("/{id}/generation-jobs/{jobId}", h.GetGenerationJob)
		r.Post("/{id}/generation-jobs/{jobId}/cancel", h.CancelGenerationJob)
		r.Post("/{id}/budget-optimization-jobs", h.CreateBudgetOptimizationJob)
		r.Get("/{id}/budget-optimization-proposals", h.ListBudgetOptimizationProposals)
		r.Get("/{id}/budget-optimization-proposals/{proposalId}", h.GetBudgetOptimizationProposal)
		r.Post("/{id}/budget-optimization-proposals/{proposalId}/apply", h.ApplyBudgetOptimizationProposal)
		r.Post("/{id}/budget-optimization-proposals/{proposalId}/discard", h.DiscardBudgetOptimizationProposal)
		r.Post("/{id}/repair-jobs", h.CreateTripRepairJob)
		r.Get("/{id}/repair-jobs/{jobId}", h.GetGenerationJob)
		r.Get("/{id}/repair-proposals", h.ListTripRepairProposals)
		r.Get("/{id}/repair-proposals/{proposalId}", h.GetTripRepairProposal)
		r.Post("/{id}/repair-proposals/{proposalId}/apply", h.ApplyTripRepairProposal)
		r.Post("/{id}/repair-proposals/{proposalId}/discard", h.DiscardTripRepairProposal)
		r.Put("/{id}/itinerary", h.UpdateItinerary)
		r.Get("/{id}/itinerary/versions", h.ListItineraryVersions)
		r.Get("/{id}/itinerary/versions/{versionId}", h.GetItineraryVersion)
		r.Post("/{id}/itinerary/versions/{versionId}/restore", h.RestoreItineraryVersion)
		r.Post("/{id}/itinerary/days/{dayNumber}/regenerate", h.RegenerateDay)
		r.Post("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate", h.RegenerateItem)
		r.Patch("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/cost-split", h.UpdateItemCostSplit)
		r.Get("/{id}/comments", h.ListComments)
		r.Post("/{id}/comments", h.CreateComment)
		r.Get("/{id}/comments/counts", h.ListCommentCounts)
		r.Patch("/{id}/comments/{commentId}", h.UpdateComment)
		r.Delete("/{id}/comments/{commentId}", h.DeleteComment)
		r.Get("/{id}/activity", h.ListActivity)
		r.Get("/{id}/activity/stream", h.StreamActivity)
		r.Get("/{id}/approval", h.GetApproval)
		r.Get("/{id}/approval-risk", h.GetApprovalRisk)
		r.Post("/{id}/approval/submit", h.SubmitApproval)
		r.Post("/{id}/approval/approve", h.ApproveTrip)
		r.Post("/{id}/approval/request-changes", h.RequestTripChanges)
		r.Post("/{id}/approval/cancel", h.CancelApproval)
		r.Get("/{id}/approval/events", h.ListApprovalEvents)
		r.Get("/{id}/policy/evaluation", h.GetTripPolicyEvaluation)
		r.Post("/{id}/policy/evaluate", h.EvaluateTripPolicy)
	})
	r.Route("/trip-templates", func(r chi.Router) {
		r.Get("/", h.ListTripTemplates)
		r.Get("/{templateId}", h.GetTripTemplate)
		r.Patch("/{templateId}", h.UpdateTripTemplate)
		r.Post("/{templateId}/archive", h.ArchiveTripTemplate)
		r.Post("/{templateId}/duplicate", h.DuplicateTripTemplate)
		r.Post("/{templateId}/create-trip", h.CreateTripFromTemplate)
		r.Post("/{templateId}/adaptation-jobs", h.CreateTemplateAdaptationJob)
	})
	r.Get("/workspaces/{workspaceId}/analytics/costs", h.GetWorkspaceCostAnalytics)
	r.Get("/workspaces/{workspaceId}/templates", h.ListWorkspaceTripTemplates)
	r.Get("/workspaces/{workspaceId}/approvals", h.ListWorkspaceApprovals)
	r.Get("/workspaces/{workspaceId}/policy", h.GetWorkspacePolicy)
	r.Put("/workspaces/{workspaceId}/policy", h.UpsertWorkspacePolicy)
	r.Post("/workspaces/{workspaceId}/policy/archive", h.ArchiveWorkspacePolicy)
	r.Route("/workspaces/{workspaceId}/budgets", func(r chi.Router) {
		r.Get("/", h.ListWorkspaceBudgets)
		r.Post("/", h.CreateWorkspaceBudget)
		r.Get("/primary/summary", h.GetPrimaryWorkspaceBudgetSummary)
		r.Get("/{budgetId}", h.GetWorkspaceBudget)
		r.Patch("/{budgetId}", h.UpdateWorkspaceBudget)
		r.Post("/{budgetId}/archive", h.ArchiveWorkspaceBudget)
		r.Post("/{budgetId}/make-primary", h.MakeWorkspaceBudgetPrimary)
		r.Get("/{budgetId}/summary", h.GetWorkspaceBudgetSummary)
	})
}

func (h *Handler) PreviewPlanningConstraints(w http.ResponseWriter, r *http.Request) {
	var req planningconstraints.PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	response, err := h.svc.PreviewPlanningConstraints(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetGoogleCalendarSyncStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	status, err := h.svc.GetGoogleCalendarSyncStatus(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) SyncGoogleCalendar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SyncTripToGoogleCalendar(r.Context(), id, req.ExpectedItineraryRevision)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RemoveGoogleCalendarSync(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.RemoveTripGoogleCalendarSync(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// RegisterPublicRoutes mounts unauthenticated read-only public routes.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/public/trips/{shareToken}/status", h.GetPublicShareStatus)
	r.Post("/public/trips/{shareToken}/unlock", h.UnlockPublicShare)
	r.Get("/public/trips/{shareToken}", h.GetPublicTrip)
}

// Create handles POST /trips.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTrip
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, response.NewTrip(created))
}

// List handles GET /trips?limit=&offset=. Pagination defaults and bounds are
// enforced by the service; the handler only parses the query parameters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return
	}
	scope := appdto.TripListScope(strings.TrimSpace(r.URL.Query().Get("scope")))
	var workspaceID *uuid.UUID
	if rawWorkspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId")); rawWorkspaceID != "" {
		parsed, err := uuid.Parse(rawWorkspaceID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workspace id")
			return
		}
		workspaceID = &parsed
	}

	trips, appliedLimit, appliedOffset, err := h.svc.ListWithFilters(r.Context(), appdto.ListTripsInput{
		Limit:       limit,
		Offset:      offset,
		Scope:       scope,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewListTrips(trips, appliedLimit, appliedOffset))
}

// Get handles GET /trips/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	t, access, err := h.svc.GetWithAccess(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripWithAccess(t, access))
}

// GetAccommodation handles GET /trips/{id}/accommodation. Any private
// owner/editor/viewer may read it.
func (h *Handler) GetAccommodation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	accommodation, err := h.svc.GetTripAccommodation(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewAccommodationEnvelope(accommodation))
}

func (h *Handler) GetRoute(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	route, err := h.svc.GetTripRoute(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripRouteEnvelope(route))
}

func (h *Handler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.UpdateTripRoute
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.svc.UpdateTripRoute(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTrip(updated))
}

// UpdateAccommodation handles PUT /trips/{id}/accommodation. Only owner/editor
// may mutate it; itinerary revision is unchanged.
func (h *Handler) UpdateAccommodation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.UpdateTripAccommodation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.svc.UpdateTripAccommodation(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewAccommodationEnvelope(updated.Accommodation))
}

// DeleteAccommodation handles DELETE /trips/{id}/accommodation.
func (h *Handler) DeleteAccommodation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if _, err := h.svc.DeleteTripAccommodation(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) UpdateAccommodationCostSplit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.UpdateAccommodationCostSplit
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.svc.UpdateAccommodationCostSplit(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTrip(updated))
}

// GetBudgetSummary handles GET /trips/{id}/budget-summary. Any accepted
// collaborator (owner/editor/viewer) may read it.
func (h *Handler) GetBudgetSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	summary, err := h.svc.GetBudgetSummary(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetTripCostAnalytics handles GET /trips/{id}/analytics/costs.
func (h *Handler) GetTripCostAnalytics(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}

	result, err := h.svc.GetTripCostAnalytics(r.Context(), id, currency)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetCostSplittingSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetCostSplittingSummary(r.Context(), id, currency)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetWorkspaceCostAnalytics handles GET /workspaces/{workspaceId}/analytics/costs.
func (h *Handler) GetWorkspaceCostAnalytics(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	from, ok := parseDateQuery(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseDateQuery(w, r, "to")
	if !ok {
		return
	}
	includeArchived, ok := parseBoolQuery(w, r, "includeArchived")
	if !ok {
		return
	}

	result, err := h.svc.GetWorkspaceCostAnalytics(r.Context(), workspaceID, appdto.WorkspaceCostAnalyticsInput{
		Currency:        currency,
		From:            from,
		To:              to,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// UpdateTripBudget handles PUT /trips/{id}/budget. Only owner/editor may update.
// It does not require expectedItineraryRevision and does not mutate the
// itinerary revision.
func (h *Handler) UpdateTripBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.UpdateTripBudget
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget")
		return
	}

	updated, err := h.svc.UpdateTripBudget(r.Context(), id, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewBudgetEnvelope(updated))
}

func (h *Handler) ListSharedTrips(w http.ResponseWriter, r *http.Request) {
	shared, err := h.svc.ListSharedTrips(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewSharedTrips(shared))
}

// GetShare handles GET /trips/{id}/share.
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	share, err := h.svc.GetTripShare(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripShareInfo(share))
}

// CreateShare handles POST /trips/{id}/share.
func (h *Handler) CreateShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.CreateTripShare
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	share, err := h.svc.CreateOrEnableTripShare(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripShareInfo(share))
}

// UpdateShare handles PATCH /trips/{id}/share.
func (h *Handler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.UpdateTripShareSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	share, err := h.svc.UpdateTripShareSettings(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripShareInfo(share))
}

// DisableShare handles DELETE /trips/{id}/share.
func (h *Handler) DisableShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.svc.DisableTripShare(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) InviteTripCollaborator(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.InviteTripCollaborator
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	collaborator, err := h.svc.InviteTripCollaborator(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripCollaborator(collaborator))
}

func (h *Handler) ListTripCollaborators(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	collaborators, err := h.svc.ListTripCollaborators(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripCollaborators(collaborators))
}

func (h *Handler) UpdateTripCollaborator(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	collaboratorID, ok := parseUUIDParam(w, r, "collaboratorId", "invalid collaborator id")
	if !ok {
		return
	}
	var req request.UpdateTripCollaborator
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	collaborator, err := h.svc.UpdateTripCollaborator(r.Context(), id, collaboratorID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripCollaborator(collaborator))
}

func (h *Handler) RemoveTripCollaborator(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	collaboratorID, ok := parseUUIDParam(w, r, "collaboratorId", "invalid collaborator id")
	if !ok {
		return
	}
	if err := h.svc.RemoveTripCollaborator(r.Context(), id, collaboratorID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ListTripTravelers(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	travelers, err := h.svc.ListTripTravelers(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripTravelers(travelers))
}

func (h *Handler) CreateTripTraveler(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateTripTraveler
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	traveler, err := h.svc.CreateTripTraveler(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripTraveler(traveler))
}

func (h *Handler) UpdateTripTraveler(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	travelerID, ok := parseUUIDParam(w, r, "travelerId", "invalid traveler id")
	if !ok {
		return
	}
	var req request.UpdateTripTraveler
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	traveler, err := h.svc.UpdateTripTraveler(r.Context(), id, travelerID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripTraveler(traveler))
}

func (h *Handler) RemoveTripTraveler(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	travelerID, ok := parseUUIDParam(w, r, "travelerId", "invalid traveler id")
	if !ok {
		return
	}
	if _, err := h.svc.RemoveTripTraveler(r.Context(), id, travelerID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) AcceptTripCollaborator(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	collaboratorID, ok := parseUUIDParam(w, r, "collaboratorId", "invalid collaborator id")
	if !ok {
		return
	}
	collaborator, err := h.svc.AcceptTripCollaborator(r.Context(), id, collaboratorID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripCollaborator(collaborator))
}

func (h *Handler) DeclineTripCollaborator(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	collaboratorID, ok := parseUUIDParam(w, r, "collaboratorId", "invalid collaborator id")
	if !ok {
		return
	}
	if err := h.svc.DeclineTripCollaborator(r.Context(), id, collaboratorID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ListCollaborationInvitations(w http.ResponseWriter, r *http.Request) {
	invitations, err := h.svc.ListCollaborationInvitations(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewCollaborationInvitations(invitations))
}

// GetPublicTrip handles GET /public/trips/{shareToken}.
func (h *Handler) GetPublicTrip(w http.ResponseWriter, r *http.Request) {
	shareToken := strings.TrimSpace(chi.URLParam(r, "shareToken"))
	shareAccessToken, _ := bearerToken(r.Header.Get("Authorization"))

	t, share, err := h.svc.GetPublicTripByShareToken(r.Context(), shareToken, shareAccessToken)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "shared trip not found")
			return
		}
		if errors.Is(err, service.ErrSharePasswordRequired) {
			writeError(w, http.StatusUnauthorized, "share password required")
			return
		}
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewPublicTrip(t, share.CreatedAt))
}

// GetPublicShareStatus handles GET /public/trips/{shareToken}/status.
func (h *Handler) GetPublicShareStatus(w http.ResponseWriter, r *http.Request) {
	shareToken := strings.TrimSpace(chi.URLParam(r, "shareToken"))

	status, err := h.svc.GetPublicTripShareStatus(r.Context(), shareToken)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "shared trip not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewPublicShareStatus(status))
}

// UnlockPublicShare handles POST /public/trips/{shareToken}/unlock.
func (h *Handler) UnlockPublicShare(w http.ResponseWriter, r *http.Request) {
	shareToken := strings.TrimSpace(chi.URLParam(r, "shareToken"))

	var req request.PublicShareUnlock
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// TODO: add per-share rate limiting before enabling password protection in production.
	unlock, err := h.svc.UnlockPublicTripShare(r.Context(), shareToken, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, domainerrs.ErrNotFound):
			writeError(w, http.StatusNotFound, "shared trip not found")
		case errors.Is(err, service.ErrInvalidSharePassword):
			writeError(w, http.StatusUnauthorized, "invalid password")
		default:
			h.writeServiceError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusOK, response.NewPublicShareUnlockResponse(unlock))
}

// Generate handles POST /trips/{id}/generate.
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.GenerateTripItinerary
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Generate(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

func (h *Handler) CreateGenerationJob(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	var req generationjobs.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	job, err := h.generationJobs.Create(r.Context(), id, req)
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, generationjobs.NewJobEnvelope(job))
}

func (h *Handler) GetGenerationJob(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	job, err := h.generationJobs.Get(r.Context(), id, jobID)
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, generationjobs.NewJobEnvelope(job))
}

func (h *Handler) ListGenerationJobs(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	jobs, appliedLimit, err := h.generationJobs.List(r.Context(), id, limit)
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, generationjobs.NewListResponse(jobs, appliedLimit))
}

func (h *Handler) CancelGenerationJob(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	jobID, ok := parseUUIDParam(w, r, "jobId", "invalid generation job id")
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	job, err := h.generationJobs.Cancel(r.Context(), id, jobID)
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, generationjobs.NewJobEnvelope(job))
}

func (h *Handler) CreateBudgetOptimizationJob(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	var req budgetoptimization.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	_, payload, err := req.NormalizeAndPayload()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	job, err := h.generationJobs.Create(r.Context(), id, generationjobs.CreateRequest{
		JobType:                   entity.GenerationJobTypeBudgetOptimizationDay,
		ExpectedItineraryRevision: req.ExpectedItineraryRevision,
		Instruction:               req.Instruction,
		DayNumber:                 req.DayNumber,
		Payload:                   payload,
	})
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, generationjobs.NewJobEnvelope(job))
}

func (h *Handler) ListBudgetOptimizationProposals(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	proposals, appliedLimit, err := h.svc.ListBudgetOptimizationProposals(r.Context(), id, status, limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, budgetoptimization.NewListResponse(proposals, appliedLimit))
}

func (h *Handler) GetBudgetOptimizationProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid budget optimization proposal id")
	if !ok {
		return
	}
	proposal, err := h.svc.GetBudgetOptimizationProposal(r.Context(), id, proposalID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, budgetoptimization.NewProposalEnvelope(proposal))
}

func (h *Handler) ApplyBudgetOptimizationProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid budget optimization proposal id")
	if !ok {
		return
	}
	var req budgetoptimization.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trip, proposal, err := h.svc.ApplyBudgetOptimizationProposal(r.Context(), id, proposalID, req.ExpectedItineraryRevision)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trip":     response.NewTrip(trip),
		"proposal": budgetoptimization.NewProposalResponse(proposal),
	})
}

func (h *Handler) DiscardBudgetOptimizationProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid budget optimization proposal id")
	if !ok {
		return
	}
	proposal, err := h.svc.DiscardBudgetOptimizationProposal(r.Context(), id, proposalID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, budgetoptimization.NewProposalEnvelope(proposal))
}

func (h *Handler) CreateTripRepairJob(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}

	var req triprepair.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	_, payload, err := req.NormalizeAndPayload()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	job, err := h.generationJobs.Create(r.Context(), id, generationjobs.CreateRequest{
		JobType:                   entity.GenerationJobTypePolicyRepair,
		ExpectedItineraryRevision: req.ExpectedItineraryRevision,
		Payload:                   payload,
	})
	if err != nil {
		h.writeGenerationJobError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, generationjobs.NewJobEnvelope(job))
}

func (h *Handler) ListTripRepairProposals(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	proposals, appliedLimit, err := h.svc.ListTripRepairProposals(r.Context(), id, status, limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, triprepair.NewListResponse(proposals, appliedLimit))
}

func (h *Handler) GetTripRepairProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid repair proposal id")
	if !ok {
		return
	}
	proposal, err := h.svc.GetTripRepairProposal(r.Context(), id, proposalID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, triprepair.NewProposalEnvelope(proposal))
}

func (h *Handler) ApplyTripRepairProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid repair proposal id")
	if !ok {
		return
	}
	var req triprepair.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trip, proposal, err := h.svc.ApplyTripRepairProposal(r.Context(), id, proposalID, req.ExpectedItineraryRevision)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trip":     response.NewTrip(trip),
		"proposal": triprepair.NewProposalResponse(proposal),
	})
}

func (h *Handler) DiscardTripRepairProposal(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	proposalID, ok := parseUUIDParam(w, r, "proposalId", "invalid repair proposal id")
	if !ok {
		return
	}
	var req triprepair.DiscardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	proposal, err := h.svc.DiscardTripRepairProposal(r.Context(), id, proposalID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, triprepair.NewProposalEnvelope(proposal))
}

// UpdateItinerary handles PUT /trips/{id}/itinerary.
func (h *Handler) UpdateItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.UpdateTripItinerary
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.UpdateItinerary(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// RegenerateDay handles POST /trips/{id}/itinerary/days/{dayNumber}/regenerate.
func (h *Handler) RegenerateDay(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}

	req, ok := decodeRegenerateRequest(w, r)
	if !ok {
		return
	}

	t, err := h.svc.RegenerateDay(r.Context(), id, dayNumber, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// RegenerateItem handles POST
// /trips/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate.
func (h *Handler) RegenerateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}
	itemIndex, ok := parseURLInt(w, r, "itemIndex")
	if !ok {
		return
	}

	req, ok := decodeRegenerateRequest(w, r)
	if !ok {
		return
	}

	t, err := h.svc.RegenerateItem(r.Context(), id, dayNumber, itemIndex, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

func (h *Handler) UpdateItemCostSplit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}
	itemIndex, ok := parseURLInt(w, r, "itemIndex")
	if !ok {
		return
	}
	var req request.UpdateItemCostSplit
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.UpdateItemCostSplit(r.Context(), id, dayNumber, itemIndex, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"trip": response.NewTrip(t)})
}

// ListItineraryVersions handles GET /trips/{id}/itinerary/versions.
func (h *Handler) ListItineraryVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return
	}

	versions, appliedLimit, appliedOffset, err := h.svc.ListItineraryVersions(r.Context(), id, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewListItineraryVersions(versions, appliedLimit, appliedOffset))
}

// GetItineraryVersion handles GET /trips/{id}/itinerary/versions/{versionId}.
func (h *Handler) GetItineraryVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid version id")
	if !ok {
		return
	}

	version, err := h.svc.GetItineraryVersion(r.Context(), id, versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewItineraryVersionDetail(version))
}

// RestoreItineraryVersion handles
// POST /trips/{id}/itinerary/versions/{versionId}/restore.
func (h *Handler) RestoreItineraryVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid version id")
	if !ok {
		return
	}

	var req request.RestoreItineraryVersion
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.RestoreItineraryVersion(r.Context(), id, versionID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return parseUUIDParam(w, r, "id", "invalid trip id")
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, key, errorMessage string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		writeError(w, http.StatusBadRequest, errorMessage)
		return uuid.Nil, false
	}
	return id, true
}

func parseURLInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(chi.URLParam(r, key))
	v, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return 0, false
	}
	return v, true
}

func decodeRegenerateRequest(w http.ResponseWriter, r *http.Request) (request.RegenerateItineraryPart, bool) {
	var req request.RegenerateItineraryPart
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return request.RegenerateItineraryPart{}, false
	}
	return req, true
}

// parseQueryInt reads an integer query parameter. A missing/empty value yields 0
// (so the service can apply its default); a non-integer value is a 400.
func parseQueryInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return 0, false
	}
	return v, true
}

func parseCurrencyQuery(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("currency"))
	if raw == "" {
		return "", true
	}
	currency := strings.ToUpper(raw)
	if len(currency) != 3 {
		writeError(w, http.StatusBadRequest, "invalid currency")
		return "", false
	}
	for _, ch := range currency {
		if ch < 'A' || ch > 'Z' {
			writeError(w, http.StatusBadRequest, "invalid currency")
			return "", false
		}
	}
	return currency, true
}

func parseDateQuery(w http.ResponseWriter, r *http.Request, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return nil, false
	}
	return &parsed, true
}

func parseBoolQuery(w http.ResponseWriter, r *http.Request, key string) (bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return false, true
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return false, false
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	value := strings.TrimSpace(header)
	if len(value) <= len(prefix) || strings.ToLower(value[:len(prefix)]) != prefix {
		return "", false
	}
	token := strings.TrimSpace(value[len(prefix):])
	return token, token != ""
}

func (h *Handler) writeValidationError(w http.ResponseWriter, err error) {
	var ve *validation.ValidationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusBadRequest, errorBody{
			Error:  "validation failed",
			Fields: ve.Fields(),
		})
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	var dependency *apperrs.DependencyError
	var budgetConversion *apperrs.BudgetConversionError
	var policyBlocking *workspacepolicies.BlockingViolationError
	var planningBlocking *planningconstraints.BlockingError
	var revisionRequired *apperrs.ExpectedItineraryRevisionRequiredError
	var conflict *apperrs.ItineraryConflictError
	var stateConflict *apperrs.ConflictError
	switch {
	case errors.As(err, &planningBlocking):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":       "planning_constraints_blocked",
			"message":     planningBlocking.Error(),
			"constraints": planningBlocking.Constraints,
			"warnings":    planningBlocking.Constraints.Warnings,
			"blockers":    planningBlocking.Constraints.Blockers,
		})
	case errors.As(err, &policyBlocking):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":      "workspace_policy_blocking_violation",
			"message":    policyBlocking.Error(),
			"evaluation": policyBlocking.Evaluation,
		})
	case errors.As(err, &stateConflict):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "conflict",
			"message": stateConflict.Error(),
		})
	case errors.As(err, &revisionRequired):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "expected_itinerary_revision_required",
			"message": revisionRequired.Error(),
		})
	case errors.As(err, &conflict):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":                    "itinerary_conflict",
			"message":                  conflict.Error(),
			"currentItineraryRevision": conflict.CurrentItineraryRevision,
		})
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.As(err, &budgetConversion):
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error":   "budget_conversion_failed",
			"message": budgetConversion.Error(),
		})
	case errors.As(err, &dependency):
		writeError(w, http.StatusBadGateway, dependency.Error())
	case errors.Is(err, apperrs.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrRegisteredUserNotFound):
		writeError(w, http.StatusNotFound, "registered user not found")
	case errors.Is(err, domainerrs.ErrNotFound):
		writeError(w, http.StatusNotFound, "trip not found")
	default:
		h.log.Error("unhandled service error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) writeGenerationJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, generationjobs.ErrDisabled):
		writeError(w, http.StatusServiceUnavailable, "generation jobs are disabled")
	case errors.Is(err, generationjobs.ErrNotCancellable):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":   "generation_job_not_cancellable",
			"message": "Only queued generation jobs can be cancelled.",
		})
	case errors.Is(err, generationjobs.ErrJobDispatchFailed):
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": "job_dispatch_failed",
		})
	default:
		h.writeServiceError(w, err)
	}
}

// errorBody is the uniform error envelope. Fields is populated only for
// validation failures.
type errorBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}
