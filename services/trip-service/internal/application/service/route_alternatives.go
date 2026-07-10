package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
)

const (
	defaultRouteAlternativeCount = 3
	maxRouteAlternativeCount     = 5
)

func (s *Service) SuggestRouteAlternatives(
	ctx context.Context,
	in routealternatives.SuggestInput,
) (routealternatives.SessionView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	normalized, err := normalizeSuggestRouteAlternativesInput(in)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	if normalized.WorkspaceID != nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return routealternatives.SessionView{}, err
		}
	}
	userCtx, err := s.loadUserContext(ctx, user, uuid.Nil)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	constraints, err := s.buildPreTripRouteAlternativeConstraints(ctx, user, normalized, userCtx, planningconstraints.SourceRouteAlternatives)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	if err := s.requireNoPlanningBlockers(constraints, planningconstraints.SourceRouteAlternatives); err != nil {
		return routealternatives.SessionView{}, err
	}
	request := aiRouteAlternativeRequest(normalized, constraints, nil, routealternatives.Refinement{})
	response, err := s.generator.SuggestRouteAlternatives(ctx, request)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	routealternatives.NormalizeAndScore(response, normalized.Budget, constraints)
	return s.persistRouteAlternativeSession(ctx, routeAlternativeSessionInput{
		UserID:         user.ID,
		WorkspaceID:    normalized.WorkspaceID,
		Source:         routealternatives.SourcePreTrip,
		Prompt:         normalized.Prompt,
		OutputLanguage: request.OutputLanguage,
		Request:        request,
		Response:       *response,
	})
}

func (s *Service) SuggestTripRouteAlternatives(
	ctx context.Context,
	tripID uuid.UUID,
	in routealternatives.ExistingTripSuggestInput,
) (routealternatives.SessionView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	trip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	count := normalizeSuggestionCount(in.SuggestionCount)
	outputLanguage := strings.TrimSpace(in.OutputLanguage)
	if outputLanguage == "" {
		outputLanguage = "en"
	}
	if err := validateOutputLanguage(outputLanguage); err != nil {
		return routealternatives.SessionView{}, err
	}
	userCtx, err := s.loadUserContext(ctx, user, tripID)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	constraints, err := s.buildPlanningConstraints(
		ctx,
		user,
		planningconstraints.SourceRouteAlternatives,
		trip,
		planningconstraints.RequestOverride{
			OutputLanguage: outputLanguage,
			Prompt: &planningconstraints.Prompt{
				UserPrompt: strings.TrimSpace(in.Prompt),
			},
		},
		userCtx,
		true,
	)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	currentRoute := trip.Route
	if !in.UseCurrentRouteAsBaseline {
		currentRoute = nil
	}
	request := routealternatives.AIRequest{
		Origin:              routeOrigin(currentRoute),
		Prompt:              strings.TrimSpace(in.Prompt),
		DurationDays:        int(trip.Days),
		StartDate:           dateString(trip.StartDate),
		Budget:              tripBudgetEstimate(trip),
		Travelers:           trip.Travelers,
		OutputLanguage:      resolveOutputLanguage(outputLanguage, userCtx.Profile),
		PlanningConstraints: constraints,
		CurrentRoute:        currentRoute,
		Refinement:          routealternatives.Refinement{},
		SuggestionCount:     count,
	}
	response, err := s.generator.SuggestRouteAlternatives(ctx, request)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	routealternatives.NormalizeAndScore(response, request.Budget, constraints)
	view, err := s.persistRouteAlternativeSession(ctx, routeAlternativeSessionInput{
		UserID:         user.ID,
		TripID:         &tripID,
		WorkspaceID:    trip.WorkspaceID,
		Source:         routealternatives.SourceExistingTrip,
		Prompt:         request.Prompt,
		OutputLanguage: request.OutputLanguage,
		Request:        request,
		Response:       *response,
	})
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventRouteAlternativesGenerated,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"sessionId":        view.ID.String(),
			"alternativeCount": len(view.Alternatives),
		},
	})
	return view, nil
}

func (s *Service) GetRouteAlternativeSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (routealternatives.SessionView, error) {
	session, err := s.authorizedRouteAlternativeSession(ctx, sessionID, false)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	return routealternatives.NewSessionView(session)
}

