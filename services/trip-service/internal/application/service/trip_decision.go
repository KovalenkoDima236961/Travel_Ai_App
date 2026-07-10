package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	maxPollTitleLength       = 120
	maxPollDescriptionLength = 500
	maxPollOptions           = 20
	maxPollOptionLabelLength = 120
	maxPollOptionDescLength  = 300
	maxGroupSummaryLength    = 500
)

func (s *Service) CreateTripPoll(ctx context.Context, tripID uuid.UUID, in appdto.CreateTripPollInput) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	trip, access, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	normalized, options, err := normalizeCreatePollInput(in)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	poll := &entity.TripPoll{
		ID:                 uuid.New(),
		TripID:             tripID,
		CreatedByUserID:    user.ID,
		Title:              normalized.Title,
		Description:        normalized.Description,
		PollType:           normalized.PollType,
		Status:             entity.PollStatusOpen,
		AllowMultipleVotes: normalized.AllowMultipleVotes,
		ClosesAt:           normalized.ClosesAt,
		Metadata:           normalized.Metadata,
	}
	created, createdOptions, err := s.repo.CreateTripPollWithOptions(ctx, poll, options)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripPollCreated,
		EntityType:  activityEntityType(activity.EntityTripPoll),
		EntityID:    activityEntityID(created.ID),
		Metadata:    pollActivityMetadata(created, len(createdOptions)),
	})
	s.notifyTripBroadcast(ctx, trip, user.ID,
		notifications.TypeTripPollCreated,
		"New trip poll",
		fmt.Sprintf("A collaborator created a poll: %s", created.Title),
		notifications.EntityTripPoll,
		activityEntityID(created.ID),
		notificationPollMetadata(tripID, created),
	)

	return s.tripPollInfo(ctx, trip, access, *created, user.ID)
}

func (s *Service) ListTripPolls(ctx context.Context, tripID uuid.UUID) ([]appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	polls, err := s.repo.ListTripPollsByTrip(ctx, tripID, false)
	if err != nil {
		return nil, err
	}
	out := make([]appdto.TripPollInfo, 0, len(polls))
	for _, poll := range polls {
		info, err := s.tripPollInfo(ctx, trip, access, poll, user.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) GetTripPoll(ctx context.Context, tripID, pollID uuid.UUID) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	poll, err := s.repo.GetTripPollByID(ctx, tripID, pollID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	return s.tripPollInfo(ctx, trip, access, *poll, user.ID)
}

func (s *Service) VoteTripPoll(ctx context.Context, tripID, pollID uuid.UUID, in appdto.VoteTripPollInput) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	poll, err := s.repo.GetTripPollByID(ctx, tripID, pollID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if poll.Status != entity.PollStatusOpen {
		return appdto.TripPollInfo{}, apperrs.NewConflict("poll is not open")
	}
	if poll.ClosesAt != nil && time.Now().After(*poll.ClosesAt) {
		return appdto.TripPollInfo{}, apperrs.NewConflict("poll is closed")
	}
	options, err := s.repo.ListPollOptions(ctx, pollID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	votes, err := buildPollVotes(*poll, options, user.ID, in)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if _, err := s.repo.ReplaceUserPollVotes(ctx, pollID, user.ID, votes); err != nil {
		return appdto.TripPollInfo{}, err
	}
	return s.tripPollInfo(ctx, trip, access, *poll, user.ID)
}

func (s *Service) CloseTripPoll(ctx context.Context, tripID, pollID uuid.UUID) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	poll, err := s.repo.GetTripPollByID(ctx, tripID, pollID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if poll.Status != entity.PollStatusOpen {
		return appdto.TripPollInfo{}, apperrs.NewConflict("poll is not open")
	}
	if !access.CanEdit() && poll.CreatedByUserID != user.ID {
		return appdto.TripPollInfo{}, apperrs.ErrForbidden
	}
	closed, err := s.repo.CloseTripPoll(ctx, tripID, pollID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	options, _ := s.repo.ListPollOptions(ctx, pollID)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripPollClosed,
		EntityType:  activityEntityType(activity.EntityTripPoll),
		EntityID:    activityEntityID(closed.ID),
		Metadata:    pollActivityMetadata(closed, len(options)),
	})
	s.notifyTripBroadcast(ctx, trip, user.ID,
		notifications.TypeTripPollClosed,
		"Trip poll closed",
		fmt.Sprintf("A poll was closed: %s", closed.Title),
		notifications.EntityTripPoll,
		activityEntityID(closed.ID),
		notificationPollMetadata(tripID, closed),
	)
	return s.tripPollInfo(ctx, trip, access, *closed, user.ID)
}

func (s *Service) ArchiveTripPoll(ctx context.Context, tripID, pollID uuid.UUID) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	trip, access, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	archived, err := s.repo.ArchiveTripPoll(ctx, tripID, pollID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	options, _ := s.repo.ListPollOptions(ctx, pollID)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripPollArchived,
		EntityType:  activityEntityType(activity.EntityTripPoll),
		EntityID:    activityEntityID(archived.ID),
		Metadata:    pollActivityMetadata(archived, len(options)),
	})
	return s.tripPollInfo(ctx, trip, access, *archived, user.ID)
}

