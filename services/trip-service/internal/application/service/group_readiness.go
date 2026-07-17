package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/groupreadiness"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
)

const (
	defaultReadinessNudgeDedupeHours = 24
	maxReadinessNudgeDedupeHours     = 168
	maxReadinessNudgeMessageLength   = 500
)

func (s *Service) GetGroupReadiness(
	ctx context.Context,
	tripID uuid.UUID,
	options groupreadiness.Options,
) (groupreadiness.Response, error) {
	started := time.Now()
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return groupreadiness.Response{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return groupreadiness.Response{}, err
	}
	cacheKey := summaryCacheKey("group_readiness", trip, user.ID, options.IncludeDetails, options.IncludeDebug, access.Role())
	if cached, ok := s.summaryCache.get("group_readiness", cacheKey); ok {
		if response, valid := cached.(groupreadiness.Response); valid {
			return response, nil
		}
	}

	snapshot, err := s.groupReadinessSnapshot(ctx, trip, access, user)
	if err != nil {
		return groupreadiness.Response{}, err
	}
	options.CanNudge = access.CanEdit()
	response := groupreadiness.Evaluate(snapshot, options)

	s.log.Info("group readiness evaluated",
		zap.String("trip_id", tripID.String()),
		zap.String("user_id", user.ID.String()),
		zap.Int("score", response.Score),
		zap.String("level", string(response.Level)),
		zap.Int("member_count", len(response.Members)),
		zap.Duration("duration", time.Since(started)),
	)
	s.summaryCache.set("group_readiness", cacheKey, response)
	return response, nil
}

func (s *Service) SendGroupReadinessNudge(
	ctx context.Context,
	tripID uuid.UUID,
	in groupreadiness.NudgeRequest,
) (groupreadiness.NudgeResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return groupreadiness.NudgeResponse{}, err
	}
	trip, access, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return groupreadiness.NudgeResponse{}, err
	}
	if !access.CanEdit() {
		return groupreadiness.NudgeResponse{}, apperrs.ErrForbidden
	}
	categories, err := normalizeReadinessNudgeCategories(in.Categories)
	if err != nil {
		return groupreadiness.NudgeResponse{}, err
	}
	message := strings.TrimSpace(in.Message)
	if len(message) > maxReadinessNudgeMessageLength {
		return groupreadiness.NudgeResponse{}, apperrs.NewInvalidInput("message must be at most %d characters", maxReadinessNudgeMessageLength)
	}
	if message == "" {
		message = "Please update your trip readiness items when you have a moment."
	}
	dedupeWindowHours := in.DedupeWindowHours
	if dedupeWindowHours <= 0 {
		dedupeWindowHours = defaultReadinessNudgeDedupeHours
	}
	if dedupeWindowHours > maxReadinessNudgeDedupeHours {
		return groupreadiness.NudgeResponse{}, apperrs.NewInvalidInput("dedupeWindowHours must be at most %d", maxReadinessNudgeDedupeHours)
	}

	members, _, err := s.groupReadinessMembers(ctx, trip, access, user)
	if err != nil {
		return groupreadiness.NudgeResponse{}, err
	}
	memberIDs := map[uuid.UUID]struct{}{}
	for _, member := range members {
		memberIDs[member.UserID] = struct{}{}
	}

	targetIDs := normalizeNudgeTargets(in.TargetUserIDs, user.ID)
	if len(targetIDs) == 0 {
		return groupreadiness.NudgeResponse{}, apperrs.NewInvalidInput("targetUserIds is required")
	}

	validTargets := make([]uuid.UUID, 0, len(targetIDs))
	skipped := 0
	deduped := 0
	for _, targetID := range targetIDs {
		if _, ok := memberIDs[targetID]; !ok {
			skipped++
			continue
		}
		if s.recentReadinessNudgeExists(ctx, tripID, user.ID, targetID, categories, time.Duration(dedupeWindowHours)*time.Hour) {
			deduped++
			continue
		}
		validTargets = append(validTargets, targetID)
	}
	if len(validTargets) == 0 {
		return groupreadiness.NudgeResponse{
			SentCount:         0,
			SkippedCount:      skipped,
			DedupedCount:      deduped,
			TargetUserIDs:     []uuid.UUID{},
			Categories:        categories,
			DedupeWindowHours: dedupeWindowHours,
		}, nil
	}

	notificationType := readinessNotificationType(categories)
	inputs := make([]notifications.NotificationCreateInput, 0, len(validTargets))
	categoryStrings := categoriesToStrings(categories)
	for _, targetID := range validTargets {
		target := targetID
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      targetID,
			TripID:      &tripID,
			ActorUserID: &user.ID,
			Type:        notificationType,
			Title:       "Trip readiness reminder",
			Message:     message,
			EntityType:  activityEntityType(notifications.EntityTrip),
			EntityID:    activityEntityID(tripID),
			Metadata: map[string]any{
				"tripId":       tripID.String(),
				"targetUserId": target.String(),
				"categories":   categoryStrings,
			},
		})
	}
	s.sendNotifications(ctx, inputs)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventGroupReadinessNudgeSent,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"targetUserCount": len(validTargets),
			"targetUserIds":   uuidStrings(validTargets),
			"categories":      categoryStrings,
			"dedupedCount":    deduped,
			"skippedCount":    skipped,
		},
	})

	return groupreadiness.NudgeResponse{
		SentCount:         len(validTargets),
		SkippedCount:      skipped,
		DedupedCount:      deduped,
		TargetUserIDs:     validTargets,
		Categories:        categories,
		DedupeWindowHours: dedupeWindowHours,
	}, nil
}