func (s *Service) ListRouteAlternativeSessions(
	ctx context.Context,
	tripID *uuid.UUID,
	limit int,
) (routealternatives.ListSessionsResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return routealternatives.ListSessionsResult{}, err
	}
	limit = normalizeSessionLimit(limit)
	var sessions []routealternatives.Session
	if tripID != nil {
		if _, _, err := s.requireViewerEditorOrOwner(ctx, *tripID, user.ID); err != nil {
			return routealternatives.ListSessionsResult{}, err
		}
		sessions, err = s.repo.ListRouteAlternativeSessionsByTrip(ctx, *tripID, limit)
	} else {
		sessions, err = s.repo.ListRouteAlternativeSessionsByUser(ctx, user.ID, limit)
	}
	if err != nil {
		return routealternatives.ListSessionsResult{}, err
	}
	items := make([]routealternatives.SessionView, 0, len(sessions))
	for i := range sessions {
		view, err := routealternatives.NewSessionView(&sessions[i])
		if err != nil {
			return routealternatives.ListSessionsResult{}, err
		}
		items = append(items, view)
	}
	return routealternatives.ListSessionsResult{Items: items, Limit: limit}, nil
}

func (s *Service) RefineRouteAlternativeSession(
	ctx context.Context,
	sessionID uuid.UUID,
	in routealternatives.RefineInput,
) (routealternatives.SessionView, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	session, err := s.authorizedRouteAlternativeSession(ctx, sessionID, true)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	instruction := strings.TrimSpace(in.Instruction)
	if instruction == "" {
		return routealternatives.SessionView{}, apperrs.NewInvalidInput("instruction is required")
	}
	if len(instruction) > maxInstructionLength {
		return routealternatives.SessionView{}, apperrs.NewInvalidInput("instruction must be at most %d characters", maxInstructionLength)
	}
	previous, err := routealternatives.DecodeResponse(session.ResponseJSON)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	var trip *entity.Trip
	var constraints *planningconstraints.PlanningConstraints
	userCtx, err := s.loadUserContext(ctx, user, uuid.Nil)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	if session.TripID != nil {
		trip, _, err = s.requireEditorOrOwner(ctx, *session.TripID, user.ID)
		if err != nil {
			return routealternatives.SessionView{}, err
		}
		constraints, err = s.buildPlanningConstraints(
			ctx,
			user,
			planningconstraints.SourceRouteAlternativeRefinement,
			trip,
			planningconstraints.RequestOverride{
				OutputLanguage: session.OutputLanguage,
				Prompt: &planningconstraints.Prompt{
					RefinementInstruction: instruction,
				},
			},
			userCtx,
			true,
		)
	} else {
		var original routealternatives.AIRequest
		_ = json.Unmarshal(session.RequestJSON, &original)
		constraints, err = s.buildPreTripRouteAlternativeConstraints(
			ctx,
			user,
			suggestInputFromAIRequest(original, session.WorkspaceID),
			userCtx,
			planningconstraints.SourceRouteAlternativeRefinement,
		)
	}
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	request := routealternatives.AIRequest{
		Prompt:              session.Prompt,
		OutputLanguage:      session.OutputLanguage,
		PlanningConstraints: constraints,
		CurrentRoute:        nil,
		Refinement: routealternatives.Refinement{
			PreviousAlternatives:  previous.Alternatives,
			Instruction:           instruction,
			SelectedAlternativeID: strings.TrimSpace(in.SelectedAlternativeID),
		},
		SuggestionCount: len(previous.Alternatives),
	}
	if trip != nil {
		request.Origin = routeOrigin(trip.Route)
		request.DurationDays = int(trip.Days)
		request.StartDate = dateString(trip.StartDate)
		request.Budget = tripBudgetEstimate(trip)
		request.Travelers = trip.Travelers
		request.CurrentRoute = trip.Route
	} else {
		var original routealternatives.AIRequest
		_ = json.Unmarshal(session.RequestJSON, &original)
		request.Origin = original.Origin
		request.DurationDays = original.DurationDays
		request.StartDate = original.StartDate
		request.Budget = original.Budget
		request.Travelers = original.Travelers
	}
	if request.SuggestionCount < 1 {
		request.SuggestionCount = defaultRouteAlternativeCount
	}
	response, err := s.generator.SuggestRouteAlternatives(ctx, request)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	routealternatives.NormalizeAndScore(response, request.Budget, constraints)
	view, err := s.persistRouteAlternativeSession(ctx, routeAlternativeSessionInput{
		UserID:          user.ID,
		TripID:          session.TripID,
		WorkspaceID:     session.WorkspaceID,
		Source:          routealternatives.SourceRouteRefinement,
		Prompt:          session.Prompt,
		OutputLanguage:  request.OutputLanguage,
		ParentSessionID: &session.ID,
		Request:         request,
		Response:        *response,
	})
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	if session.TripID != nil {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      *session.TripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventRouteAlternativeRefined,
			EntityType:  activityEntityType(activity.EntityTrip),
			EntityID:    activityEntityID(*session.TripID),
			Metadata: map[string]any{
				"sessionId":       view.ID.String(),
				"parentSessionId": session.ID.String(),
			},
		})
	}
	return view, nil
}