func (s *Service) SetItineraryItemReaction(ctx context.Context, tripID uuid.UUID, in appdto.SetItineraryItemReactionInput) (appdto.ItineraryItemReactionSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	if err := validateReactionInput(trip, in); err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	if _, err := s.repo.UpsertItineraryItemReaction(ctx, &entity.ItineraryItemReaction{
		ID:        uuid.New(),
		TripID:    tripID,
		DayNumber: in.DayNumber,
		ItemIndex: in.ItemIndex,
		ItemID:    strings.TrimSpace(in.ItemID),
		UserID:    user.ID,
		Reaction:  in.Reaction,
		Metadata:  in.Metadata,
	}); err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	return s.ListItineraryItemReactionsByItem(ctx, tripID, in.DayNumber, in.ItemIndex)
}

func (s *Service) DeleteMyItineraryItemReaction(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	if err := assertItineraryItemExists(trip, dayNumber, itemIndex); err != nil {
		return err
	}
	return s.repo.DeleteItineraryItemReaction(ctx, tripID, dayNumber, itemIndex, user.ID)
}

func (s *Service) ListItineraryItemReactions(ctx context.Context, tripID uuid.UUID) ([]appdto.ItineraryItemReactionSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	reactions, err := s.repo.ListItineraryItemReactionsByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return reactionSummaries(trip, reactions, user.ID), nil
}

func (s *Service) ListItineraryItemReactionsByItem(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int) (appdto.ItineraryItemReactionSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	if err := assertItineraryItemExists(trip, dayNumber, itemIndex); err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	reactions, err := s.repo.ListItineraryItemReactionsByItem(ctx, tripID, dayNumber, itemIndex)
	if err != nil {
		return appdto.ItineraryItemReactionSummary{}, err
	}
	summaries := reactionSummaries(trip, reactions, user.ID)
	if len(summaries) == 0 {
		return emptyReactionSummary(trip, dayNumber, itemIndex, user.ID), nil
	}
	return summaries[0], nil
}

func (s *Service) GetGroupPreferences(ctx context.Context, tripID uuid.UUID) (appdto.GroupPreferencesSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.GroupPreferencesSummary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.GroupPreferencesSummary{}, err
	}
	return s.buildGroupPreferences(ctx, trip)
}

