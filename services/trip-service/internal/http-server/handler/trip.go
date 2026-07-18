package handler

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activitystream"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiobservability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aivalidation"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/providerlimit"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triphealth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

// Handler wires the trip use case to HTTP.
type Handler struct {
	svc                  *service.Service
	validator            validation.Validator
	log                  *zap.Logger
	presence             presence.Manager
	presenceCfg          presence.Config
	activityStream       activitystream.Manager
	activityStreamCfg    activitystream.Config
	editLocks            editlocks.Manager
	editLockCfg          editlocks.Config
	generationJobs       *generationjobs.Service
	aiObservability      *aiobservability.Service
	workspacePolicies    *workspacepolicies.Service
	shareUnlockLimiter   *tripsecurity.RateLimiter
	publicShareLimiter   *tripsecurity.RateLimiter
	receiptUploadLimiter *tripsecurity.RateLimiter
}

// New constructs the trip HTTP handler.
func New(svc *service.Service, validator validation.Validator, log *zap.Logger) *Handler {
	return &Handler{
		svc:                  svc,
		validator:            validator,
		log:                  log,
		shareUnlockLimiter:   tripsecurity.NewRateLimiter(5, time.Minute),
		publicShareLimiter:   tripsecurity.NewRateLimiter(120, time.Minute),
		receiptUploadLimiter: tripsecurity.NewRateLimiter(20, time.Minute),
	}
}