func (s *Service) CreateTripFromRouteAlternative(
	ctx context.Context,
	sessionID uuid.UUID,
	alternativeID string,
	in routealternatives.CreateTripInput,
) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.authorizedRouteAlternativeSession(ctx, sessionID, true)
	if err != nil {
		return nil, err
	}
	response, err := routealternatives.DecodeResponse(session.ResponseJSON)
	if err != nil {
		return nil, err
	}
	alternative, ok := routealternatives.FindAlternative(response, alternativeID)
	if !ok {
		return nil, apperrs.NewInvalidInput("alternativeId does not belong to the session")
	}
	var original routealternatives.AIRequest
	_ = json.Unmarshal(session.RequestJSON, &original)
	workspaceID := in.WorkspaceID
	if workspaceID == nil {
		workspaceID = session.WorkspaceID
	}
	if workspaceID != nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *workspaceID); err != nil {
			return nil, err
		}
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = alternative.Title
	}
	startDate := strings.TrimSpace(in.StartDate)
	if startDate == "" {
		startDate = original.StartDate
	}
	budget := in.Budget
	if budget == nil {
		budget = alternative.EstimatedBudget
	}
	travelers := original.Travelers
	if travelers < 1 {
		travelers = 1
	}
	if in.Travelers != nil {
		travelers = *in.Travelers
	}
	days := int32(original.DurationDays)
	if days < 1 {
		days = int32(routeDurationDays(alternative.Route))
	}
	currency, amount := budgetParts(budget)
	trip, err := s.Create(ctx, appdto.CreateTripInput{
		Destination:    title,
		WorkspaceID:    workspaceID,
		TripType:       entity.TripTypeMultiDestination,
		StartDate:      startDate,
		Days:           days,
		BudgetAmount:   amount,
		BudgetCurrency: currency,
		Travelers:      travelers,
		Interests:      append([]string(nil), alternative.BestFor...),
		Pace:           "balanced",
		Route:          &alternative.Route,
	})
	if err != nil {
		return nil, err
	}
	trip, err = s.repo.UpdateTripCreationMetadata(ctx, trip.ID, user.ID, map[string]any{
		"creationSource":              "route_alternative",
		"routeAlternativeSessionId":   session.ID.String(),
		"selectedAlternativeId":       alternative.ID,
		"selectedAlternativeTitle":    alternative.Title,
		"alternativeScores":           alternative.Scores,
		"suggestedPromptForItinerary": alternative.SuggestedItineraryPrompt,
		"outputLanguage":              session.OutputLanguage,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.MarkRouteAlternativeSessionCreatedTrip(ctx, session.ID, alternative.ID, trip.ID); err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      trip.ID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripCreatedFromRouteAlternative,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(trip.ID),
		Metadata: map[string]any{
			"sessionId":        session.ID.String(),
			"alternativeId":    alternative.ID,
			"alternativeTitle": alternative.Title,
		},
	})
	return trip, nil
}