func (s *Service) buildGroupPreferences(ctx context.Context, trip *entity.Trip) (appdto.GroupPreferencesSummary, error) {
	polls, err := s.repo.ListTripPollsByTrip(ctx, trip.ID, false)
	if err != nil {
		return appdto.GroupPreferencesSummary{}, err
	}
	reactions, err := s.repo.ListItineraryItemReactionsByTrip(ctx, trip.ID)
	if err != nil {
		return appdto.GroupPreferencesSummary{}, err
	}
	discoveryVotes, err := s.repo.ListDiscoverySuggestionVotesByTrip(ctx, trip.ID)
	if err != nil {
		return appdto.GroupPreferencesSummary{}, err
	}

	result := appdto.GroupPreferencesSummary{
		TripID:      trip.ID,
		GeneratedAt: time.Now().UTC(),
		Summary: appdto.GroupPreferencesCounts{
			CollaboratorCount: s.groupCollaboratorCount(ctx, trip),
			PollCount:         len(polls),
			ReactionCount:     len(reactions),
		},
	}
	transportScores := map[string]*appdto.GroupPreferenceScore{}
	destinationScores := map[string]*appdto.GroupPreferenceScore{}
	dateScores := map[string]*appdto.GroupPreferenceScore{}
	routeAlternativeScores := map[string]*appdto.GroupRouteAlternativeVote{}

	for _, poll := range polls {
		if poll.Status == entity.PollStatusOpen {
			result.Summary.OpenPollCount++
			result.Summary.OpenDecisionCount++
		}
		options, err := s.repo.ListPollOptions(ctx, poll.ID)
		if err != nil {
			return appdto.GroupPreferencesSummary{}, err
		}
		votes, err := s.repo.ListPollVotesByPoll(ctx, poll.ID)
		if err != nil {
			return appdto.GroupPreferencesSummary{}, err
		}
		results := buildPollResults(options, votes)
		if len(results.WinningOptionIDs) > 0 {
			choice := appdto.GroupPreferencePollChoice{
				PollID:   poll.ID,
				Title:    poll.Title,
				PollType: poll.PollType,
			}
			for _, optionResult := range results.Options {
				if !containsUUID(results.WinningOptionIDs, optionResult.OptionID) || optionResult.VoteCount == 0 {
					continue
				}
				choice.WinningOptions = append(choice.WinningOptions, appdto.GroupPreferenceOptionChoice{
					OptionID:   optionResult.OptionID,
					OptionKey:  optionResult.OptionKey,
					Label:      optionResult.Label,
					VoteCount:  optionResult.VoteCount,
					Percentage: optionResult.Percentage,
				})
			}
			if len(choice.WinningOptions) > 0 {
				result.TopPollChoices = append(result.TopPollChoices, choice)
			}
		}
		accumulatePollPreferenceScores(
			poll,
			options,
			results,
			transportScores,
			destinationScores,
			dateScores,
			routeAlternativeScores,
		)
	}

	reactionSummaries := reactionSummaries(trip, reactions, uuid.Nil)
	for _, summary := range reactionSummaries {
		mustCount := summary.Counts[entity.ItineraryReactionMustHave]
		skipCount := summary.Counts[entity.ItineraryReactionSkip]
		if mustCount > 0 {
			result.Summary.MustHaveItemCount++
			result.ItineraryPreferences.MustHaveItems = append(result.ItineraryPreferences.MustHaveItems, appdto.GroupPreferenceItineraryItem{
				DayNumber: summary.DayNumber,
				ItemIndex: summary.ItemIndex,
				ItemID:    summary.ItemID,
				Name:      summary.ItemName,
				Count:     mustCount,
				Score:     summary.Score,
			})
		}
		if skipCount > 0 {
			result.Summary.SkipItemCount++
			result.ItineraryPreferences.MostSkippedItems = append(result.ItineraryPreferences.MostSkippedItems, appdto.GroupPreferenceItineraryItem{
				DayNumber: summary.DayNumber,
				ItemIndex: summary.ItemIndex,
				ItemID:    summary.ItemID,
				Name:      summary.ItemName,
				Count:     skipCount,
				Score:     summary.Score,
			})
		}
		if mustCount > 0 && skipCount > 0 {
			result.ItineraryPreferences.Controversial = append(result.ItineraryPreferences.Controversial, appdto.GroupPreferenceItineraryItem{
				DayNumber: summary.DayNumber,
				ItemIndex: summary.ItemIndex,
				ItemID:    summary.ItemID,
				Name:      summary.ItemName,
				Count:     mustCount + skipCount,
				Score:     summary.Score,
			})
		}
	}
	sortGroupItems(result.ItineraryPreferences.MustHaveItems)
	sortGroupItems(result.ItineraryPreferences.MostSkippedItems)
	sortGroupItems(result.ItineraryPreferences.Controversial)
	result.ItineraryPreferences.MustHaveItems = limitGroupItems(result.ItineraryPreferences.MustHaveItems, 5)
	result.ItineraryPreferences.MostSkippedItems = limitGroupItems(result.ItineraryPreferences.MostSkippedItems, 5)
	result.ItineraryPreferences.Controversial = limitGroupItems(result.ItineraryPreferences.Controversial, 5)

	accumulateDiscoveryScores(discoveryVotes, destinationScores)
	result.TransportPreferences = sortedScores(transportScores, 5)
	result.DestinationPreferences = sortedScores(destinationScores, 5)
	result.DatePreferences = sortedScores(dateScores, 5)
	result.RouteAlternativeVotes = sortedRouteAlternativeVotes(routeAlternativeScores, 5)
	result.AIConstraintSummary = buildAIConstraintSummary(result)
	result.AIConstraints = appdto.GroupPreferencesAIConstraints{
		Summary:                 result.AIConstraintSummary,
		MustHaveItems:           result.ItineraryPreferences.MustHaveItems,
		SkipCandidates:          result.ItineraryPreferences.MostSkippedItems,
		PreferredDestinations:   scoreLabels(result.DestinationPreferences),
		PreferredTransportModes: scoreKeys(result.TransportPreferences),
		PreferredDates:          scoreLabels(result.DatePreferences),
		RouteAlternativeVotes:   result.RouteAlternativeVotes,
		OpenDecisionCount:       result.Summary.OpenDecisionCount,
	}
	if len(result.RouteAlternativeVotes) > 0 {
		result.AIConstraints.PreferredRouteAlternativeID = result.RouteAlternativeVotes[0].AlternativeID
		result.AIConstraints.PreferredRouteSessionID = result.RouteAlternativeVotes[0].SessionID
	}
	return result, nil
}

