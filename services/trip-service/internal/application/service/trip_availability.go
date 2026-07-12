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

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
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
	maxAvailabilityRangesPerType = 20
	maxAvailabilityNotesLength   = 500
	maxAvailabilityTimezone      = 100
	defaultDateOptionLimit       = 10
	maxDateOptionLimit           = 100
	dateOptionSearchCapDays      = 365
)

type availabilityParticipant struct {
	UserID      uuid.UUID
	DisplayName string
}

type availabilityRange struct {
	start time.Time
	end   time.Time
	raw   entity.AvailabilityDateRange
}

type parsedAvailabilityResponse struct {
	response    entity.TripAvailabilityResponse
	available   []availabilityRange
	unavailable []availabilityRange
	preferred   []availabilityRange
	submitted   bool
}

type dateWindow struct {
	start time.Time
	end   time.Time
	days  int
}

func (s *Service) GetTripAvailability(ctx context.Context, tripID uuid.UUID) (appdto.TripAvailabilityList, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripAvailabilityList{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripAvailabilityList{}, err
	}
	responses, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, tripID)
	if err != nil {
		return appdto.TripAvailabilityList{}, err
	}
	participants, err := s.availabilityParticipants(ctx, trip, responses, &user)
	if err != nil {
		return appdto.TripAvailabilityList{}, err
	}
	return buildAvailabilityList(tripID, participants, responses), nil
}

func (s *Service) UpsertMyTripAvailability(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.UpsertTripAvailabilityInput,
) (appdto.TripAvailabilityResponseInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripAvailabilityResponseInfo{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripAvailabilityResponseInfo{}, err
	}
	normalized, err := normalizeAvailabilityInput(in)
	if err != nil {
		return appdto.TripAvailabilityResponseInfo{}, err
	}
	_, existingErr := s.repo.GetTripAvailabilityResponseByTripAndUser(ctx, tripID, user.ID)
	eventType := activity.EventAvailabilityUpdated
	if errors.Is(existingErr, domainerrs.ErrNotFound) {
		eventType = activity.EventAvailabilitySubmitted
	} else if existingErr != nil {
		return appdto.TripAvailabilityResponseInfo{}, existingErr
	}

	saved, err := s.repo.UpsertTripAvailabilityResponse(ctx, &entity.TripAvailabilityResponse{
		ID:                uuid.New(),
		TripID:            tripID,
		UserID:            user.ID,
		AvailableRanges:   normalized.AvailableRanges,
		UnavailableRanges: normalized.UnavailableRanges,
		PreferredRanges:   normalized.PreferredRanges,
		MinTripDays:       normalized.MinTripDays,
		MaxTripDays:       normalized.MaxTripDays,
		Timezone:          normalized.Timezone,
		Notes:             normalized.Notes,
	})
	if err != nil {
		return appdto.TripAvailabilityResponseInfo{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   eventType,
		EntityType:  activityEntityType(activity.EntityAvailability),
		EntityID:    activityEntityID(saved.ID),
		Metadata: map[string]any{
			"availableRangeCount":   len(saved.AvailableRanges),
			"unavailableRangeCount": len(saved.UnavailableRanges),
			"preferredRangeCount":   len(saved.PreferredRanges),
			"minTripDays":           nullableIntMetadataValue(saved.MinTripDays),
			"maxTripDays":           nullableIntMetadataValue(saved.MaxTripDays),
		},
	})

	displayName := displayNameForUser(user.ID, &user, trip, nil)
	return availabilityResponseInfo(displayName, *saved, true), nil
}

func (s *Service) DeleteMyTripAvailability(ctx context.Context, tripID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return err
	}
	existing, err := s.repo.GetTripAvailabilityResponseByTripAndUser(ctx, tripID, user.ID)
	if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
		return err
	}
	if err := s.repo.DeleteTripAvailabilityResponse(ctx, tripID, user.ID); err != nil {
		return err
	}
	if existing != nil {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      tripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventAvailabilityRemoved,
			EntityType:  activityEntityType(activity.EntityAvailability),
			EntityID:    activityEntityID(existing.ID),
			Metadata:    map[string]any{"userId": user.ID.String()},
		})
	}
	return nil
}

func (s *Service) GetTripDateOptions(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.DateOptionsInput,
) (appdto.DateOptionsResult, error) {
	return s.generateTripDateOptions(ctx, tripID, in)
}

func (s *Service) GenerateTripDateOptions(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.DateOptionsInput,
) (appdto.DateOptionsResult, error) {
	return s.generateTripDateOptions(ctx, tripID, in)
}