func (s *Service) ApplyRouteAlternative(
	ctx context.Context,
	tripID, sessionID uuid.UUID,
	alternativeID string,
	in routealternatives.ApplyInput,
) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.authorizedRouteAlternativeSession(ctx, sessionID, true)
	if err != nil {
		return nil, err
	}
	if session.TripID != nil && *session.TripID != tripID {
		return nil, apperrs.NewInvalidInput("session does not belong to this trip")
	}
	response, err := routealternatives.DecodeResponse(session.ResponseJSON)
	if err != nil {
		return nil, err
	}
	alternative, ok := routealternatives.FindAlternative(response, alternativeID)
	if !ok {
		return nil, apperrs.NewInvalidInput("alternativeId does not belong to the session")
	}
	updated, err := s.UpdateTripRoute(ctx, tripID, appdto.UpdateTripRouteInput{
		Route:                     &alternative.Route,
		ExpectedItineraryRevision: in.ExpectedItineraryRevision,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.MarkRouteAlternativeSessionApplied(ctx, session.ID, alternative.ID, tripID); err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventRouteAlternativeApplied,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"sessionId":           session.ID.String(),
			"alternativeId":       alternative.ID,
			"alternativeTitle":    alternative.Title,
			"regenerateItinerary": in.RegenerateItinerary,
		},
	})
	return updated, nil
}

func (s *Service) CreateRouteAlternativesPoll(
	ctx context.Context,
	tripID, sessionID uuid.UUID,
	in routealternatives.CreatePollInput,
) (appdto.TripPollInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripPollInfo{}, err
	}
	session, err := s.authorizedRouteAlternativeSession(ctx, sessionID, true)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	if session.TripID != nil && *session.TripID != tripID {
		return appdto.TripPollInfo{}, apperrs.NewInvalidInput("session does not belong to this trip")
	}
	response, err := routealternatives.DecodeResponse(session.ResponseJSON)
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	selected := normalizeAlternativeIDs(in.AlternativeIDs, response.Alternatives)
	if len(selected) == 0 {
		return appdto.TripPollInfo{}, apperrs.NewInvalidInput("alternativeIds are required")
	}
	options := make([]appdto.CreateTripPollOptionInput, 0, len(selected))
	for _, id := range selected {
		alternative, ok := routealternatives.FindAlternative(response, id)
		if !ok {
			return appdto.TripPollInfo{}, apperrs.NewInvalidInput("alternativeIds must belong to the session")
		}
		options = append(options, appdto.CreateTripPollOptionInput{
			OptionKey:   alternative.ID,
			Label:       alternative.Title,
			Description: truncateString(alternative.Summary, maxPollOptionDescLength),
			Metadata: map[string]any{
				"category":      "route_alternative",
				"sessionId":     session.ID.String(),
				"alternativeId": alternative.ID,
				"routeTitle":    alternative.Title,
				"destination":   alternative.Title,
			},
		})
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Which route should we choose?"
	}
	info, err := s.CreateTripPoll(ctx, tripID, appdto.CreateTripPollInput{
		Title:    title,
		PollType: entity.PollTypeSingleChoice,
		Metadata: map[string]any{
			"category":  "route_alternative",
			"sessionId": session.ID.String(),
		},
		Options: options,
	})
	if err != nil {
		return appdto.TripPollInfo{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventRouteAlternativesPollCreated,
		EntityType:  activityEntityType(activity.EntityTripPoll),
		EntityID:    activityEntityID(info.Poll.ID),
		Metadata: map[string]any{
			"sessionId":        session.ID.String(),
			"alternativeCount": len(options),
			"pollId":           info.Poll.ID.String(),
		},
	})
	return info, nil
}

type routeAlternativeSessionInput struct {
	UserID          uuid.UUID
	TripID          *uuid.UUID
	WorkspaceID     *uuid.UUID
	Source          string
	Prompt          string
	OutputLanguage  string
	ParentSessionID *uuid.UUID
	Request         routealternatives.AIRequest
	Response        routealternatives.Response
}