func (s *Service) tripPollInfo(ctx context.Context, _ *entity.Trip, access TripAccess, poll entity.TripPoll, userID uuid.UUID) (appdto.TripPollInfo, error) {
	options, err := s.repo.ListPollOptions(ctx, poll.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	votes, err := s.repo.ListPollVotesByPoll(ctx, poll.ID)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	userVotes := make([]entity.TripPollVote, 0)
	for _, vote := range votes {
		if vote.UserID == userID {
			userVotes = append(userVotes, vote)
		}
	}
	return appdto.TripPollInfo{
		Poll:      poll,
		Options:   options,
		Results:   buildPollResults(options, votes),
		UserVotes: userVotes,
		CanManage: access.CanEdit() || poll.CreatedByUserID == userID,
		CanVote:   access.CanView() && poll.Status == entity.PollStatusOpen && (poll.ClosesAt == nil || time.Now().Before(*poll.ClosesAt)),
	}, nil
}

func normalizeCreatePollInput(in appdto.CreateTripPollInput) (appdto.CreateTripPollInput, []entity.TripPollOption, error) {
	title := strings.TrimSpace(in.Title)
	if len(title) < 2 || len(title) > maxPollTitleLength {
		return in, nil, apperrs.NewInvalidInput("title must be between 2 and %d characters", maxPollTitleLength)
	}
	description := strings.TrimSpace(in.Description)
	if len(description) > maxPollDescriptionLength {
		return in, nil, apperrs.NewInvalidInput("description must be at most %d characters", maxPollDescriptionLength)
	}
	if !in.PollType.Valid() {
		return in, nil, apperrs.NewInvalidInput("pollType is invalid")
	}
	if in.ClosesAt != nil && !in.ClosesAt.After(time.Now()) {
		return in, nil, apperrs.NewInvalidInput("closesAt must be in the future")
	}
	if in.PollType == entity.PollTypeYesNo && len(in.Options) == 0 {
		in.Options = []appdto.CreateTripPollOptionInput{
			{OptionKey: "yes", Label: "Yes"},
			{OptionKey: "no", Label: "No"},
		}
	}
	if len(in.Options) == 0 {
		return in, nil, apperrs.NewInvalidInput("options are required")
	}
	if len(in.Options) > maxPollOptions {
		return in, nil, apperrs.NewInvalidInput("options must contain at most %d items", maxPollOptions)
	}
	if in.PollType != entity.PollTypeMultipleChoice {
		in.AllowMultipleVotes = false
	}
	options := make([]entity.TripPollOption, 0, len(in.Options))
	keys := map[string]struct{}{}
	for i, inputOption := range in.Options {
		label := strings.TrimSpace(inputOption.Label)
		if label == "" || len(label) > maxPollOptionLabelLength {
			return in, nil, apperrs.NewInvalidInput("option label must be between 1 and %d characters", maxPollOptionLabelLength)
		}
		description := strings.TrimSpace(inputOption.Description)
		if len(description) > maxPollOptionDescLength {
			return in, nil, apperrs.NewInvalidInput("option description must be at most %d characters", maxPollOptionDescLength)
		}
		key := normalizeOptionKey(inputOption.OptionKey, label, i)
		if _, exists := keys[key]; exists {
			return in, nil, apperrs.NewInvalidInput("option keys must be unique")
		}
		keys[key] = struct{}{}
		options = append(options, entity.TripPollOption{
			ID:          uuid.New(),
			OptionKey:   key,
			Label:       label,
			Description: description,
			SortOrder:   i,
			Metadata:    inputOption.Metadata,
		})
	}
	in.Title = title
	in.Description = description
	return in, options, nil
}

func buildPollVotes(poll entity.TripPoll, options []entity.TripPollOption, userID uuid.UUID, in appdto.VoteTripPollInput) ([]entity.TripPollVote, error) {
	optionSet := map[uuid.UUID]entity.TripPollOption{}
	for _, option := range options {
		optionSet[option.ID] = option
	}
	selected := dedupeUUIDs(in.OptionIDs)
	requireOptions := poll.PollType != entity.PollTypeRating
	if poll.PollType == entity.PollTypeRating && len(options) > 0 {
		requireOptions = true
	}
	if requireOptions && len(selected) == 0 {
		return nil, apperrs.NewInvalidInput("optionIds are required")
	}
	for _, optionID := range selected {
		if _, ok := optionSet[optionID]; !ok {
			return nil, apperrs.NewInvalidInput("optionIds must belong to the poll")
		}
	}
	switch poll.PollType {
	case entity.PollTypeSingleChoice, entity.PollTypeYesNo, entity.PollTypeDateChoice:
		if len(selected) != 1 {
			return nil, apperrs.NewInvalidInput("exactly one optionId is required")
		}
	case entity.PollTypeMultipleChoice:
		if len(selected) > len(options) {
			return nil, apperrs.NewInvalidInput("too many optionIds")
		}
	case entity.PollTypeRating:
		if in.RatingValue == nil || *in.RatingValue < 1 || *in.RatingValue > 5 {
			return nil, apperrs.NewInvalidInput("ratingValue must be between 1 and 5")
		}
		if len(options) > 0 && len(selected) != 1 {
			return nil, apperrs.NewInvalidInput("exactly one optionId is required for rating polls")
		}
	default:
		return nil, apperrs.NewInvalidInput("pollType is invalid")
	}
	votes := make([]entity.TripPollVote, 0, maxInt(1, len(selected)))
	if poll.PollType == entity.PollTypeRating && len(options) == 0 {
		votes = append(votes, entity.TripPollVote{
			ID:          uuid.New(),
			PollID:      poll.ID,
			UserID:      userID,
			RatingValue: in.RatingValue,
			Metadata:    in.Metadata,
		})
		return votes, nil
	}
	for _, optionID := range selected {
		id := optionID
		votes = append(votes, entity.TripPollVote{
			ID:          uuid.New(),
			PollID:      poll.ID,
			OptionID:    &id,
			UserID:      userID,
			VoteValue:   strings.TrimSpace(in.VoteValue),
			RatingValue: in.RatingValue,
			Metadata:    in.Metadata,
		})
	}
	return votes, nil
}

func buildPollResults(options []entity.TripPollOption, votes []entity.TripPollVote) appdto.PollResults {
	optionVotes := map[uuid.UUID][]entity.TripPollVote{}
	voters := map[uuid.UUID]struct{}{}
	for _, vote := range votes {
		voters[vote.UserID] = struct{}{}
		if vote.OptionID == nil {
			continue
		}
		optionVotes[*vote.OptionID] = append(optionVotes[*vote.OptionID], vote)
	}
	results := appdto.PollResults{
		TotalVoters: len(voters),
		TotalVotes:  len(votes),
		Options:     make([]appdto.PollOptionResult, 0, len(options)),
	}
	maxVotes := 0
	for _, option := range options {
		optionVoteSet := optionVotes[option.ID]
		voteCount := len(optionVoteSet)
		if voteCount > maxVotes {
			maxVotes = voteCount
		}
		percentage := 0
		if len(voters) > 0 {
			percentage = int(float64(voteCount)/float64(len(voters))*100 + 0.5)
		}
		var averageRating *float64
		ratingTotal := 0
		ratingCount := 0
		for _, vote := range optionVoteSet {
			if vote.RatingValue != nil {
				ratingTotal += *vote.RatingValue
				ratingCount++
			}
		}
		if ratingCount > 0 {
			avg := float64(ratingTotal) / float64(ratingCount)
			averageRating = &avg
		}
		results.Options = append(results.Options, appdto.PollOptionResult{
			OptionID:      option.ID,
			OptionKey:     option.OptionKey,
			Label:         option.Label,
			VoteCount:     voteCount,
			Percentage:    percentage,
			AverageRating: averageRating,
		})
	}
	if maxVotes > 0 {
		for _, optionResult := range results.Options {
			if optionResult.VoteCount == maxVotes {
				results.WinningOptionIDs = append(results.WinningOptionIDs, optionResult.OptionID)
			}
		}
	}
	return results
}

func validateReactionInput(trip *entity.Trip, in appdto.SetItineraryItemReactionInput) error {
	if in.DayNumber < 1 {
		return apperrs.NewInvalidInput("dayNumber must be >= 1")
	}
	if in.ItemIndex < 0 {
		return apperrs.NewInvalidInput("itemIndex must be >= 0")
	}
	if !in.Reaction.Valid() {
		return apperrs.NewInvalidInput("reaction is invalid")
	}
	return assertItineraryItemExists(trip, in.DayNumber, in.ItemIndex)
}

func reactionSummaries(trip *entity.Trip, reactions []entity.ItineraryItemReaction, currentUserID uuid.UUID) []appdto.ItineraryItemReactionSummary {
	byItem := map[string]*appdto.ItineraryItemReactionSummary{}
	for _, reaction := range reactions {
		key := itemKey(reaction.DayNumber, reaction.ItemIndex)
		summary := byItem[key]
		if summary == nil {
			summary = &appdto.ItineraryItemReactionSummary{
				DayNumber: reaction.DayNumber,
				ItemIndex: reaction.ItemIndex,
				ItemID:    reaction.ItemID,
				ItemName:  itineraryItemName(trip, reaction.DayNumber, reaction.ItemIndex),
				Counts: map[entity.ItineraryReaction]int{
					entity.ItineraryReactionMustHave: 0,
					entity.ItineraryReactionWantToDo: 0,
					entity.ItineraryReactionNeutral:  0,
					entity.ItineraryReactionSkip:     0,
				},
			}
			byItem[key] = summary
		}
		summary.Counts[reaction.Reaction]++
		summary.Score += reactionScore(reaction.Reaction)
		if currentUserID != uuid.Nil && reaction.UserID == currentUserID {
			selected := reaction.Reaction
			summary.CurrentUserReaction = &selected
		}
	}
	out := make([]appdto.ItineraryItemReactionSummary, 0, len(byItem))
	for _, summary := range byItem {
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DayNumber == out[j].DayNumber {
			return out[i].ItemIndex < out[j].ItemIndex
		}
		return out[i].DayNumber < out[j].DayNumber
	})
	return out
}

