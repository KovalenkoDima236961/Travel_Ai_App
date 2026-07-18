package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/personalization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/recap"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
)

const (
	maxRecapTextChars     = 4000
	maxRecapArrayItems    = 12
	maxRecapFeedbackLabel = 240
)

type recapRepository interface {
	CreateTripRecap(context.Context, *entity.TripRecap) (*entity.TripRecap, error)
	GetActiveTripRecap(context.Context, uuid.UUID) (*entity.TripRecap, error)
	UpdateTripRecap(context.Context, *entity.TripRecap) (*entity.TripRecap, error)
	ArchiveTripRecap(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*entity.TripRecap, error)
	CreateTripRecapFeedback(context.Context, *entity.TripRecapFeedback) (*entity.TripRecapFeedback, error)
	ListTripRecapFeedback(context.Context, uuid.UUID) ([]entity.TripRecapFeedback, error)
	ApproveTripRecapFeedback(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*entity.TripRecapFeedback, error)
}

func (s *Service) recapRepo() (recapRepository, error) {
	repository, ok := s.repo.(recapRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip recaps are not configured")
	}
	return repository, nil
}

func (s *Service) GetTripRecapStatus(ctx context.Context, tripID uuid.UUID) (appdto.TripRecapStatusResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripRecapStatusResponse{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripRecapStatusResponse{}, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.TripRecapStatusResponse{}, err
	}
	recapRow, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return appdto.TripRecapStatusResponse{}, err
	}
	endedAt := recapTripEndDate(trip)
	response := appdto.TripRecapStatusResponse{
		Eligible:    recapEligible(trip, time.Now().UTC()),
		Reason:      recapEligibilityReason(trip, time.Now().UTC()),
		HasRecap:    recapRow != nil && err == nil,
		CanGenerate: access.CanEdit() && s.recapEnabled,
		CanEdit:     access.CanEdit() && recapRow != nil && err == nil,
	}
	if recapRow != nil && err == nil {
		value := recapRow.ID
		response.RecapID = &value
	}
	if endedAt != nil {
		value := endedAt.Format("2006-01-02")
		response.TripEndedAt = &value
	}
	return response, nil
}

func (s *Service) GetTripRecap(ctx context.Context, tripID uuid.UUID) (appdto.GetTripRecapResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.GetTripRecapResponse{}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.GetTripRecapResponse{}, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.GetTripRecapResponse{}, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return appdto.GetTripRecapResponse{}, err
	}
	feedback, err := repository.ListTripRecapFeedback(ctx, stored.ID)
	if err != nil {
		return appdto.GetTripRecapResponse{}, err
	}
	return appdto.GetTripRecapResponse{
		Recap:       recapView(stored),
		Permissions: appdto.RecapPermissions{CanEdit: access.CanEdit(), CanFinalize: access.CanEdit(), CanCreateTemplate: access.CanEdit(), CanApplyLearning: true},
		Feedback:    recapFeedbackViews(feedback, user.ID),
	}, nil
}

func (s *Service) GenerateTripRecap(ctx context.Context, tripID uuid.UUID, input appdto.GenerateTripRecapInput) (appdto.TripRecapView, error) {
	if !s.recapEnabled {
		return appdto.TripRecapView{}, apperrs.NewDependencyError("trip recap is disabled")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	if !recapEligible(trip, time.Now().UTC()) && !input.GenerateEarly {
		return appdto.TripRecapView{}, apperrs.NewInvalidInput("trip recap is available after the trip ends; set generateEarly to confirm early generation")
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	existing, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return appdto.TripRecapView{}, err
	}
	if err == nil && existing != nil && !input.ForceRegenerate {
		return recapView(existing), nil
	}

	source, err := s.buildRecapSourceSummary(ctx, trip)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	generated, metadata, fallback, err := s.generateRecap(ctx, source, input.Language)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	if err := validateRecapJSON(generated); err != nil {
		return appdto.TripRecapView{}, err
	}
	recapRaw, _ := json.Marshal(generated)
	sourceRaw, _ := json.Marshal(source)
	metadata["fallbackUsed"] = fallback
	metadataRaw, _ := json.Marshal(metadata)

	var saved *entity.TripRecap
	if existing != nil && err == nil {
		existing.RecapJSON, existing.SourceSummary, existing.AIMetadata = recapRaw, sourceRaw, metadataRaw
		existing.Status = entity.TripRecapStatusGenerated
		existing.UpdatedByUserID = &user.ID
		existing.FinalizedAt = nil
		saved, err = repository.UpdateTripRecap(ctx, existing)
	} else {
		saved, err = repository.CreateTripRecap(ctx, &entity.TripRecap{ID: uuid.New(), TripID: tripID, CreatedByUserID: user.ID, UpdatedByUserID: &user.ID, Status: entity.TripRecapStatusGenerated, RecapJSON: recapRaw, SourceSummary: sourceRaw, AIMetadata: metadataRaw})
	}
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	recapGeneratedTotal.WithLabelValues(recapMetricLabels(recapGenerationMode(s.recapAIEnabled, fallback), "success", source.Trip.TripType, recapAIProvider(s.recapAIEnabled, fallback), fallback)...).Inc()
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripRecapGenerated, EntityType: activityEntityType(activity.EntityTripRecap), EntityID: activityEntityID(saved.ID), Metadata: map[string]any{"fallbackUsed": fallback}})
	return recapView(saved), nil
}

