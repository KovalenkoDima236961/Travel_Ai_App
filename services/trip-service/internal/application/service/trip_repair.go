package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func (s *Service) ListTripRepairProposals(
	ctx context.Context,
	tripID uuid.UUID,
	status string,
	limit int,
) ([]entity.TripRepairProposal, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}

	var filter *entity.TripRepairProposalStatus
	if trimmed := strings.TrimSpace(status); trimmed != "" {
		normalized := entity.TripRepairProposalStatus(strings.ToLower(trimmed))
		if !validTripRepairProposalStatus(normalized) {
			return nil, 0, apperrs.NewInvalidInput("status is invalid")
		}
		filter = &normalized
	}
	proposals, err := s.repo.ListTripRepairProposalsByTrip(ctx, tripID, filter, limit)
	return proposals, limit, err
}

func (s *Service) GetTripRepairProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	return s.repo.GetTripRepairProposalByIDAndTrip(ctx, proposalID, tripID)
}

func (s *Service) RepairItineraryForActor(
	ctx context.Context,
	tripID, actorUserID uuid.UUID,
	jobID *uuid.UUID,
	expectedRevision int,
	payload triprepair.JobPayload,
) (*entity.Trip, json.RawMessage, error) {
	ctx = contextWithActor(ctx, actorUserID)
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if current.WorkspaceID == nil {
		return nil, nil, apperrs.NewInvalidInput("not_supported_for_personal_trips")
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, nil, err
	}
	currentItinerary, err := currentItineraryFromTrip(current)
	if err != nil {
		return nil, nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, tripID)
	if err != nil {
		return nil, nil, err
	}
	weatherForecast, err := s.loadWeatherContext(ctx, *current, tripID)
	if err != nil {
		return nil, nil, err
	}
	constraints, err := s.buildPlanningConstraints(
		ctx,
		user,
		planningconstraints.SourcePolicyRepair,
		current,
		planningconstraints.RequestOverride{Prompt: &planningconstraints.Prompt{UserPrompt: payload.SpecialInstructions}},
		userContext,
		false,
	)
	if err != nil {
		return nil, nil, err
	}
	policy, policyEvaluation, err := s.activePolicyAndEvaluation(ctx, current)
	if err != nil {
		return nil, nil, err
	}
	risk := s.calculateApprovalRiskForTrip(ctx, user.ID, current, true)
	issues := triprepair.BuildIssues(policyEvaluation, risk, payload)
	if len(issues) == 0 {
		return nil, nil, apperrs.NewInvalidInput("no_repairable_issues")
	}

	content, err := s.generator.RepairItinerary(ctx, triprepair.Input{
		Trip:                *current,
		CurrentItinerary:    currentItinerary,
		TripContext:         repairTripContext(*current),
		Policy:              policy,
		PolicyEvaluation:    policyEvaluation,
		ApprovalRisk:        risk,
		Issues:              issues,
		Constraints:         payload,
		UserProfile:         userContext.Profile,
		UserPreferences:     userContext.Preferences,
		WeatherForecast:     weatherForecast,
		PlanningConstraints: constraints,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := triprepair.NormalizeProposalContent(content, currentItinerary, *current, payload); err != nil {
		return nil, nil, err
	}

	proposedRaw, err := json.Marshal(content.RepairedItinerary)
	if err != nil {
		return nil, nil, err
	}
	normalizedRaw, err := validateAndNormalizeItinerary(proposedRaw)
	if err != nil {
		return nil, nil, apperrs.NewDependencyError("validation_failed")
	}
	if err := json.Unmarshal(normalizedRaw, &content.RepairedItinerary); err != nil {
		return nil, nil, err
	}

	proposedTrip := *current
	proposedTrip.Itinerary = normalizedRaw
	proposedPolicyEvaluation, err := s.evaluateTripPolicyForTrip(ctx, &proposedTrip)
	if err != nil {
		content.Validation.Warnings = append(content.Validation.Warnings, "Proposed policy status could not be recalculated.")
		proposedPolicyEvaluation = workspacepolicies.NotApplicableEvaluation(current.ID, current.WorkspaceID, "policy_evaluation_failed")
	}
	proposedRisk := s.calculateApprovalRiskForTrip(ctx, user.ID, &proposedTrip, true)

	basePolicyStatus := string(policyEvaluation.Status)
	proposedPolicyStatus := string(proposedPolicyEvaluation.Status)
	contentRaw, err := json.Marshal(content)
	if err != nil {
		return nil, nil, err
	}
	issuesRaw, err := json.Marshal(issues)
	if err != nil {
		return nil, nil, err
	}

	proposal, err := s.repo.CreateTripRepairProposal(ctx, &entity.TripRepairProposal{
		ID:                    uuid.New(),
		TripID:                tripID,
		JobID:                 jobID,
		CreatedByUserID:       user.ID,
		Status:                entity.TripRepairProposalStatusPending,
		RepairMode:            string(payload.RepairMode),
		BaseItineraryRevision: current.ItineraryRevision,
		BaseRiskScore:         cloneIntPtr(risk.Score),
		ProposedRiskScore:     cloneIntPtr(proposedRisk.Score),
		BasePolicyStatus:      &basePolicyStatus,
		ProposedPolicyStatus:  &proposedPolicyStatus,
		IssuesJSON:            issuesRaw,
		ProposalJSON:          contentRaw,
	})
	if err != nil {
		return nil, nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripRepairProposalCreated,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(proposal.ID),
		Metadata: map[string]any{
			"proposalId":        proposal.ID.String(),
			"jobId":             idString(jobID),
			"repairMode":        proposal.RepairMode,
			"baseRiskScore":     valueOrNilInt(proposal.BaseRiskScore),
			"proposedRiskScore": valueOrNilInt(proposal.ProposedRiskScore),
			"changedItemCount":  content.RepairSummary.ChangedItemCount,
			"warningCount":      len(content.RepairSummary.Warnings),
		},
	})

	resultRaw, err := json.Marshal(triprepair.JobResultPayload{ProposalID: proposal.ID})
	if err != nil {
		return nil, nil, err
	}
	return current, resultRaw, nil
}

func (s *Service) ApplyTripRepairProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
	expectedItineraryRevision *int,
) (*entity.Trip, *entity.TripRepairProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(expectedItineraryRevision)
	if err != nil {
		return nil, nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, nil, err
	}
	proposal, err := s.repo.GetTripRepairProposalByIDAndTrip(ctx, proposalID, tripID)
	if err != nil {
		return nil, nil, err
	}
	if proposal.Status != entity.TripRepairProposalStatusPending {
		return nil, nil, apperrs.NewInvalidInput("proposal is not pending")
	}
	if proposal.BaseItineraryRevision != expectedRevision || proposal.BaseItineraryRevision != current.ItineraryRevision {
		_, _ = s.repo.ExpirePendingTripRepairProposalsForTripRevision(ctx, tripID, current.ItineraryRevision)
		return nil, nil, apperrs.NewItineraryConflict(current.ItineraryRevision)
	}
	currentItinerary, err := currentItineraryFromTrip(current)
	if err != nil {
		return nil, nil, err
	}
	var content triprepair.ProposalContent
	if err := json.Unmarshal(proposal.ProposalJSON, &content); err != nil {
		return nil, nil, apperrs.NewInvalidInput("proposal is invalid")
	}
	payload := triprepair.DefaultJobPayload(triprepair.RepairMode(proposal.RepairMode))
	if err := triprepair.NormalizeProposalContent(&content, currentItinerary, *current, payload); err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(content.RepairedItinerary)
	if err != nil {
		return nil, nil, err
	}
	normalizedRaw, err := validateAndNormalizeItinerary(raw)
	if err != nil {
		return nil, nil, apperrs.NewInvalidInput("proposal itinerary is invalid")
	}
	var normalizedItinerary aggregate.Itinerary
	if err := json.Unmarshal(normalizedRaw, &normalizedItinerary); err != nil {
		return nil, nil, apperrs.NewInvalidInput("proposal itinerary is invalid")
	}
	reliableItinerary, metadata, _, err := s.validateGeneratedItinerary(
		ctx,
		*current,
		normalizedItinerary,
		entity.ItineraryVersionSourceAIPolicyRepairApplied,
		map[string]any{
			"source":            "ai_policy_repair",
			"proposalId":        proposal.ID.String(),
			"repairMode":        proposal.RepairMode,
			"baseRevision":      proposal.BaseItineraryRevision,
			"baseRiskScore":     valueOrNilInt(proposal.BaseRiskScore),
			"proposedRiskScore": valueOrNilInt(proposal.ProposedRiskScore),
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, nil, err
	}
	reliableRaw, err := json.Marshal(reliableItinerary)
	if err != nil {
		return nil, nil, err
	}
	updated, err := s.saveItineraryWithVersion(
		ctx,
		tripID,
		ownerID,
		user.ID,
		reliableRaw,
		expectedRevision,
		entity.ItineraryVersionSourceAIPolicyRepairApplied,
		metadata,
	)
	if err != nil {
		return nil, nil, err
	}
	applied, err := s.repo.MarkTripRepairProposalApplied(ctx, proposal.ID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	_, _ = s.repo.ExpirePendingTripRepairProposalsForTripRevision(ctx, tripID, updated.ItineraryRevision)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripRepairProposalApplied,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(proposal.ID),
		Metadata: map[string]any{
			"proposalId":        proposal.ID.String(),
			"repairMode":        proposal.RepairMode,
			"baseRiskScore":     valueOrNilInt(proposal.BaseRiskScore),
			"proposedRiskScore": valueOrNilInt(proposal.ProposedRiskScore),
		},
	})
	return updated, applied, nil
}