func (s *Service) ApplyTripDateOption(
	ctx context.Context,
	tripID uuid.UUID,
	optionID string,
	in appdto.ApplyDateOptionInput,
) (appdto.ApplyDateOptionResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ApplyDateOptionResult{}, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ApplyDateOptionResult{}, err
	}
	expectedRevision := current.ItineraryRevision
	if len(current.Itinerary) > 0 {
		expectedRevision, err = requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
		if err != nil {
			return appdto.ApplyDateOptionResult{}, err
		}
		if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
			return appdto.ApplyDateOptionResult{}, err
		}
	} else if in.ExpectedItineraryRevision != nil {
		expectedRevision, err = requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
		if err != nil {
			return appdto.ApplyDateOptionResult{}, err
		}
		if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
			return appdto.ApplyDateOptionResult{}, err
		}
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return appdto.ApplyDateOptionResult{}, err
	}

	options, err := s.generateTripDateOptions(ctx, tripID, appdto.DateOptionsInput{Limit: maxDateOptionLimit})
	if err != nil {
		return appdto.ApplyDateOptionResult{}, err
	}
	var selected *appdto.DateOption
	for i := range options.Options {
		if options.Options[i].ID == strings.TrimSpace(optionID) {
			selected = &options.Options[i]
			break
		}
	}
	if selected == nil {
		return appdto.ApplyDateOptionResult{}, apperrs.NewInvalidInput("date option is no longer available")
	}

	start, _ := parseAvailabilityDate(selected.StartDate)
	newDays := int32(selected.DurationDays)
	metadata := cloneMetadata(current.CreationMetadata)
	metadata["selectedDateOption"] = selectedDateOptionMetadata(*selected)
	metadata["selectedDateOptionAppliedAt"] = time.Now().UTC().Format(time.RFC3339)
	metadata["selectedDateOptionAppliedByUserId"] = user.ID.String()

	updatedRoute := current.Route
	routeShifted := false
	warnings := []string{}
	if current.Route != nil && current.StartDate != nil && current.Days == newDays {
		shifted, ok := shiftedRouteDates(current.Route, *current.StartDate, start)
		if ok {
			updatedRoute = shifted
			routeShifted = true
		} else if routeHasDates(current.Route) {
			warnings = append(warnings, "Route dates could not be shifted automatically.")
		}
	} else if current.Route != nil && current.Days != newDays && routeHasDates(current.Route) {
		warnings = append(warnings, "Route dates were left unchanged because the trip duration changed.")
	}

	itineraryStale := len(current.Itinerary) > 0
	if itineraryStale {
		warnings = append(warnings, "Existing itinerary may be outdated after changing trip dates.")
	}
	if in.RegenerateItinerary {
		warnings = append(warnings, "A regeneration job should be queued by the API handler after dates are applied.")
	}

	updated, err := s.repo.UpdateTripDatesAndMetadata(ctx, tripID, ownerID, &start, newDays, updatedRoute, metadata)
	if err != nil {
		return appdto.ApplyDateOptionResult{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventDateOptionApplied,
		EntityType:  activityEntityType(activity.EntityDateOption),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"optionId":     selected.ID,
			"startDate":    selected.StartDate,
			"endDate":      selected.EndDate,
			"durationDays": selected.DurationDays,
			"score":        selected.Score,
			"routeShifted": routeShifted,
		},
	})
	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Trip dates changed")
	s.notifyTripBroadcast(ctx, updated, user.ID,
		notifications.TypeDateOptionApplied,
		"Trip dates selected",
		fmt.Sprintf("Dates were selected for %s: %s to %s.", tripDestination(updated), selected.StartDate, selected.EndDate),
		notifications.EntityDateOption,
		activityEntityID(tripID),
		map[string]any{
			"tripId":       tripID.String(),
			"optionId":     selected.ID,
			"startDate":    selected.StartDate,
			"endDate":      selected.EndDate,
			"durationDays": selected.DurationDays,
			"score":        selected.Score,
		},
	)

	return appdto.ApplyDateOptionResult{
		Trip:                      updated,
		AppliedOption:             *selected,
		ItineraryStale:            itineraryStale,
		RouteShifted:              routeShifted,
		RegenerateItinerary:       in.RegenerateItinerary,
		Warnings:                  warnings,
		ExpectedItineraryRevision: expectedRevision,
	}, nil
}

func (s *Service) CreateDateOptionsPoll(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.CreateDateOptionsPollInput,
) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripPollInfo{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Which dates work best?"
	}
	if len(in.OptionIDs) == 0 {
		return appdto.TripPollInfo{}, apperrs.NewInvalidInput("optionIds is required")
	}
	if len(in.OptionIDs) > maxPollOptions {
		return appdto.TripPollInfo{}, apperrs.NewInvalidInput("optionIds must contain at most %d options", maxPollOptions)
	}
	optionsResult, err := s.generateTripDateOptions(ctx, tripID, appdto.DateOptionsInput{Limit: maxDateOptionLimit})
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	byID := make(map[string]appdto.DateOption, len(optionsResult.Options))
	for _, option := range optionsResult.Options {
		byID[option.ID] = option
	}
	pollOptions := make([]appdto.CreateTripPollOptionInput, 0, len(in.OptionIDs))
	for index, optionID := range in.OptionIDs {
		option, ok := byID[strings.TrimSpace(optionID)]
		if !ok {
			return appdto.TripPollInfo{}, apperrs.NewInvalidInput("date option %q is no longer available", optionID)
		}
		pollOptions = append(pollOptions, appdto.CreateTripPollOptionInput{
			OptionKey: fmt.Sprintf("date_%d", index+1),
			Label: fmt.Sprintf(
				"%s-%s · %d days · %d/%d available",
				option.StartDate,
				option.EndDate,
				option.DurationDays,
				option.AvailableUserCount,
				option.TotalUserCount,
			),
			Description: fmt.Sprintf("Score %d · %d conflict(s) · %d missing response(s)", option.Score, option.ConflictUserCount, option.MissingResponseUserCount),
			Metadata: map[string]any{
				"category":     "date_option",
				"optionId":     option.ID,
				"startDate":    option.StartDate,
				"endDate":      option.EndDate,
				"durationDays": option.DurationDays,
				"score":        option.Score,
			},
		})
	}

	info, err := s.CreateTripPoll(ctx, tripID, appdto.CreateTripPollInput{
		Title:              title,
		PollType:           entity.PollTypeDateChoice,
		AllowMultipleVotes: false,
		Metadata: map[string]any{
			"category": "date_options",
			"source":   "group_availability",
		},
		Options: pollOptions,
	})
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventDateOptionsPollCreated,
		EntityType:  activityEntityType(activity.EntityTripPoll),
		EntityID:    activityEntityID(info.Poll.ID),
		Metadata: map[string]any{
			"pollId":      info.Poll.ID.String(),
			"optionCount": len(info.Options),
		},
	})
	return info, nil
}

