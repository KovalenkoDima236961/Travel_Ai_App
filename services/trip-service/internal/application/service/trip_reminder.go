package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
)

const (
	defaultReminderTime       = "09:00"
	defaultReminderTimezone   = "Europe/Bratislava"
	maxReminderTitleLength    = 140
	maxReminderDescLength     = 600
	maxReminderInstructions   = 1000
	maxReminderFailureReason  = 240
	maxReminderMetadataString = 160
)

func (s *Service) ListTripReminders(ctx context.Context, tripID uuid.UUID, filters appdto.ReminderListFilters) (*appdto.ReminderViewResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	repoFilters := entity.TripReminderFilters{
		Status:           filters.Status,
		Category:         filters.Category,
		UpcomingOnly:     filters.UpcomingOnly,
		HighPriorityOnly: filters.HighPriorityOnly,
		FromDate:         filters.FromDate,
		ToDate:           filters.ToDate,
	}
	if filters.AssignedToMe {
		repoFilters.AssignedToUserID = &user.ID
	}
	reminders, err := s.repo.ListTripRemindersByTrip(ctx, tripID, repoFilters)
	if err != nil {
		return nil, err
	}
	summary := s.reminderSummary(ctx, trip, reminders, user.ID)
	return &appdto.ReminderViewResponse{
		Reminders: appdto.NewTripReminderDTOs(reminders),
		Summary:   summary,
	}, nil
}

func (s *Service) ListAssignedTripReminders(ctx context.Context, filters appdto.ReminderListFilters) (*appdto.ReminderViewResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repoFilters := entity.TripReminderFilters{
		Status:           filters.Status,
		Category:         filters.Category,
		UpcomingOnly:     filters.UpcomingOnly,
		HighPriorityOnly: filters.HighPriorityOnly,
		FromDate:         filters.FromDate,
		ToDate:           filters.ToDate,
	}
	reminders, err := s.repo.ListTripRemindersAssignedToUser(ctx, user.ID, repoFilters)
	if err != nil {
		return nil, err
	}
	return &appdto.ReminderViewResponse{
		Reminders: appdto.NewTripReminderDTOs(reminders),
		Summary:   buildReminderSummary(reminders, user.ID, false),
	}, nil
}

func (s *Service) GenerateTripReminders(ctx context.Context, tripID uuid.UUID, in appdto.GenerateRemindersInput) (*appdto.ReminderViewResponse, error) {
	normalized, err := normalizeGenerateRemindersInput(in)
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
	if trip.StartDate == nil {
		return nil, apperrs.NewInvalidInput("trip startDate is required to generate reminders")
	}

	checklist, err := s.activeChecklistWithItems(ctx, tripID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			return nil, err
		}
		checklist = nil
	}
	weather, err := s.loadWeatherContext(ctx, *trip, tripID)
	if err != nil {
		return nil, err
	}

	if normalized.ReplaceGeneratedPendingReminders {
		if _, err := s.repo.DeleteGeneratedPendingRemindersForTrip(ctx, tripID, user.ID, normalized.Categories); err != nil {
			return nil, err
		}
	}

	existing, err := s.repo.ListTripRemindersByTrip(ctx, tripID, entity.TripReminderFilters{})
	if err != nil {
		return nil, err
	}
	candidates := s.generateReminderCandidates(trip, checklist, weather, user.ID, normalized)
	toCreate := mergeReminderCandidates(existing, candidates)
	if len(toCreate) > 0 {
		if _, err := s.repo.BatchCreateTripReminders(ctx, toCreate); err != nil {
			return nil, err
		}
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventRemindersGenerated,
		EntityType:  activityEntityType(activity.EntityReminder),
		Metadata: map[string]any{
			"mode":       string(normalized.Mode),
			"addedCount": len(toCreate),
		},
	})

	return s.ListTripReminders(ctx, tripID, appdto.ReminderListFilters{})
}

func (s *Service) CreateTripReminder(ctx context.Context, tripID uuid.UUID, in appdto.CreateReminderInput) (appdto.TripReminderDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	normalized, err := normalizeCreateReminderInput(in)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if !access.CanEdit() {
		if normalized.AssignedToUserID != nil && *normalized.AssignedToUserID != user.ID {
			return appdto.TripReminderDTO{}, apperrs.ErrForbidden
		}
		normalized.AssignedToUserID = &user.ID
	}
	if err := s.validateReminderAssignee(ctx, tripID, normalized.AssignedToUserID); err != nil {
		return appdto.TripReminderDTO{}, err
	}
	reminder := &entity.TripReminder{
		ID:                 uuid.New(),
		TripID:             tripID,
		Title:              normalized.Title,
		Description:        normalized.Description,
		Category:           normalized.Category,
		Priority:           normalized.Priority,
		Source:             entity.ReminderSourceManual,
		Status:             entity.ReminderStatusPending,
		TriggerDate:        dateOnly(normalized.TriggerDate),
		TriggerTime:        normalized.TriggerTime,
		Timezone:           normalized.Timezone,
		RelativeOffsetDays: normalized.RelativeOffsetDays,
		AssignedToUserID:   normalized.AssignedToUserID,
		ChecklistItemID:    normalized.ChecklistItemID,
		RelatedDayNumber:   normalized.RelatedDayNumber,
		RelatedItemIndex:   normalized.RelatedItemIndex,
		RelatedItemID:      normalized.RelatedItemID,
		Metadata:           sanitizeReminderMetadata(normalized.Metadata),
		CreatedByUserID:    &user.ID,
		UpdatedByUserID:    &user.ID,
	}
	created, err := s.repo.CreateTripReminder(ctx, reminder)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReminderCreated,
		EntityType:  activityEntityType(activity.EntityReminder),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"reminderTitle": truncateReminderString(created.Title, maxReminderMetadataString),
			"category":      string(created.Category),
			"priority":      string(created.Priority),
		},
	})
	if created.AssignedToUserID != nil && *created.AssignedToUserID != user.ID {
		s.notifyReminderAssigned(ctx, trip, user.ID, created)
	}
	return appdto.NewTripReminderDTO(created), nil
}