func (s *Service) persistRouteAlternativeSession(
	ctx context.Context,
	in routeAlternativeSessionInput,
) (routealternatives.SessionView, error) {
	requestJSON, err := json.Marshal(in.Request)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	responseJSON, err := json.Marshal(in.Response)
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	session, err := s.repo.CreateRouteAlternativeSession(ctx, &routealternatives.Session{
		ID:              uuid.New(),
		UserID:          in.UserID,
		TripID:          in.TripID,
		WorkspaceID:     in.WorkspaceID,
		Source:          in.Source,
		Prompt:          truncateString(in.Prompt, maxInstructionLength),
		OutputLanguage:  in.OutputLanguage,
		Status:          routealternatives.StatusCompleted,
		RequestJSON:     requestJSON,
		ResponseJSON:    responseJSON,
		ParentSessionID: in.ParentSessionID,
	})
	if err != nil {
		return routealternatives.SessionView{}, err
	}
	return routealternatives.NewSessionView(session)
}

func (s *Service) authorizedRouteAlternativeSession(
	ctx context.Context,
	sessionID uuid.UUID,
	requireEdit bool,
) (*routealternatives.Session, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.repo.GetRouteAlternativeSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.TripID == nil {
		if session.UserID != user.ID {
			return nil, apperrs.ErrForbidden
		}
		return session, nil
	}
	if requireEdit {
		_, _, err = s.requireEditorOrOwner(ctx, *session.TripID, user.ID)
	} else {
		_, _, err = s.requireViewerEditorOrOwner(ctx, *session.TripID, user.ID)
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) buildPreTripRouteAlternativeConstraints(
	ctx context.Context,
	user auth.AuthenticatedUser,
	in routealternatives.SuggestInput,
	userCtx usercontext.UserContext,
	source planningconstraints.Source,
) (*planningconstraints.PlanningConstraints, error) {
	policy, err := s.activeWorkspacePolicy(ctx, in.WorkspaceID, true)
	if err != nil {
		return nil, err
	}
	previousTrips, err := s.previousTripsForPlanningConstraints(ctx, user.ID, true)
	if err != nil {
		return nil, err
	}
	duration := in.DurationDays
	travelers := in.Travelers
	if travelers < 1 {
		travelers = 1
	}
	request := planningconstraints.RequestOverride{
		OutputLanguage: in.OutputLanguage,
		StartDate:      in.StartDate,
		DurationDays:   &duration,
		Budget:         budgetOverride(in.Budget),
		Travelers:      &planningconstraints.TravelerOverride{Count: &travelers},
		Transport: &planningconstraints.TransportOverride{
			PreferredModes:         append([]string(nil), in.Transport.PreferredModes...),
			AvoidModes:             append([]string(nil), in.Transport.AvoidModes...),
			CarAvailable:           &in.Transport.CarAvailable,
			MaxTransferHoursPerDay: in.Transport.MaxTransferHoursPerDay,
		},
		TripStyles: append([]string(nil), in.TripStyles...),
		Prompt: &planningconstraints.Prompt{
			UserPrompt: in.Prompt,
		},
	}
	constraints := planningconstraints.Build(planningconstraints.BuildInput{
		UserID:                     user.ID,
		WorkspaceID:                in.WorkspaceID,
		Source:                     source,
		Request:                    request,
		UserContext:                userCtx,
		WorkspacePolicy:            policy,
		PreviousTrips:              previousTrips,
		IncludePreviousTripSignals: true,
		IncludeRoute:               true,
	})
	s.logPlanningConstraintsSummary(nil, constraints)
	return &constraints, nil
}

func aiRouteAlternativeRequest(
	in routealternatives.SuggestInput,
	constraints *planningconstraints.PlanningConstraints,
	currentRoute *aggregate.TripRoute,
	refinement routealternatives.Refinement,
) routealternatives.AIRequest {
	travelers := in.Travelers
	if travelers < 1 {
		travelers = 1
	}
	return routealternatives.AIRequest{
		Origin:              in.Origin,
		Prompt:              in.Prompt,
		DurationDays:        in.DurationDays,
		StartDate:           in.StartDate,
		Budget:              in.Budget,
		Travelers:           travelers,
		OutputLanguage:      normalizeOutputLanguageValue(in.OutputLanguage),
		PlanningConstraints: constraints,
		CurrentRoute:        currentRoute,
		Refinement:          refinement,
		SuggestionCount:     normalizeSuggestionCount(in.SuggestionCount),
	}
}

func normalizeSuggestRouteAlternativesInput(in routealternatives.SuggestInput) (routealternatives.SuggestInput, error) {
	in.Prompt = strings.TrimSpace(in.Prompt)
	in.OutputLanguage = normalizeOutputLanguageValue(in.OutputLanguage)
	if err := validateOutputLanguage(in.OutputLanguage); err != nil {
		return in, err
	}
	if in.DurationDays < 1 || in.DurationDays > maxDays {
		return in, apperrs.NewInvalidInput("durationDays must be between 1 and %d", maxDays)
	}
	if in.Travelers < 1 {
		in.Travelers = 1
	}
	in.SuggestionCount = normalizeSuggestionCount(in.SuggestionCount)
	if in.Budget != nil {
		in.Budget.Currency = normalizeCurrencyOrDefault(in.Budget.Currency)
		if in.Budget.Amount != nil && *in.Budget.Amount < 0 {
			return in, apperrs.NewInvalidInput("budget.amount must be >= 0")
		}
	}
	return in, nil
}

func normalizeSuggestionCount(value int) int {
	if value <= 0 {
		return defaultRouteAlternativeCount
	}
	if value > maxRouteAlternativeCount {
		return maxRouteAlternativeCount
	}
	return value
}

func normalizeSessionLimit(value int) int {
	if value <= 0 {
		return 20
	}
	if value > 100 {
		return 100
	}
	return value
}

func normalizeOutputLanguageValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "en", "es", "uk", "fr":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "en"
	}
}