func (s *Service) RequestTripAvailability(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.RequestAvailabilityInput,
) (appdto.TripAvailabilitySummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripAvailabilitySummary{}, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripAvailabilitySummary{}, err
	}
	message := strings.TrimSpace(in.Message)
	if len(message) > maxAvailabilityNotesLength {
		return appdto.TripAvailabilitySummary{}, apperrs.NewInvalidInput("message must be at most %d characters", maxAvailabilityNotesLength)
	}
	responses, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, tripID)
	if err != nil {
		return appdto.TripAvailabilitySummary{}, err
	}
	participants, err := s.availabilityParticipants(ctx, trip, responses, &user)
	if err != nil {
		return appdto.TripAvailabilitySummary{}, err
	}
	list := buildAvailabilityList(tripID, participants, responses)
	for _, missing := range list.Summary.MissingUsers {
		s.notifyDirect(ctx, missing.UserID, tripID, user.ID,
			notifications.TypeAvailabilityRequested,
			"Availability requested",
			availabilityRequestMessage(trip, message),
			notifications.EntityAvailability,
			activityEntityID(tripID),
			map[string]any{
				"tripId":      tripID.String(),
				"destination": tripDestination(trip),
			},
		)
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventAvailabilityRequested,
		EntityType:  activityEntityType(activity.EntityAvailability),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"missingCount": len(list.Summary.MissingUsers),
			"message":      truncateForMetadata(message, 160),
		},
	})
	return list.Summary, nil
}

func (s *Service) groupAvailabilityForPlanning(
	ctx context.Context,
	trip *entity.Trip,
) (*planningconstraints.GroupAvailability, error) {
	if trip == nil {
		return nil, nil
	}
	responses, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, trip.ID)
	if err != nil {
		return nil, err
	}
	participants, err := s.availabilityParticipants(ctx, trip, responses, nil)
	if err != nil {
		return nil, err
	}
	list := buildAvailabilityList(trip.ID, participants, responses)
	out := &planningconstraints.GroupAvailability{
		SubmittedCount:       list.Summary.SubmittedCount,
		TotalCollaborators:   list.Summary.TotalCollaborators,
		MissingResponseCount: list.Summary.MissingCount,
		Notes:                groupAvailabilityNotes(responses),
	}
	if selected := selectedDateOptionForPlanning(trip.CreationMetadata); selected != nil {
		out.SelectedDateOption = selected
	}
	if out.SubmittedCount == 0 && out.SelectedDateOption == nil {
		return nil, nil
	}
	return out, nil
}

func (s *Service) generateTripDateOptions(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.DateOptionsInput,
) (appdto.DateOptionsResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.DateOptionsResult{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.DateOptionsResult{}, err
	}
	normalized, err := normalizeDateOptionsInput(in)
	if err != nil {
		return appdto.DateOptionsResult{}, err
	}
	responses, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, tripID)
	if err != nil {
		return appdto.DateOptionsResult{}, err
	}
	participants, err := s.availabilityParticipants(ctx, trip, responses, &user)
	if err != nil {
		return appdto.DateOptionsResult{}, err
	}
	list := buildAvailabilityList(tripID, participants, responses)
	result := calculateDateOptions(trip, participants, responses, normalized)
	result.Summary.ResponseCount = list.Summary.SubmittedCount
	result.Summary.TotalCollaborators = list.Summary.TotalCollaborators
	result.Summary.MissingResponseCount = list.Summary.MissingCount
	if len(result.Options) > 0 {
		result.Summary.RecommendedOptionID = result.Options[0].ID
	}
	return result, nil
}

func (s *Service) availabilityParticipants(
	ctx context.Context,
	trip *entity.Trip,
	responses []entity.TripAvailabilityResponse,
	currentUser *auth.AuthenticatedUser,
) ([]availabilityParticipant, error) {
	byID := map[uuid.UUID]availabilityParticipant{}
	add := func(userID uuid.UUID, displayName string) {
		if userID == uuid.Nil {
			return
		}
		if existing, ok := byID[userID]; ok && existing.DisplayName != "" {
			return
		}
		byID[userID] = availabilityParticipant{UserID: userID, DisplayName: displayName}
	}
	if trip != nil && trip.UserID != nil {
		add(*trip.UserID, displayNameForUser(*trip.UserID, currentUser, trip, nil))
	}
	if trip != nil {
		collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
		if err != nil {
			return nil, err
		}
		for i := range collaborators {
			c := collaborators[i]
			if c.Status != entity.CollaboratorStatusAccepted {
				continue
			}
			add(c.UserID, displayNameForUser(c.UserID, currentUser, trip, &c))
		}
	}
	for _, response := range responses {
		add(response.UserID, displayNameForUser(response.UserID, currentUser, trip, nil))
	}
	if currentUser != nil {
		add(currentUser.ID, displayNameForUser(currentUser.ID, currentUser, trip, nil))
	}
	participants := make([]availabilityParticipant, 0, len(byID))
	for _, participant := range byID {
		if participant.DisplayName == "" {
			participant.DisplayName = fallbackDisplayName(participant.UserID)
		}
		participants = append(participants, participant)
	}
	sort.SliceStable(participants, func(i, j int) bool {
		return strings.ToLower(participants[i].DisplayName) < strings.ToLower(participants[j].DisplayName)
	})
	return participants, nil
}