func (s *Service) DiscardTripRepairProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
) (*entity.TripRepairProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	proposal, err := s.repo.GetTripRepairProposalByIDAndTrip(ctx, proposalID, tripID)
	if err != nil {
		return nil, err
	}
	if proposal.Status != entity.TripRepairProposalStatusPending {
		return proposal, nil
	}
	discarded, err := s.repo.MarkTripRepairProposalDiscarded(ctx, proposalID, user.ID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return proposal, nil
		}
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripRepairProposalDiscarded,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(discarded.ID),
		Metadata: map[string]any{
			"proposalId": discarded.ID.String(),
			"repairMode": discarded.RepairMode,
		},
	})
	return discarded, nil
}

func (s *Service) activePolicyAndEvaluation(
	ctx context.Context,
	trip *entity.Trip,
) (*workspacepolicies.Policy, workspacepolicies.Evaluation, error) {
	evaluation, err := s.evaluateTripPolicyForTrip(ctx, trip)
	if err != nil {
		return nil, workspacepolicies.Evaluation{}, err
	}
	if trip == nil || trip.WorkspaceID == nil || s.workspacePolicyProvider == nil {
		return nil, evaluation, nil
	}
	policy, err := s.workspacePolicyProvider.GetActive(ctx, *trip.WorkspaceID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, evaluation, nil
		}
		return nil, workspacepolicies.Evaluation{}, err
	}
	return policy, evaluation, nil
}