func (s *Service) UpdateTripRecap(ctx context.Context, tripID uuid.UUID, recapJSON appdto.RecapJSON) (appdto.TripRecapView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	_, _, err = s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	if err := validateRecapJSON(recapJSON); err != nil {
		return appdto.TripRecapView{}, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	raw, _ := json.Marshal(recapJSON)
	stored.RecapJSON, stored.UpdatedByUserID = raw, &user.ID
	if stored.Status != entity.TripRecapStatusFinalized {
		stored.Status = entity.TripRecapStatusEdited
	}
	updated, err := repository.UpdateTripRecap(ctx, stored)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	recapUpdatedTotal.Inc()
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripRecapUpdated, EntityType: activityEntityType(activity.EntityTripRecap), EntityID: activityEntityID(updated.ID)})
	return recapView(updated), nil
}

func (s *Service) FinalizeTripRecap(ctx context.Context, tripID uuid.UUID) (appdto.TripRecapView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	_, _, err = s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	now := time.Now().UTC()
	stored.Status, stored.FinalizedAt, stored.UpdatedByUserID = entity.TripRecapStatusFinalized, &now, &user.ID
	updated, err := repository.UpdateTripRecap(ctx, stored)
	if err != nil {
		return appdto.TripRecapView{}, err
	}
	recapFinalizedTotal.Inc()
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripRecapFinalized, EntityType: activityEntityType(activity.EntityTripRecap), EntityID: activityEntityID(updated.ID)})
	return recapView(updated), nil
}

func (s *Service) ArchiveTripRecap(ctx context.Context, tripID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, _, err = s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return err
	}
	_, err = repository.ArchiveTripRecap(ctx, tripID, stored.ID, user.ID)
	return err
}

func (s *Service) SubmitTripRecapFeedback(ctx context.Context, tripID uuid.UUID, input appdto.SubmitRecapFeedbackInput) (appdto.RecapFeedbackView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.RecapFeedbackView{}, err
	}
	_, _, err = s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.RecapFeedbackView{}, err
	}
	if !validRecapFeedbackType(input.FeedbackType) || strings.TrimSpace(input.Label) == "" {
		return appdto.RecapFeedbackView{}, apperrs.NewInvalidInput("feedbackType and label are required")
	}
	repository, err := s.recapRepo()
	if err != nil {
		return appdto.RecapFeedbackView{}, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return appdto.RecapFeedbackView{}, err
	}
	feedback, err := repository.CreateTripRecapFeedback(ctx, &entity.TripRecapFeedback{ID: uuid.New(), TripID: tripID, RecapID: stored.ID, UserID: user.ID, FeedbackType: input.FeedbackType, EntityType: optionalRecapString(input.EntityType), EntityID: optionalRecapString(input.EntityID), Label: recapTrim(input.Label, maxRecapFeedbackLabel), Value: optionalRecapString(input.Value), ApprovedForPersonalization: input.ApprovedForPersonalization, Metadata: sanitizeRecapFeedbackMetadata(input.Metadata)})
	if err != nil {
		return appdto.RecapFeedbackView{}, err
	}
	if input.ApprovedForPersonalization {
		if err := s.applyFeedbackToPersonalization(ctx, user.ID, feedback); err != nil {
			return appdto.RecapFeedbackView{}, err
		}
	}
	return recapFeedbackView(feedback), nil
}