func normalizeCurrencyOrDefault(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return defaultCurrency
	}
	return value
}

func budgetOverride(budget *routealternatives.BudgetEstimate) *planningconstraints.BudgetOverride {
	if budget == nil {
		return nil
	}
	return &planningconstraints.BudgetOverride{
		Amount:     budget.Amount,
		Currency:   budget.Currency,
		Strictness: "target",
	}
}

func tripBudgetEstimate(trip *entity.Trip) *routealternatives.BudgetEstimate {
	if trip == nil {
		return nil
	}
	return &routealternatives.BudgetEstimate{
		Amount:     trip.BudgetAmount,
		Currency:   normalizeCurrencyOrDefault(trip.BudgetCurrency),
		Confidence: "medium",
	}
}

func routeOrigin(route *aggregate.TripRoute) *aggregate.RoutePlace {
	if route == nil {
		return nil
	}
	return route.Origin
}

func suggestInputFromAIRequest(
	request routealternatives.AIRequest,
	workspaceID *uuid.UUID,
) routealternatives.SuggestInput {
	return routealternatives.SuggestInput{
		Origin:          request.Origin,
		Prompt:          request.Prompt,
		DurationDays:    request.DurationDays,
		StartDate:       request.StartDate,
		Budget:          request.Budget,
		Travelers:       request.Travelers,
		WorkspaceID:     workspaceID,
		OutputLanguage:  request.OutputLanguage,
		SuggestionCount: request.SuggestionCount,
	}
}

func budgetParts(budget *routealternatives.BudgetEstimate) (string, *float64) {
	if budget == nil {
		return defaultCurrency, nil
	}
	return normalizeCurrencyOrDefault(budget.Currency), budget.Amount
}

func routeDurationDays(route aggregate.TripRoute) int {
	total := 0
	for _, stop := range route.Stops {
		if stop.Nights != nil && *stop.Nights > 0 {
			total += *stop.Nights
		}
	}
	if total < len(route.Stops) {
		total = len(route.Stops)
	}
	if total < 1 {
		return 1
	}
	if total > maxDays {
		return maxDays
	}
	return total
}

func normalizeAlternativeIDs(ids []string, alternatives []routealternatives.Alternative) []string {
	if len(ids) == 0 {
		out := make([]string, 0, len(alternatives))
		for _, alternative := range alternatives {
			out = append(out, alternative.ID)
		}
		return out
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
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

func truncateString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max > 0 && len(value) > max {
		return strings.TrimSpace(value[:max])
	}
	return value
}
