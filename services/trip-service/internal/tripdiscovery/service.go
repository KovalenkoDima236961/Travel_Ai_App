package tripdiscovery

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	maxPromptLength        = 1000
	maxPreviousTrips       = 15
	defaultSuggestionCount = 5
)

type Repository interface {
	CreateTripDiscoverySession(context.Context, *Session) (*Session, error)
	GetTripDiscoverySessionByID(context.Context, uuid.UUID) (*Session, error)
	GetTripDiscoverySessionByIDAndUser(context.Context, uuid.UUID, uuid.UUID) (*Session, error)
	ListTripDiscoverySessionsByUser(context.Context, uuid.UUID, int) ([]Session, error)
	MarkTripDiscoverySessionCreatedTrip(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
	) (*Session, error)
	ListByUser(context.Context, uuid.UUID, int, int) ([]entity.Trip, error)
	UpdateTripCreationMetadata(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		map[string]any,
	) (*entity.Trip, error)
	UpsertDiscoverySuggestionVote(context.Context, *entity.DiscoverySuggestionVote) (*entity.DiscoverySuggestionVote, error)
	ListDiscoverySuggestionVotesBySession(context.Context, uuid.UUID) ([]entity.DiscoverySuggestionVote, error)
}

type TripCreator interface {
	Create(context.Context, appdto.CreateTripInput) (*entity.Trip, error)
	Get(context.Context, uuid.UUID) (*entity.Trip, error)
}

type GenerationJobCreator interface {
	Create(context.Context, uuid.UUID, generationjobs.CreateRequest) (*entity.GenerationJob, error)
}

type UserContextProvider interface {
	GetUserContext(context.Context, string) (*usercontext.UserContext, error)
}

type WorkspaceProvider interface {
	AccessCheck(context.Context, uuid.UUID, uuid.UUID) (*workspaces.Access, error)
}

type WorkspacePolicyProvider interface {
	GetActive(context.Context, uuid.UUID) (*workspacepolicies.Policy, error)
}

type Config struct {
	Enabled                bool
	MaxPreviousTrips       int
	DefaultSuggestionCount int
}

type Service struct {
	repo       Repository
	ai         AIClient
	trips      TripCreator
	jobs       GenerationJobCreator
	users      UserContextProvider
	workspaces WorkspaceProvider
	policies   WorkspacePolicyProvider
	cfg        Config
	log        *zap.Logger
}

func NewService(
	repo Repository,
	ai AIClient,
	trips TripCreator,
	jobs GenerationJobCreator,
	users UserContextProvider,
	workspaces WorkspaceProvider,
	policies WorkspacePolicyProvider,
	cfg Config,
	log *zap.Logger,
) *Service {
	if cfg.MaxPreviousTrips <= 0 {
		cfg.MaxPreviousTrips = maxPreviousTrips
	}
	if cfg.MaxPreviousTrips > 20 {
		cfg.MaxPreviousTrips = 20
	}
	if cfg.DefaultSuggestionCount < 3 || cfg.DefaultSuggestionCount > 5 {
		cfg.DefaultSuggestionCount = defaultSuggestionCount
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo: repo, ai: ai, trips: trips, jobs: jobs, users: users,
		workspaces: workspaces, policies: policies, cfg: cfg, log: log,
	}
}

func (s *Service) Discover(ctx context.Context, input DiscoverInput) (*Session, error) {
	return s.discover(ctx, ModePrompt, input, nil, nil)
}

func (s *Service) Surprise(ctx context.Context, input DiscoverInput) (*Session, error) {
	return s.discover(ctx, ModeSurprise, input, nil, nil)
}

