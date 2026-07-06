package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
)

// ResetApprovalIfApproved moves an approved or pending_approval workspace trip
// back to draft after a material edit, so a stale plan is never left approved.
//
// It is deliberately best-effort and must be called ONLY after the originating
// mutation has already succeeded: it swallows all errors (like recordActivity /
// sendNotifications) so an approval bookkeeping failure can never roll back or
// fail the itinerary/budget/traveler change that triggered it. It is a no-op for
// personal trips and for trips in any status other than approved/pending.
func (s *Service) ResetApprovalIfApproved(ctx context.Context, tripID, actorUserID uuid.UUID, reason string) {
	result, err := s.repo.ResetApprovalStatusForTripIfActive(ctx, tripID, actorUserID)
	if err != nil {
		s.log.Warn("failed to reset approval after material edit",
			zap.String("trip_id", tripID.String()), zap.Error(err))
		return
	}
	if result == nil || !result.Reset {
		return
	}

	from := result.FromStatus
	reasonPtr := trimmedPtr(reason)
	event := &entity.TripApprovalEvent{
		TripID:      tripID,
		WorkspaceID: result.WorkspaceID,
		ActorUserID: actorUserID,
		EventType:   string(approvals.EventResetToDraft),
		FromStatus:  &from,
		ToStatus:    string(approvals.StatusDraft),
		Note:        reasonPtr,
	}
	if _, err := s.repo.InsertTripApprovalEvent(ctx, event); err != nil {
		s.log.Warn("failed to record approval reset event",
			zap.String("trip_id", tripID.String()), zap.Error(err))
	}

	s.recordApprovalActivity(ctx, tripID, actorUserID, activity.EventTripApprovalResetToDraft, from, string(approvals.StatusDraft), reason)
	s.notifyApprovalReset(ctx, tripID, result.WorkspaceID, actorUserID)
}

// notifyApprovalReset tells the previous submitter and previous approver that the
// trip changed after submission/approval and needs to be resubmitted. Only fires
// on a real reset (prior status was approved or pending), which bounds the spam.
func (s *Service) notifyApprovalReset(ctx context.Context, tripID, workspaceID, actorUserID uuid.UUID) {
	if !s.notificationsEnabled || s.notifier == nil {
		return
	}
	fields, err := s.repo.GetTripApprovalFields(ctx, tripID)
	if err != nil {
		s.log.Warn("failed to load approval fields for reset notification",
			zap.String("trip_id", tripID.String()), zap.Error(err))
		return
	}
	recipients := make([]uuid.UUID, 0, 2)
	seen := map[uuid.UUID]struct{}{actorUserID: {}}
	for _, candidate := range []*uuid.UUID{fields.SubmittedByUserID, fields.ApprovedByUserID} {
		if candidate == nil {
			continue
		}
		if _, ok := seen[*candidate]; ok {
			continue
		}
		seen[*candidate] = struct{}{}
		recipients = append(recipients, *candidate)
	}
	if len(recipients) == 0 {
		return
	}
	trip := &entity.Trip{ID: tripID, WorkspaceID: &workspaceID}
	s.notifyApproval(ctx, recipients, trip, actorUserID,
		notifications.TypeTripApprovalResetToDraft,
		"Approval reset",
		"This trip changed after submission and needs to be resubmitted.",
		string(approvals.StatusDraft))
}