func normalizeAvailabilityInput(in appdto.UpsertTripAvailabilityInput) (appdto.UpsertTripAvailabilityInput, error) {
	available, err := normalizeAvailabilityRanges(in.AvailableRanges, "availableRanges")
	if err != nil {
		return in, err
	}
	unavailable, err := normalizeAvailabilityRanges(in.UnavailableRanges, "unavailableRanges")
	if err != nil {
		return in, err
	}
	preferred, err := normalizeAvailabilityRanges(in.PreferredRanges, "preferredRanges")
	if err != nil {
		return in, err
	}
	if in.MinTripDays != nil && (*in.MinTripDays < 1 || *in.MinTripDays > 60) {
		return in, apperrs.NewInvalidInput("minTripDays must be between 1 and 60")
	}
	if in.MaxTripDays != nil && (*in.MaxTripDays < 1 || *in.MaxTripDays > 90) {
		return in, apperrs.NewInvalidInput("maxTripDays must be between 1 and 90")
	}
	if in.MinTripDays != nil && in.MaxTripDays != nil && *in.MaxTripDays < *in.MinTripDays {
		return in, apperrs.NewInvalidInput("maxTripDays must be greater than or equal to minTripDays")
	}
	timezone := strings.TrimSpace(in.Timezone)
	if len(timezone) > maxAvailabilityTimezone {
		return in, apperrs.NewInvalidInput("timezone must be at most %d characters", maxAvailabilityTimezone)
	}
	notes := strings.TrimSpace(in.Notes)
	if len(notes) > maxAvailabilityNotesLength {
		return in, apperrs.NewInvalidInput("notes must be at most %d characters", maxAvailabilityNotesLength)
	}
	in.AvailableRanges = available
	in.UnavailableRanges = unavailable
	in.PreferredRanges = preferred
	in.Timezone = timezone
	in.Notes = notes
	return in, nil
}

func normalizeAvailabilityRanges(ranges []entity.AvailabilityDateRange, field string) ([]entity.AvailabilityDateRange, error) {
	if len(ranges) > maxAvailabilityRangesPerType {
		return nil, apperrs.NewInvalidInput("%s must contain at most %d ranges", field, maxAvailabilityRangesPerType)
	}
	out := make([]entity.AvailabilityDateRange, 0, len(ranges))
	for i, r := range ranges {
		start := strings.TrimSpace(r.StartDate)
		end := strings.TrimSpace(r.EndDate)
		if start == "" || end == "" {
			return nil, apperrs.NewInvalidInput("%s[%d].startDate and endDate are required", field, i)
		}
		startDate, err := parseAvailabilityDate(start)
		if err != nil {
			return nil, apperrs.NewInvalidInput("%s[%d].startDate must be in YYYY-MM-DD format", field, i)
		}
		endDate, err := parseAvailabilityDate(end)
		if err != nil {
			return nil, apperrs.NewInvalidInput("%s[%d].endDate must be in YYYY-MM-DD format", field, i)
		}
		if endDate.Before(startDate) {
			return nil, apperrs.NewInvalidInput("%s[%d].endDate must be on or after startDate", field, i)
		}
		out = append(out, entity.AvailabilityDateRange{StartDate: start, EndDate: end})
	}
	return out, nil
}

func normalizeDateOptionsInput(in appdto.DateOptionsInput) (appdto.DateOptionsInput, error) {
	if in.MinDays != nil && (*in.MinDays < 1 || *in.MinDays > 60) {
		return in, apperrs.NewInvalidInput("minDays must be between 1 and 60")
	}
	if in.MaxDays != nil && (*in.MaxDays < 1 || *in.MaxDays > 90) {
		return in, apperrs.NewInvalidInput("maxDays must be between 1 and 90")
	}
	if in.MinDays != nil && in.MaxDays != nil && *in.MaxDays < *in.MinDays {
		return in, apperrs.NewInvalidInput("maxDays must be greater than or equal to minDays")
	}
	if strings.TrimSpace(in.SearchStartDate) != "" {
		if _, err := parseAvailabilityDate(in.SearchStartDate); err != nil {
			return in, apperrs.NewInvalidInput("searchStartDate must be in YYYY-MM-DD format")
		}
	}
	if strings.TrimSpace(in.SearchEndDate) != "" {
		if _, err := parseAvailabilityDate(in.SearchEndDate); err != nil {
			return in, apperrs.NewInvalidInput("searchEndDate must be in YYYY-MM-DD format")
		}
	}
	if in.SearchStartDate != "" && in.SearchEndDate != "" {
		start, _ := parseAvailabilityDate(in.SearchStartDate)
		end, _ := parseAvailabilityDate(in.SearchEndDate)
		if end.Before(start) {
			return in, apperrs.NewInvalidInput("searchEndDate must be on or after searchStartDate")
		}
	}
	if in.Limit == 0 {
		in.Limit = defaultDateOptionLimit
	}
	if in.Limit < 1 || in.Limit > maxDateOptionLimit {
		return in, apperrs.NewInvalidInput("limit must be between 1 and %d", maxDateOptionLimit)
	}
	in.SearchStartDate = strings.TrimSpace(in.SearchStartDate)
	in.SearchEndDate = strings.TrimSpace(in.SearchEndDate)
	return in, nil
}

