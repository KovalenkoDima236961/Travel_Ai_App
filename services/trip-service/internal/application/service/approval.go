package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const approvalDecisionNoteMaxLen = 1000

// approvalRepository is the persistence port for approval state and history. The
// concrete postgres repository satisfies it; it is embedded in tripRepository so
// the trip use case reaches these methods through the same s.repo.
type approvalRepository interface {
	GetTripApprovalFields(ctx context.Context, tripID uuid.UUID) (*entity.TripApprovalFields, error)
	UpdateTripApprovalStatus(ctx context.Context, fields *entity.TripApprovalFields) (*entity.TripApprovalFields, error)
	InsertTripApprovalEvent(ctx context.Context, event *entity.TripApprovalEvent) (*entity.TripApprovalEvent, error)
	ListTripApprovalEventsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.TripApprovalEvent, error)
	ListWorkspaceApprovals(ctx context.Context, params entity.ListWorkspaceApprovalsParams) ([]entity.WorkspaceApprovalRow, error)
	CountWorkspaceApprovalsByStatus(ctx context.Context, workspaceID uuid.UUID) (entity.WorkspaceApprovalCounts, error)
	ResetApprovalStatusForTripIfActive(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.ApprovalResetResult, error)
}

// GetTripApproval returns the approval state (plus a freshly computed checklist
// and the caller's allowed actions) for a trip the caller can view. Personal
// trips return status not_required with no checklist and no allowed actions.
func (s *Service) GetTripApproval(ctx context.Context, tripID uuid.UUID) (appdto.TripApprovalState, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	fields, err := s.repo.GetTripApprovalFields(ctx, tripID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	if trip.WorkspaceID == nil {
		return personalApprovalState(fields), nil
	}

	checklist, err := s.computeChecklist(ctx, trip)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	isOwnerAdmin, err := s.isWorkspaceApprover(ctx, user.ID, *trip.WorkspaceID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	return s.buildApprovalState(fields, &checklist, access.CanEdit(), isOwnerAdmin, user.ID), nil
}

// SubmitTripApproval moves a draft/changes_requested/cancelled workspace trip to
// pending_approval. It requires trip edit permission and a checklist with no
// unmet blocker (a missing itinerary in v1); warnings never block.
func (s *Service) SubmitTripApproval(ctx context.Context, tripID uuid.UUID, input appdto.SubmitApprovalInput) (appdto.TripApprovalState, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	trip, access, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	if trip.WorkspaceID == nil {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("approval is only available for workspace trips")
	}
	fields, err := s.repo.GetTripApprovalFields(ctx, tripID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	current := approvals.Status(fields.Status)
	if !approvals.CanSubmitFrom(current) {
		return appdto.TripApprovalState{}, apperrs.NewConflict("this trip cannot be submitted from its current status")
	}
	checklist, err := s.computeChecklist(ctx, trip)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	if !checklist.CanSubmit() {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("this trip has unmet requirements and cannot be submitted for approval")
	}

	now := time.Now().UTC()
	note := trimmedPtr(input.Note)
	fields.Status = string(approvals.StatusPendingApproval)
	fields.SubmittedAt = &now
	fields.SubmittedByUserID = &user.ID
	fields.Note = note
	fields.DecisionNote = nil
	fields.ApprovedAt, fields.ApprovedByUserID = nil, nil
	fields.ChangesRequestedAt, fields.ChangesRequestedByUserID = nil, nil
	fields.CancelledAt, fields.CancelledByUserID = nil, nil
	fields.LastStatusChangedAt, fields.LastStatusChangedByUserID = &now, &user.ID

	updated, err := s.repo.UpdateTripApprovalStatus(ctx, fields)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}

	snapshot := approvalChecklistSnapshot(checklist, input.AcknowledgedWarnings)
	s.recordApprovalEvent(ctx, updated, user.ID, approvals.EventSubmitted, string(current), note, snapshot)
	s.recordApprovalActivity(ctx, tripID, user.ID, activity.EventTripSubmittedForApproval, string(current), fields.Status, input.Note)
	s.notifyApproval(
		ctx,
		s.workspaceApproverRecipients(ctx, *trip.WorkspaceID, user.ID),
		trip, user.ID,
		notifications.TypeTripSubmittedForApproval,
		"Trip submitted for approval",
		"A trip in your workspace was submitted for approval.",
		fields.Status,
	)

	return s.buildApprovalState(updated, &checklist, access.CanEdit(), false, user.ID), nil
}

// ApproveTrip approves a pending workspace trip. Only workspace owners/admins may
// approve.
func (s *Service) ApproveTrip(ctx context.Context, tripID uuid.UUID, input appdto.ApprovalDecisionInput) (appdto.TripApprovalState, error) {
	return s.decide(ctx, tripID, input.DecisionNote, decideConfig{
		to:               approvals.StatusApproved,
		event:            approvals.EventApproved,
		activityType:     activity.EventTripApproved,
		notificationType: notifications.TypeTripApproved,
		title:            "Trip approved",
		message:          "Your trip was approved.",
		allowed:          approvals.CanApproveFrom,
		requireNote:      false,
		notifyBroadcast:  true,
	})
}

// RequestTripChanges moves a pending workspace trip to changes_requested. Only
// workspace owners/admins may request changes, and a decision note is required.
func (s *Service) RequestTripChanges(ctx context.Context, tripID uuid.UUID, input appdto.ApprovalDecisionInput) (appdto.TripApprovalState, error) {
	return s.decide(ctx, tripID, input.DecisionNote, decideConfig{
		to:               approvals.StatusChangesRequested,
		event:            approvals.EventChangesRequested,
		activityType:     activity.EventTripChangesRequested,
		notificationType: notifications.TypeTripChangesRequested,
		title:            "Changes requested",
		message:          "Changes were requested on your trip.",
		allowed:          approvals.CanRequestChangesFrom,
		requireNote:      true,
		notifyBroadcast:  true,
	})
}

type decideConfig struct {
	to               approvals.Status
	event            approvals.EventType
	activityType     string
	notificationType string
	title            string
	message          string
	allowed          func(approvals.Status) bool
	requireNote      bool
	notifyBroadcast  bool
}

// decide is the shared approve / request-changes flow: both require workspace
// owner/admin, a pending trip, and notify the submitter and trip editors.
func (s *Service) decide(ctx context.Context, tripID uuid.UUID, decisionNote string, cfg decideConfig) (appdto.TripApprovalState, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	if trip.WorkspaceID == nil {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("approval is only available for workspace trips")
	}
	if err := s.requireWorkspaceApprover(ctx, user.ID, *trip.WorkspaceID); err != nil {
		return appdto.TripApprovalState{}, err
	}

	note := strings.TrimSpace(decisionNote)
	if cfg.requireNote {
		if note == "" || len(note) > approvalDecisionNoteMaxLen {
			return appdto.TripApprovalState{}, apperrs.NewInvalidInput("decisionNote is required and must be 1-%d characters", approvalDecisionNoteMaxLen)
		}
	} else if len(note) > approvalDecisionNoteMaxLen {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("decisionNote must be at most %d characters", approvalDecisionNoteMaxLen)
	}

	fields, err := s.repo.GetTripApprovalFields(ctx, tripID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	current := approvals.Status(fields.Status)
	if !cfg.allowed(current) {
		return appdto.TripApprovalState{}, apperrs.NewConflict("this action is not allowed from the trip's current approval status")
	}

	now := time.Now().UTC()
	notePtr := trimmedPtr(decisionNote)
	fields.Status = string(cfg.to)
	fields.DecisionNote = notePtr
	fields.LastStatusChangedAt, fields.LastStatusChangedByUserID = &now, &user.ID
	switch cfg.to {
	case approvals.StatusApproved:
		fields.ApprovedAt, fields.ApprovedByUserID = &now, &user.ID
	case approvals.StatusChangesRequested:
		fields.ChangesRequestedAt, fields.ChangesRequestedByUserID = &now, &user.ID
	}

	updated, err := s.repo.UpdateTripApprovalStatus(ctx, fields)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}

	s.recordApprovalEvent(ctx, updated, user.ID, cfg.event, string(current), notePtr, nil)
	s.recordApprovalActivity(ctx, tripID, user.ID, cfg.activityType, string(current), fields.Status, decisionNote)
	if cfg.notifyBroadcast {
		s.notifyApproval(ctx, s.approvalDecisionRecipients(ctx, trip, updated, user.ID), trip, user.ID,
			cfg.notificationType, cfg.title, cfg.message, fields.Status)
	}

	checklist, err := s.computeChecklist(ctx, trip)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	isOwnerAdmin, _ := s.isWorkspaceApprover(ctx, user.ID, *trip.WorkspaceID)
	return s.buildApprovalState(updated, &checklist, false, isOwnerAdmin, user.ID), nil
}

// CancelTripApproval cancels a pending submission. The submitter can cancel their
// own submission; workspace owners/admins can cancel any pending submission.
func (s *Service) CancelTripApproval(ctx context.Context, tripID uuid.UUID, input appdto.CancelApprovalInput) (appdto.TripApprovalState, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	if trip.WorkspaceID == nil {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("approval is only available for workspace trips")
	}
	if len(strings.TrimSpace(input.Note)) > approvalDecisionNoteMaxLen {
		return appdto.TripApprovalState{}, apperrs.NewInvalidInput("note must be at most %d characters", approvalDecisionNoteMaxLen)
	}
	fields, err := s.repo.GetTripApprovalFields(ctx, tripID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	current := approvals.Status(fields.Status)
	if !approvals.CanCancelFrom(current) {
		return appdto.TripApprovalState{}, apperrs.NewConflict("only a pending submission can be cancelled")
	}
	isOwnerAdmin, err := s.isWorkspaceApprover(ctx, user.ID, *trip.WorkspaceID)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	isSubmitter := fields.SubmittedByUserID != nil && *fields.SubmittedByUserID == user.ID
	if !isSubmitter && !isOwnerAdmin {
		return appdto.TripApprovalState{}, apperrs.ErrForbidden
	}

	now := time.Now().UTC()
	note := trimmedPtr(input.Note)
	previousSubmitter := fields.SubmittedByUserID
	fields.Status = string(approvals.StatusCancelled)
	fields.CancelledAt, fields.CancelledByUserID = &now, &user.ID
	fields.DecisionNote = note
	fields.LastStatusChangedAt, fields.LastStatusChangedByUserID = &now, &user.ID

	updated, err := s.repo.UpdateTripApprovalStatus(ctx, fields)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}

	s.recordApprovalEvent(ctx, updated, user.ID, approvals.EventCancelled, string(current), note, nil)
	s.recordApprovalActivity(ctx, tripID, user.ID, activity.EventTripApprovalCancelled, string(current), fields.Status, input.Note)
	// If the submitter cancelled, tell the owners/admins; if an owner/admin
	// cancelled, tell the original submitter.
	var recipients []uuid.UUID
	if isSubmitter {
		recipients = s.workspaceApproverRecipients(ctx, *trip.WorkspaceID, user.ID)
	} else if previousSubmitter != nil {
		recipients = excludeActor([]uuid.UUID{*previousSubmitter}, user.ID)
	}
	s.notifyApproval(ctx, recipients, trip, user.ID,
		notifications.TypeTripApprovalCancelled, "Approval cancelled",
		"An approval submission was cancelled.", fields.Status)

	checklist, err := s.computeChecklist(ctx, trip)
	if err != nil {
		return appdto.TripApprovalState{}, err
	}
	return s.buildApprovalState(updated, &checklist, access.CanEdit(), isOwnerAdmin, user.ID), nil
}

// ListTripApprovalEvents returns a trip's approval history for any caller with
// view access.
func (s *Service) ListTripApprovalEvents(ctx context.Context, tripID uuid.UUID) (appdto.TripApprovalEventsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripApprovalEventsResponse{}, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripApprovalEventsResponse{}, err
	}
	events, err := s.repo.ListTripApprovalEventsByTrip(ctx, tripID, 100)
	if err != nil {
		return appdto.TripApprovalEventsResponse{}, err
	}
	out := make([]appdto.TripApprovalEventDTO, 0, len(events))
	for i := range events {
		out = append(out, approvalEventDTO(events[i]))
	}
	return appdto.TripApprovalEventsResponse{Events: out}, nil
}

// --- helpers ---

// computeChecklist gathers the trip's signals and runs the pure calculator. It
// reuses the cost-splitting summary (traveler/split/estimate signals) and the
// itinerary structure. Workspace-budget evaluation is kept to an existence check
// to keep the checklist lightweight.
func (s *Service) computeChecklist(ctx context.Context, trip *entity.Trip) (approvals.Checklist, error) {
	return s.computeChecklistCore(ctx, trip, s.workspaceHasPrimaryBudget(ctx, trip.WorkspaceID))
}

// workspaceHasPrimaryBudget reports whether the workspace has an active primary
// budget. A lookup failure is treated as "no budget" (a warning, never a blocker)
// so the checklist never fails on a budget-service hiccup.
func (s *Service) workspaceHasPrimaryBudget(ctx context.Context, workspaceID *uuid.UUID) bool {
	if workspaceID == nil {
		return false
	}
	if _, err := s.repo.GetPrimaryWorkspaceBudget(ctx, *workspaceID); err == nil {
		return true
	} else if !errors.Is(err, domainerrs.ErrNotFound) {
		s.log.Warn("approval checklist: failed to load workspace budget",
			zap.String("workspace_id", workspaceID.String()), zap.Error(err))
	}
	return false
}

// computeChecklistCore runs the checklist with the workspace-budget existence
// already resolved, so the approvals queue can fetch it once for the whole
// workspace instead of per trip.
func (s *Service) computeChecklistCore(ctx context.Context, trip *entity.Trip, hasWorkspaceBudget bool) (approvals.Checklist, error) {
	checklist, _, err := s.computeChecklistWithInput(ctx, trip, hasWorkspaceBudget)
	return checklist, err
}

// computeChecklistWithInput returns the checklist plus the gathered input so
// callers (the queue) can also read derived signals like the estimated total.
func (s *Service) computeChecklistWithInput(ctx context.Context, trip *entity.Trip, hasWorkspaceBudget bool) (approvals.Checklist, approvals.ChecklistInput, error) {
	itinerary := parseItineraryLenient(trip.Itinerary)
	itemCount, bookable, unchecked := countItineraryItems(itinerary)

	travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, trip.ID)
	if err != nil {
		return approvals.Checklist{}, approvals.ChecklistInput{}, err
	}
	currency, err := resolveCostSplitCurrency("", trip, itinerary)
	if err != nil {
		currency = trip.BudgetCurrency
	}
	summary, err := s.calculateCostSplittingSummary(ctx, trip, itinerary, travelers, currency, time.Now().UTC())
	if err != nil {
		return approvals.Checklist{}, approvals.ChecklistInput{}, err
	}

	in := approvals.ChecklistInput{
		ItineraryDayCount:          len(itinerary.Days),
		ItineraryItemCount:         itemCount,
		HasTripBudget:              trip.BudgetAmount != nil,
		TripBudgetAmount:           valueOrZeroFloat(trip.BudgetAmount),
		EstimatedTotal:             summary.Summary.EstimatedTotal,
		HasWorkspaceBudget:         hasWorkspaceBudget,
		TravelerCount:              summary.Summary.TravelerCount,
		UnassignedCostCount:        len(summary.UnassignedCosts),
		InvalidSplitCount:          summary.Summary.InvalidSplitCount,
		MissingEstimateCount:       summary.Summary.MissingEstimateCount,
		DefaultSplitCount:          summary.Summary.DefaultSplitCount,
		BookableItemCount:          bookable,
		AvailabilityUncheckedCount: unchecked,
	}
	return approvals.Calculate(in), in, nil
}