func (s *Service) ApplyTripRecapLearning(ctx context.Context, tripID uuid.UUID, input appdto.ApplyRecapLearningInput) ([]appdto.RecapFeedbackView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return nil, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return nil, err
	}
	existing, err := repository.ListTripRecapFeedback(ctx, stored.ID)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]entity.TripRecapFeedback{}
	for _, item := range existing {
		byID[item.ID] = item
	}
	applied := 0
	for _, id := range input.FeedbackIDs {
		item, ok := byID[id]
		if !ok || item.UserID != user.ID {
			return nil, apperrs.ErrForbidden
		}
		if err := s.applyFeedbackToPersonalization(ctx, user.ID, &item); err != nil {
			return nil, err
		}
		if !item.ApprovedForPersonalization {
			updated, approveErr := repository.ApproveTripRecapFeedback(ctx, stored.ID, id, user.ID)
			if approveErr != nil {
				return nil, approveErr
			}
			byID[id] = *updated
		}
		applied++
	}
	for _, candidate := range input.LearningCandidates {
		if !validRecapFeedbackType(candidate.FeedbackType) || strings.TrimSpace(candidate.Label) == "" {
			return nil, apperrs.NewInvalidInput("invalid learning candidate")
		}
		created, createErr := repository.CreateTripRecapFeedback(ctx, &entity.TripRecapFeedback{ID: uuid.New(), TripID: tripID, RecapID: stored.ID, UserID: user.ID, FeedbackType: candidate.FeedbackType, EntityType: optionalRecapString(candidate.EntityType), EntityID: optionalRecapString(candidate.EntityID), Label: recapTrim(candidate.Label, maxRecapFeedbackLabel), Value: optionalRecapString(candidate.Value), ApprovedForPersonalization: true, Metadata: sanitizeRecapFeedbackMetadata(candidate.Metadata)})
		if createErr != nil {
			return nil, createErr
		}
		if applyErr := s.applyFeedbackToPersonalization(ctx, user.ID, created); applyErr != nil {
			return nil, applyErr
		}
		byID[created.ID] = *created
		applied++
	}
	if applied > 0 {
		recapLearningAppliedTotal.Add(float64(applied))
		s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripRecapLearningApplied, EntityType: activityEntityType(activity.EntityTripRecap), EntityID: activityEntityID(stored.ID), Metadata: map[string]any{"count": applied, "workspaceTrip": trip.WorkspaceID != nil}})
	}
	all, err := repository.ListTripRecapFeedback(ctx, stored.ID)
	if err != nil {
		return nil, err
	}
	return recapFeedbackViews(all, user.ID), nil
}

func (s *Service) CreateTemplateFromTripRecap(ctx context.Context, tripID uuid.UUID, input appdto.CreateTemplateFromRecapInput) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	repository, err := s.recapRepo()
	if err != nil {
		return nil, err
	}
	stored, err := repository.GetActiveTripRecap(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, apperrs.NewInvalidInput("template title is required")
	}
	description := recapTrim(input.Description, 1000)
	if input.UseRecapLessons {
		recapJSON, decodeErr := decodeRecap(stored)
		if decodeErr != nil {
			return nil, decodeErr
		}
		if lessons := safeRecapLessons(recapJSON.LessonsLearned); len(lessons) > 0 {
			lessonText := "Lessons for a similar trip: " + strings.Join(lessons, "; ")
			if description != "" {
				description += "\n\n"
			}
			description = recapTrim(description+lessonText, 1000)
		}
	}
	workspaceID := (*uuid.UUID)(nil)
	if input.Visibility == entity.TripTemplateVisibilityWorkspace {
		workspaceID = trip.WorkspaceID
	}
	template, err := s.SaveTripAsTemplate(ctx, tripID, appdto.SaveTripAsTemplateInput{Title: input.Title, Description: optionalRecapString(description), Visibility: input.Visibility, WorkspaceID: workspaceID, DestinationHint: optionalRecapString(trip.Destination), DefaultCurrency: optionalRecapString(trip.BudgetCurrency), Tags: input.Tags})
	if err != nil {
		return nil, err
	}
	recapTemplateCreatedTotal.Inc()
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: tripID, ActorUserID: &user.ID, EventType: activity.EventTripTemplateCreatedFromRecap, EntityType: activityEntityType(activity.EntityTripTemplate), EntityID: activityEntityID(template.Template.ID), Metadata: map[string]any{"recapId": stored.ID.String(), "visibility": string(template.Template.Visibility)}})
	return template, nil
}

func (s *Service) generateRecap(ctx context.Context, source recap.SourceSummary, language string) (appdto.RecapJSON, map[string]any, bool, error) {
	if strings.TrimSpace(language) == "" {
		language = "en"
	}
	if s.recapAIEnabled && s.recapClient != nil {
		started := time.Now()
		response, err := s.recapClient.Generate(ctx, recap.GenerateRequest{Language: language, SourceSummary: source, Style: "concise", IncludeLearningCandidates: true})
		status := "success"
		if err != nil {
			status = "error"
		} else if validateRecapJSON(response.Recap) != nil {
			status = "validation_error"
		}
		recapGenerationDuration.WithLabelValues(recapMetricLabels("ai", status, source.Trip.TripType, "ai_planning", false)...).Observe(time.Since(started).Seconds())
		if err == nil {
			if validateErr := validateRecapJSON(response.Recap); validateErr == nil {
				return response.Recap, map[string]any{"mode": "ai", "warnings": response.Warnings, "assumptions": response.Assumptions}, false, nil
			}
			recapAIFailuresTotal.WithLabelValues(recapMetricLabels("ai", "validation_error", source.Trip.TripType, "ai_planning", false)...).Inc()
		} else {
			recapAIFailuresTotal.WithLabelValues(recapMetricLabels("ai", "error", source.Trip.TripType, "ai_planning", false)...).Inc()
			s.log.Warn("trip recap AI generation failed", zap.Error(err))
		}
	}
	if !s.recapFailOpen {
		return appdto.RecapJSON{}, nil, false, fmt.Errorf("trip recap generation is unavailable")
	}
	recapFallbacksTotal.WithLabelValues(recapMetricLabels("deterministic", "success", source.Trip.TripType, recapAIProvider(s.recapAIEnabled, true), true)...).Inc()
	return deterministicRecap(source), map[string]any{"mode": "deterministic"}, true, nil
}

