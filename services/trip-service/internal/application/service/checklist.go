package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
)

const (
	defaultChecklistTitle = "Packing & preparation checklist"

	maxChecklistGeneratedItems = 100
	maxChecklistTitleLength    = 120
	maxChecklistDescLength     = 500
	maxChecklistReasonLength   = 500
	maxChecklistInstructions   = 1000
)

func (s *Service) GetTripChecklist(ctx context.Context, tripID uuid.UUID) (*appdto.ChecklistViewResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	checklist, err := s.activeChecklistWithItems(ctx, tripID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return &appdto.ChecklistViewResponse{
				Checklist:   nil,
				CanGenerate: access.CanEdit(),
			}, nil
		}
		return nil, err
	}
	summary := checklistSummary(checklist.Items, user.ID)
	return &appdto.ChecklistViewResponse{
		Checklist:   appdto.NewTripChecklistDTO(checklist),
		Summary:     &summary,
		CanGenerate: access.CanEdit(),
	}, nil
}

func (s *Service) GenerateTripChecklist(ctx context.Context, tripID uuid.UUID, in appdto.GenerateChecklistInput) (*appdto.ChecklistViewResponse, error) {
	if err := validateOutputLanguage(in.OutputLanguage); err != nil {
		return nil, err
	}
	in, err := normalizeGenerateChecklistInput(in)
	if err != nil {
		return nil, err
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}

	userCtx, err := s.loadUserContext(ctx, user, tripID)
	if err != nil {
		return nil, err
	}
	weatherForecast, err := s.loadWeatherContext(ctx, *trip, tripID)
	if err != nil {
		return nil, err
	}
	constraints, err := s.buildPlanningConstraints(
		ctx,
		user,
		planningconstraints.SourceTripGeneration,
		trip,
		planningconstraints.RequestOverride{
			OutputLanguage: in.OutputLanguage,
			Prompt: &planningconstraints.Prompt{
				UserPrompt: in.Instructions,
			},
		},
		userCtx,
		true,
	)
	if err != nil {
		return nil, err
	}

	existing, err := s.activeChecklistWithItems(ctx, tripID)
	if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, domainerrs.ErrNotFound) {
		existing = nil
	}
	currentItinerary := parseItineraryLenient(trip.Itinerary)
	var itineraryPtr *aggregate.Itinerary
	if len(currentItinerary.Days) > 0 {
		itineraryPtr = &currentItinerary
	}

	generated, err := s.generator.GenerateChecklist(ctx, application.GenerateChecklistInput{
		Trip:                       *trip,
		CurrentItinerary:           itineraryPtr,
		OutputLanguage:             resolveOutputLanguage(in.OutputLanguage, userCtx.Profile),
		Options:                    in,
		ExistingChecklist:          existing,
		UserProfile:                userCtx.Profile,
		UserPreferences:            userCtx.Preferences,
		WeatherForecast:            weatherForecast,
		WorkspacePolicyConstraints: s.workspacePolicyAIConstraints(ctx, trip),
		PlanningConstraints:        constraints,
	})
	if err != nil {
		return nil, err
	}
	normalizedGenerated, err := normalizeGeneratedChecklist(generated, in)
	if err != nil {
		return nil, err
	}

	checklist, err := s.ensureActiveChecklist(ctx, trip, user.ID)
	if err != nil {
		return nil, err
	}
	checklist.Title = normalizedGenerated.Title
	checklist.Summary = stringPtrOrNil(normalizedGenerated.Summary)
	checklist.GeneratedFromItineraryRevision = intPtrValue(trip.ItineraryRevision)
	checklist.GeneratedByUserID = &user.ID
	checklist.UpdatedByUserID = &user.ID
	checklist.Metadata = mergeMetadata(checklist.Metadata, map[string]any{
		"mode":           string(in.Mode),
		"warnings":       normalizedGenerated.Warnings,
		"generatedAt":    time.Now().UTC().Format(time.RFC3339),
		"generatedCount": len(normalizedGenerated.Items),
	})
	checklist, err = s.repo.UpdateChecklist(ctx, checklist)
	if err != nil {
		return nil, err
	}
	existingItems, err := s.repo.ListChecklistItemsByChecklist(ctx, checklist.ID)
	if err != nil {
		return nil, err
	}
	if in.ReplaceAIItems {
		if _, err := s.repo.SoftDeleteGeneratedChecklistItems(ctx, checklist.ID, user.ID, in.Categories, in.PreserveCheckedItems); err != nil {
			return nil, err
		}
		existingItems, err = s.repo.ListChecklistItemsByChecklist(ctx, checklist.ID)
		if err != nil {
			return nil, err
		}
	}
	newItems := generatedItemsToCreate(checklist.ID, tripID, user.ID, normalizedGenerated.Items, existingItems, in)
	if len(newItems) > 0 {
		if _, err := s.repo.BatchCreateChecklistItems(ctx, newItems); err != nil {
			return nil, err
		}
	}

	checklist, err = s.activeChecklistWithItems(ctx, tripID)
	if err != nil {
		return nil, err
	}
	eventType := activity.EventChecklistGenerated
	if existing != nil {
		eventType = activity.EventChecklistRegenerated
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   eventType,
		EntityType:  activityEntityType(activity.EntityChecklist),
		EntityID:    activityEntityID(checklist.ID),
		Metadata: map[string]any{
			"mode":       string(in.Mode),
			"itemCount":  len(checklist.Items),
			"addedCount": len(newItems),
		},
	})
	summary := checklistSummary(checklist.Items, user.ID)
	return &appdto.ChecklistViewResponse{
		Checklist:   appdto.NewTripChecklistDTO(checklist),
		Summary:     &summary,
		CanGenerate: true,
	}, nil
}