func countItineraryItems(it aggregate.Itinerary) (items, bookable, unchecked int) {
	for _, day := range it.Days {
		for i := range day.Items {
			items++
			if day.Items[i].Place != nil {
				bookable++
				if day.Items[i].PriceEnrichment == nil {
					unchecked++
				}
			}
		}
	}
	return items, bookable, unchecked
}

func (s *Service) buildApprovalState(
	fields *entity.TripApprovalFields,
	checklist *approvals.Checklist,
	canEdit, isOwnerAdmin bool,
	actorID uuid.UUID,
) appdto.TripApprovalState {
	state := approvalStateBase(fields)
	state.Checklist = checklist
	if fields.WorkspaceID == nil || approvals.Status(fields.Status) == approvals.StatusNotRequired {
		return state
	}
	status := approvals.Status(fields.Status)
	isSubmitter := fields.SubmittedByUserID != nil && *fields.SubmittedByUserID == actorID
	checklistOK := checklist == nil || checklist.CanSubmit()
	state.CanSubmit = canEdit && approvals.CanSubmitFrom(status) && checklistOK
	state.CanApprove = isOwnerAdmin && approvals.CanApproveFrom(status)
	state.CanRequestChanges = isOwnerAdmin && approvals.CanRequestChangesFrom(status)
	state.CanCancel = approvals.CanCancelFrom(status) && (isSubmitter || isOwnerAdmin)
	return state
}