func (s *Service) UpdateTripReminder(ctx context.Context, tripID, reminderID uuid.UUID, in appdto.UpdateReminderInput) (appdto.TripReminderDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	reminder, err := s.repo.GetTripReminderByID(ctx, tripID, reminderID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if !access.CanEdit() && !canEditOwnManualReminder(user.ID, reminder) {
		return appdto.TripReminderDTO{}, apperrs.ErrForbidden
	}
	previousAssignee := reminder.AssignedToUserID
	if err := applyReminderPatch(reminder, in); err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if !access.CanEdit() {
		if reminder.AssignedToUserID != nil && *reminder.AssignedToUserID != user.ID {
			return appdto.TripReminderDTO{}, apperrs.ErrForbidden
		}
	}
	if err := s.validateReminderAssignee(ctx, tripID, reminder.AssignedToUserID); err != nil {
		return appdto.TripReminderDTO{}, err
	}
	reminder.UpdatedByUserID = &user.ID
	updated, err := s.repo.UpdateTripReminder(ctx, reminder)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReminderUpdated,
		EntityType:  activityEntityType(activity.EntityReminder),
		EntityID:    activityEntityID(updated.ID),
		Metadata: map[string]any{
			"reminderTitle": truncateReminderString(updated.Title, maxReminderMetadataString),
			"category":      string(updated.Category),
			"priority":      string(updated.Priority),
		},
	})
	if !uuidPtrEqual(previousAssignee, updated.AssignedToUserID) && updated.AssignedToUserID != nil && *updated.AssignedToUserID != user.ID {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      tripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventReminderAssigned,
			EntityType:  activityEntityType(activity.EntityReminder),
			EntityID:    activityEntityID(updated.ID),
			Metadata: map[string]any{
				"reminderTitle":    truncateReminderString(updated.Title, maxReminderMetadataString),
				"assignedToUserId": updated.AssignedToUserID.String(),
			},
		})
		s.notifyReminderAssigned(ctx, trip, user.ID, updated)
	}
	return appdto.NewTripReminderDTO(updated), nil
}

func (s *Service) CompleteTripReminder(ctx context.Context, tripID, reminderID uuid.UUID) (appdto.TripReminderDTO, error) {
	return s.setTripReminderDoneState(ctx, tripID, reminderID, true)
}

func (s *Service) ReopenTripReminder(ctx context.Context, tripID, reminderID uuid.UUID) (appdto.TripReminderDTO, error) {
	return s.setTripReminderDoneState(ctx, tripID, reminderID, false)
}

func (s *Service) DisableTripReminder(ctx context.Context, tripID, reminderID uuid.UUID) (appdto.TripReminderDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	reminder, err := s.repo.GetTripReminderByID(ctx, tripID, reminderID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if !access.CanEdit() && !isAssignedTo(user.ID, reminder) {
		return appdto.TripReminderDTO{}, apperrs.ErrForbidden
	}
	updated, err := s.repo.DisableTripReminder(ctx, tripID, reminderID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReminderDisabled,
		EntityType:  activityEntityType(activity.EntityReminder),
		EntityID:    activityEntityID(updated.ID),
		Metadata: map[string]any{
			"reminderTitle": truncateReminderString(updated.Title, maxReminderMetadataString),
			"category":      string(updated.Category),
			"priority":      string(updated.Priority),
		},
	})
	return appdto.NewTripReminderDTO(updated), nil
}

func (s *Service) EnableTripReminder(ctx context.Context, tripID, reminderID uuid.UUID) (appdto.TripReminderDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripReminderDTO{}, err
	}
	updated, err := s.repo.EnableTripReminder(ctx, tripID, reminderID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	return appdto.NewTripReminderDTO(updated), nil
}

func (s *Service) DeleteTripReminder(ctx context.Context, tripID, reminderID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	reminder, err := s.repo.GetTripReminderByID(ctx, tripID, reminderID)
	if err != nil {
		return err
	}
	if !access.CanEdit() && !canEditOwnManualReminder(user.ID, reminder) {
		return apperrs.ErrForbidden
	}
	deleted, err := s.repo.SoftDeleteTripReminder(ctx, tripID, reminderID, user.ID)
	if err != nil {
		return err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventReminderDeleted,
		EntityType:  activityEntityType(activity.EntityReminder),
		EntityID:    activityEntityID(deleted.ID),
		Metadata: map[string]any{
			"reminderTitle": truncateReminderString(deleted.Title, maxReminderMetadataString),
		},
	})
	return nil
}

func (s *Service) ProcessDueTripReminders(ctx context.Context, in appdto.ProcessDueRemindersInput) (*appdto.ProcessDueRemindersResult, error) {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	limit := in.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	candidates, err := s.repo.ListDueTripReminders(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	result := &appdto.ProcessDueRemindersResult{}
	for i := range candidates {
		reminder := candidates[i]
		if !reminderIsDue(reminder, now) {
			continue
		}
		result.Processed++
		if err := s.processOneDueReminder(ctx, &reminder); err != nil {
			result.Failed++
			reason := truncateReminderString(err.Error(), maxReminderFailureReason)
			if _, markErr := s.repo.MarkTripReminderFailed(ctx, reminder.TripID, reminder.ID, reason); markErr != nil {
				s.log.Warn("failed to mark trip reminder failed",
					zap.String("trip_id", reminder.TripID.String()),
					zap.String("reminder_id", reminder.ID.String()),
					zap.Error(markErr),
				)
			}
			continue
		}
		if _, err := s.repo.MarkTripReminderSent(ctx, reminder.TripID, reminder.ID); err != nil {
			result.Failed++
			s.log.Warn("failed to mark trip reminder sent",
				zap.String("trip_id", reminder.TripID.String()),
				zap.String("reminder_id", reminder.ID.String()),
				zap.Error(err),
			)
			continue
		}
		result.Sent++
	}
	return result, nil
}

func (s *Service) setTripReminderDoneState(ctx context.Context, tripID, reminderID uuid.UUID, done bool) (appdto.TripReminderDTO, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	reminder, err := s.repo.GetTripReminderByID(ctx, tripID, reminderID)
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	if !access.CanEdit() && !isAssignedTo(user.ID, reminder) {
		return appdto.TripReminderDTO{}, apperrs.ErrForbidden
	}
	var updated *entity.TripReminder
	if done {
		updated, err = s.repo.CompleteTripReminder(ctx, tripID, reminderID, user.ID)
	} else {
		updated, err = s.repo.ReopenTripReminder(ctx, tripID, reminderID, user.ID)
	}
	if err != nil {
		return appdto.TripReminderDTO{}, err
	}
	return appdto.NewTripReminderDTO(updated), nil
}

func (s *Service) processOneDueReminder(ctx context.Context, reminder *entity.TripReminder) error {
	if !s.notificationsEnabled || s.notifier == nil {
		return fmt.Errorf("notifications disabled")
	}
	trip, err := s.repo.GetByID(ctx, reminder.TripID)
	if err != nil {
		return err
	}
	recipients := s.reminderRecipients(ctx, trip, reminder)
	if len(recipients) == 0 {
		return fmt.Errorf("no reminder recipients")
	}
	inputs := make([]notifications.NotificationCreateInput, 0, len(recipients))
	tripID := trip.ID
	entityType := notifications.EntityTripReminder
	reminderID := reminder.ID
	for _, recipient := range recipients {
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:     recipient,
			TripID:     &tripID,
			Type:       notifications.TypePreTripReminderDue,
			Title:      "Trip reminder: " + reminder.Title,
			Message:    reminderNotificationMessage(trip, reminder),
			EntityType: &entityType,
			EntityID:   &reminderID,
			Metadata: map[string]any{
				"tripId":     trip.ID.String(),
				"reminderId": reminder.ID.String(),
				"category":   string(reminder.Category),
				"priority":   string(reminder.Priority),
			},
		})
	}
	return s.notifier.CreateNotifications(ctx, inputs)
}