func buildAvailabilityList(
	tripID uuid.UUID,
	participants []availabilityParticipant,
	responses []entity.TripAvailabilityResponse,
) appdto.TripAvailabilityList {
	responseByUser := make(map[uuid.UUID]entity.TripAvailabilityResponse, len(responses))
	for _, response := range responses {
		responseByUser[response.UserID] = response
	}
	items := make([]appdto.TripAvailabilityResponseInfo, 0, len(participants))
	missing := make([]appdto.TripAvailabilityUserSummary, 0)
	submitted := 0
	for _, participant := range participants {
		if response, ok := responseByUser[participant.UserID]; ok {
			items = append(items, availabilityResponseInfo(participant.DisplayName, response, true))
			submitted++
			continue
		}
		items = append(items, appdto.TripAvailabilityResponseInfo{
			UserID:            participant.UserID,
			DisplayName:       participant.DisplayName,
			AvailableRanges:   []entity.AvailabilityDateRange{},
			UnavailableRanges: []entity.AvailabilityDateRange{},
			PreferredRanges:   []entity.AvailabilityDateRange{},
			Submitted:         false,
		})
		missing = append(missing, appdto.TripAvailabilityUserSummary{
			UserID:      participant.UserID,
			DisplayName: participant.DisplayName,
		})
	}
	return appdto.TripAvailabilityList{
		TripID:    tripID,
		Responses: items,
		Summary: appdto.TripAvailabilitySummary{
			TotalCollaborators: len(participants),
			SubmittedCount:     submitted,
			MissingCount:       len(missing),
			MissingUsers:       missing,
		},
	}
}

func availabilityResponseInfo(displayName string, response entity.TripAvailabilityResponse, submitted bool) appdto.TripAvailabilityResponseInfo {
	updatedAt := response.UpdatedAt
	return appdto.TripAvailabilityResponseInfo{
		UserID:            response.UserID,
		DisplayName:       displayName,
		AvailableRanges:   cloneAvailabilityRanges(response.AvailableRanges),
		UnavailableRanges: cloneAvailabilityRanges(response.UnavailableRanges),
		PreferredRanges:   cloneAvailabilityRanges(response.PreferredRanges),
		MinTripDays:       cloneIntPtr(response.MinTripDays),
		MaxTripDays:       cloneIntPtr(response.MaxTripDays),
		Timezone:          response.Timezone,
		Notes:             response.Notes,
		Submitted:         submitted,
		UpdatedAt:         &updatedAt,
	}
}