func (s *Service) buildRecapSourceSummary(ctx context.Context, trip *entity.Trip) (recap.SourceSummary, error) {
	result := recap.SourceSummary{ItineraryOutcome: recap.SourceItineraryOutcome{TopCompletedItems: []string{}, TopSkippedItems: []string{}}, BudgetOutcome: recap.SourceBudgetOutcome{TopCategories: []appdto.RecapCategoryTotal{}}, RouteOutcome: recap.SourceRouteOutcome{Stops: []string{}, TransportModes: []string{}, Issues: []string{}}, ChecklistOutcome: recap.SourceChecklistOutcome{}, VerificationOutcome: recap.SourceVerificationOutcome{Issues: []string{}}, LearningCandidates: []appdto.LearningCandidate{}}
	result.Trip = recap.SourceTrip{Title: recapTrim(trip.Destination+" Trip", 160), Destination: recapTrim(trip.Destination, 160), DurationDays: int(trip.Days), TripType: trip.TripType}
	if trip.StartDate != nil {
		result.Trip.StartDate = trip.StartDate.Format("2006-01-02")
	}
	if end := recapTripEndDate(trip); end != nil {
		result.Trip.EndDate = end.Format("2006-01-02")
	}

	var itinerary aggregate.Itinerary
	_ = json.Unmarshal(trip.Itinerary, &itinerary)
	for dayIndex, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			result.ItineraryOutcome.PlannedItemCount++
			status := "unknown"
			if item.TravelStatus != nil {
				status = strings.ToLower(strings.TrimSpace(item.TravelStatus.Status))
			}
			switch status {
			case "done", "completed", "complete":
				result.ItineraryOutcome.DoneItemCount++
				if len(result.ItineraryOutcome.TopCompletedItems) < 5 {
					result.ItineraryOutcome.TopCompletedItems = append(result.ItineraryOutcome.TopCompletedItems, recapTrim(item.Name, 120))
				}
			case "skipped", "cancelled", "canceled":
				result.ItineraryOutcome.SkippedItemCount++
				if len(result.ItineraryOutcome.TopSkippedItems) < 5 {
					result.ItineraryOutcome.TopSkippedItems = append(result.ItineraryOutcome.TopSkippedItems, recapTrim(item.Name, 120))
				}
			case "delayed":
				result.ItineraryOutcome.DelayedItemCount++
			default:
				result.ItineraryOutcome.UnknownItemCount++
			}
			_ = dayIndex
			_ = itemIndex
		}
	}
	if trip.BudgetAmount != nil {
		result.BudgetOutcome.PlannedTotal = &appdto.RecapMoney{Amount: *trip.BudgetAmount, Currency: recapCurrency(trip.BudgetCurrency)}
	}
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, trip.ID, appdto.ListExpensesInput{Limit: 1000})
	if err != nil {
		return recap.SourceSummary{}, err
	}
	actual, categories := 0.0, map[string]float64{}
	currency := recapCurrency(trip.BudgetCurrency)
	for _, expense := range expenses {
		if expense.Status != entity.ExpenseStatusActive || !strings.EqualFold(expense.Currency, currency) {
			continue
		}
		actual += expense.Amount
		categories[string(expense.Category)] += expense.Amount
	}
	if len(expenses) > 0 {
		result.BudgetOutcome.ActualTotal = &appdto.RecapMoney{Amount: actual, Currency: currency}
	}
	if result.BudgetOutcome.PlannedTotal != nil && result.BudgetOutcome.ActualTotal != nil {
		result.BudgetOutcome.Variance = &appdto.RecapMoney{Amount: actual - result.BudgetOutcome.PlannedTotal.Amount, Currency: currency}
	}
	for category, total := range categories {
		result.BudgetOutcome.TopCategories = append(result.BudgetOutcome.TopCategories, appdto.RecapCategoryTotal{Category: category, Total: appdto.RecapMoney{Amount: total, Currency: currency}})
	}
	sort.Slice(result.BudgetOutcome.TopCategories, func(i, j int) bool {
		return result.BudgetOutcome.TopCategories[i].Total.Amount > result.BudgetOutcome.TopCategories[j].Total.Amount
	})
	if len(result.BudgetOutcome.TopCategories) > 5 {
		result.BudgetOutcome.TopCategories = result.BudgetOutcome.TopCategories[:5]
	}
	receipts, err := s.repo.ListTripExpenseReceipts(ctx, trip.ID, appdto.ListReceiptsInput{Limit: 1000})
	if err != nil {
		return recap.SourceSummary{}, err
	}
	if len(expenses) > 0 {
		result.BudgetOutcome.ReceiptCoveragePercent = int(math.Round(float64(len(receipts)) / float64(len(expenses)) * 100))
	}

	if trip.Route != nil {
		for _, stop := range trip.Route.Stops {
			if value := recapTrim(stop.Destination, 120); value != "" {
				result.RouteOutcome.Stops = append(result.RouteOutcome.Stops, value)
			}
		}
		for _, leg := range trip.Route.Legs {
			mode := strings.TrimSpace(leg.Mode)
			if leg.SelectedTransportOption != nil {
				mode = leg.SelectedTransportOption.Mode
				if strings.TrimSpace(mode) == "" {
					mode = leg.Mode
				}
				result.RouteOutcome.VerifiedTransportCount++
				if len(leg.SelectedTransportOption.Warnings) > 0 {
					result.RouteOutcome.Issues = append(result.RouteOutcome.Issues, "A selected "+mode+" segment needs review.")
				}
			} else if mode != "" {
				result.RouteOutcome.UnverifiedTransportCount++
			}
			result.RouteOutcome.TransportModes = appendUniqueRecap(result.RouteOutcome.TransportModes, mode)
		}
	}
	checklistItems, err := s.repo.ListChecklistItemsByTrip(ctx, trip.ID)
	if err != nil {
		return recap.SourceSummary{}, err
	}
	for _, item := range checklistItems {
		result.ChecklistOutcome.TotalChecklistItems++
		if item.Checked {
			result.ChecklistOutcome.CompletedChecklistItems++
		}
	}
	reminders, err := s.repo.ListTripRemindersByTrip(ctx, trip.ID, entity.TripReminderFilters{})
	if err != nil {
		return recap.SourceSummary{}, err
	}
	for _, reminder := range reminders {
		result.ChecklistOutcome.TotalReminders++
		if reminder.Status == entity.ReminderStatusCompleted {
			result.ChecklistOutcome.CompletedReminders++
		}
	}
	verificationResult, verifyErr := s.GetTripVerification(ctx, trip.ID)
	if verifyErr == nil {
		result.VerificationOutcome.Score, result.VerificationOutcome.VerifiedCount, result.VerificationOutcome.StaleCount, result.VerificationOutcome.MissingCount = verificationResult.Score, verificationResult.Summary.VerifiedCount, verificationResult.Summary.StaleCount, verificationResult.Summary.MissingCount
		result.VerificationOutcome.Summary = fmt.Sprintf("%d verified items; %d stale or missing items need review.", verificationResult.Summary.VerifiedCount, verificationResult.Summary.StaleCount+verificationResult.Summary.MissingCount)
		for _, issue := range verificationResult.TopIssues {
			if issue.Status == verification.StatusVerified || issue.Status == verification.StatusNotApplicable {
				continue
			}
			result.VerificationOutcome.Issues = append(result.VerificationOutcome.Issues, "A "+string(issue.Scope)+" item needs review.")
			if len(result.VerificationOutcome.Issues) == 5 {
				break
			}
		}
	}
	result.LearningCandidates = recapLearningCandidates(result)
	return capRecapSource(result, s.recapMaxSourceChars), nil
}