func personalApprovalState(fields *entity.TripApprovalFields) appdto.TripApprovalState {
	state := approvalStateBase(fields)
	state.Status = string(approvals.StatusNotRequired)
	return state
}

func approvalStateBase(fields *entity.TripApprovalFields) appdto.TripApprovalState {
	return appdto.TripApprovalState{
		TripID:                    fields.TripID,
		WorkspaceID:               fields.WorkspaceID,
		Status:                    fields.Status,
		SubmittedAt:               fields.SubmittedAt,
		SubmittedByUserID:         fields.SubmittedByUserID,
		ApprovedAt:                fields.ApprovedAt,
		ApprovedByUserID:          fields.ApprovedByUserID,
		ChangesRequestedAt:        fields.ChangesRequestedAt,
		ChangesRequestedByUserID:  fields.ChangesRequestedByUserID,
		CancelledAt:               fields.CancelledAt,
		CancelledByUserID:         fields.CancelledByUserID,
		Note:                      fields.Note,
		DecisionNote:              fields.DecisionNote,
		LastStatusChangedAt:       fields.LastStatusChangedAt,
		LastStatusChangedByUserID: fields.LastStatusChangedByUserID,
	}
}

func approvalEventDTO(e entity.TripApprovalEvent) appdto.TripApprovalEventDTO {
	return appdto.TripApprovalEventDTO{
		ID:                e.ID,
		EventType:         e.EventType,
		FromStatus:        e.FromStatus,
		ToStatus:          e.ToStatus,
		ActorUserID:       e.ActorUserID,
		Note:              e.Note,
		ChecklistSnapshot: e.ChecklistSnapshot,
		CreatedAt:         e.CreatedAt,
	}
}