func (s *Service) reminderRecipients(ctx context.Context, trip *entity.Trip, reminder *entity.TripReminder) []uuid.UUID {
	if reminder.AssignedToUserID != nil {
		return []uuid.UUID{*reminder.AssignedToUserID}
	}
	if reminder.Priority != entity.ReminderPriorityHigh && reminder.Priority != entity.ReminderPriorityCritical {
		return nil
	}
	seen := map[uuid.UUID]struct{}{}
	recipients := make([]uuid.UUID, 0, 2)
	add := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		recipients = append(recipients, id)
	}
	if trip.UserID != nil {
		add(*trip.UserID)
	}
	collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
	if err == nil {
		for _, collaborator := range collaborators {
			if collaborator.Status == entity.CollaboratorStatusAccepted && collaborator.Role == entity.CollaboratorRoleEditor {
				add(collaborator.UserID)
			}
		}
	}
	if trip.WorkspaceID != nil && s.workspacesEnabled && s.workspaceProvider != nil {
		members, err := s.workspaceProvider.ListMembers(ctx, *trip.WorkspaceID)
		if err == nil {
			for _, member := range members {
				switch member.Role {
				case "owner", "admin", "member":
					add(member.UserID)
				}
			}
		}
	}
	if reminder.CreatedByUserID != nil {
		add(*reminder.CreatedByUserID)
	}
	return recipients
}

func (s *Service) validateReminderAssignee(ctx context.Context, tripID uuid.UUID, userID *uuid.UUID) error {
	if userID == nil {
		return nil
	}
	if _, access, err := s.tripForAccess(ctx, tripID, *userID); err != nil || !access.CanView() {
		return apperrs.NewInvalidInput("assignedToUserId must have trip access")
	}
	return nil
}

func (s *Service) reminderSummary(ctx context.Context, trip *entity.Trip, reminders []entity.TripReminder, userID uuid.UUID) appdto.ReminderSummary {
	stale := s.reminderTimelineStale(ctx, trip, reminders)
	return buildReminderSummary(reminders, userID, stale)
}

func buildReminderSummary(reminders []entity.TripReminder, userID uuid.UUID, stale bool) appdto.ReminderSummary {
	now := time.Now()
	today := dateOnly(now)
	var summary appdto.ReminderSummary
	summary.Total = len(reminders)
	summary.Stale = stale
	for i := range reminders {
		reminder := reminders[i]
		switch reminder.Status {
		case entity.ReminderStatusPending:
			summary.Pending++
			if reminder.TriggerDate.Before(today) {
				summary.Overdue++
			}
			if reminder.TriggerDate.Equal(today) {
				summary.DueToday++
			}
			if reminder.Priority == entity.ReminderPriorityHigh || reminder.Priority == entity.ReminderPriorityCritical {
				summary.HighPriorityPending++
			}
		case entity.ReminderStatusCompleted:
			summary.Completed++
		}
		if reminder.AssignedToUserID != nil && *reminder.AssignedToUserID == userID {
			summary.AssignedToMe++
		}
	}
	return summary
}

func (s *Service) reminderTimelineStale(ctx context.Context, trip *entity.Trip, reminders []entity.TripReminder) bool {
	if trip == nil {
		return false
	}
	var latest *entity.TripReminder
	for i := range reminders {
		reminder := &reminders[i]
		if reminder.Source == entity.ReminderSourceManual {
			continue
		}
		if latest == nil || reminder.CreatedAt.After(latest.CreatedAt) {
			latest = reminder
		}
	}
	if latest == nil {
		return false
	}
	meta := latest.Metadata
	if trip.StartDate != nil && metaStringValue(meta, "generationTripStartDate") != trip.StartDate.Format("2006-01-02") {
		return true
	}
	if intFromMeta(meta, "generationItineraryRevision") != trip.ItineraryRevision {
		return true
	}
	if metaStringValue(meta, "generationRouteSignature") != routeReminderSignature(trip) {
		return true
	}
	checklist, err := s.repo.GetActiveChecklistByTripID(ctx, trip.ID)
	if err == nil {
		if metaStringValue(meta, "generationChecklistUpdatedAt") != checklist.UpdatedAt.UTC().Format(time.RFC3339) {
			return true
		}
	}
	return false
}