func emptyReactionSummary(trip *entity.Trip, dayNumber, itemIndex int, _ uuid.UUID) appdto.ItineraryItemReactionSummary {
	return appdto.ItineraryItemReactionSummary{
		DayNumber: dayNumber,
		ItemIndex: itemIndex,
		ItemName:  itineraryItemName(trip, dayNumber, itemIndex),
		Counts: map[entity.ItineraryReaction]int{
			entity.ItineraryReactionMustHave: 0,
			entity.ItineraryReactionWantToDo: 0,
			entity.ItineraryReactionNeutral:  0,
			entity.ItineraryReactionSkip:     0,
		},
	}
}

func (s *Service) groupCollaboratorCount(ctx context.Context, trip *entity.Trip) int {
	seen := map[uuid.UUID]struct{}{}
	if trip.UserID != nil {
		seen[*trip.UserID] = struct{}{}
	}
	collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
	if err == nil {
		for _, collaborator := range collaborators {
			if collaborator.Status == entity.CollaboratorStatusAccepted {
				seen[collaborator.UserID] = struct{}{}
			}
		}
	}
	if trip.WorkspaceID != nil && s.workspaceProvider != nil {
		members, err := s.workspaceProvider.ListMembers(ctx, *trip.WorkspaceID)
		if err == nil {
			for _, member := range members {
				if member.Status == workspaces.MemberStatusActive {
					seen[member.UserID] = struct{}{}
				}
			}
		}
	}
	return len(seen)
}