// isWorkspaceApprover reports whether the user is a workspace owner or admin.
func (s *Service) isWorkspaceApprover(ctx context.Context, userID, workspaceID uuid.UUID) (bool, error) {
	access, err := s.workspaceAccess(ctx, userID, workspaceID)
	if err != nil {
		if errors.Is(err, apperrs.ErrForbidden) {
			return false, nil
		}
		return false, err
	}
	return access.Role == workspaces.RoleOwner || access.Role == workspaces.RoleAdmin, nil
}

// requireWorkspaceApprover enforces owner/admin for approve/request-changes.
func (s *Service) requireWorkspaceApprover(ctx context.Context, userID, workspaceID uuid.UUID) error {
	access, err := s.workspaceAccess(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	switch access.Role {
	case workspaces.RoleOwner, workspaces.RoleAdmin:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

func (s *Service) recordApprovalEvent(
	ctx context.Context,
	fields *entity.TripApprovalFields,
	actorID uuid.UUID,
	eventType approvals.EventType,
	fromStatus string,
	note *string,
	checklistSnapshot []byte,
) {
	if fields.WorkspaceID == nil {
		return
	}
	from := fromStatus
	event := &entity.TripApprovalEvent{
		TripID:            fields.TripID,
		WorkspaceID:       *fields.WorkspaceID,
		ActorUserID:       actorID,
		EventType:         string(eventType),
		FromStatus:        &from,
		ToStatus:          fields.Status,
		Note:              note,
		ChecklistSnapshot: checklistSnapshot,
	}
	if _, err := s.repo.InsertTripApprovalEvent(ctx, event); err != nil {
		s.log.Warn("failed to record approval event",
			zap.String("trip_id", fields.TripID.String()),
			zap.String("event_type", string(eventType)),
			zap.Error(err))
	}
}

func (s *Service) recordApprovalActivity(ctx context.Context, tripID, actorID uuid.UUID, eventType, fromStatus, toStatus, note string) {
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &actorID,
		EventType:   eventType,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"fromStatus":  fromStatus,
			"toStatus":    toStatus,
			"noteSnippet": snippet(note, 120),
		},
	})
}