func recapTripEndDate(trip *entity.Trip) *time.Time {
	if trip == nil || trip.StartDate == nil || trip.Days < 1 {
		return nil
	}
	value := trip.StartDate.UTC().AddDate(0, 0, int(trip.Days)-1)
	return &value
}

func recapEligible(trip *entity.Trip, now time.Time) bool {
	end := recapTripEndDate(trip)
	if end == nil {
		return false
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return end.Before(today)
}

func recapEligibilityReason(trip *entity.Trip, now time.Time) string {
	if recapEligible(trip, now) {
		return "trip_ended"
	}
	if recapTripEndDate(trip) == nil {
		return "trip_dates_missing"
	}
	return "trip_not_ended"
}

func recapCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return defaultCurrency
	}
	return value
}

func capRecapSource(source recap.SourceSummary, maximum int) recap.SourceSummary {
	if maximum <= 0 {
		return source
	}
	raw, _ := json.Marshal(source)
	if len(raw) <= maximum {
		return source
	}
	source.ItineraryOutcome.TopCompletedItems = []string{}
	source.ItineraryOutcome.TopSkippedItems = []string{}
	source.RouteOutcome.Stops = []string{}
	source.RouteOutcome.Issues = []string{}
	source.VerificationOutcome.Issues = []string{}
	return source
}