func accumulatePollPreferenceScores(
	poll entity.TripPoll,
	options []entity.TripPollOption,
	results appdto.PollResults,
	transportScores map[string]*appdto.GroupPreferenceScore,
	destinationScores map[string]*appdto.GroupPreferenceScore,
	dateScores map[string]*appdto.GroupPreferenceScore,
	routeAlternativeScores map[string]*appdto.GroupRouteAlternativeVote,
) {
	optionByID := map[uuid.UUID]entity.TripPollOption{}
	for _, option := range options {
		optionByID[option.ID] = option
	}
	for _, result := range results.Options {
		if result.VoteCount == 0 {
			continue
		}
		option := optionByID[result.OptionID]
		category := metadataString(option.Metadata, "category")
		if category == "" {
			category = metadataString(poll.Metadata, "category")
		}
		switch strings.ToLower(category) {
		case "transport":
			key := metadataString(option.Metadata, "mode")
			if key == "" {
				key = option.OptionKey
			}
			addScore(transportScores, key, option.Label, result.VoteCount, result.VoteCount)
		case "route_alternative":
			alternativeID := metadataString(option.Metadata, "alternativeId")
			if alternativeID == "" {
				alternativeID = option.OptionKey
			}
			sessionID := metadataString(option.Metadata, "sessionId")
			if sessionID == "" {
				sessionID = metadataString(poll.Metadata, "sessionId")
			}
			label := metadataString(option.Metadata, "routeTitle")
			if label == "" {
				label = metadataString(option.Metadata, "destination")
			}
			if label == "" {
				label = option.Label
			}
			addRouteAlternativeScore(routeAlternativeScores, poll.ID, sessionID, alternativeID, label, result.VoteCount, result.VoteCount)
			addScore(destinationScores, normalizeScoreKey(label), label, result.VoteCount, result.VoteCount)
		case "destination", "route":
			label := metadataString(option.Metadata, "destination")
			if label == "" {
				label = option.Label
			}
			addScore(destinationScores, normalizeScoreKey(label), label, result.VoteCount, result.VoteCount)
		case "date", "dates":
			label := metadataString(option.Metadata, "date")
			if label == "" {
				label = metadataString(option.Metadata, "dateRange")
			}
			if label == "" {
				label = option.Label
			}
			addScore(dateScores, normalizeScoreKey(label), label, result.VoteCount, result.VoteCount)
		}
	}
}