func calculateDateOptions(
	trip *entity.Trip,
	participants []availabilityParticipant,
	responses []entity.TripAvailabilityResponse,
	in appdto.DateOptionsInput,
) appdto.DateOptionsResult {
	if len(responses) == 0 || len(participants) == 0 {
		return appdto.DateOptionsResult{Options: []appdto.DateOption{}}
	}
	parsed := parseAvailabilityResponses(responses)
	searchStart, searchEnd := dateOptionSearchWindow(trip, parsed, in)
	if searchEnd.Before(searchStart) {
		return appdto.DateOptionsResult{Options: []appdto.DateOption{}}
	}
	minDays, maxDays := dateOptionDurations(trip, parsed, in)
	options := make([]appdto.DateOption, 0)
	for start := searchStart; !start.After(searchEnd); start = start.AddDate(0, 0, 1) {
		for days := minDays; days <= maxDays; days++ {
			end := start.AddDate(0, 0, days-1)
			if end.After(searchEnd) {
				continue
			}
			option := scoreDateWindow(dateWindow{start: start, end: end, days: days}, participants, parsed, trip, in)
			options = append(options, option)
		}
	}
	sort.SliceStable(options, func(i, j int) bool {
		a, b := options[i], options[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.ConflictUserCount != b.ConflictUserCount {
			return a.ConflictUserCount < b.ConflictUserCount
		}
		if a.MissingResponseUserCount != b.MissingResponseUserCount {
			return a.MissingResponseUserCount < b.MissingResponseUserCount
		}
		if a.PreferredUserCount != b.PreferredUserCount {
			return a.PreferredUserCount > b.PreferredUserCount
		}
		if a.DurationDays != b.DurationDays {
			return a.DurationDays < b.DurationDays
		}
		return a.StartDate < b.StartDate
	})
	if len(options) > in.Limit {
		options = options[:in.Limit]
	}
	return appdto.DateOptionsResult{Options: options}
}

func scoreDateWindow(
	window dateWindow,
	participants []availabilityParticipant,
	responses map[uuid.UUID]parsedAvailabilityResponse,
	trip *entity.Trip,
	in appdto.DateOptionsInput,
) appdto.DateOption {
	total := len(participants)
	availableUsers := []appdto.DateOptionUserSummary{}
	conflicts := []appdto.DateOptionConflict{}
	missing := []appdto.DateOptionUserSummary{}
	preferredCount := 0
	outsideDurationCount := 0
	unknownCount := 0

	for _, participant := range participants {
		summary := appdto.DateOptionUserSummary{UserID: participant.UserID, DisplayName: participant.DisplayName}
		response, ok := responses[participant.UserID]
		if !ok || !response.submitted {
			missing = append(missing, summary)
			continue
		}
		if outsideUserDuration(window.days, response.response) {
			outsideDurationCount++
		}
		status, reason := availabilityStatusForWindow(window, response)
		switch status {
		case "unavailable", "partial":
			conflicts = append(conflicts, appdto.DateOptionConflict{
				UserID:      participant.UserID,
				DisplayName: participant.DisplayName,
				Reason:      reason,
			})
		case "preferred":
			preferredCount++
			availableUsers = append(availableUsers, summary)
		case "available":
			availableUsers = append(availableUsers, summary)
		default:
			unknownCount++
		}
	}

	score := 0.0
	if total > 0 {
		score += float64(len(availableUsers)) / float64(total) * 60
		score += float64(preferredCount) / float64(total) * 20
	}
	weekend := includesWeekend(window.start, window.end)
	if weekend && preferWeekends(in) {
		score += 10
	}
	if durationFitsTrip(window.days, trip) && outsideDurationCount == 0 {
		score += 10
	} else if outsideDurationCount == 0 {
		score += 5
	}
	score -= float64(len(conflicts)) * 15
	score -= float64(len(missing)) * 5
	score -= float64(outsideDurationCount) * 5
	if !weekend && preferredCount > 0 && preferWeekends(in) {
		score -= 5
	}
	score = math.Max(0, math.Min(100, score))

	option := appdto.DateOption{
		ID:                       window.start.Format("2006-01-02") + "_" + window.end.Format("2006-01-02"),
		StartDate:                window.start.Format("2006-01-02"),
		EndDate:                  window.end.Format("2006-01-02"),
		DurationDays:             window.days,
		Score:                    int(math.Round(score)),
		AvailableUserCount:       len(availableUsers),
		TotalUserCount:           total,
		PreferredUserCount:       preferredCount,
		ConflictUserCount:        len(conflicts),
		MissingResponseUserCount: len(missing),
		AvailableUsers:           availableUsers,
		Conflicts:                conflicts,
		MissingResponses:         missing,
	}
	option.Pros, option.Cons, option.Warnings = dateOptionExplanations(option, weekend, outsideDurationCount, unknownCount, durationFitsTrip(window.days, trip))
	return option
}

func dateOptionExplanations(option appdto.DateOption, weekend bool, outsideDurationCount, unknownCount int, matchesTripDuration bool) ([]string, []string, []string) {
	pros := []string{}
	cons := []string{}
	warnings := []string{}
	if option.AvailableUserCount == option.TotalUserCount && option.TotalUserCount > 0 {
		pros = append(pros, "Best overlap for all collaborators.")
	} else if option.AvailableUserCount > 0 {
		pros = append(pros, "Best overlap for most collaborators.")
	}
	if weekend {
		pros = append(pros, "Includes a weekend.")
	}
	if matchesTripDuration {
		pros = append(pros, "Matches preferred trip length.")
	}
	if option.ConflictUserCount == 0 {
		pros = append(pros, "No reported conflicts.")
	}
	if option.ConflictUserCount > 0 {
		cons = append(cons, pluralSentence(option.ConflictUserCount, "One collaborator has a conflict.", "%d collaborators have conflicts."))
	}
	if option.MissingResponseUserCount > 0 {
		cons = append(cons, pluralSentence(option.MissingResponseUserCount, "One collaborator has not submitted availability.", "%d collaborators have not submitted availability."))
		warnings = append(warnings, pluralSentence(option.MissingResponseUserCount, "One collaborator has not submitted availability.", "%d collaborators have not submitted availability."))
	}
	if outsideDurationCount > 0 {
		cons = append(cons, pluralSentence(outsideDurationCount, "Longer or shorter than one collaborator's trip length preference.", "Outside %d collaborators' trip length preferences."))
	}
	if unknownCount > 0 {
		warnings = append(warnings, pluralSentence(unknownCount, "One collaborator's availability is unknown for this window.", "%d collaborators have unknown availability for this window."))
	}
	return pros, cons, warnings
}

func availabilityStatusForWindow(window dateWindow, response parsedAvailabilityResponse) (string, string) {
	if len(response.unavailable) > 0 {
		for _, r := range response.unavailable {
			if rangesOverlap(window.start, window.end, r.start, r.end) {
				return "unavailable", "Unavailable on " + maxTime(window.start, r.start).Format("2006-01-02")
			}
		}
	}
	for _, r := range response.preferred {
		if rangesOverlap(window.start, window.end, r.start, r.end) {
			return "preferred", ""
		}
	}
	if len(response.available) == 0 {
		return "unknown", "Availability unknown"
	}
	for _, r := range response.available {
		if fullyCovers(r.start, r.end, window.start, window.end) {
			return "available", ""
		}
	}
	for _, r := range response.available {
		if rangesOverlap(window.start, window.end, r.start, r.end) {
			return "partial", "Only partially available"
		}
	}
	return "unknown", "Availability unknown"
}

func parseAvailabilityResponses(responses []entity.TripAvailabilityResponse) map[uuid.UUID]parsedAvailabilityResponse {
	out := make(map[uuid.UUID]parsedAvailabilityResponse, len(responses))
	for _, response := range responses {
		parsed := parsedAvailabilityResponse{
			response:  response,
			submitted: true,
		}
		parsed.available = parseRangesBestEffort(response.AvailableRanges)
		parsed.unavailable = parseRangesBestEffort(response.UnavailableRanges)
		parsed.preferred = parseRangesBestEffort(response.PreferredRanges)
		out[response.UserID] = parsed
	}
	return out
}

func parseRangesBestEffort(ranges []entity.AvailabilityDateRange) []availabilityRange {
	out := make([]availabilityRange, 0, len(ranges))
	for _, r := range ranges {
		start, err1 := parseAvailabilityDate(r.StartDate)
		end, err2 := parseAvailabilityDate(r.EndDate)
		if err1 != nil || err2 != nil || end.Before(start) {
			continue
		}
		out = append(out, availabilityRange{start: start, end: end, raw: r})
	}
	return out
}

func dateOptionSearchWindow(
	trip *entity.Trip,
	responses map[uuid.UUID]parsedAvailabilityResponse,
	in appdto.DateOptionsInput,
) (time.Time, time.Time) {
	if in.SearchStartDate != "" || in.SearchEndDate != "" {
		start, end := defaultDateOptionWindow(trip)
		if in.SearchStartDate != "" {
			start, _ = parseAvailabilityDate(in.SearchStartDate)
		}
		if in.SearchEndDate != "" {
			end, _ = parseAvailabilityDate(in.SearchEndDate)
		}
		return capSearchWindow(start, end)
	}
	var earliest, latest time.Time
	for _, response := range responses {
		for _, ranges := range [][]availabilityRange{response.available, response.unavailable, response.preferred} {
			for _, r := range ranges {
				if earliest.IsZero() || r.start.Before(earliest) {
					earliest = r.start
				}
				if latest.IsZero() || r.end.After(latest) {
					latest = r.end
				}
			}
		}
	}
	if !earliest.IsZero() && !latest.IsZero() {
		return capSearchWindow(earliest, latest)
	}
	return capSearchWindow(defaultDateOptionWindow(trip))
}

func defaultDateOptionWindow(trip *entity.Trip) (time.Time, time.Time) {
	if trip != nil && trip.StartDate != nil {
		start := truncateToDate(*trip.StartDate).AddDate(0, 0, -30)
		return start, truncateToDate(*trip.StartDate).AddDate(0, 0, 90)
	}
	today := truncateToDate(time.Now().UTC())
	return today, today.AddDate(0, 0, 180)
}

func capSearchWindow(start, end time.Time) (time.Time, time.Time) {
	start = truncateToDate(start)
	end = truncateToDate(end)
	capEnd := start.AddDate(0, 0, dateOptionSearchCapDays)
	if end.After(capEnd) {
		end = capEnd
	}
	return start, end
}

func dateOptionDurations(
	trip *entity.Trip,
	responses map[uuid.UUID]parsedAvailabilityResponse,
	in appdto.DateOptionsInput,
) (int, int) {
	if in.MinDays != nil || in.MaxDays != nil {
		minDays, maxDays := 1, 7
		if in.MinDays != nil {
			minDays = *in.MinDays
		}
		if in.MaxDays != nil {
			maxDays = *in.MaxDays
		}
		if maxDays < minDays {
			maxDays = minDays
		}
		return minDays, maxDays
	}
	minDays, maxDays := 0, 0
	for _, response := range responses {
		if response.response.MinTripDays != nil {
			if minDays == 0 || *response.response.MinTripDays > minDays {
				minDays = *response.response.MinTripDays
			}
		}
		if response.response.MaxTripDays != nil {
			if maxDays == 0 || *response.response.MaxTripDays < maxDays {
				maxDays = *response.response.MaxTripDays
			}
		}
	}
	if minDays > 0 && maxDays > 0 && minDays <= maxDays {
		return minDays, maxDays
	}
	if trip != nil && trip.Days > 0 {
		return int(trip.Days), int(trip.Days)
	}
	return 2, 7
}

func selectedDateOptionMetadata(option appdto.DateOption) map[string]any {
	return map[string]any{
		"id":                       option.ID,
		"startDate":                option.StartDate,
		"endDate":                  option.EndDate,
		"durationDays":             option.DurationDays,
		"score":                    option.Score,
		"availableUserCount":       option.AvailableUserCount,
		"totalUserCount":           option.TotalUserCount,
		"preferredUserCount":       option.PreferredUserCount,
		"conflictUserCount":        option.ConflictUserCount,
		"missingResponseUserCount": option.MissingResponseUserCount,
		"warnings":                 option.Warnings,
	}
}

func selectedDateOptionForPlanning(metadata map[string]any) *planningconstraints.SelectedDateOption {
	if metadata == nil {
		return nil
	}
	raw, ok := metadata["selectedDateOption"].(map[string]any)
	if !ok {
		return nil
	}
	startDate, _ := raw["startDate"].(string)
	endDate, _ := raw["endDate"].(string)
	durationDays := numericMetadataInt(raw["durationDays"])
	score := numericMetadataInt(raw["score"])
	conflicts := numericMetadataInt(raw["conflictUserCount"])
	if strings.TrimSpace(startDate) == "" || strings.TrimSpace(endDate) == "" || durationDays <= 0 {
		return nil
	}
	return &planningconstraints.SelectedDateOption{
		StartDate:         startDate,
		EndDate:           endDate,
		DurationDays:      durationDays,
		Score:             score,
		ConflictUserCount: conflicts,
	}
}

func groupAvailabilityNotes(responses []entity.TripAvailabilityResponse) string {
	preferred := 0
	weekendPreferred := 0
	for _, response := range responses {
		if len(response.PreferredRanges) > 0 {
			preferred++
		}
		for _, r := range parseRangesBestEffort(response.PreferredRanges) {
			if includesWeekend(r.start, r.end) {
				weekendPreferred++
				break
			}
		}
	}
	parts := []string{}
	if preferred > 0 {
		parts = append(parts, pluralSentence(preferred, "One collaborator submitted preferred dates.", "%d collaborators submitted preferred dates."))
	}
	if weekendPreferred > 0 {
		parts = append(parts, pluralSentence(weekendPreferred, "One collaborator prefers a weekend overlap.", "%d collaborators prefer weekend overlap."))
	}
	return strings.Join(parts, " ")
}

func shiftedRouteDates(route *aggregate.TripRoute, oldStart, newStart time.Time) (*aggregate.TripRoute, bool) {
	if route == nil {
		return nil, false
	}
	raw, err := json.Marshal(route)
	if err != nil {
		return nil, false
	}
	var cloned aggregate.TripRoute
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return nil, false
	}
	delta := int(newStart.Sub(truncateToDate(oldStart)).Hours() / 24)
	shift := func(value string) (string, bool) {
		if strings.TrimSpace(value) == "" {
			return "", true
		}
		parsed, err := parseAvailabilityDate(value)
		if err != nil {
			return value, false
		}
		return parsed.AddDate(0, 0, delta).Format("2006-01-02"), true
	}
	for i := range cloned.Stops {
		var ok bool
		if cloned.Stops[i].ArrivalDate, ok = shift(cloned.Stops[i].ArrivalDate); !ok {
			return nil, false
		}
		if cloned.Stops[i].DepartureDate, ok = shift(cloned.Stops[i].DepartureDate); !ok {
			return nil, false
		}
	}
	for i := range cloned.Legs {
		var ok bool
		if cloned.Legs[i].DepartureDate, ok = shift(cloned.Legs[i].DepartureDate); !ok {
			return nil, false
		}
	}
	return &cloned, true
}