// notifyApproval fans out a single approval notification to a resolved recipient
// list. Metadata is limited to safe identifiers (never notes or itinerary).
func (s *Service) notifyApproval(
	ctx context.Context,
	recipients []uuid.UUID,
	trip *entity.Trip,
	actorID uuid.UUID,
	notificationType, title, message, status string,
) {
	if !s.notificationsEnabled || s.notifier == nil || trip == nil || len(recipients) == 0 {
		return
	}
	metadata := map[string]any{
		"tripId":         trip.ID.String(),
		"approvalStatus": status,
	}
	if trip.WorkspaceID != nil {
		metadata["workspaceId"] = trip.WorkspaceID.String()
	}
	tripID := trip.ID
	actor := actorID
	inputs := make([]notifications.NotificationCreateInput, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient == uuid.Nil || recipient == actorID {
			continue
		}
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      recipient,
			TripID:      &tripID,
			ActorUserID: &actor,
			Type:        notificationType,
			Title:       title,
			Message:     message,
			EntityType:  activityEntityType(notifications.EntityTrip),
			EntityID:    &tripID,
			Metadata:    metadata,
		})
	}
	s.sendNotifications(ctx, inputs)
}

// workspaceApproverRecipients returns active workspace owners/admins (excluding
// the actor) for submit/cancel-by-member notifications.
func (s *Service) workspaceApproverRecipients(ctx context.Context, workspaceID, actorID uuid.UUID) []uuid.UUID {
	if s.workspaceProvider == nil {
		return nil
	}
	members, err := s.workspaceProvider.ListMembers(ctx, workspaceID)
	if err != nil {
		s.log.Warn("failed to list workspace members for approval notification",
			zap.String("workspace_id", workspaceID.String()), zap.Error(err))
		return nil
	}
	seen := map[uuid.UUID]struct{}{actorID: {}}
	recipients := make([]uuid.UUID, 0)
	for _, member := range members {
		if member.Status != workspaces.MemberStatusActive {
			continue
		}
		if member.Role != workspaces.RoleOwner && member.Role != workspaces.RoleAdmin {
			continue
		}
		if _, ok := seen[member.UserID]; ok {
			continue
		}
		seen[member.UserID] = struct{}{}
		recipients = append(recipients, member.UserID)
	}
	return recipients
}