func currentItineraryFromTrip(trip *entity.Trip) (aggregate.Itinerary, error) {
	if trip == nil || len(trip.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(trip.Itinerary)), "null") {
		return aggregate.Itinerary{}, currentItineraryInvalidError()
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(trip.Itinerary, &itinerary); err != nil {
		return aggregate.Itinerary{}, currentItineraryInvalidError()
	}
	if err := validateCurrentItinerary(itinerary); err != nil {
		return aggregate.Itinerary{}, err
	}
	return itinerary, nil
}

func repairTripContext(trip entity.Trip) triprepair.TripContext {
	ctx := triprepair.TripContext{
		Destination:  trip.Destination,
		DurationDays: trip.Days,
		Travelers:    trip.Travelers,
		Pace:         trip.Pace,
	}
	if trip.StartDate != nil {
		ctx.StartDate = trip.StartDate.Format("2006-01-02")
	}
	if trip.BudgetAmount != nil {
		ctx.Budget = &triprepair.Money{Amount: *trip.BudgetAmount, Currency: trip.BudgetCurrency}
	}
	return ctx
}

func validTripRepairProposalStatus(status entity.TripRepairProposalStatus) bool {
	switch status {
	case entity.TripRepairProposalStatusPending,
		entity.TripRepairProposalStatusApplied,
		entity.TripRepairProposalStatusDiscarded,
		entity.TripRepairProposalStatusExpired,
		entity.TripRepairProposalStatusFailed:
		return true
	default:
		return false
	}
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func valueOrNilInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