func routeHasDates(route *aggregate.TripRoute) bool {
	if route == nil {
		return false
	}
	for _, stop := range route.Stops {
		if strings.TrimSpace(stop.ArrivalDate) != "" || strings.TrimSpace(stop.DepartureDate) != "" {
			return true
		}
	}
	for _, leg := range route.Legs {
		if strings.TrimSpace(leg.DepartureDate) != "" {
			return true
		}
	}
	return false
}

func availabilityRequestMessage(trip *entity.Trip, custom string) string {
	if custom != "" {
		return custom
	}
	return fmt.Sprintf("Please submit availability for %s.", tripDestination(trip))
}

func displayNameForUser(userID uuid.UUID, currentUser *auth.AuthenticatedUser, trip *entity.Trip, collaborator *entity.TripCollaborator) string {
	if currentUser != nil && userID == currentUser.ID {
		if currentUser.Email != "" {
			return emailDisplayName(currentUser.Email)
		}
		return "You"
	}
	if trip != nil && trip.UserID != nil && userID == *trip.UserID {
		return "Trip owner"
	}
	if collaborator != nil {
		return "Collaborator " + shortUUID(userID)
	}
	return fallbackDisplayName(userID)
}

func fallbackDisplayName(userID uuid.UUID) string {
	return "User " + shortUUID(userID)
}