func (s *Service) groupReadinessSnapshot(
	ctx context.Context,
	trip *entity.Trip,
	access TripAccess,
	user auth.AuthenticatedUser,
) (groupreadiness.Snapshot, error) {
	members, collaborators, err := s.groupReadinessMembers(ctx, trip, access, user)
	if err != nil {
		return groupreadiness.Snapshot{}, err
	}
	snapshot := groupreadiness.Snapshot{
		Trip:    trip,
		Members: members,
		Now:     time.Now().UTC(),
	}

	availability, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, trip.ID)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "availability")
	} else {
		snapshot.AvailabilityResponses = availability
	}

	polls, err := s.repo.ListTripPollsByTrip(ctx, trip.ID, false)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "polls")
	} else {
		for _, poll := range polls {
			votes, err := s.repo.ListPollVotesByPoll(ctx, poll.ID)
			if err != nil {
				snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "poll_votes")
			}
			snapshot.Polls = append(snapshot.Polls, groupreadiness.PollSnapshot{Poll: poll, Votes: votes})
		}
	}

	checklist, err := s.activeChecklistWithItems(ctx, trip.ID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "checklist")
		}
	} else {
		snapshot.Checklist = checklist
	}

	reminders, err := s.repo.ListTripRemindersByTrip(ctx, trip.ID, entity.TripReminderFilters{})
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "reminders")
	} else {
		snapshot.Reminders = reminders
	}

	settlements, err := s.repo.ListTripSettlementsByTrip(ctx, trip.ID)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "settlements")
	} else {
		snapshot.Settlements = settlements
	}

	if trip.WorkspaceID != nil {
		approval, err := s.repo.GetTripApprovalFields(ctx, trip.ID)
		if err != nil {
			snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "approval")
		} else {
			snapshot.Approval = approval
		}
	}

	_ = collaborators
	return snapshot, nil
}

func (s *Service) groupReadinessMembers(
	ctx context.Context,
	trip *entity.Trip,
	access TripAccess,
	user auth.AuthenticatedUser,
) ([]groupreadiness.Member, []entity.TripCollaborator, error) {
	collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
	if err != nil {
		return nil, nil, err
	}

	type memberEntry struct {
		member groupreadiness.Member
		rank   int
	}
	byID := map[uuid.UUID]memberEntry{}
	add := func(userID uuid.UUID, displayName string, role string, rank int) {
		if userID == uuid.Nil {
			return
		}
		if displayName == "" {
			displayName = fallbackDisplayName(userID)
		}
		isCurrentUser := userID == user.ID
		entry, exists := byID[userID]
		if exists && entry.rank <= rank {
			if isCurrentUser {
				entry.member.IsCurrentUser = true
				byID[userID] = entry
			}
			return
		}
		byID[userID] = memberEntry{
			rank: rank,
			member: groupreadiness.Member{
				UserID:        userID,
				DisplayName:   displayName,
				Role:          role,
				IsCurrentUser: isCurrentUser,
			},
		}
	}

	if trip.UserID != nil {
		add(*trip.UserID, displayNameForUser(*trip.UserID, &user, trip, nil), "owner", 1)
	}
	for i := range collaborators {
		collaborator := collaborators[i]
		if collaborator.Status != entity.CollaboratorStatusAccepted {
			continue
		}
		add(collaborator.UserID, displayNameForUser(collaborator.UserID, &user, trip, &collaborator), string(collaborator.Role), 2)
	}
	travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, trip.ID)
	if err == nil {
		for _, traveler := range travelers {
			if traveler.LinkedUserID == nil {
				continue
			}
			add(*traveler.LinkedUserID, strings.TrimSpace(traveler.Name), string(traveler.Role), 4)
		}
	}
	add(user.ID, displayNameForUser(user.ID, &user, trip, nil), access.Role(), 3)

	members := make([]groupreadiness.Member, 0, len(byID))
	for _, entry := range byID {
		members = append(members, entry.member)
	}
	sort.SliceStable(members, func(i, j int) bool {
		if readinessRoleRank(members[i].Role) != readinessRoleRank(members[j].Role) {
			return readinessRoleRank(members[i].Role) < readinessRoleRank(members[j].Role)
		}
		return strings.ToLower(members[i].DisplayName) < strings.ToLower(members[j].DisplayName)
	})
	return members, collaborators, nil
}