func (h *Handler) EnableSecurityLimits(shareUnlock, publicShare, receiptUpload int) *Handler {
	h.shareUnlockLimiter = tripsecurity.NewRateLimiter(shareUnlock, time.Minute)
	h.publicShareLimiter = tripsecurity.NewRateLimiter(publicShare, time.Minute)
	h.receiptUploadLimiter = tripsecurity.NewRateLimiter(receiptUpload, time.Minute)
	return h
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

func (h *Handler) EnableAIObservability(svc *aiobservability.Service) *Handler {
	h.aiObservability = svc
	return h
}

func (h *Handler) EnableWorkspacePolicies(svc *workspacepolicies.Service) *Handler {
	h.workspacePolicies = svc
	return h
}

// RegisterRoutes mounts the trip routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/collaboration/invitations", h.ListCollaborationInvitations)
	r.Get("/reminders/assigned-to-me", h.ListAssignedReminders)
	r.Post("/planning-constraints/preview", h.PreviewPlanningConstraints)
	r.Get("/personalization/context", h.GetPersonalizationContext)
	r.Post("/personalization/feedback", h.SubmitPersonalizationFeedback)
	r.Get("/personalization/feedback/summary", h.GetPersonalizationFeedbackSummary)
	r.Delete("/personalization/feedback", h.ClearPersonalizationFeedback)
	r.Get("/trip-templates/recommended", h.GetRecommendedTemplates)
	r.Post("/route-alternatives/suggest", h.SuggestRouteAlternatives)
	r.Get("/route-alternatives/sessions", h.ListRouteAlternativeSessions)
	r.Get("/route-alternatives/sessions/{sessionId}", h.GetRouteAlternativeSession)
	r.Post("/route-alternatives/sessions/{sessionId}/refine", h.RefineRouteAlternativeSession)
	r.Post("/route-alternatives/sessions/{sessionId}/alternatives/{alternativeId}/create-trip", h.CreateTripFromRouteAlternative)
	r.Route("/trips", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/library", h.GetTripLibrary)
		r.Get("/library/insights", h.GetTripLibraryInsights)
		r.Get("/shared-with-me", h.ListSharedTrips)
		r.Get("/{id}", h.Get)
		r.Post("/{id}/export/archive", h.CreateTripArchiveExport)
		r.Get("/{id}/export/{exportId}", h.GetTripExport)
		r.Get("/{id}/export/{exportId}/download", h.DownloadTripExport)
		r.Post("/{id}/archive", h.ArchiveTrip)
		r.Post("/{id}/restore", h.RestoreTrip)
		r.Get("/{id}/command-center-summary", h.GetCommandCenterSummary)
		r.Get("/{id}/recap/status", h.GetTripRecapStatus)
		r.Get("/{id}/recap", h.GetTripRecap)
		r.Post("/{id}/recap/generate", h.GenerateTripRecap)
		r.Patch("/{id}/recap", h.UpdateTripRecap)
		r.Post("/{id}/recap/finalize", h.FinalizeTripRecap)
		r.Delete("/{id}/recap", h.ArchiveTripRecap)
		r.Post("/{id}/recap/feedback", h.SubmitTripRecapFeedback)
		r.Post("/{id}/recap/apply-learning", h.ApplyTripRecapLearning)
		r.Post("/{id}/recap/create-template", h.CreateTemplateFromTripRecap)
		r.Get("/{id}/travel-day", h.GetTravelDay)
		r.Get("/{id}/health", h.GetTripHealth)
		r.Get("/{id}/verification", h.GetTripVerification)
		r.Post("/{id}/verification/actions", h.RunTripVerificationAction)
		r.Get("/{id}/group-readiness", h.GetGroupReadiness)
		r.Post("/{id}/group-readiness/nudge", h.SendGroupReadinessNudge)
		r.Post("/{id}/group-readiness/nudge-missing-availability", h.NudgeMissingAvailability)
		r.Post("/{id}/group-readiness/nudge-assigned-tasks", h.NudgeAssignedTasks)
		r.Post("/{id}/group-readiness/nudge-pending-votes", h.NudgePendingVotes)
		r.Post("/{id}/group-readiness/nudge-pending-settlements", h.NudgePendingSettlements)
		r.Post("/{id}/templates", h.SaveTripAsTemplate)
		r.Get("/{id}/accommodation", h.GetAccommodation)
		r.Put("/{id}/accommodation", h.UpdateAccommodation)
		r.Delete("/{id}/accommodation", h.DeleteAccommodation)
		r.Get("/{id}/route", h.GetRoute)
		r.Put("/{id}/route", h.UpdateRoute)
		r.Post("/{id}/route/legs/{legId}/transport/search", h.SearchRouteLegTransportOptions)
		r.Put("/{id}/route/legs/{legId}/transport-option", h.AttachRouteLegTransportOption)
		r.Delete("/{id}/route/legs/{legId}/transport-option", h.RemoveRouteLegTransportOption)
		r.Get("/{id}/checklist", h.GetChecklist)
		r.Post("/{id}/checklist/generate", h.GenerateChecklist)
		r.Post("/{id}/checklist/items", h.CreateChecklistItem)
		r.Patch("/{id}/checklist/items/{itemId}", h.UpdateChecklistItem)
		r.Delete("/{id}/checklist/items/{itemId}", h.DeleteChecklistItem)
		r.Post("/{id}/checklist/items/{itemId}/check", h.CheckChecklistItem)
		r.Post("/{id}/checklist/items/{itemId}/uncheck", h.UncheckChecklistItem)
		r.Post("/{id}/checklist/reorder", h.ReorderChecklistItems)
		r.Get("/{id}/reminders", h.ListReminders)
		r.Post("/{id}/reminders/generate", h.GenerateReminders)
		r.Post("/{id}/reminders", h.CreateReminder)
		r.Patch("/{id}/reminders/{reminderId}", h.UpdateReminder)
		r.Post("/{id}/reminders/{reminderId}/complete", h.CompleteReminder)
		r.Post("/{id}/reminders/{reminderId}/reopen", h.ReopenReminder)
		r.Post("/{id}/reminders/{reminderId}/disable", h.DisableReminder)
		r.Post("/{id}/reminders/{reminderId}/enable", h.EnableReminder)
		r.Delete("/{id}/reminders/{reminderId}", h.DeleteReminder)
		r.Post("/{id}/route-alternatives", h.SuggestTripRouteAlternatives)
		r.Post("/{id}/route-alternatives/{sessionId}/alternatives/{alternativeId}/apply", h.ApplyRouteAlternative)
		r.Post("/{id}/route-alternatives/{sessionId}/create-poll", h.CreateRouteAlternativesPoll)
		r.Get("/{id}/availability", h.GetTripAvailability)
		r.Put("/{id}/availability/me", h.UpsertMyTripAvailability)
		r.Delete("/{id}/availability/me", h.DeleteMyTripAvailability)
		r.Post("/{id}/availability/request", h.RequestTripAvailability)
		r.Post("/{id}/availability/import-calendar/preview", h.PreviewCalendarAvailabilityImport)
		r.Post("/{id}/availability/import-calendar/apply", h.ApplyCalendarAvailabilityImport)
		r.Get("/{id}/date-options", h.GetTripDateOptions)
		r.Post("/{id}/date-options/generate", h.GenerateTripDateOptions)
		r.Post("/{id}/date-options/{optionId}/apply", h.ApplyTripDateOption)
		r.Post("/{id}/date-options/create-poll", h.CreateDateOptionsPoll)
		r.Patch("/{id}/accommodation/cost-split", h.UpdateAccommodationCostSplit)
		r.Get("/{id}/budget-summary", h.GetBudgetSummary)
		r.Get("/{id}/budget-confidence", h.GetBudgetConfidence)
		r.Get("/{id}/budget-suggestion", h.GetPersonalizedBudgetSuggestion)
		r.Get("/{id}/analytics/costs", h.GetTripCostAnalytics)
		r.Get("/{id}/cost-splitting/summary", h.GetCostSplittingSummary)
		r.Get("/{id}/expenses", h.ListTripExpenses)
		r.Post("/{id}/expenses", h.CreateTripExpense)
		r.Get("/{id}/expenses/export.csv", h.ExportExpensesCSV)
		r.Get("/{id}/expenses/summary", h.GetTripExpenseSummary)
		r.Post("/{id}/expenses/receipts/upload", h.UploadReceipt)
		r.Get("/{id}/expenses/receipts", h.ListReceipts)
		r.Get("/{id}/expenses/receipts/export-metadata.csv", h.ExportReceiptMetadataCSV)
		r.Get("/{id}/expenses/receipts/{receiptId}", h.GetReceipt)
		r.Get("/{id}/expenses/receipts/{receiptId}/file", h.GetReceiptFile)
		r.Post("/{id}/expenses/receipts/{receiptId}/extract", h.ExtractReceipt)
		r.Post("/{id}/expenses/receipts/{receiptId}/create-expense", h.CreateExpenseFromReceipt)
		r.Delete("/{id}/expenses/receipts/{receiptId}", h.DeleteReceipt)
		r.Get("/{id}/expenses/{expenseId}", h.GetTripExpense)
		r.Patch("/{id}/expenses/{expenseId}", h.UpdateTripExpense)
		r.Delete("/{id}/expenses/{expenseId}", h.DeleteTripExpense)
		r.Post("/{id}/expenses/{expenseId}/receipts", h.AttachReceiptToExpense)
		r.Get("/{id}/settlements", h.GetTripSettlements)
		r.Get("/{id}/settlements/export.csv", h.ExportSettlementsCSV)
		r.Post("/{id}/settlements/recalculate", h.RecalculateTripSettlements)
		r.Post("/{id}/settlements/{settlementId}/mark-paid", h.MarkTripSettlementPaid)
		r.Post("/{id}/settlements/{settlementId}/cancel", h.CancelTripSettlement)
		r.Put("/{id}/budget", h.UpdateTripBudget)
		r.Get("/{id}/budget/export.csv", h.ExportBudgetCSV)
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
		r.Get("/{id}/group-preferences", h.GetGroupPreferences)
		r.Post("/{id}/polls", h.CreateTripPoll)
		r.Get("/{id}/polls", h.ListTripPolls)
		r.Get("/{id}/polls/{pollId}", h.GetTripPoll)
		r.Post("/{id}/polls/{pollId}/vote", h.VoteTripPoll)
		r.Post("/{id}/polls/{pollId}/close", h.CloseTripPoll)
		r.Post("/{id}/polls/{pollId}/archive", h.ArchiveTripPoll)
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
		r.Patch("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/travel-status", h.UpdateTravelItemStatus)
		r.Post("/{id}/itinerary/reactions", h.SetItineraryItemReaction)
		r.Get("/{id}/itinerary/reactions", h.ListItineraryItemReactions)
		r.Get("/{id}/itinerary/versions", h.ListItineraryVersions)
		r.Get("/{id}/itinerary/versions/{versionId}", h.GetItineraryVersion)
		r.Post("/{id}/itinerary/versions/{versionId}/restore", h.RestoreItineraryVersion)
		r.Post("/{id}/itinerary/days/{dayNumber}/regenerate", h.RegenerateDay)
		r.Post("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate", h.RegenerateItem)
		r.Get("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/reactions", h.GetItineraryItemReactions)
		r.Delete("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/reactions/me", h.DeleteMyItineraryItemReaction)
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

// GetCommandCenterSummary returns the compact, private initial payload used by
// the Trip Command Center. Public routes never mount this handler.
func (h *Handler) GetCommandCenterSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.GetCommandCenterSummary(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) RegisterInternalRoutes(r chi.Router) {
	r.Post("/internal/reminders/process-due", h.ProcessDueReminders)
	r.Post("/internal/data-exports/account-package", h.BuildAccountTripPackage)
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
	includeArchived, ok := parseBoolQuery(w, r, "includeArchived")
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
		Limit:           limit,
		Offset:          offset,
		Scope:           scope,
		WorkspaceID:     workspaceID,
		IncludeArchived: includeArchived,
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

func (h *Handler) SearchRouteLegTransportOptions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	legID := strings.TrimSpace(chi.URLParam(r, "legId"))
	if legID == "" {
		writeError(w, http.StatusBadRequest, "invalid route leg id")
		return
	}
	var req request.SearchRouteLegTransport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SearchRouteLegTransportOptions(r.Context(), id, legID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AttachRouteLegTransportOption(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	legID := strings.TrimSpace(chi.URLParam(r, "legId"))
	if legID == "" {
		writeError(w, http.StatusBadRequest, "invalid route leg id")
		return
	}
	var req request.AttachRouteLegTransportOption
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.svc.AttachRouteLegTransportOption(r.Context(), id, legID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTrip(updated))
}

func (h *Handler) RemoveRouteLegTransportOption(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	legID := strings.TrimSpace(chi.URLParam(r, "legId"))
	if legID == "" {
		writeError(w, http.StatusBadRequest, "invalid route leg id")
		return
	}
	var req request.RemoveRouteLegTransportOption
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.svc.RemoveRouteLegTransportOption(r.Context(), id, legID, req.ToInput())
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

// GetBudgetConfidence handles GET /trips/{id}/budget-confidence. Private
// owner/editor/viewer access is required because the response can include actual
// expense and receipt-derived signals.
func (h *Handler) GetBudgetConfidence(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	includeDebug, ok := parseBoolQuery(w, r, "includeDebug")
	if !ok {
		return
	}

	result, err := h.svc.GetBudgetConfidence(r.Context(), id, budgetconfidence.Options{
		Currency:     currency,
		IncludeDebug: includeDebug,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetTripHealth handles GET /trips/{id}/health. Health is computed live for
// private trip viewers; public share routes do not expose this endpoint.
func (h *Handler) GetTripHealth(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	includeResolved, ok := parseBoolQuery(w, r, "includeResolved")
	if !ok {
		return
	}
	includeDebug, ok := parseBoolQuery(w, r, "includeDebug")
	if !ok {
		return
	}

	health, err := h.svc.GetTripHealth(r.Context(), id, triphealth.Options{
		IncludeResolved: includeResolved,
		IncludeDebug:    includeDebug,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, health)
}

// GetTripVerification returns the private, advisory real-world readiness
// report. Public-share routes intentionally do not expose this data.
func (h *Handler) GetTripVerification(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripVerification(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// RunTripVerificationAction executes an explicit, permission-checked refresh
// only. It never books or purchases travel.
func (h *Handler) RunTripVerificationAction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var input verification.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.RunTripVerificationAction(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	shareRef := safeShareRef(shareToken)
	if !h.publicShareLimiter.Allow(shareRef + ":" + requestClientKey(r)) {
		h.auditSecurity("share_access", "trip_share", shareRef, "rate_limited")
		writeRateLimited(w)
		return
	}
	shareAccessToken, _ := bearerToken(r.Header.Get("Authorization"))

	t, share, err := h.svc.GetPublicTripByShareToken(r.Context(), shareToken, shareAccessToken)
	if err != nil {
		h.auditSecurity("share_access", "trip_share", shareRef, "denied")
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

	h.auditSecurity("share_access", "trip_share", shareRef, "success")
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
	shareRef := safeShareRef(shareToken)
	if !h.shareUnlockLimiter.Allow(shareRef + ":" + requestClientKey(r)) {
		tripsecurity.ShareUnlockAttempts.WithLabelValues("rate_limited").Inc()
		h.auditSecurity("share_unlock", "trip_share", shareRef, "rate_limited")
		writeRateLimited(w)
		return
	}

	var req request.PublicShareUnlock
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	unlock, err := h.svc.UnlockPublicTripShare(r.Context(), shareToken, req.Password)
	if err != nil {
		tripsecurity.ShareUnlockAttempts.WithLabelValues("failure").Inc()
		h.auditSecurity("share_unlock", "trip_share", shareRef, "failure")
		switch {
		case errors.Is(err, domainerrs.ErrNotFound):
			writeError(w, http.StatusUnauthorized, "Invalid or expired share link.")
		case errors.Is(err, service.ErrInvalidSharePassword):
			writeError(w, http.StatusUnauthorized, "Incorrect password.")
		default:
			h.writeServiceError(w, err)
		}
		return
	}

	tripsecurity.ShareUnlockAttempts.WithLabelValues("success").Inc()
	h.auditSecurity("share_unlock", "trip_share", shareRef, "success")
	writeJSON(w, http.StatusOK, response.NewPublicShareUnlockResponse(unlock))
}

func safeShareRef(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", sum[:6])
}

func requestClientKey(r *http.Request) string {
	value := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil && host != "" {
		return host
	}
	if value != "" {
		return value
	}
	return "unknown"
}

func (h *Handler) auditSecurity(action, resourceType, resourceID, outcome string) {
	tripsecurity.SecurityAuditEvents.WithLabelValues(action, outcome).Inc()
	h.log.Info("security_audit",
		zap.String("action", action),
		zap.String("resource_type", resourceType),
		zap.String("resource_id", resourceID),
		zap.String("outcome", outcome),
	)
}

func writeRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error": map[string]string{
			"code":    "rate_limited",
			"message": "Too many attempts. Please try again later.",
		},
	})
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
	var aiValidation *aivalidation.ValidationError
	var policyBlocking *workspacepolicies.BlockingViolationError
	var planningBlocking *planningconstraints.BlockingError
	var providerLimit *providerlimit.Error
	var revisionRequired *apperrs.ExpectedItineraryRevisionRequiredError
	var conflict *apperrs.ItineraryConflictError
	var stateConflict *apperrs.ConflictError
	switch {
	case errors.As(err, &aiValidation):
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]any{
				"code":    aiValidation.Code,
				"message": aiValidation.Message,
				"details": map[string]any{
					"issues":            aiValidation.Issues,
					"generationQuality": aiValidation.Quality,
				},
			},
		})
	case errors.As(err, &planningBlocking):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":       "planning_constraints_blocked",
			"message":     planningBlocking.Error(),
			"constraints": planningBlocking.Constraints,
			"warnings":    planningBlocking.Constraints.Warnings,
			"blockers":    planningBlocking.Constraints.Blockers,
		})
	case errors.As(err, &providerLimit):
		status := http.StatusTooManyRequests
		if providerLimit.Code == providerlimit.CodeQuotaExceeded {
			status = http.StatusTooManyRequests
		}
		if providerLimit.Code == providerlimit.CodeLimitsUnavailable {
			status = http.StatusServiceUnavailable
		}
		if providerLimit.RetryAfterSeconds > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(providerLimit.RetryAfterSeconds))
		}
		writeJSON(w, status, map[string]any{
			"error":             providerLimit.Code,
			"message":           providerLimit.Error(),
			"provider":          providerLimit.Provider,
			"operation":         providerLimit.Operation,
			"retryAfterSeconds": providerLimit.RetryAfterSeconds,
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
	case errors.Is(err, service.ErrTripHealthDisabled):
		writeError(w, http.StatusServiceUnavailable, "trip health is disabled")
	case errors.Is(err, service.ErrBudgetConfidenceDisabled):
		writeError(w, http.StatusServiceUnavailable, "budget confidence is disabled")
	case errors.Is(err, service.ErrVerificationDisabled):
		writeError(w, http.StatusServiceUnavailable, "verification is disabled")
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
