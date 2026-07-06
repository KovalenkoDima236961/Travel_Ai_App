package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

// fakeWorkspaceProvider models a single workspace with per-user roles so
// permission tests can exercise owner/admin/member/viewer/non-member callers.
type fakeWorkspaceProvider struct {
	roles   map[uuid.UUID]workspaces.Role
	members []workspaces.WorkspaceMember
}

func (f *fakeWorkspaceProvider) AccessCheck(_ context.Context, userID, _ uuid.UUID) (*workspaces.Access, error) {
	role, ok := f.roles[userID]
	if !ok {
		return &workspaces.Access{HasAccess: false}, nil
	}
	return &workspaces.Access{HasAccess: true, Role: role, Status: "active"}, nil
}

func (f *fakeWorkspaceProvider) ListForUser(_ context.Context, _ uuid.UUID) ([]workspaces.UserWorkspace, error) {
	return nil, nil
}

func (f *fakeWorkspaceProvider) ListMembers(_ context.Context, _ uuid.UUID) ([]workspaces.WorkspaceMember, error) {
	return f.members, nil
}

var (
	wsID       = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ownerID    = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	memberID   = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")
	viewerID   = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003")
	otherMemID = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000004")
)

func approvalTestProvider() *fakeWorkspaceProvider {
	return &fakeWorkspaceProvider{
		roles: map[uuid.UUID]workspaces.Role{
			ownerID:    workspaces.RoleOwner,
			memberID:   workspaces.RoleMember,
			viewerID:   workspaces.RoleViewer,
			otherMemID: workspaces.RoleMember,
		},
		members: []workspaces.WorkspaceMember{
			{UserID: ownerID, Role: workspaces.RoleOwner, Status: workspaces.MemberStatusActive},
			{UserID: memberID, Role: workspaces.RoleMember, Status: workspaces.MemberStatusActive},
		},
	}
}

func newApprovalService(repo tripRepository, ws workspaceProvider, n notifier) *Service {
	return New(repo, &mockGenerator{}, zap.NewNop(),
		WithWorkspaces(ws, true),
		WithNotifications(n, true, true),
	)
}

// workspaceTrip returns a mock trip that belongs to the workspace and carries a
// valid itinerary so the itinerary blocker passes.
func workspaceTrip(t *testing.T, status string) *entity.Trip {
	return &entity.Trip{
		ID:             uuid.New(),
		UserID:         &memberID,
		WorkspaceID:    &wsID,
		Destination:    "Tokyo",
		Days:           2,
		BudgetCurrency: "EUR",
		Itinerary:      validExistingItineraryRaw(t),
	}
}

func ctxFor(userID uuid.UUID) context.Context {
	return auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID})
}

func approvalFields(tripID uuid.UUID, status string, submitter *uuid.UUID) *entity.TripApprovalFields {
	return &entity.TripApprovalFields{TripID: tripID, WorkspaceID: &wsID, Status: status, SubmittedByUserID: submitter}
}

func TestGetTripApproval_PersonalTrip_NotRequired(t *testing.T) {
	repo := &mockRepo{getByIDResult: &entity.Trip{ID: uuid.New(), UserID: &memberID, Destination: "Rome", Days: 2}}
	repo.approvalFields = &entity.TripApprovalFields{Status: "not_required"}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	state, err := svc.GetTripApproval(ctxFor(memberID), repo.getByIDResult.ID)
	if err != nil {
		t.Fatalf("GetTripApproval: %v", err)
	}
	if state.Status != "not_required" {
		t.Fatalf("expected not_required, got %q", state.Status)
	}
	if state.CanSubmit || state.CanApprove || state.CanRequestChanges || state.CanCancel {
		t.Fatal("personal trip must expose no approval actions")
	}
	if state.Checklist != nil {
		t.Fatal("personal trip must not carry a checklist")
	}
}