func readinessRoleRank(role string) int {
	switch role {
	case "owner":
		return 1
	case "editor", "admin", "member":
		return 2
	case "viewer":
		return 3
	default:
		return 4
	}
}

func normalizeReadinessNudgeCategories(in []groupreadiness.Category) ([]groupreadiness.Category, error) {
	if len(in) == 0 {
		return nil, apperrs.NewInvalidInput("categories is required")
	}
	seen := map[groupreadiness.Category]struct{}{}
	out := make([]groupreadiness.Category, 0, len(in))
	for _, category := range in {
		category = groupreadiness.Category(strings.TrimSpace(string(category)))
		if !readinessNudgeCategoryAllowed(category) {
			return nil, apperrs.NewInvalidInput("unsupported readiness category %q", category)
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		out = append(out, category)
	}
	return out, nil
}

func readinessNudgeCategoryAllowed(category groupreadiness.Category) bool {
	switch category {
	case groupreadiness.CategoryAvailability,
		groupreadiness.CategoryPolls,
		groupreadiness.CategoryChecklist,
		groupreadiness.CategoryReminders,
		groupreadiness.CategorySettlements:
		return true
	default:
		return false
	}
}

func normalizeNudgeTargets(ids []uuid.UUID, actorID uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{actorID: {}}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Service) recentReadinessNudgeExists(
	ctx context.Context,
	tripID uuid.UUID,
	actorID uuid.UUID,
	targetID uuid.UUID,
	categories []groupreadiness.Category,
	window time.Duration,
) bool {
	if s.activity == nil || window <= 0 {
		return false
	}
	result, err := s.activity.List(ctx, activity.ListActivityInput{
		TripID: tripID,
		Limit:  activity.MaxLimit,
	})
	if err != nil {
		return false
	}
	cutoff := time.Now().UTC().Add(-window)
	wanted := map[string]struct{}{}
	for _, category := range categories {
		wanted[string(category)] = struct{}{}
	}
	for _, event := range result.Events {
		if event.EventType != activity.EventGroupReadinessNudgeSent || event.ActorUserID == nil || *event.ActorUserID != actorID {
			continue
		}
		if event.CreatedAt.Before(cutoff) {
			continue
		}
		targets := metadataStringSet(event.Metadata, "targetUserIds")
		if _, ok := targets[targetID.String()]; !ok {
			continue
		}
		eventCategories := metadataStringSet(event.Metadata, "categories")
		for category := range wanted {
			if _, ok := eventCategories[category]; ok {
				return true
			}
		}
	}
	return false
}

func readinessNotificationType(categories []groupreadiness.Category) string {
	if len(categories) != 1 {
		return notifications.TypeGroupReadinessNudge
	}
	switch categories[0] {
	case groupreadiness.CategoryAvailability:
		return notifications.TypeAvailabilityNudge
	case groupreadiness.CategoryChecklist:
		return notifications.TypeChecklistAssignmentNudge
	case groupreadiness.CategoryReminders:
		return notifications.TypeReminderTaskNudge
	case groupreadiness.CategoryPolls:
		return notifications.TypePollVoteNudge
	case groupreadiness.CategorySettlements:
		return notifications.TypeSettlementNudge
	default:
		return notifications.TypeGroupReadinessNudge
	}
}

func categoriesToStrings(categories []groupreadiness.Category) []string {
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		out = append(out, string(category))
	}
	sort.Strings(out)
	return out
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func metadataStringSet(metadata map[string]any, key string) map[string]struct{} {
	out := map[string]struct{}{}
	if metadata == nil {
		return out
	}
	switch value := metadata[key].(type) {
	case []string:
		for _, item := range value {
			out[item] = struct{}{}
		}
	case []any:
		for _, item := range value {
			if s, ok := item.(string); ok {
				out[s] = struct{}{}
			}
		}
	case string:
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func ReadinessNudgeRequest(categories ...groupreadiness.Category) groupreadiness.NudgeRequest {
	return groupreadiness.NudgeRequest{Categories: categories}
}

func readinessConvenienceInput(targetUserIDs []uuid.UUID, message string, category groupreadiness.Category) groupreadiness.NudgeRequest {
	return groupreadiness.NudgeRequest{
		TargetUserIDs: targetUserIDs,
		Categories:    []groupreadiness.Category{category},
		Message:       message,
	}
}

func nudgeRequestFromCategory(category groupreadiness.Category, targetUserIDs []uuid.UUID, message string) groupreadiness.NudgeRequest {
	return readinessConvenienceInput(targetUserIDs, message, category)
}

func readinessNudgeDescription(categories []groupreadiness.Category) string {
	parts := categoriesToStrings(categories)
	if len(parts) == 0 {
		return "readiness"
	}
	return fmt.Sprintf("readiness: %s", strings.Join(parts, ", "))
}