func (s *Service) discover(
	ctx context.Context,
	mode Mode,
	input DiscoverInput,
	parent *Session,
	refinement *Refinement,
) (*Session, error) {
	if !s.cfg.Enabled {
		return nil, apperrs.NewDependencyError("Trip discovery is disabled.")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateDiscoverInput(mode, input); err != nil {
		return nil, err
	}
	if input.Scope == "" {
		input.Scope = "personal"
	}
	if input.Scope == "workspace" {
		if input.WorkspaceID == nil {
			return nil, apperrs.NewInvalidInput("workspaceId is required for workspace discovery")
		}
		if err := s.requireWorkspaceCreateAccess(ctx, user.ID, *input.WorkspaceID); err != nil {
			return nil, err
		}
	}

	request, err := s.buildAIRequest(ctx, user, mode, input, refinement)
	if err != nil {
		return nil, err
	}
	response, err := s.ai.SuggestDestinations(ctx, request)
	if err != nil {
		s.log.Warn(
			"trip discovery AI request failed",
			zap.String("user_id", user.ID.String()),
			zap.String("mode", string(mode)),
			zap.Error(err),
		)
		return nil, apperrs.NewDependencyError("Could not generate destination suggestions.")
	}
	normalizeSuggestions(response, request.Constraints.SuggestionCount)
	if len(response.Suggestions) == 0 {
		return nil, apperrs.NewDependencyError("AI returned no usable destination suggestions.")
	}
	session := &Session{
		ID:             uuid.New(),
		UserID:         user.ID,
		WorkspaceID:    input.WorkspaceID,
		Mode:           mode,
		Prompt:         strings.TrimSpace(input.Prompt),
		OutputLanguage: request.OutputLanguage,
		Status:         "completed",
		Request:        request,
		Response:       *response,
	}
	if parent != nil {
		session.ParentSessionID = &parent.ID
	}
	created, err := s.repo.CreateTripDiscoverySession(ctx, session)
	if err != nil {
		return nil, err
	}
	s.log.Info(
		"trip discovery suggestions created",
		zap.String("session_id", created.ID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("mode", string(mode)),
		zap.Int("suggestion_count", len(created.Response.Suggestions)),
	)
	return created, nil
}

func (s *Service) Refine(
	ctx context.Context,
	sessionID uuid.UUID,
	input RefineInput,
) (*Session, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	instruction := strings.TrimSpace(input.Instruction)
	if instruction == "" {
		return nil, apperrs.NewInvalidInput("instruction is required")
	}
	if len(instruction) > maxPromptLength {
		return nil, apperrs.NewInvalidInput("instruction must be 1000 characters or fewer")
	}
	parent, err := s.repo.GetTripDiscoverySessionByIDAndUser(ctx, sessionID, user.ID)
	if err != nil {
		return nil, err
	}
	discoverInput := discoverInputFromRequest(parent.Request)
	discoverInput.WorkspaceID = parent.WorkspaceID
	if normalized := normalizeLanguage(input.OutputLanguage); normalized != "" {
		discoverInput.OutputLanguage = normalized
	}
	refinement := &Refinement{
		PreviousSuggestions:  parent.Response.Suggestions,
		SelectedSuggestionID: strings.TrimSpace(input.SelectedSuggestionID),
		Instruction:          instruction,
	}
	if input.FeedbackType != "" {
		refinement.Instruction += " Feedback: " + strings.TrimSpace(input.FeedbackType) + "."
	}
	return s.discover(ctx, ModeRefine, discoverInput, parent, refinement)
}

func (s *Service) Get(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.accessibleSession(ctx, sessionID, user.ID)
}

func (s *Service) List(ctx context.Context, limit int) ([]Session, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListTripDiscoverySessionsByUser(ctx, user.ID, limit)
}

func (s *Service) VoteSuggestion(
	ctx context.Context,
	sessionID uuid.UUID,
	suggestionID string,
	input VoteSuggestionInput,
) (*SuggestionVotesResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.accessibleSession(ctx, sessionID, user.ID)
	if err != nil {
		return nil, err
	}
	suggestion, ok := findSuggestion(session.Response.Suggestions, suggestionID)
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	if !input.Vote.Valid() {
		return nil, apperrs.NewInvalidInput("vote is invalid")
	}
	metadata := map[string]any{}
	for key, value := range input.Metadata {
		metadata[key] = value
	}
	metadata["destination"] = suggestion.Destination
	if suggestion.SuggestionType != "" {
		metadata["suggestionType"] = suggestion.SuggestionType
	}
	if _, err := s.repo.UpsertDiscoverySuggestionVote(ctx, &entity.DiscoverySuggestionVote{
		ID:           uuid.New(),
		SessionID:    session.ID,
		SuggestionID: suggestion.ID,
		TripID:       session.CreatedTripID,
		UserID:       user.ID,
		Vote:         input.Vote,
		Metadata:     metadata,
	}); err != nil {
		return nil, err
	}
	return s.SuggestionVotes(ctx, sessionID)
}

func (s *Service) SuggestionVotes(ctx context.Context, sessionID uuid.UUID) (*SuggestionVotesResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.accessibleSession(ctx, sessionID, user.ID)
	if err != nil {
		return nil, err
	}
	votes, err := s.repo.ListDiscoverySuggestionVotesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	bySuggestion := map[string]*SuggestionVoteSummary{}
	for _, suggestion := range session.Response.Suggestions {
		bySuggestion[suggestion.ID] = &SuggestionVoteSummary{
			SuggestionID: suggestion.ID,
			Counts: map[string]int{
				string(entity.DiscoverySuggestionVoteLike):          0,
				string(entity.DiscoverySuggestionVoteDislike):       0,
				string(entity.DiscoverySuggestionVoteFavorite):      0,
				string(entity.DiscoverySuggestionVoteNotInterested): 0,
			},
		}
	}
	for _, vote := range votes {
		summary := bySuggestion[vote.SuggestionID]
		if summary == nil {
			summary = &SuggestionVoteSummary{
				SuggestionID: vote.SuggestionID,
				Counts:       map[string]int{},
			}
			bySuggestion[vote.SuggestionID] = summary
		}
		summary.Counts[string(vote.Vote)]++
		summary.Score += suggestionVoteScore(vote.Vote)
		if vote.UserID == user.ID {
			summary.CurrentUser = string(vote.Vote)
		}
	}
	items := make([]SuggestionVoteSummary, 0, len(bySuggestion))
	for _, summary := range bySuggestion {
		items = append(items, *summary)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SuggestionID < items[j].SuggestionID
	})
	return &SuggestionVotesResponse{SessionID: sessionID, Items: items}, nil
}

func (s *Service) accessibleSession(ctx context.Context, sessionID, userID uuid.UUID) (*Session, error) {
	session, err := s.repo.GetTripDiscoverySessionByIDAndUser(ctx, sessionID, userID)
	if err == nil {
		return session, nil
	}
	if !errors.Is(err, domainerrs.ErrNotFound) {
		return nil, err
	}
	session, err = s.repo.GetTripDiscoverySessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.CreatedTripID == nil {
		return nil, domainerrs.ErrNotFound
	}
	if s.trips == nil {
		return nil, domainerrs.ErrNotFound
	}
	if _, err := s.trips.Get(ctx, *session.CreatedTripID); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) CreateTrip(
	ctx context.Context,
	sessionID uuid.UUID,
	suggestionID string,
	input CreateTripInput,
) (*CreateTripResult, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.repo.GetTripDiscoverySessionByIDAndUser(ctx, sessionID, user.ID)
	if err != nil {
		return nil, err
	}
	suggestion, ok := findSuggestion(session.Response.Suggestions, suggestionID)
	if !ok {
		return nil, domainerrs.ErrNotFound
	}
	if session.CreatedTripID != nil {
		return nil, apperrs.NewConflict("A trip has already been created from this discovery session.")
	}
	if !sameUUID(input.WorkspaceID, session.WorkspaceID) {
		return nil, apperrs.NewInvalidInput(
			"workspaceId must match the discovery session scope",
		)
	}
	duration := input.DurationDays
	if duration == 0 {
		duration = suggestion.RecommendedDurationDays
	}
	if duration < 1 || duration > 30 {
		return nil, apperrs.NewInvalidInput("durationDays must be between 1 and 30")
	}
	travelers := input.Travelers
	if travelers == 0 {
		travelers = int32(session.Request.TripContext.Travelers)
	}
	if travelers < 1 || travelers > 50 {
		return nil, apperrs.NewInvalidInput("travelers must be between 1 and 50")
	}
	budget := input.Budget
	if budget == nil {
		budget = &Budget{
			Amount:   suggestion.EstimatedBudget.Amount,
			Currency: suggestion.EstimatedBudget.Currency,
		}
	}
	startDate := strings.TrimSpace(input.StartDate)
	if startDate == "" && session.Request.TripContext.StartDate != nil {
		startDate = *session.Request.TripContext.StartDate
	}
	route := input.Route
	if route == nil {
		route = suggestion.Route
	}
	tripType := strings.TrimSpace(input.TripType)
	if tripType == "" && route != nil {
		tripType = entity.TripTypeMultiDestination
	}
	destination := strings.TrimSpace(input.Title)
	if destination == "" {
		destination = suggestion.Destination
	}
	trip, err := s.trips.Create(ctx, appdto.CreateTripInput{
		TripType:       tripType,
		Destination:    destination,
		WorkspaceID:    input.WorkspaceID,
		StartDate:      startDate,
		Days:           int32(duration),
		BudgetAmount:   &budget.Amount,
		BudgetCurrency: budget.Currency,
		Travelers:      travelers,
		Interests:      append([]string(nil), suggestion.Tags...),
		Pace:           paceFromSession(session),
		Route:          route,
	})
	if err != nil {
		return nil, err
	}
	trip, err = s.repo.UpdateTripCreationMetadata(ctx, trip.ID, user.ID, map[string]any{
		"creationSource":              "trip_discovery",
		"discoverySessionId":          session.ID.String(),
		"discoverySuggestionId":       suggestion.ID,
		"discoveryMatchScore":         suggestion.MatchScore,
		"discoverySuggestionType":     suggestion.SuggestionType,
		"discoveryPrompt":             truncate(session.Prompt, maxPromptLength),
		"suggestedPromptForItinerary": suggestion.SuggestedPromptForItinerary,
		"outputLanguage":              session.OutputLanguage,
	})
	if err != nil {
		return nil, err
	}
	var job *entity.GenerationJob
	if input.AutoGenerateItinerary {
		instruction := suggestion.SuggestedPromptForItinerary
		expectedRevision := trip.ItineraryRevision
		job, err = s.jobs.Create(ctx, trip.ID, generationjobs.CreateRequest{
			JobType:                   entity.GenerationJobTypeFullGeneration,
			ExpectedItineraryRevision: &expectedRevision,
			Instruction:               &instruction,
		})
		if err != nil {
			return nil, err
		}
	}
	if _, err := s.repo.MarkTripDiscoverySessionCreatedTrip(
		ctx,
		session.ID,
		user.ID,
		trip.ID,
	); err != nil {
		return nil, err
	}
	s.log.Info(
		"trip created from discovery",
		zap.String("session_id", session.ID.String()),
		zap.String("suggestion_id", suggestion.ID),
		zap.String("trip_id", trip.ID.String()),
	)
	return &CreateTripResult{Trip: trip, GenerationJob: job}, nil
}

func (s *Service) buildAIRequest(
	ctx context.Context,
	user auth.AuthenticatedUser,
	mode Mode,
	input DiscoverInput,
	refinement *Refinement,
) (AIRequest, error) {
	language := normalizeLanguage(input.OutputLanguage)
	var trustedContext *usercontext.UserContext
	if s.users != nil && user.AccessToken != "" {
		value, err := s.users.GetUserContext(ctx, user.AccessToken)
		if err != nil {
			s.log.Warn("trip discovery user context unavailable", zap.Error(err))
		} else {
			trustedContext = value
		}
	}
	if language == "" && trustedContext != nil && trustedContext.Profile != nil {
		language = normalizeLanguage(trustedContext.Profile.PreferredLanguage)
	}
	if language == "" {
		language = "en"
	}
	currency := "EUR"
	if input.Budget != nil {
		input.Budget.Currency = strings.ToUpper(strings.TrimSpace(input.Budget.Currency))
		if input.Budget.Currency == "" {
			input.Budget.Currency = currency
		}
	}
	userContext := mapUserContext(trustedContext)
	if userContext != nil && userContext.PreferredCurrency != "" {
		currency = userContext.PreferredCurrency
	}
	if input.Budget != nil && input.Budget.Currency == "" {
		input.Budget.Currency = currency
	}
	previous, err := s.repo.ListByUser(ctx, user.ID, s.cfg.MaxPreviousTrips, 0)
	if err != nil {
		return AIRequest{}, err
	}
	var policyConstraints *PolicyConstraints
	var policy *workspacepolicies.Policy
	if input.WorkspaceID != nil && s.policies != nil {
		policyValue, policyErr := s.policies.GetActive(ctx, *input.WorkspaceID)
		if policyErr != nil && !errors.Is(policyErr, domainerrs.ErrNotFound) {
			return AIRequest{}, policyErr
		}
		policy = policyValue
		if policy != nil {
			if constraints := workspacepolicies.BuildAIConstraints(policy); constraints != nil {
				policyConstraints = &PolicyConstraints{
					Summary: constraints.Summary,
					Rules:   constraints.Rules,
				}
			}
		}
	}
	prompt := strings.TrimSpace(input.Prompt)
	if len(input.QuickChips) > 0 {
		chips := strings.Join(input.QuickChips, ", ")
		if prompt == "" {
			prompt = chips
		} else {
			prompt += "\nPreferences: " + chips
		}
	}
	avoidVisited := true
	if input.AvoidPreviouslyVisited != nil {
		avoidVisited = *input.AvoidPreviouslyVisited
	}
	preferNovelty := true
	if input.PreferNovelty != nil {
		preferNovelty = *input.PreferNovelty
	}
	complexity := "medium"
	if input.NoveltyLevel == "safe" {
		complexity = "low"
	} else if input.NoveltyLevel == "adventurous" {
		complexity = "high"
	}
	origin := strings.TrimSpace(input.Origin)
	if origin == "" && userContext != nil {
		origin = joinLocation(userContext.HomeCity, userContext.HomeCountry)
	}
	planningUserContext := usercontext.UserContext{}
	if trustedContext != nil {
		planningUserContext = *trustedContext
	}
	planning := planningconstraints.Build(planningconstraints.BuildInput{
		UserID:      user.ID,
		WorkspaceID: input.WorkspaceID,
		Source:      planningconstraints.SourceTripDiscovery,
		Request: planningconstraints.RequestOverride{
			OutputLanguage:  language,
			StartDate:       stringPtrValue(input.StartDate),
			DurationDays:    input.DurationDays,
			DateFlexibility: input.DateFlexibility,
			Budget:          discoveryBudgetOverride(input.Budget),
			Travelers:       discoveryTravelersOverride(input.Travelers),
			Prompt: &planningconstraints.Prompt{
				UserPrompt: prompt,
				QuickChips: input.QuickChips,
			},
		},
		UserContext:                planningUserContext,
		WorkspacePolicy:            policy,
		PreviousTrips:              previous,
		IncludePreviousTripSignals: true,
		IncludeRoute:               false,
	})
	if len(planning.Blockers) > 0 {
		return AIRequest{}, planningconstraints.NewBlockingError(planning)
	}
	return AIRequest{
		Prompt:         prompt,
		Mode:           mode,
		OutputLanguage: language,
		UserContext:    userContext,
		TripContext: TripContext{
			DurationDays:    input.DurationDays,
			StartDate:       input.StartDate,
			DateFlexibility: input.DateFlexibility,
			Budget:          input.Budget,
			Travelers:       max(input.Travelers, 1),
			Origin:          origin,
			Scope:           input.Scope,
		},
		PreviousTrips:              summarizeTrips(previous),
		WorkspacePolicyConstraints: policyConstraints,
		PlanningConstraints:        &planning,
		Refinement:                 refinement,
		Constraints: Constraints{
			SuggestionCount:        s.cfg.DefaultSuggestionCount,
			AvoidPreviouslyVisited: avoidVisited,
			PreferNovelty:          preferNovelty,
			IncludeReasoning:       true,
			MaxTravelComplexity:    complexity,
		},
	}, nil
}

func (s *Service) requireWorkspaceCreateAccess(
	ctx context.Context,
	userID, workspaceID uuid.UUID,
) error {
	if s.workspaces == nil {
		return apperrs.ErrForbidden
	}
	access, err := s.workspaces.AccessCheck(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if access == nil || !access.HasAccess || access.WorkspaceArchived || access.Status != "active" {
		return apperrs.ErrForbidden
	}
	if access.Role == workspaces.RoleViewer {
		return apperrs.ErrForbidden
	}
	return nil
}

func validateDiscoverInput(mode Mode, input DiscoverInput) error {
	prompt := strings.TrimSpace(input.Prompt)
	if len(prompt) > maxPromptLength {
		return apperrs.NewInvalidInput("prompt must be 1000 characters or fewer")
	}
	if mode == ModePrompt && prompt == "" && len(input.QuickChips) == 0 {
		return apperrs.NewInvalidInput("prompt or quickChips is required")
	}
	if input.Scope != "" && input.Scope != "personal" && input.Scope != "workspace" {
		return apperrs.NewInvalidInput("scope must be personal or workspace")
	}
	if input.DurationDays != nil && (*input.DurationDays < 1 || *input.DurationDays > 30) {
		return apperrs.NewInvalidInput("durationDays must be between 1 and 30")
	}
	if input.StartDate != nil && *input.StartDate != "" {
		if _, err := time.Parse("2006-01-02", *input.StartDate); err != nil {
			return apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
		}
	}
	if input.Budget != nil {
		if input.Budget.Amount < 0 {
			return apperrs.NewInvalidInput("budget amount must be zero or greater")
		}
		currency := strings.TrimSpace(input.Budget.Currency)
		if currency != "" && len(currency) != 3 {
			return apperrs.NewInvalidInput("budget currency must be 3 letters")
		}
	}
	if input.Travelers < 0 || input.Travelers > 50 {
		return apperrs.NewInvalidInput("travelers must be between 1 and 50")
	}
	if len(input.QuickChips) > 20 {
		return apperrs.NewInvalidInput("quickChips must contain 20 items or fewer")
	}
	if input.OutputLanguage != "" && normalizeLanguage(input.OutputLanguage) == "" {
		return apperrs.NewInvalidInput("outputLanguage must be en, es, uk, or fr")
	}
	if input.NoveltyLevel != "" &&
		input.NoveltyLevel != "safe" &&
		input.NoveltyLevel != "balanced" &&
		input.NoveltyLevel != "adventurous" {
		return apperrs.NewInvalidInput("noveltyLevel must be safe, balanced, or adventurous")
	}
	return nil
}

func normalizeSuggestions(response *SuggestionResponse, limit int) {
	seen := make(map[string]struct{})
	result := make([]Suggestion, 0, len(response.Suggestions))
	for _, suggestion := range response.Suggestions {
		suggestion.ID = strings.TrimSpace(suggestion.ID)
		suggestion.Destination = strings.TrimSpace(suggestion.Destination)
		if suggestion.ID == "" || suggestion.Destination == "" {
			continue
		}
		if _, exists := seen[suggestion.ID]; exists {
			continue
		}
		seen[suggestion.ID] = struct{}{}
		suggestion.SuggestionType = strings.TrimSpace(suggestion.SuggestionType)
		if suggestion.SuggestionType == "" {
			suggestion.SuggestionType = "single_destination"
		}
		if suggestion.SuggestionType != "route" {
			suggestion.SuggestionType = "single_destination"
			suggestion.Route = nil
		}
		suggestion.MatchScore = min(max(suggestion.MatchScore, 0), 100)
		if suggestion.RecommendedDurationDays < 1 {
			suggestion.RecommendedDurationDays = 1
		}
		if suggestion.RecommendedDurationDays > 30 {
			suggestion.RecommendedDurationDays = 30
		}
		result = append(result, suggestion)
		if len(result) == limit {
			break
		}
	}
	response.Suggestions = result
}

func mapUserContext(value *usercontext.UserContext) *UserContext {
	if value == nil {
		return nil
	}
	result := &UserContext{PreferredCurrency: "EUR", PreferredLanguage: "en"}
	if value.Profile != nil {
		result.HomeCity = value.Profile.HomeCity
		result.HomeCountry = value.Profile.HomeCountry
		if currency := strings.ToUpper(strings.TrimSpace(value.Profile.PreferredCurrency)); currency != "" {
			result.PreferredCurrency = currency
		}
		if language := normalizeLanguage(value.Profile.PreferredLanguage); language != "" {
			result.PreferredLanguage = language
		}
	}
	if value.Preferences != nil {
		result.Preferences = &UserPreferences{
			TravelStyles:       append([]string(nil), value.Preferences.TravelStyles...),
			Pace:               value.Preferences.Pace,
			MaxWalkingKmPerDay: value.Preferences.MaxWalkingKmPerDay,
			FoodPreferences:    append([]string(nil), value.Preferences.FoodPreferences...),
			Avoid:              append([]string(nil), value.Preferences.Avoid...),
			PreferredTransport: append([]string(nil), value.Preferences.PreferredTransport...),
		}
	}
	return result
}

func summarizeTrips(trips []entity.Trip) []PreviousTripSummary {
	result := make([]PreviousTripSummary, 0, len(trips))
	for _, trip := range trips {
		destination, country := splitDestination(trip.Destination)
		item := PreviousTripSummary{
			Destination:  destination,
			Country:      country,
			DurationDays: trip.Days,
			Tags:         append([]string(nil), trip.Interests...),
			Pace:         trip.Pace,
			CreatedAt:    trip.CreatedAt.Format("2006-01-02"),
		}
		if trip.BudgetAmount != nil {
			item.Budget = &Budget{Amount: *trip.BudgetAmount, Currency: trip.BudgetCurrency}
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt > result[j].CreatedAt })
	return result
}

func discoverInputFromRequest(request AIRequest) DiscoverInput {
	return DiscoverInput{
		Prompt:          request.Prompt,
		Scope:           request.TripContext.Scope,
		DurationDays:    request.TripContext.DurationDays,
		StartDate:       request.TripContext.StartDate,
		DateFlexibility: request.TripContext.DateFlexibility,
		Budget:          request.TripContext.Budget,
		Travelers:       request.TripContext.Travelers,
		Origin:          request.TripContext.Origin,
		OutputLanguage:  request.OutputLanguage,
	}
}

func findSuggestion(suggestions []Suggestion, id string) (Suggestion, bool) {
	for _, suggestion := range suggestions {
		if suggestion.ID == strings.TrimSpace(id) {
			return suggestion, true
		}
	}
	return Suggestion{}, false
}

func normalizeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "en", "es", "uk", "fr":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func splitDestination(value string) (string, string) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(value), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func joinLocation(city, country *string) string {
	values := make([]string, 0, 2)
	if city != nil && strings.TrimSpace(*city) != "" {
		values = append(values, strings.TrimSpace(*city))
	}
	if country != nil && strings.TrimSpace(*country) != "" {
		values = append(values, strings.TrimSpace(*country))
	}
	return strings.Join(values, ", ")
}

func paceFromSession(session *Session) string {
	if session.Request.UserContext != nil &&
		session.Request.UserContext.Preferences != nil &&
		strings.TrimSpace(session.Request.UserContext.Preferences.Pace) != "" {
		pace := strings.TrimSpace(session.Request.UserContext.Preferences.Pace)
		if pace == "intensive" {
			return "packed"
		}
		return pace
	}
	return "balanced"
}

func sameUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func discoveryBudgetOverride(value *Budget) *planningconstraints.BudgetOverride {
	if value == nil {
		return nil
	}
	amount := value.Amount
	return &planningconstraints.BudgetOverride{Amount: &amount, Currency: value.Currency}
}

func discoveryTravelersOverride(value int) *planningconstraints.TravelerOverride {
	if value <= 0 {
		return nil
	}
	count := int32(value)
	return &planningconstraints.TravelerOverride{Count: &count}
}

func suggestionVoteScore(vote entity.DiscoverySuggestionVoteValue) int {
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