func (s *Service) generateReminderCandidates(
	trip *entity.Trip,
	checklist *entity.TripChecklist,
	forecast *weathercontext.WeatherForecast,
	actorID uuid.UUID,
	options appdto.GenerateRemindersInput,
) []entity.TripReminder {
	commonMeta := map[string]any{
		"generationTripStartDate":     trip.StartDate.Format("2006-01-02"),
		"generationItineraryRevision": trip.ItineraryRevision,
		"generationRouteSignature":    routeReminderSignature(trip),
		"generatedAt":                 time.Now().UTC().Format(time.RFC3339),
	}
	if checklist != nil {
		commonMeta["generationChecklistUpdatedAt"] = checklist.UpdatedAt.UTC().Format(time.RFC3339)
	}
	out := make([]entity.TripReminder, 0)
	add := func(reminder entity.TripReminder) {
		if options.Mode == appdto.GenerateRemindersModeCategory && !reminderCategorySelected(reminder.Category, options.Categories) {
			return
		}
		reminder.ID = uuid.New()
		reminder.TripID = trip.ID
		reminder.Status = entity.ReminderStatusPending
		if reminder.Priority == "" {
			reminder.Priority = entity.ReminderPriorityMedium
		}
		if reminder.TriggerTime == nil {
			reminder.TriggerTime = stringPtr(defaultReminderTime)
		}
		if reminder.Timezone == nil {
			reminder.Timezone = stringPtr(defaultReminderTimezone)
		}
		reminder.Metadata = mergeMetadata(commonMeta, sanitizeReminderMetadata(reminder.Metadata))
		reminder.CreatedByUserID = &actorID
		reminder.UpdatedByUserID = &actorID
		out = append(out, reminder)
	}

	if checklist != nil {
		for i := range checklist.Items {
			item := checklist.Items[i]
			if item.Checked || item.DeletedAt != nil {
				continue
			}
			add(reminderFromChecklistItem(trip, item))
		}
	}

	for _, reminder := range routeReminderCandidates(trip) {
		add(reminder)
	}
	for _, reminder := range itineraryTransferReminderCandidates(trip) {
		add(reminder)
	}
	for _, reminder := range accommodationReminderCandidates(trip) {
		add(reminder)
	}
	for _, reminder := range weatherReminderCandidates(trip, forecast) {
		add(reminder)
	}
	for _, reminder := range baselineReminderCandidates(trip) {
		add(reminder)
	}
	if s.tripHasCollaborators(context.Background(), trip.ID) {
		add(entity.TripReminder{
			Title:              "Check group readiness",
			Description:        stringPtr("Review assigned high-priority preparation items with collaborators."),
			Category:           entity.ReminderCategoryGroup,
			Priority:           entity.ReminderPriorityHigh,
			Source:             entity.ReminderSourceSystem,
			TriggerDate:        offsetDate(trip.StartDate, -3),
			RelativeOffsetDays: reminderIntPtr(-3),
			Metadata:           map[string]any{"reason": "Trip has collaborators"},
		})
	}
	return dedupeReminderCandidates(out)
}

func reminderFromChecklistItem(trip *entity.Trip, item entity.TripChecklistItem) entity.TripReminder {
	category := mapChecklistReminderCategory(item.Category, item.ItemType)
	priority := entity.ReminderPriority(item.Priority)
	offset := checklistReminderOffset(item)
	triggerDate := offsetDate(trip.StartDate, offset)
	if item.DueDate != nil {
		triggerDate = dateOnly(*item.DueDate)
		offset = int(triggerDate.Sub(dateOnly(*trip.StartDate)).Hours() / 24)
	}
	return entity.TripReminder{
		Title:              item.Title,
		Description:        item.Description,
		Category:           category,
		Priority:           priority,
		Source:             entity.ReminderSourceChecklist,
		TriggerDate:        triggerDate,
		RelativeOffsetDays: &offset,
		AssignedToUserID:   item.AssignedToUserID,
		ChecklistItemID:    &item.ID,
		RelatedDayNumber:   item.RelatedDayNumber,
		RelatedItemIndex:   item.RelatedItemIndex,
		RelatedItemID:      item.RelatedItemID,
		Metadata: map[string]any{
			"reason": "Generated from checklist item",
		},
	}
}