func (s *Service) CreateTripChecklistItem(ctx context.Context, tripID uuid.UUID, in appdto.CreateChecklistItemInput) (appdto.TripChecklistItemDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	normalized, err := normalizeCreateChecklistItemInput(in)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	if err := s.validateChecklistAssignee(ctx, tripID, normalized.AssignedToUserID); err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	checklist, err := s.ensureActiveChecklist(ctx, trip, user.ID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	existingItems, err := s.repo.ListChecklistItemsByChecklist(ctx, checklist.ID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	item := entity.TripChecklistItem{
		ID:               uuid.New(),
		ChecklistID:      checklist.ID,
		TripID:           tripID,
		Title:            normalized.Title,
		Description:      normalized.Description,
		Category:         normalized.Category,
		ItemType:         normalized.ItemType,
		Priority:         normalized.Priority,
		Quantity:         normalized.Quantity,
		AssignedToUserID: normalized.AssignedToUserID,
		DueDate:          normalized.DueDate,
		Checked:          false,
		Source:           entity.ChecklistSourceManual,
		Reason:           normalized.Reason,
		RelatedDayNumber: normalized.RelatedDayNumber,
		RelatedItemIndex: normalized.RelatedItemIndex,
		RelatedItemID:    normalized.RelatedItemID,
		SortOrder:        nextChecklistSortOrder(existingItems),
		Metadata:         normalized.Metadata,
		CreatedByUserID:  &user.ID,
		UpdatedByUserID:  &user.ID,
	}
	created, err := s.repo.CreateChecklistItem(ctx, &item)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventChecklistItemAdded,
		EntityType:  activityEntityType(activity.EntityChecklistItem),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"itemTitle": created.Title,
			"category":  string(created.Category),
		},
	})
	s.notifyChecklistAssignment(ctx, trip, user.ID, created, nil)
	return appdto.NewTripChecklistItemDTO(created), nil
}

func (s *Service) UpdateTripChecklistItem(ctx context.Context, tripID, itemID uuid.UUID, in appdto.UpdateChecklistItemInput) (appdto.TripChecklistItemDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	item, err := s.repo.GetChecklistItemByID(ctx, tripID, itemID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	previousAssignee := item.AssignedToUserID
	if err := applyChecklistItemPatch(item, in); err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	if err := s.validateChecklistAssignee(ctx, tripID, item.AssignedToUserID); err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	item.UpdatedByUserID = &user.ID
	updated, err := s.repo.UpdateChecklistItem(ctx, item)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	if !uuidPtrEqual(previousAssignee, updated.AssignedToUserID) {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      tripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventChecklistItemAssigned,
			EntityType:  activityEntityType(activity.EntityChecklistItem),
			EntityID:    activityEntityID(updated.ID),
			Metadata: map[string]any{
				"itemTitle":        updated.Title,
				"assignedToUserId": uuidPtrStringValue(updated.AssignedToUserID),
			},
		})
		s.notifyChecklistAssignment(ctx, trip, user.ID, updated, previousAssignee)
	}
	return appdto.NewTripChecklistItemDTO(updated), nil
}