func deterministicRecap(source recap.SourceSummary) appdto.RecapJSON {
	planned := source.ItineraryOutcome.PlannedItemCount
	completion := 0.0
	if planned > 0 {
		completion = float64(source.ItineraryOutcome.DoneItemCount) / float64(planned)
	}
	skipped, delayed := append([]string{}, source.ItineraryOutcome.TopSkippedItems...), []string{}
	if source.ItineraryOutcome.DelayedItemCount > 0 {
		delayed = append(delayed, fmt.Sprintf("%d itinerary item(s) were marked delayed.", source.ItineraryOutcome.DelayedItemCount))
	}
	budget := appdto.BudgetRecap{PlannedTotal: source.BudgetOutcome.PlannedTotal, ActualTotal: source.BudgetOutcome.ActualTotal, VarianceAmount: source.BudgetOutcome.Variance, ReceiptCoveragePercent: source.BudgetOutcome.ReceiptCoveragePercent, TopCategories: source.BudgetOutcome.TopCategories}
	if source.BudgetOutcome.PlannedTotal != nil && source.BudgetOutcome.Variance != nil && source.BudgetOutcome.PlannedTotal.Amount != 0 {
		percent := source.BudgetOutcome.Variance.Amount / source.BudgetOutcome.PlannedTotal.Amount * 100
		budget.VariancePercent = &percent
	}
	if source.BudgetOutcome.ActualTotal != nil {
		budget.Notes = "Actual spend is based on tracked expenses."
	}
	routeSummary := "No route or selected transport outcomes were recorded."
	if len(source.RouteOutcome.TransportModes) > 0 {
		routeSummary = "Recorded transport modes: " + strings.Join(source.RouteOutcome.TransportModes, ", ") + "."
	}
	verificationSummary := source.VerificationOutcome.Summary
	if verificationSummary == "" {
		verificationSummary = "No verification summary was available."
	}
	checklistNotes := fmt.Sprintf("%d of %d checklist items and %d of %d reminders were completed.", source.ChecklistOutcome.CompletedChecklistItems, source.ChecklistOutcome.TotalChecklistItems, source.ChecklistOutcome.CompletedReminders, source.ChecklistOutcome.TotalReminders)
	lessons := deterministicLessons(source)
	return appdto.RecapJSON{
		SchemaVersion:         appdto.TripRecapSchemaVersion,
		Title:                 source.Trip.Title + " Recap",
		Summary:               fmt.Sprintf("%d of %d planned itinerary items were marked done. This private recap is editable; review details before finalizing.", source.ItineraryOutcome.DoneItemCount, planned),
		Highlights:            recapHighlights(source.ItineraryOutcome.TopCompletedItems),
		PlannedVsActual:       appdto.RecapPlannedVsActual{PlannedItemCount: planned, DoneItemCount: source.ItineraryOutcome.DoneItemCount, SkippedItemCount: source.ItineraryOutcome.SkippedItemCount, DelayedItemCount: source.ItineraryOutcome.DelayedItemCount, UnknownItemCount: source.ItineraryOutcome.UnknownItemCount, CompletionRate: completion, Notes: "Based on recorded Travel Day item statuses.", SkippedItems: skipped, DelayedItems: delayed},
		Budget:                budget,
		RouteAndTransport:     appdto.RouteTransportRecap{Summary: routeSummary, Issues: source.RouteOutcome.Issues, SuccessfulModes: source.RouteOutcome.TransportModes, ProblemModes: []string{}},
		Verification:          appdto.VerificationRecap{Summary: verificationSummary, Issues: source.VerificationOutcome.Issues},
		ChecklistAndReminders: appdto.ChecklistReminderRecap{CompletedChecklistItems: source.ChecklistOutcome.CompletedChecklistItems, TotalChecklistItems: source.ChecklistOutcome.TotalChecklistItems, CompletedReminders: source.ChecklistOutcome.CompletedReminders, TotalReminders: source.ChecklistOutcome.TotalReminders, Notes: checklistNotes},
		LessonsLearned:        lessons,
		FuturePreferences:     source.LearningCandidates,
		TemplateSuggestion:    appdto.TemplateSuggestion{Recommended: completion >= .6, Title: source.Trip.Destination + " trip template", Reason: "A reusable template will keep only the itinerary structure and safe planning details."},
		UserEditableNotes:     "",
	}
}

func recapHighlights(items []string) []appdto.RecapHighlight {
	result := make([]appdto.RecapHighlight, 0, len(items))
	for _, item := range items {
		if item != "" {
			result = append(result, appdto.RecapHighlight{Title: item, Description: "Completed itinerary moment."})
		}
	}
	return result
}