func emailDisplayName(email string) string {
	local := strings.TrimSpace(strings.Split(email, "@")[0])
	if local == "" {
		return email
	}
	return local
}

func shortUUID(id uuid.UUID) string {
	value := id.String()
	if len(value) < 8 {
		return value
	}
	return value[:8]
}

func parseAvailabilityDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return truncateToDate(parsed), nil
}

func truncateToDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func rangesOverlap(startA, endA, startB, endB time.Time) bool {
	return !endA.Before(startB) && !endB.Before(startA)
}

func fullyCovers(outerStart, outerEnd, innerStart, innerEnd time.Time) bool {
	return (outerStart.Equal(innerStart) || outerStart.Before(innerStart)) &&
		(outerEnd.Equal(innerEnd) || outerEnd.After(innerEnd))
}

func includesWeekend(start, end time.Time) bool {
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			return true
		}
	}
	return false
}

func preferWeekends(in appdto.DateOptionsInput) bool {
	return in.PreferWeekends == nil || *in.PreferWeekends
}

func outsideUserDuration(days int, response entity.TripAvailabilityResponse) bool {
	if response.MinTripDays != nil && days < *response.MinTripDays {
		return true
	}
	if response.MaxTripDays != nil && days > *response.MaxTripDays {
		return true
	}
	return false
}

func durationFitsTrip(days int, trip *entity.Trip) bool {
	return trip != nil && trip.Days > 0 && int(trip.Days) == days
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func pluralSentence(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return fmt.Sprintf(many, count)
}

func cloneAvailabilityRanges(ranges []entity.AvailabilityDateRange) []entity.AvailabilityDateRange {
	if ranges == nil {
		return []entity.AvailabilityDateRange{}
	}
	return append([]entity.AvailabilityDateRange(nil), ranges...)
}

func nullableIntMetadataValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func cloneMetadata(metadata map[string]any) map[string]any {
	out := make(map[string]any, len(metadata)+4)
	for k, v := range metadata {
		out[k] = v
	}
	return out
}

func truncateForMetadata(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func numericMetadataInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}