func (s *Service) DeleteTripChecklistItem(ctx context.Context, tripID, itemID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return err
	}
	deleted, err := s.repo.SoftDeleteChecklistItem(ctx, tripID, itemID, user.ID)
	if err != nil {
		return err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventChecklistItemDeleted,
		EntityType:  activityEntityType(activity.EntityChecklistItem),
		EntityID:    activityEntityID(deleted.ID),
		Metadata:    map[string]any{"itemTitle": deleted.Title},
	})
	return nil
}

func (s *Service) SetTripChecklistItemChecked(ctx context.Context, tripID, itemID uuid.UUID, checked bool) (appdto.TripChecklistItemDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	item, err := s.repo.GetChecklistItemByID(ctx, tripID, itemID)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	if !canCheckChecklistItem(access, user.ID, item) {
		return appdto.TripChecklistItemDTO{}, apperrs.ErrForbidden
	}
	updated, err := s.repo.SetChecklistItemChecked(ctx, tripID, itemID, user.ID, checked)
	if err != nil {
		return appdto.TripChecklistItemDTO{}, err
	}
	return appdto.NewTripChecklistItemDTO(updated), nil
}

func (s *Service) ReorderTripChecklistItems(ctx context.Context, tripID uuid.UUID, in appdto.ChecklistReorderInput) (*appdto.ChecklistViewResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	if len(in.ItemIDs) == 0 {
		return nil, apperrs.NewInvalidInput("itemIds is required")
	}
	if err := s.repo.ReorderChecklistItems(ctx, tripID, in.ItemIDs, user.ID); err != nil {
		return nil, err
	}
	return s.GetTripChecklist(ctx, tripID)
}

func (s *Service) activeChecklistWithItems(ctx context.Context, tripID uuid.UUID) (*entity.TripChecklist, error) {
	checklist, err := s.repo.GetActiveChecklistByTripID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListChecklistItemsByChecklist(ctx, checklist.ID)
	if err != nil {
		return nil, err
	}
	checklist.Items = items
	return checklist, nil
}

func (s *Service) ensureActiveChecklist(ctx context.Context, trip *entity.Trip, actorID uuid.UUID) (*entity.TripChecklist, error) {
	checklist, err := s.repo.GetActiveChecklistByTripID(ctx, trip.ID)
	if err == nil {
		return checklist, nil
	}
	if !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}
	return s.repo.CreateChecklist(ctx, &entity.TripChecklist{
		ID:              uuid.New(),
		TripID:          trip.ID,
		Status:          entity.ChecklistStatusActive,
		Title:           defaultChecklistTitle,
		CreatedByUserID: actorID,
		UpdatedByUserID: &actorID,
		Metadata:        map[string]any{},
	})
}

func (s *Service) validateChecklistAssignee(ctx context.Context, tripID uuid.UUID, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	if _, access, err := s.tripForAccess(ctx, tripID, *userID); err != nil || !access.CanView() {
		return apperrs.NewInvalidInput("assignedToUserId must have trip access")
	}
	return nil
}

func canCheckChecklistItem(access TripAccess, actorID uuid.UUID, item *entity.TripChecklistItem) bool {
	if access.CanEdit() {
		return true
	}
	if item.AssignedToUserID != nil && *item.AssignedToUserID == actorID {
		return true
	}
	return item.AssignedToUserID == nil && access.CanView()
}