func accumulateDiscoveryScores(votes []entity.DiscoverySuggestionVote, destinationScores map[string]*appdto.GroupPreferenceScore) {
	for _, vote := range votes {
		score := discoveryVoteScore(vote.Vote)
		if score == 0 {
			continue
		}
		label := metadataString(vote.Metadata, "destination")
		if label == "" {
			label = vote.SuggestionID
		}
		addScore(destinationScores, normalizeScoreKey(label), label, score, 1)
	}
}

func buildAIConstraintSummary(summary appdto.GroupPreferencesSummary) string {
	lines := []string{}
	if len(summary.ItineraryPreferences.MustHaveItems) > 0 {
		lines = append(lines, "Keep must-have activities: "+joinItemNames(summary.ItineraryPreferences.MustHaveItems)+".")
	}
	if len(summary.ItineraryPreferences.MostSkippedItems) > 0 {
		lines = append(lines, "Avoid or replace skipped items where possible: "+joinItemNames(summary.ItineraryPreferences.MostSkippedItems)+".")
	}
	if len(summary.DestinationPreferences) > 0 {
		lines = append(lines, "Preferred destination: "+summary.DestinationPreferences[0].Label+".")
	}
	if len(summary.TransportPreferences) > 0 {
		lines = append(lines, "Preferred transport: "+summary.TransportPreferences[0].Label+".")
	}
	if len(summary.DatePreferences) > 0 {
		lines = append(lines, "Date preference: "+summary.DatePreferences[0].Label+".")
	}
	if len(summary.RouteAlternativeVotes) > 0 {
		lines = append(lines, "Preferred route alternative: "+summary.RouteAlternativeVotes[0].Label+".")
	}
	if summary.Summary.OpenDecisionCount > 0 {
		lines = append(lines, fmt.Sprintf("%d group decision(s) are still open.", summary.Summary.OpenDecisionCount))
	}
	if len(lines) == 0 {
		return "There is no clear group winner yet."
	}
	out := strings.Join(lines, " ")
	if len(out) > maxGroupSummaryLength {
		out = strings.TrimSpace(out[:maxGroupSummaryLength-1]) + "…"
	}
	return out
}

func pollActivityMetadata(poll *entity.TripPoll, optionCount int) map[string]any {
	return map[string]any{
		"pollId":      poll.ID.String(),
		"pollTitle":   poll.Title,
		"pollType":    string(poll.PollType),
		"optionCount": optionCount,
	}
}

func notificationPollMetadata(tripID uuid.UUID, poll *entity.TripPoll) map[string]any {
	return map[string]any{
		"tripId":    tripID.String(),
		"pollId":    poll.ID.String(),
		"pollTitle": poll.Title,
	}
}