func deterministicLessons(source recap.SourceSummary) []string {
	lessons := []string{}
	if source.ItineraryOutcome.SkippedItemCount > 0 {
		lessons = append(lessons, "Review skipped activities before reusing this itinerary.")
	}
	if source.RouteOutcome.UnverifiedTransportCount > 0 {
		lessons = append(lessons, "Verify selected transport closer to departure for future trips.")
	}
	if source.BudgetOutcome.ReceiptCoveragePercent < 100 && source.BudgetOutcome.ActualTotal != nil {
		lessons = append(lessons, "Add receipts consistently to make future budget comparisons more complete.")
	}
	if len(lessons) == 0 {
		lessons = append(lessons, "Keep tracking statuses and expenses to make the next recap more useful.")
	}
	return limitRecapStrings(lessons, 3, 300)
}

func recapLearningCandidates(source recap.SourceSummary) []appdto.LearningCandidate {
	items := []appdto.LearningCandidate{}
	for _, mode := range source.RouteOutcome.TransportModes {
		if mode == "train" && source.RouteOutcome.UnverifiedTransportCount == 0 {
			items = append(items, appdto.LearningCandidate{FeedbackType: "prefer_next_time", Label: "Prefer train routes", EntityType: "transport_mode", Value: "train", Metadata: map[string]any{"transport": "train"}, Approved: false})
		}
	}
	if source.ItineraryOutcome.SkippedItemCount > 0 {
		items = append(items, appdto.LearningCandidate{FeedbackType: "pace_too_packed", Label: "Leave more room for changes", EntityType: "general", Value: "pace", Metadata: map[string]any{}, Approved: false})
	}
	if source.BudgetOutcome.PlannedTotal != nil && source.BudgetOutcome.Variance != nil && source.BudgetOutcome.Variance.Amount > 0 {
		items = append(items, appdto.LearningCandidate{FeedbackType: "budget_inaccurate", Label: "Review budget estimates for a similar trip", EntityType: "general", Value: "budget", Metadata: map[string]any{"currency": source.BudgetOutcome.Variance.Currency}, Approved: false})
	}
	return items
}

func validateRecapJSON(value appdto.RecapJSON) error {
	if value.SchemaVersion != appdto.TripRecapSchemaVersion {
		return apperrs.NewInvalidInput("recap schemaVersion must be %s", appdto.TripRecapSchemaVersion)
	}
	if recapTrim(value.Title, 160) == "" || recapTrim(value.Summary, maxRecapTextChars) == "" {
		return apperrs.NewInvalidInput("recap title and summary are required")
	}
	if len(value.Highlights) > maxRecapArrayItems || len(value.LessonsLearned) > maxRecapArrayItems || len(value.FuturePreferences) > maxRecapArrayItems {
		return apperrs.NewInvalidInput("recap contains too many items")
	}
	for _, candidate := range value.FuturePreferences {
		if !validRecapFeedbackType(candidate.FeedbackType) {
			return apperrs.NewInvalidInput("recap contains an invalid learning feedback type")
		}
	}
	if value.Budget.PlannedTotal != nil && !validRecapMoney(*value.Budget.PlannedTotal) {
		return apperrs.NewInvalidInput("recap planned budget is invalid")
	}
	if value.Budget.ActualTotal != nil && !validRecapMoney(*value.Budget.ActualTotal) {
		return apperrs.NewInvalidInput("recap actual budget is invalid")
	}
	raw, _ := json.Marshal(value)
	lower := strings.ToLower(string(raw))
	for _, sensitive := range []string{"raw ocr", "receipt ocr text", "share token", "public share password", "api key", "access token"} {
		if strings.Contains(lower, sensitive) {
			return apperrs.NewInvalidInput("recap contains restricted private data")
		}
	}
	return nil
}

func validRecapMoney(value appdto.RecapMoney) bool {
	return !math.IsNaN(value.Amount) && !math.IsInf(value.Amount, 0) && len(strings.TrimSpace(value.Currency)) == 3
}

func decodeRecap(stored *entity.TripRecap) (appdto.RecapJSON, error) {
	var result appdto.RecapJSON
	if err := json.Unmarshal(stored.RecapJSON, &result); err != nil {
		return appdto.RecapJSON{}, fmt.Errorf("decode stored recap: %w", err)
	}
	return result, nil
}

func recapView(stored *entity.TripRecap) appdto.TripRecapView {
	value, _ := decodeRecap(stored)
	return appdto.TripRecapView{ID: stored.ID, TripID: stored.TripID, Status: stored.Status, Recap: value, FinalizedAt: stored.FinalizedAt, CreatedAt: stored.CreatedAt, UpdatedAt: stored.UpdatedAt}
}