func TestSubmit_Draft_ToPending(t *testing.T) {
	trip := workspaceTrip(t, "draft")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "draft", nil)}
	notifier := &fakeNotifier{}
	svc := newApprovalService(repo, approvalTestProvider(), notifier)

	state, err := svc.SubmitTripApproval(ctxFor(memberID), trip.ID, appdto.SubmitApprovalInput{Note: "ready"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if state.Status != "pending_approval" {
		t.Fatalf("expected pending_approval, got %q", state.Status)
	}
	if repo.updatedApprovalFields == nil || repo.updatedApprovalFields.SubmittedByUserID == nil || *repo.updatedApprovalFields.SubmittedByUserID != memberID {
		t.Fatal("expected submitted_by to be the member")
	}
	if len(repo.insertedApprovalEvents) != 1 || repo.insertedApprovalEvents[0].EventType != "submitted" {
		t.Fatalf("expected one submitted event, got %+v", repo.insertedApprovalEvents)
	}
	if len(repo.insertedApprovalEvents[0].ChecklistSnapshot) == 0 {
		t.Fatal("submit event should carry a checklist snapshot")
	}
	// Owners/admins (except the actor) should be notified.
	if !notifier.recipients()[ownerID] {
		t.Fatal("expected owner to be notified of submission")
	}
	if notifier.recipients()[memberID] {
		t.Fatal("actor must not be notified")
	}
}

func TestSubmit_NoItinerary_Blocked(t *testing.T) {
	trip := workspaceTrip(t, "draft")
	trip.Itinerary = json.RawMessage(`{"days":[]}`)
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "draft", nil)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.SubmitTripApproval(ctxFor(memberID), trip.ID, appdto.SubmitApprovalInput{})
	if err == nil {
		t.Fatal("expected submission to be blocked without an itinerary")
	}
	if repo.updatedApprovalFields != nil {
		t.Fatal("blocked submission must not update approval state")
	}
}

func TestSubmit_Viewer_Forbidden(t *testing.T) {
	trip := workspaceTrip(t, "draft")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "draft", nil)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.SubmitTripApproval(ctxFor(viewerID), trip.ID, appdto.SubmitApprovalInput{})
	if err != apperrs.ErrForbidden {
		t.Fatalf("expected forbidden for viewer, got %v", err)
	}
}

func TestSubmit_PersonalTrip_Rejected(t *testing.T) {
	trip := &entity.Trip{ID: uuid.New(), UserID: &memberID, Destination: "Rome", Days: 2, Itinerary: validExistingItineraryRaw(t)}
	repo := &mockRepo{getByIDResult: trip, approvalFields: &entity.TripApprovalFields{TripID: trip.ID, Status: "not_required"}}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.SubmitTripApproval(ctxFor(memberID), trip.ID, appdto.SubmitApprovalInput{})
	var invalid *apperrs.InvalidInputError
	if err == nil || !asErr(err, &invalid) {
		t.Fatalf("expected invalid input for personal trip, got %v", err)
	}
}

func TestApprove_Pending_ByOwner(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	notifier := &fakeNotifier{}
	svc := newApprovalService(repo, approvalTestProvider(), notifier)

	state, err := svc.ApproveTrip(ctxFor(ownerID), trip.ID, appdto.ApprovalDecisionInput{DecisionNote: "great"})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if state.Status != "approved" {
		t.Fatalf("expected approved, got %q", state.Status)
	}
	if !notifier.recipients()[memberID] {
		t.Fatal("expected submitter to be notified of approval")
	}
	if notifier.recipients()[ownerID] {
		t.Fatal("actor (owner) must not be notified")
	}
}

func TestApprove_ByMember_Forbidden(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.ApproveTrip(ctxFor(memberID), trip.ID, appdto.ApprovalDecisionInput{})
	if err != apperrs.ErrForbidden {
		t.Fatalf("expected forbidden for member approving, got %v", err)
	}
}

func TestApprove_NotPending_Conflict(t *testing.T) {
	trip := workspaceTrip(t, "draft")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "draft", nil)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.ApproveTrip(ctxFor(ownerID), trip.ID, appdto.ApprovalDecisionInput{})
	var conflict *apperrs.ConflictError
	if err == nil || !asErr(err, &conflict) {
		t.Fatalf("expected conflict approving a draft, got %v", err)
	}
}

func TestRequestChanges_RequiresNote(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.RequestTripChanges(ctxFor(ownerID), trip.ID, appdto.ApprovalDecisionInput{DecisionNote: "  "})
	var invalid *apperrs.InvalidInputError
	if err == nil || !asErr(err, &invalid) {
		t.Fatalf("expected invalid input when note missing, got %v", err)
	}

	state, err := svc.RequestTripChanges(ctxFor(ownerID), trip.ID, appdto.ApprovalDecisionInput{DecisionNote: "please fix"})
	if err != nil {
		t.Fatalf("request changes: %v", err)
	}
	if state.Status != "changes_requested" {
		t.Fatalf("expected changes_requested, got %q", state.Status)
	}
}

func TestCancel_BySubmitter(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	state, err := svc.CancelTripApproval(ctxFor(memberID), trip.ID, appdto.CancelApprovalInput{Note: "not ready"})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if state.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %q", state.Status)
	}
}

func TestCancel_ByAnotherMember_Forbidden(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.CancelTripApproval(ctxFor(otherMemID), trip.ID, appdto.CancelApprovalInput{})
	if err != apperrs.ErrForbidden {
		t.Fatalf("expected forbidden when another member cancels, got %v", err)
	}
}