func normalizeOptionKey(raw, label string, index int) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(label))
	}
	var b strings.Builder
	lastDash := false
	for _, r := range key {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	normalized := strings.Trim(b.String(), "-")
	if normalized == "" {
		normalized = "option-" + strconv.Itoa(index+1)
	}
	return normalized
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
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

func reactionScore(reaction entity.ItineraryReaction) int {
	switch reaction {
	case entity.ItineraryReactionMustHave:
		return 3
	case entity.ItineraryReactionWantToDo:
		return 1
	case entity.ItineraryReactionSkip:
		return -2
	default:
		return 0
	}
}

func discoveryVoteScore(vote entity.DiscoverySuggestionVoteValue) int {
	switch vote {
	case entity.DiscoverySuggestionVoteFavorite:
		return 3
	case entity.DiscoverySuggestionVoteLike:
		return 1
	case entity.DiscoverySuggestionVoteDislike:
		return -1
	case entity.DiscoverySuggestionVoteNotInterested:
		return -2
	default:
		return 0
	}
}

func itemKey(dayNumber, itemIndex int) string {
	return fmt.Sprintf("%d:%d", dayNumber, itemIndex)
}

func containsUUID(values []uuid.UUID, needle uuid.UUID) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func normalizeScoreKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func addScore(scores map[string]*appdto.GroupPreferenceScore, key, label string, score int, votes int) {
	key = normalizeScoreKey(key)
	label = strings.TrimSpace(label)
	if key == "" {
		return
	}
	if label == "" {
		label = key
	}
	existing := scores[key]
	if existing == nil {
		scores[key] = &appdto.GroupPreferenceScore{Key: key, Label: label, Score: score, Votes: votes}
		return
	}
	existing.Score += score
	existing.Votes += votes
}

func addRouteAlternativeScore(
	scores map[string]*appdto.GroupRouteAlternativeVote,
	pollID uuid.UUID,
	sessionID string,
	alternativeID string,
	label string,
	score int,
	votes int,
) {
	sessionID = strings.TrimSpace(sessionID)
	alternativeID = strings.TrimSpace(alternativeID)
	if sessionID == "" || alternativeID == "" {
		return
	}
	key := sessionID + ":" + alternativeID
	label = strings.TrimSpace(label)
	if label == "" {
		label = alternativeID
	}
	existing := scores[key]
	if existing == nil {
		scores[key] = &appdto.GroupRouteAlternativeVote{
			SessionID:     sessionID,
			AlternativeID: alternativeID,
			Label:         label,
			Score:         score,
			Votes:         votes,
			PollID:        pollID,
		}
		return
	}
	existing.Score += score
	existing.Votes += votes
}

func sortedScores(scores map[string]*appdto.GroupPreferenceScore, limit int) []appdto.GroupPreferenceScore {
	out := make([]appdto.GroupPreferenceScore, 0, len(scores))
	for _, score := range scores {
		if score.Score <= 0 {
			continue
		}
		out = append(out, *score)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Label < out[j].Label
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func sortedRouteAlternativeVotes(
	scores map[string]*appdto.GroupRouteAlternativeVote,
	limit int,
) []appdto.GroupRouteAlternativeVote {
	out := make([]appdto.GroupRouteAlternativeVote, 0, len(scores))
	for _, score := range scores {
		if score.Score <= 0 {
			continue
		}
		out = append(out, *score)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Label < out[j].Label
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func sortGroupItems(items []appdto.GroupPreferenceItineraryItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			if items[i].DayNumber == items[j].DayNumber {
				return items[i].ItemIndex < items[j].ItemIndex
			}
			return items[i].DayNumber < items[j].DayNumber
		}
		return items[i].Count > items[j].Count
	})
}

func limitGroupItems(items []appdto.GroupPreferenceItineraryItem, limit int) []appdto.GroupPreferenceItineraryItem {
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func scoreLabels(scores []appdto.GroupPreferenceScore) []string {
	out := make([]string, 0, len(scores))
	for _, score := range scores {
		out = append(out, score.Label)
	}
	return out
}

func scoreKeys(scores []appdto.GroupPreferenceScore) []string {
	out := make([]string, 0, len(scores))
	for _, score := range scores {
		out = append(out, score.Key)
	}
	return out
}

func joinItemNames(items []appdto.GroupPreferenceItineraryItem) string {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprintf("Day %d item %d", item.DayNumber, item.ItemIndex+1)
		}
		labels = append(labels, label)
		if len(labels) == 3 {
			break
		}
	}
	return strings.Join(labels, ", ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