// approvalDecisionRecipients returns the submitter plus the trip owner and
// accepted collaborators (excluding the actor) for approve/request-changes.
func (s *Service) approvalDecisionRecipients(ctx context.Context, trip *entity.Trip, fields *entity.TripApprovalFields, actorID uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{actorID: {}}
	recipients := make([]uuid.UUID, 0)
	add := func(id *uuid.UUID) {
		if id == nil {
			return
		}
		if _, ok := seen[*id]; ok {
			return
		}
		seen[*id] = struct{}{}
		recipients = append(recipients, *id)
	}
	add(fields.SubmittedByUserID)
	for _, id := range s.broadcastRecipients(ctx, trip, actorID) {
		copyID := id
		add(&copyID)
	}
	return recipients
}

func excludeActor(ids []uuid.UUID, actorID uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != actorID && id != uuid.Nil {
			out = append(out, id)
		}
	}
	return out
}

func trimmedPtr(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func valueOrZeroFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func snippet(s string, max int) string {
	trimmed := strings.TrimSpace(s)
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max])
}

// approvalChecklistSnapshot serialises the checklist plus the warnings the
// submitter acknowledged into the JSONB stored on the submit event. It never
// stores private notes or itinerary content. A marshal failure degrades to a nil
// snapshot rather than failing the submit.
func approvalChecklistSnapshot(checklist approvals.Checklist, acknowledgedWarnings []string) []byte {
	if acknowledgedWarnings == nil {
		acknowledgedWarnings = []string{}
	}
	payload := struct {
		Checklist            approvals.Checklist `json:"checklist"`
		AcknowledgedWarnings []string            `json:"acknowledgedWarnings"`
	}{Checklist: checklist, AcknowledgedWarnings: acknowledgedWarnings}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return raw
}