func recapFeedbackView(value *entity.TripRecapFeedback) appdto.RecapFeedbackView {
	return appdto.RecapFeedbackView{ID: value.ID, FeedbackType: value.FeedbackType, EntityType: value.EntityType, EntityID: value.EntityID, Label: value.Label, Value: value.Value, ApprovedForPersonalization: value.ApprovedForPersonalization, Metadata: value.Metadata, CreatedAt: value.CreatedAt}
}

func recapFeedbackViews(values []entity.TripRecapFeedback, userID uuid.UUID) []appdto.RecapFeedbackView {
	result := make([]appdto.RecapFeedbackView, 0, len(values))
	for index := range values {
		if values[index].UserID == userID {
			result = append(result, recapFeedbackView(&values[index]))
		}
	}
	return result
}

func validRecapFeedbackType(value string) bool {
	switch strings.TrimSpace(value) {
	case "liked_place", "disliked_place", "too_expensive", "budget_worked_well", "budget_inaccurate", "too_much_walking", "pace_too_packed", "pace_too_slow", "route_worked_well", "transport_issue", "accommodation_issue", "weather_affected_plan", "checklist_missing_item", "reminder_helpful", "availability_issue", "favorite_activity_type", "avoid_next_time", "prefer_next_time", "other":
		return true
	default:
		return false
	}
}

func (s *Service) applyFeedbackToPersonalization(ctx context.Context, userID uuid.UUID, feedback *entity.TripRecapFeedback) error {
	if s.personalization == nil {
		return apperrs.NewDependencyError("personalization feedback is not configured")
	}
	feedbackType, ok := recapPersonalizationFeedbackType(feedback.FeedbackType, feedback.Value)
	if !ok {
		return apperrs.NewInvalidInput("recap feedback cannot be applied to personalization")
	}
	entityType := "general"
	if feedback.EntityType != nil && strings.TrimSpace(*feedback.EntityType) == "itinerary_item" {
		entityType = "itinerary_item"
	}
	entityID := ""
	if feedback.EntityID != nil {
		entityID = *feedback.EntityID
	}
	_, err := s.personalization.Submit(ctx, userID, personalization.SubmitFeedbackInput{TripID: &feedback.TripID, EntityType: entityType, EntityID: entityID, FeedbackType: feedbackType, FeedbackValue: recapTrim(feedback.Label, 300), Metadata: sanitizeRecapFeedbackMetadata(feedback.Metadata)})
	return err
}

func recapPersonalizationFeedbackType(value string, feedbackValue *string) (personalization.FeedbackType, bool) {
	switch value {
	case "liked_place":
		return personalization.FeedbackLike, true
	case "disliked_place", "avoid_next_time", "accommodation_issue", "availability_issue":
		return personalization.FeedbackDislike, true
	case "too_expensive", "budget_inaccurate":
		return personalization.FeedbackTooExpensive, true
	case "too_much_walking":
		return personalization.FeedbackTooMuchWalking, true
	case "pace_too_packed":
		return personalization.FeedbackTooPacked, true
	case "prefer_next_time", "route_worked_well":
		if feedbackValue != nil && strings.EqualFold(*feedbackValue, "train") {
			return personalization.FeedbackPreferTrains, true
		}
		return personalization.FeedbackOther, true
	case "favorite_activity_type":
		return personalization.FeedbackLike, true
	case "pace_too_slow":
		return personalization.FeedbackPreferFastPaced, true
	default:
		return personalization.FeedbackOther, true
	}
}

func sanitizeRecapFeedbackMetadata(input map[string]any) map[string]any {
	allowed := map[string]struct{}{"source": {}, "destination": {}, "style": {}, "transport": {}, "currency": {}, "category": {}}
	result := map[string]any{"source": "trip_recap"}
	for key, value := range input {
		if _, ok := allowed[key]; !ok {
			continue
		}
		if text, ok := value.(string); ok && recapTrim(text, 160) != "" {
			result[key] = recapTrim(text, 160)
		}
	}
	return result
}

func safeRecapLessons(input []string) []string { return limitRecapStrings(input, 3, 240) }
func limitRecapStrings(input []string, maximum, chars int) []string {
	result := []string{}
	for _, item := range input {
		if value := recapTrim(item, chars); value != "" {
			result = append(result, value)
			if len(result) == maximum {
				break
			}
		}
	}
	return result
}
func recapTrim(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}
func optionalRecapString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
func appendUniqueRecap(values []string, value string) []string {
	value = recapTrim(value, 40)
	if value == "" {
		return values
	}
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return values
		}
	}
	return append(values, value)
}
func recapGenerationMode(aiEnabled, fallback bool) string {
	if aiEnabled && !fallback {
		return "ai"
	}
	return "deterministic"
}

func recapAIProvider(aiEnabled, fallback bool) string {
	if aiEnabled && !fallback {
		return "ai_planning"
	}
	if aiEnabled {
		return "ai_planning_fallback"
	}
	return "none"
}