func normalizeGenerateChecklistInput(in appdto.GenerateChecklistInput) (appdto.GenerateChecklistInput, error) {
	switch in.Mode {
	case "":
		in.Mode = appdto.GenerateChecklistModeFull
	case appdto.GenerateChecklistModeFull, appdto.GenerateChecklistModeAddMissing, appdto.GenerateChecklistModeCategory:
	default:
		return in, apperrs.NewInvalidInput("mode must be full, add_missing, or category")
	}
	if in.OutputLanguage == "" {
		in.OutputLanguage = "en"
	}
	if len(in.Instructions) > maxChecklistInstructions {
		return in, apperrs.NewInvalidInput("instructions must be at most %d characters", maxChecklistInstructions)
	}
	categories, err := normalizeChecklistCategories(in.Categories)
	if err != nil {
		return in, err
	}
	if in.Mode == appdto.GenerateChecklistModeCategory && len(categories) == 0 {
		return in, apperrs.NewInvalidInput("categories is required for category mode")
	}
	in.Categories = categories
	return in, nil
}

func normalizeCreateChecklistItemInput(in appdto.CreateChecklistItemInput) (appdto.CreateChecklistItemInput, error) {
	title, err := normalizeChecklistTitle(in.Title)
	if err != nil {
		return in, err
	}
	in.Title = title
	category, err := normalizeChecklistCategory(in.Category)
	if err != nil {
		return in, err
	}
	in.Category = category
	itemType, err := normalizeChecklistItemType(in.ItemType)
	if err != nil {
		return in, err
	}
	in.ItemType = itemType
	priority, err := normalizeChecklistPriority(in.Priority)
	if err != nil {
		return in, err
	}
	in.Priority = priority
	if err := validateChecklistQuantity(in.Quantity); err != nil {
		return in, err
	}
	in.Description = normalizeOptionalStringPtr(in.Description, maxChecklistDescLength)
	in.Reason = normalizeOptionalStringPtr(in.Reason, maxChecklistReasonLength)
	in.RelatedItemID = normalizeOptionalStringPtr(in.RelatedItemID, 100)
	return in, nil
}