func routeReminderCandidates(trip *entity.Trip) []entity.TripReminder {
	if trip.Route == nil {
		return nil
	}
	seenModes := map[string]struct{}{}
	for _, mode := range trip.Route.Preferences.PreferredModes {
		seenModes[aggregate.NormalizeRouteToken(mode)] = struct{}{}
	}
	for _, leg := range trip.Route.Legs {
		mode := aggregate.NormalizeRouteToken(leg.Mode)
		if mode != "" {
			seenModes[mode] = struct{}{}
		}
	}
	modes := make([]string, 0, len(seenModes))
	for mode := range seenModes {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	out := make([]entity.TripReminder, 0, len(modes))
	for _, mode := range modes {
		if reminder, ok := transportModeReminder(trip, mode); ok {
			out = append(out, reminder)
		}
	}
	for _, stop := range trip.Route.Stops {
		if aggregate.NormalizeRouteToken(stop.AccommodationHint) == "campsite" {
			out = append(out, campsiteReminderCandidates(trip)...)
			break
		}
	}
	return out
}

func itineraryTransferReminderCandidates(trip *entity.Trip) []entity.TripReminder {
	itinerary := parseItineraryLenient(trip.Itinerary)
	seen := map[string]struct{}{}
	out := []entity.TripReminder{}
	for _, day := range itinerary.Days {
		for index, item := range day.Items {
			mode := aggregate.NormalizeRouteToken(item.TransportMode)
			if item.Transfer != nil && mode == "" {
				mode = aggregate.NormalizeRouteToken(item.Transfer.Mode)
			}
			if mode == "" {
				continue
			}
			key := fmt.Sprintf("%s:%d:%d", mode, day.Day, index)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			reminder, ok := transportModeReminder(trip, mode)
			if !ok {
				continue
			}
			reminder.RelatedDayNumber = &day.Day
			reminder.RelatedItemIndex = &index
			reminder.Metadata = mergeMetadata(reminder.Metadata, map[string]any{
				"reason":        "Generated from itinerary transfer item",
				"transportMode": mode,
			})
			out = append(out, reminder)
		}
	}
	return out
}

func transportModeReminder(trip *entity.Trip, mode string) (entity.TripReminder, bool) {
	switch mode {
	case aggregate.TransportModeTrain, aggregate.TransportModeBus, aggregate.TransportModePublicTransport:
		return reminderWithOffset(trip, "Verify transport tickets", "Check departure time, platform or stop details, and save tickets offline.", entity.ReminderCategoryTransport, entity.ReminderPriorityHigh, entity.ReminderSourceTransport, -3, map[string]any{"transportMode": mode}), true
	case aggregate.TransportModeFlight:
		return reminderWithOffset(trip, "Prepare for flight", "Check boarding pass, baggage rules, airport transfer, and travel documents.", entity.ReminderCategoryTransport, entity.ReminderPriorityHigh, entity.ReminderSourceTransport, -2, map[string]any{"transportMode": mode}), true
	case aggregate.TransportModeCar, aggregate.TransportModeRentalCar:
		return reminderWithOffset(trip, "Prepare car route", "Check driving documents, fuel, tolls, parking, and route offline access.", entity.ReminderCategoryTransport, entity.ReminderPriorityHigh, entity.ReminderSourceTransport, -3, map[string]any{"transportMode": mode}), true
	case aggregate.TransportModeFerry, aggregate.TransportModeBoat:
		return reminderWithOffset(trip, "Verify ferry or boat details", "Confirm schedule, reservation, weather conditions, and boarding location.", entity.ReminderCategoryTransport, entity.ReminderPriorityHigh, entity.ReminderSourceTransport, -2, map[string]any{"transportMode": mode}), true
	case aggregate.TransportModeHiking:
		return reminderWithOffset(trip, "Check hiking route", "Review trail conditions, weather, safety gear, and offline maps.", entity.ReminderCategoryRoute, entity.ReminderPriorityHigh, entity.ReminderSourceRoute, -2, map[string]any{"transportMode": mode}), true
	case aggregate.TransportModeBike:
		return reminderWithOffset(trip, "Prepare bike route", "Check bike or rental details, helmet, lights, and route safety.", entity.ReminderCategoryRoute, entity.ReminderPriorityMedium, entity.ReminderSourceRoute, -2, map[string]any{"transportMode": mode}), true
	default:
		return entity.TripReminder{}, false
	}
}

func accommodationReminderCandidates(trip *entity.Trip) []entity.TripReminder {
	if trip.Accommodation == nil {
		return nil
	}
	out := []entity.TripReminder{
		reminderWithOffset(trip, "Confirm accommodation details", "Confirm reservation, check-in time, address, and cancellation details.", entity.ReminderCategoryAccommodation, entity.ReminderPriorityHigh, entity.ReminderSourceAccommodation, -5, map[string]any{"accommodationName": trip.Accommodation.Name}),
		reminderWithOffset(trip, "Save accommodation check-in info", "Save address, check-in instructions, and contact details offline.", entity.ReminderCategoryAccommodation, entity.ReminderPriorityMedium, entity.ReminderSourceAccommodation, -2, map[string]any{"accommodationName": trip.Accommodation.Name}),
	}
	if strings.Contains(strings.ToLower(string(trip.Accommodation.Type)+" "+trip.Accommodation.Name+" "+trip.Accommodation.Address), "camp") {
		out = append(out, campsiteReminderCandidates(trip)...)
	}
	return out
}

func campsiteReminderCandidates(trip *entity.Trip) []entity.TripReminder {
	return []entity.TripReminder{
		reminderWithOffset(trip, "Confirm campsite arrival rules", "Confirm campsite reservation, arrival window, quiet hours, and required equipment.", entity.ReminderCategoryAccommodation, entity.ReminderPriorityHigh, entity.ReminderSourceAccommodation, -5, map[string]any{"reason": "Camping accommodation"}),
		reminderWithOffset(trip, "Check camping gear", "Check tent, sleeping gear, lights, cooking gear, and weather protection.", entity.ReminderCategoryPacking, entity.ReminderPriorityHigh, entity.ReminderSourceSystem, -7, map[string]any{"reason": "Camping trip"}),
	}
}

func weatherReminderCandidates(trip *entity.Trip, forecast *weathercontext.WeatherForecast) []entity.TripReminder {
	out := []entity.TripReminder{
		reminderWithOffset(trip, "Check weather forecast", "Review the latest forecast and adjust packing before departure.", entity.ReminderCategoryWeather, entity.ReminderPriorityMedium, entity.ReminderSourceWeather, -2, map[string]any{"reason": "Weather review"}),
		reminderWithOffset(trip, "Re-check weather before departure", "Re-check the latest forecast one day before departure.", entity.ReminderCategoryWeather, entity.ReminderPriorityMedium, entity.ReminderSourceWeather, -1, map[string]any{"reason": "Weather review"}),
	}
	if forecast == nil || len(forecast.Days) == 0 {
		return out
	}
	var rain, hot, cold bool
	for _, day := range forecast.Days {
		condition := strings.ToLower(day.Condition + " " + day.Summary + " " + strings.Join(day.Warnings, " "))
		if day.PrecipitationChance >= 40 || strings.Contains(condition, "rain") || strings.Contains(condition, "storm") {
			rain = true
		}
		if day.TemperatureMaxC >= 28 || strings.Contains(condition, "hot") || strings.Contains(condition, "sun") {
			hot = true
		}
		if day.TemperatureMinC <= 5 || strings.Contains(condition, "cold") || strings.Contains(condition, "snow") {
			cold = true
		}
	}
	if rain {
		out = append(out, reminderWithOffset(trip, "Pack rain gear", "Rain is possible. Pack waterproof layers, shoe protection, and bag covers.", entity.ReminderCategoryWeather, entity.ReminderPriorityMedium, entity.ReminderSourceWeather, -2, map[string]any{"weatherSignal": "rain"}))
	}
	if hot {
		out = append(out, reminderWithOffset(trip, "Prepare sun and heat protection", "Pack sunscreen, hat, sunglasses, and reusable water bottle.", entity.ReminderCategoryWeather, entity.ReminderPriorityMedium, entity.ReminderSourceWeather, -2, map[string]any{"weatherSignal": "hot"}))
	}
	if cold {
		out = append(out, reminderWithOffset(trip, "Pack warm layers", "Cold weather is possible. Pack warm layers and weather-appropriate footwear.", entity.ReminderCategoryWeather, entity.ReminderPriorityMedium, entity.ReminderSourceWeather, -2, map[string]any{"weatherSignal": "cold"}))
	}
	return out
}

func baselineReminderCandidates(trip *entity.Trip) []entity.TripReminder {
	out := []entity.TripReminder{
		reminderWithOffset(trip, "Check travel documents", "Verify IDs, travel documents, insurance, and required confirmations. Verify official requirements yourself.", entity.ReminderCategoryDocuments, entity.ReminderPriorityCritical, entity.ReminderSourceSystem, -7, map[string]any{"reason": "Critical documents"}),
		reminderWithOffset(trip, "Download offline maps and tickets", "Save tickets, reservations, addresses, and offline maps before departure.", entity.ReminderCategoryBeforeDeparture, entity.ReminderPriorityHigh, entity.ReminderSourceSystem, -2, map[string]any{"reason": "Offline access"}),
		reminderWithOffset(trip, "Charge devices and power bank", "Charge phone, camera, headphones, and power bank.", entity.ReminderCategoryBeforeDeparture, entity.ReminderPriorityMedium, entity.ReminderSourceSystem, -1, map[string]any{"reason": "Before departure"}),
		reminderWithOffset(trip, "Finish packing", "Finish packing key items and check weather-sensitive gear.", entity.ReminderCategoryPacking, entity.ReminderPriorityMedium, entity.ReminderSourceSystem, -2, map[string]any{"reason": "Packing"}),
	}
	for _, style := range trip.Interests {
		token := aggregate.NormalizeRouteToken(style)
		if token == "hiking" || token == "camping" || token == "adventure" {
			out = append(out, reminderWithOffset(trip, "Check hiking or camping gear", "Check footwear, layers, first aid, lights, route files, and emergency basics.", entity.ReminderCategorySafety, entity.ReminderPriorityHigh, entity.ReminderSourceSystem, -7, map[string]any{"tripStyle": token}))
			break
		}
	}
	if trip.Route != nil {
		for _, style := range trip.Route.Preferences.TripStyles {
			token := aggregate.NormalizeRouteToken(style)
			if token == "hiking" || token == "camping" || token == "road_trip" || token == "island_hopping" {
				out = append(out, reminderWithOffset(trip, "Review route-specific preparation", "Review route constraints, equipment, weather exposure, and transport timing.", entity.ReminderCategoryRoute, entity.ReminderPriorityHigh, entity.ReminderSourceRoute, -3, map[string]any{"tripStyle": token}))
				break
			}
		}
	}
	return out
}

func reminderWithOffset(trip *entity.Trip, title, description string, category entity.ReminderCategory, priority entity.ReminderPriority, source entity.ReminderSource, offset int, metadata map[string]any) entity.TripReminder {
	return entity.TripReminder{
		Title:              title,
		Description:        stringPtr(description),
		Category:           category,
		Priority:           priority,
		Source:             source,
		TriggerDate:        offsetDate(trip.StartDate, offset),
		RelativeOffsetDays: &offset,
		Metadata:           metadata,
	}
}

func mergeReminderCandidates(existing []entity.TripReminder, candidates []entity.TripReminder) []entity.TripReminder {
	seen := map[string]struct{}{}
	for i := range existing {
		seen[reminderDuplicateKey(existing[i])] = struct{}{}
		if existing[i].ChecklistItemID != nil {
			seen["checklist:"+existing[i].ChecklistItemID.String()] = struct{}{}
		}
	}
	out := make([]entity.TripReminder, 0, len(candidates))
	for _, candidate := range candidates {
		keys := []string{reminderDuplicateKey(candidate)}
		if candidate.ChecklistItemID != nil {
			keys = append(keys, "checklist:"+candidate.ChecklistItemID.String())
		}
		duplicate := false
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		for _, key := range keys {
			seen[key] = struct{}{}
		}
		out = append(out, candidate)
	}
	return out
}

func dedupeReminderCandidates(candidates []entity.TripReminder) []entity.TripReminder {
	return mergeReminderCandidates(nil, candidates)
}

func reminderDuplicateKey(reminder entity.TripReminder) string {
	assigned := "unassigned"
	if reminder.AssignedToUserID != nil {
		assigned = reminder.AssignedToUserID.String()
	}
	return fmt.Sprintf("%s:%s:%s:%s",
		normalizeDuplicateText(reminder.Title),
		reminder.Category,
		assigned,
		reminder.TriggerDate.Format("2006-01-02"),
	)
}

func normalizeGenerateRemindersInput(in appdto.GenerateRemindersInput) (appdto.GenerateRemindersInput, error) {
	switch in.Mode {
	case "":
		in.Mode = appdto.GenerateRemindersModeFull
	case appdto.GenerateRemindersModeFull, appdto.GenerateRemindersModeAddMissing, appdto.GenerateRemindersModeCategory:
	default:
		return in, apperrs.NewInvalidInput("mode must be full, add_missing, or category")
	}
	if len(in.Instructions) > maxReminderInstructions {
		return in, apperrs.NewInvalidInput("instructions must be at most %d characters", maxReminderInstructions)
	}
	categories, err := normalizeReminderCategories(in.Categories)
	if err != nil {
		return in, err
	}
	if in.Mode == appdto.GenerateRemindersModeCategory && len(categories) == 0 {
		return in, apperrs.NewInvalidInput("categories is required for category mode")
	}
	in.Categories = categories
	return in, nil
}

func normalizeCreateReminderInput(in appdto.CreateReminderInput) (appdto.CreateReminderInput, error) {
	title, err := normalizeReminderTitle(in.Title)
	if err != nil {
		return in, err
	}
	in.Title = title
	category, err := normalizeReminderCategory(in.Category)
	if err != nil {
		return in, err
	}
	in.Category = category
	priority, err := normalizeReminderPriority(in.Priority)
	if err != nil {
		return in, err
	}
	in.Priority = priority
	if in.TriggerDate.IsZero() {
		return in, apperrs.NewInvalidInput("triggerDate is required")
	}
	if in.TriggerTime != nil {
		normalized, err := normalizeReminderTime(*in.TriggerTime)
		if err != nil {
			return in, err
		}
		in.TriggerTime = &normalized
	}
	in.Description = normalizeOptionalStringPtr(in.Description, maxReminderDescLength)
	in.Timezone = normalizeOptionalStringPtr(in.Timezone, 80)
	in.RelatedItemID = normalizeOptionalStringPtr(in.RelatedItemID, 100)
	in.Metadata = sanitizeReminderMetadata(in.Metadata)
	return in, nil
}

func applyReminderPatch(reminder *entity.TripReminder, in appdto.UpdateReminderInput) error {
	if in.Title != nil {
		title, err := normalizeReminderTitle(*in.Title)
		if err != nil {
			return err
		}
		reminder.Title = title
	}
	if in.ClearDescription {
		reminder.Description = nil
	} else if in.Description != nil {
		reminder.Description = normalizeOptionalStringPtr(in.Description, maxReminderDescLength)
	}
	if in.Category != nil {
		category, err := normalizeReminderCategory(*in.Category)
		if err != nil {
			return err
		}
		reminder.Category = category
	}
	if in.Priority != nil {
		priority, err := normalizeReminderPriority(*in.Priority)
		if err != nil {
			return err
		}
		reminder.Priority = priority
	}
	if in.TriggerDate != nil {
		reminder.TriggerDate = dateOnly(*in.TriggerDate)
	}
	if in.ClearTriggerTime {
		reminder.TriggerTime = nil
	} else if in.TriggerTime != nil {
		normalized, err := normalizeReminderTime(*in.TriggerTime)
		if err != nil {
			return err
		}
		reminder.TriggerTime = &normalized
	}
	if in.ClearTimezone {
		reminder.Timezone = nil
	} else if in.Timezone != nil {
		reminder.Timezone = normalizeOptionalStringPtr(in.Timezone, 80)
	}
	if in.ClearRelativeOffset {
		reminder.RelativeOffsetDays = nil
	} else if in.RelativeOffsetDays != nil {
		reminder.RelativeOffsetDays = in.RelativeOffsetDays
	}
	if in.ClearAssignee {
		reminder.AssignedToUserID = nil
	} else if in.AssignedToUserID != nil {
		reminder.AssignedToUserID = in.AssignedToUserID
	}
	if in.Metadata != nil {
		reminder.Metadata = mergeMetadata(reminder.Metadata, sanitizeReminderMetadata(in.Metadata))
	}
	if reminder.Metadata == nil {
		reminder.Metadata = map[string]any{}
	}
	reminder.Metadata["edited"] = true
	return nil
}

func normalizeReminderTitle(value string) (string, error) {
	title := strings.TrimSpace(value)
	if title == "" {
		return "", apperrs.NewInvalidInput("title is required")
	}
	if len(title) > maxReminderTitleLength {
		return "", apperrs.NewInvalidInput("title must be at most %d characters", maxReminderTitleLength)
	}
	return title, nil
}

func normalizeReminderCategory(value entity.ReminderCategory) (entity.ReminderCategory, error) {
	switch value {
	case "":
		return entity.ReminderCategoryOther, nil
	case entity.ReminderCategoryDocuments,
		entity.ReminderCategoryPacking,
		entity.ReminderCategoryTransport,
		entity.ReminderCategoryAccommodation,
		entity.ReminderCategoryWeather,
		entity.ReminderCategoryActivities,
		entity.ReminderCategoryGroup,
		entity.ReminderCategoryChecklist,
		entity.ReminderCategoryBeforeDeparture,
		entity.ReminderCategoryRoute,
		entity.ReminderCategorySafety,
		entity.ReminderCategoryOther:
		return value, nil
	default:
		return value, apperrs.NewInvalidInput("category is invalid")
	}
}

func normalizeReminderCategories(values []entity.ReminderCategory) ([]entity.ReminderCategory, error) {
	seen := map[entity.ReminderCategory]struct{}{}
	out := make([]entity.ReminderCategory, 0, len(values))
	for _, value := range values {
		normalized, err := normalizeReminderCategory(value)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeReminderPriority(value entity.ReminderPriority) (entity.ReminderPriority, error) {
	switch value {
	case "":
		return entity.ReminderPriorityMedium, nil
	case entity.ReminderPriorityLow, entity.ReminderPriorityMedium, entity.ReminderPriorityHigh, entity.ReminderPriorityCritical:
		return value, nil
	default:
		return value, apperrs.NewInvalidInput("priority is invalid")
	}
}

func normalizeReminderStatus(value entity.ReminderStatus) (entity.ReminderStatus, error) {
	switch value {
	case entity.ReminderStatusPending,
		entity.ReminderStatusSent,
		entity.ReminderStatusCompleted,
		entity.ReminderStatusDisabled,
		entity.ReminderStatusCancelled,
		entity.ReminderStatusFailed:
		return value, nil
	default:
		return value, apperrs.NewInvalidInput("status is invalid")
	}
}

func normalizeReminderTime(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := time.Parse("15:04", trimmed)
	if err != nil {
		return "", apperrs.NewInvalidInput("triggerTime must be in HH:MM format")
	}
	return parsed.Format("15:04"), nil
}

func mapChecklistReminderCategory(category entity.ChecklistCategory, itemType entity.ChecklistItemType) entity.ReminderCategory {
	switch category {
	case entity.ChecklistCategoryDocuments, entity.ChecklistCategoryMoney:
		return entity.ReminderCategoryDocuments
	case entity.ChecklistCategoryClothing, entity.ChecklistCategoryElectronics, entity.ChecklistCategoryCampingHiking:
		return entity.ReminderCategoryPacking
	case entity.ChecklistCategoryTransport:
		return entity.ReminderCategoryTransport
	case entity.ChecklistCategoryAccommodation:
		return entity.ReminderCategoryAccommodation
	case entity.ChecklistCategoryActivities:
		return entity.ReminderCategoryActivities
	case entity.ChecklistCategoryGroupItems:
		return entity.ReminderCategoryGroup
	case entity.ChecklistCategoryBeforeDeparture:
		return entity.ReminderCategoryBeforeDeparture
	case entity.ChecklistCategoryWeather:
		return entity.ReminderCategoryWeather
	case entity.ChecklistCategoryHealthSafety:
		return entity.ReminderCategorySafety
	default:
		if itemType == entity.ChecklistItemTypeReminder {
			return entity.ReminderCategoryChecklist
		}
		return entity.ReminderCategoryOther
	}
}

func checklistReminderOffset(item entity.TripChecklistItem) int {
	if item.Priority == entity.ChecklistPriorityCritical {
		return -7
	}
	switch item.Category {
	case entity.ChecklistCategoryDocuments:
		return -7
	case entity.ChecklistCategoryTransport:
		return -3
	case entity.ChecklistCategoryAccommodation:
		return -5
	case entity.ChecklistCategoryGroupItems:
		return -3
	case entity.ChecklistCategoryCampingHiking:
		return -7
	case entity.ChecklistCategoryBeforeDeparture:
		return -1
	case entity.ChecklistCategoryWeather:
		return -2
	default:
		if item.ItemType == entity.ChecklistItemTypeDocument {
			return -7
		}
		if item.ItemType == entity.ChecklistItemTypeBookingCheck {
			return -3
		}
		return -2
	}
}

func reminderCategorySelected(category entity.ReminderCategory, selected []entity.ReminderCategory) bool {
	if len(selected) == 0 {
		return true
	}
	for _, value := range selected {
		if value == category {
			return true
		}
	}
	return false
}

func canEditOwnManualReminder(actorID uuid.UUID, reminder *entity.TripReminder) bool {
	return reminder != nil &&
		reminder.Source == entity.ReminderSourceManual &&
		reminder.CreatedByUserID != nil &&
		*reminder.CreatedByUserID == actorID
}

func isAssignedTo(actorID uuid.UUID, reminder *entity.TripReminder) bool {
	return reminder != nil && reminder.AssignedToUserID != nil && *reminder.AssignedToUserID == actorID
}

func reminderIsDue(reminder entity.TripReminder, now time.Time) bool {
	dueAt := reminderDueAt(reminder)
	return !dueAt.After(now)
}

func reminderDueAt(reminder entity.TripReminder) time.Time {
	tz := defaultReminderTimezone
	if reminder.Timezone != nil && strings.TrimSpace(*reminder.Timezone) != "" {
		tz = strings.TrimSpace(*reminder.Timezone)
	}
	location, err := time.LoadLocation(tz)
	if err != nil {
		location = time.UTC
	}
	hour, minute := 9, 0
	if reminder.TriggerTime != nil && strings.TrimSpace(*reminder.TriggerTime) != "" {
		if parsed, err := time.Parse("15:04", *reminder.TriggerTime); err == nil {
			hour, minute = parsed.Hour(), parsed.Minute()
		}
	}
	return time.Date(reminder.TriggerDate.Year(), reminder.TriggerDate.Month(), reminder.TriggerDate.Day(), hour, minute, 0, 0, location).UTC()
}

func reminderNotificationMessage(trip *entity.Trip, reminder *entity.TripReminder) string {
	destination := tripDestination(trip)
	return fmt.Sprintf("%s starts soon. %s", destination, descriptionOrReminderTitle(reminder))
}

func descriptionOrReminderTitle(reminder *entity.TripReminder) string {
	if reminder == nil {
		return "Open your reminder timeline."
	}
	if reminder.Source == entity.ReminderSourceManual {
		return "Open your reminder timeline to review the task."
	}
	if reminder.Description != nil && strings.TrimSpace(*reminder.Description) != "" {
		return truncateReminderString(*reminder.Description, 180)
	}
	return reminder.Title
}

func (s *Service) notifyReminderAssigned(ctx context.Context, trip *entity.Trip, actorID uuid.UUID, reminder *entity.TripReminder) {
	if reminder == nil || reminder.AssignedToUserID == nil {
		return
	}
	s.notifyDirect(ctx,
		*reminder.AssignedToUserID,
		reminder.TripID,
		actorID,
		notifications.TypeReminderAssigned,
		"Trip reminder assigned",
		fmt.Sprintf("%s: %s", tripDestination(trip), reminder.Title),
		notifications.EntityTripReminder,
		&reminder.ID,
		map[string]any{
			"tripId":     reminder.TripID.String(),
			"reminderId": reminder.ID.String(),
			"category":   string(reminder.Category),
			"priority":   string(reminder.Priority),
		},
	)
}

func (s *Service) tripHasCollaborators(ctx context.Context, tripID uuid.UUID) bool {
	collaborators, err := s.repo.ListTripCollaborators(ctx, tripID)
	if err != nil {
		return false
	}
	for _, collaborator := range collaborators {
		if collaborator.Status == entity.CollaboratorStatusAccepted {
			return true
		}
	}
	return false
}

func routeReminderSignature(trip *entity.Trip) string {
	if trip == nil {
		return ""
	}
	payload := struct {
		StartDate     string                   `json:"startDate,omitempty"`
		Route         *aggregate.TripRoute     `json:"route,omitempty"`
		Accommodation *aggregate.Accommodation `json:"accommodation,omitempty"`
		Interests     []string                 `json:"interests,omitempty"`
	}{
		Route:         trip.Route,
		Accommodation: trip.Accommodation,
		Interests:     trip.Interests,
	}
	if trip.StartDate != nil {
		payload.StartDate = trip.StartDate.Format("2006-01-02")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func sanitizeReminderMetadata(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for key, value := range in {
		if len(out) >= 24 {
			break
		}
		if value == nil {
			continue
		}
		if s, ok := value.(string); ok {
			out[key] = truncateReminderString(s, maxReminderMetadataString)
			continue
		}
		out[key] = value
	}
	return out
}

func offsetDate(start *time.Time, offsetDays int) time.Time {
	if start == nil {
		return dateOnly(time.Now())
	}
	return dateOnly(start.AddDate(0, 0, offsetDays))
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func stringPtr(value string) *string {
	return &value
}

func reminderIntPtr(value int) *int {
	return &value
}

func truncateReminderString(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func metaStringValue(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, _ := meta[key].(string)
	return value
}

func intFromMeta(meta map[string]any, key string) int {
	if meta == nil {
		return 0
	}
	switch value := meta[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