func TestCancel_ByOwner_Allowed(t *testing.T) {
	trip := workspaceTrip(t, "pending_approval")
	repo := &mockRepo{getByIDResult: trip, approvalFields: approvalFields(trip.ID, "pending_approval", &memberID)}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	state, err := svc.CancelTripApproval(ctxFor(ownerID), trip.ID, appdto.CancelApprovalInput{})
	if err != nil {
		t.Fatalf("owner cancel: %v", err)
	}
	if state.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %q", state.Status)
	}
}

func TestResetApprovalIfApproved_RecordsEvent(t *testing.T) {
	tripID := uuid.New()
	repo := &mockRepo{
		resetResult:    &entity.ApprovalResetResult{Reset: true, FromStatus: "approved", ToStatus: "draft", WorkspaceID: wsID},
		approvalFields: approvalFields(tripID, "draft", &memberID),
	}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	svc.ResetApprovalIfApproved(context.Background(), tripID, ownerID, "Itinerary changed")

	if len(repo.insertedApprovalEvents) != 1 || repo.insertedApprovalEvents[0].EventType != "reset_to_draft" {
		t.Fatalf("expected one reset_to_draft event, got %+v", repo.insertedApprovalEvents)
	}
}

func TestResetApprovalIfApproved_NoResetWhenInactive(t *testing.T) {
	repo := &mockRepo{resetResult: &entity.ApprovalResetResult{Reset: false}}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	svc.ResetApprovalIfApproved(context.Background(), uuid.New(), ownerID, "comment added")

	if len(repo.insertedApprovalEvents) != 0 {
		t.Fatal("no event should be recorded when nothing was reset")
	}
}

func TestMaterialEdit_TriggersReset(t *testing.T) {
	trip := workspaceTrip(t, "approved")
	repo := &mockRepo{getByIDResult: trip}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	if _, err := svc.UpdateTripBudget(ctxFor(memberID), trip.ID, appdto.UpdateTripBudgetInput{Amount: floatPtr(1200), Currency: "EUR"}); err != nil {
		t.Fatalf("update budget: %v", err)
	}
	if repo.resetCalls != 1 {
		t.Fatalf("expected budget update to trigger one reset, got %d", repo.resetCalls)
	}
}

func TestQueue_NonMember_Denied(t *testing.T) {
	repo := &mockRepo{}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	_, err := svc.ListWorkspaceApprovals(ctxFor(uuid.New()), wsID, appdto.ListWorkspaceApprovalsInput{})
	if err != apperrs.ErrForbidden {
		t.Fatalf("expected forbidden for non-member, got %v", err)
	}
}

func TestQueue_Member_CanList_WithCountsAndFilter(t *testing.T) {
	repo := &mockRepo{
		workspaceApprovalRows: []entity.WorkspaceApprovalRow{
			{TripID: uuid.New(), WorkspaceID: wsID, Destination: "Tokyo", ApprovalStatus: "pending_approval", BudgetCurrency: "EUR"},
		},
		workspaceApprovalCounts: entity.WorkspaceApprovalCounts{PendingApproval: 1, ChangesRequested: 2, Approved: 3, Draft: 4},
	}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	resp, err := svc.ListWorkspaceApprovals(ctxFor(memberID), wsID, appdto.ListWorkspaceApprovalsInput{Status: "pending_approval"})
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(resp.Approvals) != 1 || resp.Approvals[0].Destination != "Tokyo" {
		t.Fatalf("unexpected approvals: %+v", resp.Approvals)
	}
	if resp.Counts.PendingApproval != 1 || resp.Counts.ChangesRequested != 2 || resp.Counts.Approved != 3 || resp.Counts.Draft != 4 {
		t.Fatalf("unexpected counts: %+v", resp.Counts)
	}
	if len(repo.listApprovalsParams.Statuses) != 1 || repo.listApprovalsParams.Statuses[0] != "pending_approval" {
		t.Fatalf("expected status filter to be applied, got %+v", repo.listApprovalsParams.Statuses)
	}
}

func TestQueue_DefaultStatusFocusesActiveSet(t *testing.T) {
	repo := &mockRepo{}
	svc := newApprovalService(repo, approvalTestProvider(), &fakeNotifier{})

	if _, err := svc.ListWorkspaceApprovals(ctxFor(memberID), wsID, appdto.ListWorkspaceApprovalsInput{}); err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(repo.listApprovalsParams.Statuses) != 3 {
		t.Fatalf("expected default active set of 3 statuses, got %+v", repo.listApprovalsParams.Statuses)
	}
}

// asErr is a tiny errors.As wrapper so table tests stay terse.
func asErr(err error, target any) bool {
	return errors.As(err, target)
}

func floatPtr(v float64) *float64 { return &v }