func applyChecklistItemPatch(item *entity.TripChecklistItem, in appdto.UpdateChecklistItemInput) error {
	if in.Title != nil {
		title, err := normalizeChecklistTitle(*in.Title)
		if err != nil {
			return err
		}
		item.Title = title
	}
	if in.ClearDescription {
		item.Description = nil
	} else if in.Description != nil {
		item.Description = normalizeOptionalStringPtr(in.Description, maxChecklistDescLength)
	}
	if in.Category != nil {
		category, err := normalizeChecklistCategory(*in.Category)
		if err != nil {
			return err
		}
		item.Category = category
	}
	if in.ItemType != nil {
		itemType, err := normalizeChecklistItemType(*in.ItemType)
		if err != nil {
			return err
		}
		item.ItemType = itemType
	}
	if in.Priority != nil {
		priority, err := normalizeChecklistPriority(*in.Priority)
		if err != nil {
			return err
		}
		item.Priority = priority
	}
	if in.ClearQuantity {
		item.Quantity = nil
	} else if in.Quantity != nil {
		if err := validateChecklistQuantity(in.Quantity); err != nil {
			return err
		}
		item.Quantity = in.Quantity
	}
	if in.ClearAssignee {
		item.AssignedToUserID = nil
	} else if in.AssignedToUserID != nil {
		item.AssignedToUserID = in.AssignedToUserID
	}
	if in.ClearDueDate {
		item.DueDate = nil
	} else if in.DueDate != nil {
		item.DueDate = in.DueDate
	}
	if in.ClearReason {
		item.Reason = nil
	} else if in.Reason != nil {
		item.Reason = normalizeOptionalStringPtr(in.Reason, maxChecklistReasonLength)
	}
	if in.ClearRelatedDay {
		item.RelatedDayNumber = nil
	} else if in.RelatedDayNumber != nil {
		item.RelatedDayNumber = in.RelatedDayNumber
	}
	if in.ClearRelatedIndex {
		item.RelatedItemIndex = nil
	} else if in.RelatedItemIndex != nil {
		item.RelatedItemIndex = in.RelatedItemIndex
	}
	if in.ClearRelatedItem {
		item.RelatedItemID = nil
	} else if in.RelatedItemID != nil {
		item.RelatedItemID = normalizeOptionalStringPtr(in.RelatedItemID, 100)
	}
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	if in.Metadata != nil {
		item.Metadata = mergeMetadata(item.Metadata, in.Metadata)
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	item.Metadata["edited"] = true
	return nil
}

func normalizeGeneratedChecklist(in *appdto.GeneratedChecklist, options appdto.GenerateChecklistInput) (*appdto.GeneratedChecklist, error) {
	if in == nil {
		return nil, apperrs.NewDependencyError("AI returned empty checklist")
	}
	title, err := normalizeChecklistTitle(defaultChecklistString(in.Title, defaultChecklistTitle))
	if err != nil {
		return nil, apperrs.NewDependencyError("AI returned invalid checklist title")
	}
	out := &appdto.GeneratedChecklist{
		Title:    title,
		Summary:  truncateChecklistString(strings.TrimSpace(in.Summary), 500),
		Warnings: nonNilStringsCopy(in.Warnings),
	}
	if len(in.Items) == 0 || len(in.Items) > maxChecklistGeneratedItems {
		return nil, apperrs.NewDependencyError("AI returned invalid checklist item count")
	}
	seen := map[string]struct{}{}
	for _, item := range in.Items {
		normalized, err := normalizeGeneratedChecklistItem(item)
		if err != nil {
			continue
		}
		if options.Mode == appdto.GenerateChecklistModeCategory && !categorySelected(normalized.Category, options.Categories) {
			continue
		}
		key := duplicateChecklistKey(normalized.Title, normalized.Category)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out.Items = append(out.Items, normalized)
	}
	if len(out.Items) == 0 {
		return nil, apperrs.NewDependencyError("AI returned no valid checklist items")
	}
	return out, nil
}

func normalizeGeneratedChecklistItem(in appdto.GeneratedChecklistItem) (appdto.GeneratedChecklistItem, error) {
	title, err := normalizeChecklistTitle(in.Title)
	if err != nil {
		return in, err
	}
	category, err := normalizeChecklistCategory(in.Category)
	if err != nil {
		return in, err
	}
	itemType, err := normalizeChecklistItemType(in.ItemType)
	if err != nil {
		return in, err
	}
	priority, err := normalizeChecklistPriority(in.Priority)
	if err != nil {
		return in, err
	}
	if err := validateChecklistQuantity(in.Quantity); err != nil {
		return in, err
	}
	in.Title = title
	in.Description = truncateChecklistString(strings.TrimSpace(in.Description), maxChecklistDescLength)
	in.Category = category
	in.ItemType = itemType
	in.Priority = priority
	in.Reason = truncateChecklistString(strings.TrimSpace(in.Reason), maxChecklistReasonLength)
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return in, nil
}

func generatedItemsToCreate(
	checklistID uuid.UUID,
	tripID uuid.UUID,
	actorID uuid.UUID,
	generated []appdto.GeneratedChecklistItem,
	existing []entity.TripChecklistItem,
	options appdto.GenerateChecklistInput,
) []entity.TripChecklistItem {
	existingKeys := map[string]entity.TripChecklistItem{}
	maxSort := -1
	for _, item := range existing {
		if item.SortOrder > maxSort {
			maxSort = item.SortOrder
		}
		existingKeys[duplicateChecklistKey(item.Title, item.Category)] = item
	}
	out := make([]entity.TripChecklistItem, 0, len(generated))
	nextSort := maxSort + 1
	for _, item := range generated {
		key := duplicateChecklistKey(item.Title, item.Category)
		if _, exists := existingKeys[key]; exists {
			continue
		}
		if options.Mode == appdto.GenerateChecklistModeCategory && !categorySelected(item.Category, options.Categories) {
			continue
		}
		description := stringPtrOrNil(item.Description)
		reason := stringPtrOrNil(item.Reason)
		out = append(out, entity.TripChecklistItem{
			ID:               uuid.New(),
			ChecklistID:      checklistID,
			TripID:           tripID,
			Title:            item.Title,
			Description:      description,
			Category:         item.Category,
			ItemType:         item.ItemType,
			Priority:         item.Priority,
			Quantity:         item.Quantity,
			Source:           entity.ChecklistSourceAI,
			Reason:           reason,
			RelatedDayNumber: item.RelatedDayNumber,
			RelatedItemIndex: item.RelatedItemIndex,
			RelatedItemID:    item.RelatedItemID,
			SortOrder:        nextSort,
			Metadata:         item.Metadata,
			CreatedByUserID:  &actorID,
			UpdatedByUserID:  &actorID,
		})
		nextSort++
	}
	return out
}

func checklistSummary(items []entity.TripChecklistItem, currentUserID uuid.UUID) appdto.ChecklistSummary {
	categoryMap := map[entity.ChecklistCategory]*appdto.ChecklistCategorySummary{}
	summary := appdto.ChecklistSummary{Categories: []appdto.ChecklistCategorySummary{}}
	for _, item := range items {
		if item.DeletedAt != nil {
			continue
		}
		summary.TotalItems++
		if item.Checked {
			summary.CheckedItems++
		} else {
			summary.UncheckedItems++
			if item.Priority == entity.ChecklistPriorityHigh || item.Priority == entity.ChecklistPriorityCritical {
				summary.HighPriorityUnchecked++
			}
		}
		if item.AssignedToUserID != nil && *item.AssignedToUserID == currentUserID && !item.Checked {
			summary.AssignedToMe++
		}
		categorySummary := categoryMap[item.Category]
		if categorySummary == nil {
			categorySummary = &appdto.ChecklistCategorySummary{Category: item.Category}
			categoryMap[item.Category] = categorySummary
		}
		categorySummary.Total++
		if item.Checked {
			categorySummary.Checked++
		}
	}
	for _, category := range checklistCategoryOrder() {
		if categorySummary := categoryMap[category]; categorySummary != nil {
			summary.Categories = append(summary.Categories, *categorySummary)
		}
	}
	return summary
}

func (s *Service) notifyChecklistAssignment(
	ctx context.Context,
	trip *entity.Trip,
	actorID uuid.UUID,
	item *entity.TripChecklistItem,
	previousAssignee *uuid.UUID,
) {
	if item == nil || item.AssignedToUserID == nil || uuidPtrEqual(previousAssignee, item.AssignedToUserID) {
		return
	}
	destination := tripDestination(trip)
	s.notifyDirect(ctx, *item.AssignedToUserID, item.TripID, actorID,
		notifications.TypeChecklistItemAssigned,
		"Checklist item assigned",
		fmt.Sprintf("You were assigned %q for %s.", item.Title, destination),
		notifications.EntityChecklistItem, activityEntityID(item.ID),
		map[string]any{
			"tripId":          item.TripID.String(),
			"checklistItemId": item.ID.String(),
			"itemTitle":       item.Title,
			"destination":     destination,
		})
}

func normalizeChecklistCategories(in []entity.ChecklistCategory) ([]entity.ChecklistCategory, error) {
	out := make([]entity.ChecklistCategory, 0, len(in))
	seen := map[entity.ChecklistCategory]struct{}{}
	for _, category := range in {
		normalized, err := normalizeChecklistCategory(category)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeChecklistTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", apperrs.NewInvalidInput("title is required")
	}
	if len(trimmed) > maxChecklistTitleLength {
		return "", apperrs.NewInvalidInput("title must be at most %d characters", maxChecklistTitleLength)
	}
	return trimmed, nil
}

func normalizeChecklistCategory(category entity.ChecklistCategory) (entity.ChecklistCategory, error) {
	if category == "" {
		return entity.ChecklistCategoryOther, nil
	}
	category = entity.ChecklistCategory(strings.TrimSpace(strings.ToLower(string(category))))
	for _, candidate := range checklistCategoryOrder() {
		if category == candidate {
			return category, nil
		}
	}
	return "", apperrs.NewInvalidInput("invalid category")
}

func normalizeChecklistPriority(priority entity.ChecklistPriority) (entity.ChecklistPriority, error) {
	if priority == "" {
		return entity.ChecklistPriorityMedium, nil
	}
	switch entity.ChecklistPriority(strings.TrimSpace(strings.ToLower(string(priority)))) {
	case entity.ChecklistPriorityLow:
		return entity.ChecklistPriorityLow, nil
	case entity.ChecklistPriorityMedium:
		return entity.ChecklistPriorityMedium, nil
	case entity.ChecklistPriorityHigh:
		return entity.ChecklistPriorityHigh, nil
	case entity.ChecklistPriorityCritical:
		return entity.ChecklistPriorityCritical, nil
	default:
		return "", apperrs.NewInvalidInput("invalid priority")
	}
}

func normalizeChecklistItemType(itemType entity.ChecklistItemType) (entity.ChecklistItemType, error) {
	if itemType == "" {
		return entity.ChecklistItemTypePacking, nil
	}
	switch entity.ChecklistItemType(strings.TrimSpace(strings.ToLower(string(itemType)))) {
	case entity.ChecklistItemTypePacking:
		return entity.ChecklistItemTypePacking, nil
	case entity.ChecklistItemTypePreparation:
		return entity.ChecklistItemTypePreparation, nil
	case entity.ChecklistItemTypeBookingCheck:
		return entity.ChecklistItemTypeBookingCheck, nil
	case entity.ChecklistItemTypeDocument:
		return entity.ChecklistItemTypeDocument, nil
	case entity.ChecklistItemTypeSharedGroupItem:
		return entity.ChecklistItemTypeSharedGroupItem, nil
	case entity.ChecklistItemTypeReminder:
		return entity.ChecklistItemTypeReminder, nil
	case entity.ChecklistItemTypeSafetyCheck:
		return entity.ChecklistItemTypeSafetyCheck, nil
	case entity.ChecklistItemTypeOther:
		return entity.ChecklistItemTypeOther, nil
	default:
		return "", apperrs.NewInvalidInput("invalid itemType")
	}
}

func validateChecklistQuantity(quantity *int) error {
	if quantity == nil {
		return nil
	}
	if *quantity < 1 || *quantity > 99 {
		return apperrs.NewInvalidInput("quantity must be between 1 and 99")
	}
	return nil
}

func checklistCategoryOrder() []entity.ChecklistCategory {
	return []entity.ChecklistCategory{
		entity.ChecklistCategoryDocuments,
		entity.ChecklistCategoryClothing,
		entity.ChecklistCategoryElectronics,
		entity.ChecklistCategoryHealthSafety,
		entity.ChecklistCategoryTransport,
		entity.ChecklistCategoryAccommodation,
		entity.ChecklistCategoryActivities,
		entity.ChecklistCategoryFoodWater,
		entity.ChecklistCategoryMoney,
		entity.ChecklistCategoryBeforeDeparture,
		entity.ChecklistCategoryGroupItems,
		entity.ChecklistCategoryCampingHiking,
		entity.ChecklistCategoryWeather,
		entity.ChecklistCategoryOther,
	}
}

func categorySelected(category entity.ChecklistCategory, selected []entity.ChecklistCategory) bool {
	if len(selected) == 0 {
		return true
	}
	for _, candidate := range selected {
		if candidate == category {
			return true
		}
	}
	return false
}

func duplicateChecklistKey(title string, category entity.ChecklistCategory) string {
	return fmt.Sprintf("%s:%s", category, normalizeDuplicateText(title))
}

func normalizeDuplicateText(value string) string {
	var b strings.Builder
	previousSpace := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			previousSpace = false
			continue
		}
		if unicode.IsSpace(r) && !previousSpace {
			b.WriteRune(' ')
			previousSpace = true
		}
	}
	normalized := strings.TrimSpace(b.String())
	parts := strings.Fields(normalized)
	for i, part := range parts {
		if len(part) > 3 && strings.HasSuffix(part, "s") {
			parts[i] = strings.TrimSuffix(part, "s")
		}
	}
	return strings.Join(parts, " ")
}

func nextChecklistSortOrder(items []entity.TripChecklistItem) int {
	maxSort := -1
	for _, item := range items {
		if item.SortOrder > maxSort {
			maxSort = item.SortOrder
		}
	}
	return maxSort + 1
}

func normalizeOptionalStringPtr(value *string, maxLength int) *string {
	if value == nil {
		return nil
	}
	trimmed := truncateChecklistString(strings.TrimSpace(*value), maxLength)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func truncateChecklistString(value string, maxLength int) string {
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}

func stringPtrOrNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func intPtrValue(value int) *int {
	v := value
	return &v
}

func defaultChecklistString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func nonNilStringsCopy(values []string) []string {
	if values == nil {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func mergeMetadata(current map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range current {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func uuidPtrEqual(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func uuidPtrStringValue(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
