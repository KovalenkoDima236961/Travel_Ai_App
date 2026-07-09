package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/providerlimit"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/sharing"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	defaultCurrency = "EUR"
	defaultPace     = "balanced"

	maxDays                    = 30
	maxItineraryDays           = 60
	maxItineraryItemsPerDay    = 30
	maxInstructionLength       = 2000
	maxPlaceURLLength          = 2048
	maxPlaceEnrichmentQuery    = 300
	maxPlaceEnrichmentProvider = 50
	maxPlaceEnrichmentReason   = 200
	maxPriceEnrichmentProvider = 50
	maxPriceEnrichmentReason   = 200
	maxAccommodationNameLength = 200
	maxAccommodationAddress    = 500
	maxAccommodationNotes      = 1000
	defaultLimit               = 20
	maxLimit                   = 100
)

type editableItinerary struct {
	Destination string                   `json:"destination,omitempty"`
	Summary     string                   `json:"summary,omitempty"`
	Travelers   int32                    `json:"travelers,omitempty"`
	Pace        string                   `json:"pace,omitempty"`
	Currency    string                   `json:"currency,omitempty"`
	TotalBudget *float64                 `json:"totalBudget,omitempty"`
	Days        []aggregate.ItineraryDay `json:"days"`
	GeneratedAt *time.Time               `json:"generatedAt,omitempty"`
	Source      string                   `json:"source,omitempty"`
}

// tripRepository is the persistence port the use case depends on. The concrete
// postgres adapter satisfies it; tests substitute a mock. It is intentionally
// unexported — the use case owns the abstraction it consumes.
type tripRepository interface {
	Create(ctx context.Context, t *entity.Trip) (*entity.Trip, error)
	UpdateTripBudget(ctx context.Context, id, userID uuid.UUID, amount *float64, currency string) (*entity.Trip, error)
	UpdateTripAccommodation(ctx context.Context, id, userID uuid.UUID, accommodation *aggregate.Accommodation) (*entity.Trip, error)
	ClearTripAccommodation(ctx context.Context, id, userID uuid.UUID) (*entity.Trip, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Trip, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*entity.Trip, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Trip, error)
	ListAccessible(ctx context.Context, userID uuid.UUID, workspaceIDs []uuid.UUID, scope appdto.TripListScope, workspaceID *uuid.UUID, limit, offset int) ([]entity.Trip, error)
	UpdateStatusByUserID(ctx context.Context, id, userID uuid.UUID, status entity.Status) (*entity.Trip, error)
	UpdateItineraryAndCreateVersion(
		ctx context.Context,
		id, ownerUserID, actorUserID uuid.UUID,
		itinerary json.RawMessage,
		status entity.Status,
		expectedItineraryRevision int,
		source entity.ItineraryVersionSource,
		metadata map[string]any,
	) (*entity.Trip, *entity.ItineraryVersion, error)
	UpdateItineraryByUserIDAndCreateVersion(
		ctx context.Context,
		id, userID uuid.UUID,
		itinerary json.RawMessage,
		status entity.Status,
		expectedItineraryRevision int,
		source entity.ItineraryVersionSource,
		metadata map[string]any,
	) (*entity.Trip, *entity.ItineraryVersion, error)
	ListItineraryVersionsByTrip(ctx context.Context, tripID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error)
	ListItineraryVersionsByTripAndUser(ctx context.Context, tripID, userID uuid.UUID, limit, offset int) ([]entity.ItineraryVersion, error)
	GetItineraryVersionByIDTrip(ctx context.Context, id, tripID uuid.UUID) (*entity.ItineraryVersion, error)
	GetItineraryVersionByIDTripAndUser(ctx context.Context, id, tripID, userID uuid.UUID) (*entity.ItineraryVersion, error)
	UpsertTripCollaborator(ctx context.Context, collaborator *entity.TripCollaborator) (*entity.TripCollaborator, error)
	GetTripCollaboratorByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripCollaborator, error)
	GetTripCollaboratorByID(ctx context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error)
	ListTripCollaborators(ctx context.Context, tripID uuid.UUID) ([]entity.TripCollaborator, error)
	UpdateTripCollaboratorRole(ctx context.Context, tripID, collaboratorID uuid.UUID, role entity.CollaboratorRole) (*entity.TripCollaborator, error)
	RemoveTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID) (*entity.TripCollaborator, error)
	AcceptTripCollaborator(ctx context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error)
	DeclineTripCollaborator(ctx context.Context, tripID, collaboratorID, userID uuid.UUID) (*entity.TripCollaborator, error)
	ListPendingCollaborationInvitations(ctx context.Context, userID uuid.UUID) ([]entity.SharedTrip, error)
	ListSharedTripsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SharedTrip, error)
	CreateTripTraveler(ctx context.Context, traveler *entity.TripTraveler) (*entity.TripTraveler, error)
	GetTripTravelerByID(ctx context.Context, tripID, travelerID uuid.UUID) (*entity.TripTraveler, error)
	ListTripTravelersByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripTraveler, error)
	ListActiveTripTravelersByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripTraveler, error)
	UpdateTripTraveler(ctx context.Context, traveler *entity.TripTraveler) (*entity.TripTraveler, error)
	RemoveTripTraveler(ctx context.Context, tripID, travelerID uuid.UUID) (*entity.TripTraveler, error)
	GetTripTravelerByLinkedUser(ctx context.Context, tripID, linkedUserID uuid.UUID) (*entity.TripTraveler, error)
	CountActiveTravelersByTrip(ctx context.Context, tripID uuid.UUID) (int, error)
	CreateTripShare(ctx context.Context, share *entity.TripShare) (*entity.TripShare, error)
	GetTripShareByTripAndUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error)
	GetTripShareByToken(ctx context.Context, shareToken string) (*entity.TripShare, error)
	EnableTripShare(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error)
	UpdateTripShareSettings(ctx context.Context, tripID, userID uuid.UUID, expiresAt *time.Time, passwordRequired bool, passwordHash *string) (*entity.TripShare, error)
	DisableTripShare(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripShare, error)
	CreateItineraryComment(ctx context.Context, comment *entity.ItineraryComment) (*entity.ItineraryComment, error)
	ListItineraryCommentsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.ItineraryComment, error)
	ListItineraryCommentsByItem(ctx context.Context, tripID uuid.UUID, dayNumber, itemIndex int) ([]entity.ItineraryComment, error)
	GetItineraryCommentByID(ctx context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error)
	UpdateItineraryCommentBody(ctx context.Context, tripID, commentID uuid.UUID, body string) (*entity.ItineraryComment, error)
	SoftDeleteItineraryComment(ctx context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error)
	CountItineraryCommentsByTripGrouped(ctx context.Context, tripID uuid.UUID) ([]entity.ItineraryCommentCount, error)
	UpsertTripCalendarSync(ctx context.Context, sync *entity.TripCalendarSync) (*entity.TripCalendarSync, error)
	ListTripCalendarSyncsByTripUserProvider(ctx context.Context, tripID, userID uuid.UUID, provider string) ([]entity.TripCalendarSync, error)
	GetTripCalendarSyncStatus(ctx context.Context, tripID, userID uuid.UUID, provider string) (int, *time.Time, int, error)
	GetActiveTripCalendarSyncByKey(ctx context.Context, tripID, userID uuid.UUID, provider, syncKey string) (*entity.TripCalendarSync, error)
	MarkTripCalendarSyncDeleted(ctx context.Context, tripID, userID uuid.UUID, provider, syncKey string) error
	MarkAllTripCalendarSyncsDeleted(ctx context.Context, tripID, userID uuid.UUID, provider string) error
	CreateBudgetOptimizationProposal(ctx context.Context, proposal *entity.BudgetOptimizationProposal) (*entity.BudgetOptimizationProposal, error)
	GetBudgetOptimizationProposalByIDAndTrip(ctx context.Context, id, tripID uuid.UUID) (*entity.BudgetOptimizationProposal, error)
	ListBudgetOptimizationProposalsByTrip(ctx context.Context, tripID uuid.UUID, status *entity.BudgetOptimizationProposalStatus, limit int) ([]entity.BudgetOptimizationProposal, error)
	ListPendingBudgetOptimizationProposalsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.BudgetOptimizationProposal, error)
	MarkBudgetOptimizationProposalApplied(ctx context.Context, id uuid.UUID, appliedItineraryRevision int) (*entity.BudgetOptimizationProposal, error)
	MarkBudgetOptimizationProposalDiscarded(ctx context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error)
	MarkBudgetOptimizationProposalExpired(ctx context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error)
	MarkBudgetOptimizationProposalFailed(ctx context.Context, id uuid.UUID) (*entity.BudgetOptimizationProposal, error)
	CreateWorkspaceBudget(ctx context.Context, budget *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error)
	GetWorkspaceBudgetByID(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error)
	ListWorkspaceBudgetsByWorkspace(ctx context.Context, workspaceID uuid.UUID, status *entity.WorkspaceBudgetStatus) ([]entity.WorkspaceBudget, error)
	ListActiveWorkspaceBudgetsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]entity.WorkspaceBudget, error)
	GetPrimaryWorkspaceBudget(ctx context.Context, workspaceID uuid.UUID) (*entity.WorkspaceBudget, error)
	UpdateWorkspaceBudget(ctx context.Context, budget *entity.WorkspaceBudget) (*entity.WorkspaceBudget, error)
	ArchiveWorkspaceBudget(ctx context.Context, workspaceID, budgetID, actorUserID uuid.UUID) (*entity.WorkspaceBudget, error)
	SetWorkspaceBudgetPrimary(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error)
	CountWorkspaceBudgets(ctx context.Context, workspaceID uuid.UUID, status *entity.WorkspaceBudgetStatus) (int, error)
	approvalRepository
}

type userContextProvider interface {
	GetUserContext(ctx context.Context, accessToken string) (*usercontext.UserContext, error)
}

type weatherContextProvider interface {
	GetForecast(ctx context.Context, destination string, startDate string, days int) (*weathercontext.WeatherForecast, error)
}

type placeEnrichmentProvider interface {
	EnrichItinerary(ctx context.Context, input placeenrichment.EnrichItineraryInput) (*placeenrichment.EnrichItineraryResult, error)
}

type priceEnrichmentProvider interface {
	EnrichItinerary(ctx context.Context, input priceenrichment.EnrichItineraryInput) (*priceenrichment.EnrichItineraryResult, error)
}

type userLookupProvider interface {
	LookupByEmail(ctx context.Context, email string) (*appdto.UserLookupResult, error)
}

type calendarSyncProvider interface {
	GetGoogleCalendarStatus(ctx context.Context, accessToken string) (*calendarclient.ConnectionStatus, error)
	SyncGoogleCalendarEvents(ctx context.Context, input calendarclient.SyncRequest) (*calendarclient.SyncResult, error)
	DeleteGoogleCalendarEvents(ctx context.Context, input calendarclient.DeleteRequest) (*calendarclient.DeleteResult, error)
}

type budgetConversionProvider interface {
	Convert(ctx context.Context, amount float64, from string, to string) (*budget.CurrencyConversionResult, error)
}

type workspaceProvider interface {
	AccessCheck(ctx context.Context, userID, workspaceID uuid.UUID) (*workspaces.Access, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]workspaces.UserWorkspace, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]workspaces.WorkspaceMember, error)
}

type workspacePolicyProvider interface {
	GetActive(ctx context.Context, workspaceID uuid.UUID) (*workspacepolicies.Policy, error)
}

// Option customizes Service dependencies that are not required for the core
// trip CRUD flow.
type Option func(*Service)

// WithUserContext enables optional user profile/preferences loading during
// itinerary generation.
func WithUserContext(provider userContextProvider, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.userContextProvider = provider
		s.userContextEnabled = enabled
		s.userContextFailOpen = failOpen
	}
}

// WithWeatherContext enables optional weather forecast loading during itinerary
// generation and regeneration.
func WithWeatherContext(provider weatherContextProvider, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.weatherContextProvider = provider
		s.weatherContextEnabled = enabled
		s.weatherContextFailOpen = failOpen
	}
}

// WithPlaceEnrichment enables optional automatic place metadata attachment
// after generated itinerary payloads are returned by AI Planning Service.
func WithPlaceEnrichment(provider placeEnrichmentProvider, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.placeEnrichmentProvider = provider
		s.placeEnrichmentEnabled = enabled
		s.placeEnrichmentFailOpen = failOpen
	}
}

// WithPriceEnrichment enables optional provider-based ticket/attraction price
// estimates after place enrichment.
func WithPriceEnrichment(provider priceEnrichmentProvider, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.priceEnrichmentProvider = provider
		s.priceEnrichmentEnabled = enabled
		s.priceEnrichmentFailOpen = failOpen
	}
}

func WithUserLookup(provider userLookupProvider) Option {
	return func(s *Service) {
		s.userLookupProvider = provider
	}
}

func WithCalendarSync(provider calendarSyncProvider, enabled bool, publicWebBaseURL, defaultTimeZone string) Option {
	return func(s *Service) {
		s.calendarSyncProvider = provider
		s.calendarSyncEnabled = enabled
		s.calendarSyncPublicWebBaseURL = strings.TrimRight(strings.TrimSpace(publicWebBaseURL), "/")
		s.calendarSyncDefaultTimeZone = strings.TrimSpace(defaultTimeZone)
	}
}

func WithBudgetConversion(provider budgetConversionProvider, enabled bool, failOpen bool) Option {
	return func(s *Service) {
		s.budgetConversionProvider = provider
		s.budgetConversionEnabled = enabled
		s.budgetConversionFailOpen = failOpen
	}
}

func WithWorkspaces(provider workspaceProvider, enabled bool) Option {
	return func(s *Service) {
		s.workspaceProvider = provider
		s.workspacesEnabled = enabled
	}
}

func WithWorkspacePolicies(provider workspacePolicyProvider) Option {
	return func(s *Service) {
		s.workspacePolicyProvider = provider
	}
}

// WithPublicSharing configures owner-managed public read-only trip links.
func WithPublicSharing(
	enabled bool,
	publicWebBaseURL string,
	shareTokenBytes int,
	publicShareAccessSecret string,
	publicShareAccessTTLMinutes int,
) Option {
	return func(s *Service) {
		s.publicSharingEnabled = enabled
		s.publicWebBaseURL = strings.TrimRight(strings.TrimSpace(publicWebBaseURL), "/")
		if shareTokenBytes >= 32 {
			s.shareTokenBytes = shareTokenBytes
		}
		s.publicShareTokens = sharing.NewPublicShareTokenManager(
			publicShareAccessSecret,
			time.Duration(publicShareAccessTTLMinutes)*time.Minute,
		)
	}
}

// Service holds the trip business logic. It depends on the repository and
// generator ports and a logger.
type Service struct {
	repo                         tripRepository
	generator                    application.ItineraryGenerator
	userContextProvider          userContextProvider
	userContextEnabled           bool
	userContextFailOpen          bool
	weatherContextProvider       weatherContextProvider
	weatherContextEnabled        bool
	weatherContextFailOpen       bool
	placeEnrichmentProvider      placeEnrichmentProvider
	placeEnrichmentEnabled       bool
	placeEnrichmentFailOpen      bool
	priceEnrichmentProvider      priceEnrichmentProvider
	priceEnrichmentEnabled       bool
	priceEnrichmentFailOpen      bool
	userLookupProvider           userLookupProvider
	calendarSyncProvider         calendarSyncProvider
	calendarSyncEnabled          bool
	calendarSyncPublicWebBaseURL string
	calendarSyncDefaultTimeZone  string
	budgetConversionProvider     budgetConversionProvider
	budgetConversionEnabled      bool
	budgetConversionFailOpen     bool
	workspaceProvider            workspaceProvider
	workspacePolicyProvider      workspacePolicyProvider
	workspacesEnabled            bool
	activity                     activityService
	notifier                     notifier
	notificationsEnabled         bool
	notificationsFailOpen        bool
	publicSharingEnabled         bool
	publicWebBaseURL             string
	shareTokenBytes              int
	publicShareTokens            *sharing.PublicShareTokenManager
	log                          *zap.Logger
}

// New constructs the trip service.
func New(repo tripRepository, generator application.ItineraryGenerator, log *zap.Logger, opts ...Option) *Service {
	s := &Service{
		repo:                     repo,
		generator:                generator,
		publicSharingEnabled:     true,
		publicWebBaseURL:         "http://localhost:3000",
		shareTokenBytes:          32,
		publicShareTokens:        sharing.NewPublicShareTokenManager("dev-public-share-secret-change-me", time.Hour),
		budgetConversionFailOpen: true,
		log:                      log,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create validates input, applies defaults, and stores a new DRAFT trip.
func (s *Service) Create(ctx context.Context, in appdto.CreateTripInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	destination := strings.TrimSpace(in.Destination)
	if destination == "" {
		return nil, apperrs.NewInvalidInput("destination is required")
	}
	if in.Days < 1 || in.Days > maxDays {
		return nil, apperrs.NewInvalidInput("days must be between 1 and %d", maxDays)
	}
	if in.Travelers < 1 {
		return nil, apperrs.NewInvalidInput("travelers must be at least 1")
	}
	if in.WorkspaceID != nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *in.WorkspaceID); err != nil {
			return nil, err
		}
	}

	currency := strings.ToUpper(strings.TrimSpace(in.BudgetCurrency))
	if currency == "" {
		currency = defaultCurrency
	}
	pace := strings.TrimSpace(in.Pace)
	if pace == "" {
		pace = defaultPace
	}

	var startDate *time.Time
	if strings.TrimSpace(in.StartDate) != "" {
		parsed, err := time.Parse("2006-01-02", in.StartDate)
		if err != nil {
			return nil, apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
		}
		startDate = &parsed
	}

	interests := in.Interests
	if interests == nil {
		interests = []string{}
	}

	created, err := s.repo.Create(ctx, &entity.Trip{
		UserID:         &user.ID,
		WorkspaceID:    in.WorkspaceID,
		Destination:    destination,
		StartDate:      startDate,
		Days:           in.Days,
		BudgetAmount:   in.BudgetAmount,
		BudgetCurrency: currency,
		Travelers:      in.Travelers,
		Interests:      interests,
		Pace:           pace,
		Status:         entity.StatusDraft,
	})
	if err != nil {
		return nil, err
	}

	s.log.Info("trip created",
		zap.String("trip_id", created.ID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("destination", created.Destination),
	)

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      created.ID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripCreated,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"destination": created.Destination,
			"days":        int(created.Days),
		},
	})

	return created, nil
}

// Get returns a trip by id.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*entity.Trip, error) {
	t, _, err := s.GetWithAccess(ctx, id)
	return t, err
}

func (s *Service) GetWithAccess(ctx context.Context, id uuid.UUID) (*entity.Trip, TripAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, TripAccess{Level: AccessLevelNone}, err
	}
	return s.requireViewerEditorOrOwner(ctx, id, user.ID)
}

// List returns trips ordered by created_at DESC. It normalises and validates the
// pagination parameters: limit defaults to 20 (when 0) and must be 1..100;
// offset must be >= 0.
func (s *Service) List(ctx context.Context, limit, offset int) ([]entity.Trip, int, int, error) {
	return s.ListWithFilters(ctx, appdto.ListTripsInput{Limit: limit, Offset: offset, Scope: appdto.TripListScopeAll})
}

func (s *Service) ListWithFilters(ctx context.Context, in appdto.ListTripsInput) ([]entity.Trip, int, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, 0, err
	}

	limit := in.Limit
	offset := in.Offset
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}
	if offset < 0 {
		return nil, 0, 0, apperrs.NewInvalidInput("offset must be >= 0")
	}

	workspaceIDs, err := s.accessibleWorkspaceIDs(ctx, user.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	trips, err := s.repo.ListAccessible(ctx, user.ID, workspaceIDs, normalizeTripListScope(in.Scope), in.WorkspaceID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return trips, limit, offset, nil
}

// Generate runs the planning flow: PROCESSING -> generate itinerary -> COMPLETED
// (or FAILED on error). The itinerary itself is produced by the injected
// ItineraryGenerator port.
func (s *Service) Generate(ctx context.Context, id uuid.UUID, in appdto.GenerateItineraryInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	current, _, err := s.requireEditorOrOwner(ctx, id, user.ID)
	if err != nil {
		return nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		return nil, err
	}

	weatherForecast, err := s.loadWeatherContext(ctx, *current, id)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpdateStatusByUserID(ctx, id, ownerID, entity.StatusProcessing); err != nil {
		return nil, err
	}
	s.log.Info("trip processing started",
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	)

	itinerary, err := s.generator.Generate(ctx, application.GenerateItineraryInput{
		Trip:                       *current,
		UserProfile:                userContext.Profile,
		UserPreferences:            userContext.Preferences,
		WeatherForecast:            weatherForecast,
		WorkspacePolicyConstraints: s.workspacePolicyAIConstraints(ctx, current),
	})
	if err != nil {
		s.markFailed(ctx, id, ownerID)
		return nil, err
	}

	itinerary, err = s.enrichGeneratedItinerary(ctx, id, *current, itinerary, userContext)
	if err != nil {
		s.markFailed(ctx, id, ownerID)
		return nil, err
	}

	raw, err := json.Marshal(itinerary)
	if err != nil {
		s.markFailed(ctx, id, ownerID)
		return nil, err
	}

	updated, err := s.saveItineraryWithVersion(
		ctx,
		id,
		ownerID,
		user.ID,
		raw,
		expectedRevision,
		entity.ItineraryVersionSourceGenerated,
		map[string]any{
			"generator": "full",
		},
	)
	if err != nil {
		if !isItineraryConflict(err) {
			s.markFailed(ctx, id, ownerID)
		}
		return nil, err
	}

	s.log.Info("trip completed",
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	)

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      id,
		ActorUserID: &user.ID,
		EventType:   activity.EventItineraryGenerated,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(id),
		Metadata:    map[string]any{"source": "GENERATED"},
	})

	// Notify accepted collaborators (if any) that the itinerary was generated.
	// When the owner generates an initial itinerary with no collaborators, the
	// recipient set is empty and no notification is created.
	destination := tripDestination(current)
	s.notifyTripBroadcast(ctx, current, user.ID,
		notifications.TypeItineraryGenerated,
		"Itinerary generated",
		fmt.Sprintf("The itinerary for %s was generated.", destination),
		notifications.EntityItinerary, activityEntityID(id),
		map[string]any{"tripId": id.String(), "destination": destination})

	return updated, nil
}

func (s *Service) GenerateForActor(
	ctx context.Context,
	id, actorUserID uuid.UUID,
	expectedRevision int,
) (*entity.Trip, error) {
	ctx = contextWithActor(ctx, actorUserID)
	return s.Generate(ctx, id, appdto.GenerateItineraryInput{
		ExpectedItineraryRevision: &expectedRevision,
	})
}

// UpdateItinerary validates and replaces the full itinerary JSON for a trip
// owned by the authenticated user. It does not call the itinerary generator.
func (s *Service) UpdateItinerary(ctx context.Context, id uuid.UUID, in appdto.UpdateItineraryInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, id, user.ID)
	if err != nil {
		return nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	fields := []zap.Field{
		zap.String("action", "itinerary_update"),
		zap.String("trip_id", id.String()),
		zap.String("user_id", user.ID.String()),
	}

	normalized, err := validateAndNormalizeItinerary(in.Itinerary)
	if err != nil {
		s.log.Warn("itinerary update failed",
			append(fields,
				zap.Bool("success", false),
				zap.Error(err),
			)...,
		)
		return nil, err
	}

	updated, err := s.saveItineraryWithVersion(
		ctx,
		id,
		ownerID,
		user.ID,
		normalized,
		expectedRevision,
		entity.ItineraryVersionSourceManualEdit,
		map[string]any{},
	)
	if err != nil {
		s.log.Warn("itinerary update failed",
			append(fields,
				zap.Bool("success", false),
				zap.Error(err),
			)...,
		)
		return nil, err
	}

	s.log.Info("itinerary updated",
		append(fields,
			zap.Bool("success", true),
		)...,
	)

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      id,
		ActorUserID: &user.ID,
		EventType:   activity.EventItineraryUpdated,
		EntityType:  activityEntityType(activity.EntityItinerary),
		EntityID:    activityEntityID(id),
		Metadata:    map[string]any{"source": "MANUAL_EDIT"},
	})

	destination := tripDestination(current)
	s.notifyTripBroadcast(ctx, current, user.ID,
		notifications.TypeItineraryUpdated,
		"Itinerary updated",
		fmt.Sprintf("The itinerary for %s was updated.", destination),
		notifications.EntityItinerary, activityEntityID(id),
		map[string]any{"tripId": id.String(), "destination": destination})

	return updated, nil
}

// RegenerateDay replaces only one existing itinerary day with an AI-generated
// replacement. DayNumber is one-based and matched against itinerary.days[].day.
func (s *Service) RegenerateDay(ctx context.Context, id uuid.UUID, dayNumber int, in appdto.RegenerateItineraryPartInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	instruction, err := normalizeRegenerationInstruction(in.Instruction)
	fields := regenerateLogFields("regenerate_day", id, user.ID, dayNumber, nil, instruction)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, id, user.ID)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}

	currentItinerary, dayIndex, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, false, err)
		return nil, err
	}
	userContextLoaded := userContext.Profile != nil || userContext.Preferences != nil

	weatherForecast, err := s.loadWeatherContext(ctx, *current, id)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	replacement, err := s.generator.RegenerateDay(ctx, application.RegenerateDayInput{
		Trip:                       *current,
		CurrentItinerary:           currentItinerary,
		DayNumber:                  dayNumber,
		Instruction:                instruction,
		UserProfile:                userContext.Profile,
		UserPreferences:            userContext.Preferences,
		WeatherForecast:            weatherForecast,
		WorkspacePolicyConstraints: s.workspacePolicyAIConstraints(ctx, current),
	})
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	normalizedReplacement, err := normalizeReplacementDay(replacement, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, apperrs.NewDependencyError("AI returned invalid replacement")
	}

	normalizedReplacement, err = s.enrichReplacementDay(ctx, id, *current, normalizedReplacement, userContext)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	currentItinerary.Days[dayIndex] = normalizedReplacement
	updated, err := s.saveRegeneratedItinerary(
		ctx,
		id,
		ownerID,
		user.ID,
		currentItinerary,
		expectedRevision,
		entity.ItineraryVersionSourceRegenerateDay,
		map[string]any{
			"dayNumber":          dayNumber,
			"instructionPresent": instruction != "",
		},
	)
	if err != nil {
		s.logRegenerationFailure("itinerary day regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	s.logRegenerationSuccess("itinerary day regenerated", fields, started, userContextLoaded)

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      id,
		ActorUserID: &user.ID,
		EventType:   activity.EventDayRegenerated,
		EntityType:  activityEntityType(activity.EntityItineraryDay),
		Metadata:    map[string]any{"dayNumber": dayNumber},
	})

	destination := tripDestination(current)
	s.notifyTripBroadcast(ctx, current, user.ID,
		notifications.TypeDayRegenerated,
		"Day regenerated",
		fmt.Sprintf("Day %d of %s was regenerated.", dayNumber, destination),
		notifications.EntityItineraryDay, nil,
		map[string]any{"tripId": id.String(), "destination": destination, "dayNumber": dayNumber})

	return updated, nil
}

func (s *Service) RegenerateDayForActor(
	ctx context.Context,
	id, actorUserID uuid.UUID,
	dayNumber int,
	instruction string,
	expectedRevision int,
) (*entity.Trip, error) {
	ctx = contextWithActor(ctx, actorUserID)
	return s.RegenerateDay(ctx, id, dayNumber, appdto.RegenerateItineraryPartInput{
		Instruction:               instruction,
		ExpectedItineraryRevision: &expectedRevision,
	})
}

// RegenerateItem replaces only one item in one itinerary day. DayNumber is
// one-based; ItemIndex is zero-based to match the items array index.
func (s *Service) RegenerateItem(ctx context.Context, id uuid.UUID, dayNumber, itemIndex int, in appdto.RegenerateItineraryPartInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	instruction, err := normalizeRegenerationInstruction(in.Instruction)
	fields := regenerateLogFields("regenerate_item", id, user.ID, dayNumber, &itemIndex, instruction)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, id, user.ID)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}

	currentItinerary, dayIndex, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	if itemIndex < 0 || itemIndex >= len(currentItinerary.Days[dayIndex].Items) {
		err := currentItineraryInvalidError()
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, id)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, false, err)
		return nil, err
	}
	userContextLoaded := userContext.Profile != nil || userContext.Preferences != nil

	weatherForecast, err := s.loadWeatherContext(ctx, *current, id)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	replacement, err := s.generator.RegenerateItem(ctx, application.RegenerateItemInput{
		Trip:                       *current,
		CurrentItinerary:           currentItinerary,
		DayNumber:                  dayNumber,
		ItemIndex:                  itemIndex,
		Instruction:                instruction,
		UserProfile:                userContext.Profile,
		UserPreferences:            userContext.Preferences,
		WeatherForecast:            weatherForecast,
		WorkspacePolicyConstraints: s.workspacePolicyAIConstraints(ctx, current),
	})
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	normalizedReplacement, err := normalizeReplacementItem(replacement)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, apperrs.NewDependencyError("AI returned invalid replacement")
	}

	normalizedReplacement, err = s.enrichReplacementItem(ctx, id, *current, dayNumber, normalizedReplacement, userContext)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	currentItinerary.Days[dayIndex].Items[itemIndex] = normalizedReplacement
	updated, err := s.saveRegeneratedItinerary(
		ctx,
		id,
		ownerID,
		user.ID,
		currentItinerary,
		expectedRevision,
		entity.ItineraryVersionSourceRegenerateItem,
		map[string]any{
			"dayNumber":          dayNumber,
			"itemIndex":          itemIndex,
			"instructionPresent": instruction != "",
		},
	)
	if err != nil {
		s.logRegenerationFailure("itinerary item regeneration failed", fields, started, userContextLoaded, err)
		return nil, err
	}

	s.logRegenerationSuccess("itinerary item regenerated", fields, started, userContextLoaded)

	itemMetadata := map[string]any{
		"dayNumber": dayNumber,
		"itemIndex": itemIndex,
	}
	if name := strings.TrimSpace(normalizedReplacement.Name); name != "" {
		itemMetadata["itemName"] = name
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      id,
		ActorUserID: &user.ID,
		EventType:   activity.EventItemRegenerated,
		EntityType:  activityEntityType(activity.EntityItineraryItem),
		Metadata:    itemMetadata,
	})

	destination := tripDestination(current)
	itemNotificationMetadata := map[string]any{
		"tripId":      id.String(),
		"destination": destination,
		"dayNumber":   dayNumber,
		"itemIndex":   itemIndex,
	}
	if name := strings.TrimSpace(normalizedReplacement.Name); name != "" {
		itemNotificationMetadata["itemName"] = name
	}
	s.notifyTripBroadcast(ctx, current, user.ID,
		notifications.TypeItemRegenerated,
		"Item regenerated",
		fmt.Sprintf("An item on Day %d of %s was regenerated.", dayNumber, destination),
		notifications.EntityItineraryItem, nil,
		itemNotificationMetadata)

	return updated, nil
}

func (s *Service) RegenerateItemForActor(
	ctx context.Context,
	id, actorUserID uuid.UUID,
	dayNumber, itemIndex int,
	instruction string,
	expectedRevision int,
) (*entity.Trip, error) {
	ctx = contextWithActor(ctx, actorUserID)
	return s.RegenerateItem(ctx, id, dayNumber, itemIndex, appdto.RegenerateItineraryPartInput{
		Instruction:               instruction,
		ExpectedItineraryRevision: &expectedRevision,
	})
}

func validateAndNormalizeItinerary(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil, apperrs.NewInvalidInput("itinerary is required")
	}

	var itinerary editableItinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		return nil, apperrs.NewInvalidInput("invalid itinerary")
	}

	itinerary.Destination = strings.TrimSpace(itinerary.Destination)
	itinerary.Summary = strings.TrimSpace(itinerary.Summary)
	itinerary.Pace = strings.TrimSpace(itinerary.Pace)
	itinerary.Currency = strings.ToUpper(strings.TrimSpace(itinerary.Currency))
	itinerary.Source = strings.TrimSpace(itinerary.Source)
	if itinerary.TotalBudget != nil && *itinerary.TotalBudget < 0 {
		return nil, apperrs.NewInvalidInput("itinerary.totalBudget must be >= 0")
	}

	if len(itinerary.Days) == 0 {
		return nil, apperrs.NewInvalidInput("itinerary.days must contain at least 1 day")
	}
	if len(itinerary.Days) > maxItineraryDays {
		return nil, apperrs.NewInvalidInput("itinerary.days must contain at most %d days", maxItineraryDays)
	}

	for dayIndex := range itinerary.Days {
		day := &itinerary.Days[dayIndex]
		if day.Day < 1 {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].day must be >= 1", dayIndex)
		}
		day.Title = strings.TrimSpace(day.Title)
		if day.Title == "" {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].title is required", dayIndex)
		}
		if len(day.Items) == 0 {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].items must contain at least 1 item", dayIndex)
		}
		if len(day.Items) > maxItineraryItemsPerDay {
			return nil, apperrs.NewInvalidInput("itinerary.days[%d].items must contain at most %d items", dayIndex, maxItineraryItemsPerDay)
		}

		for itemIndex := range day.Items {
			item := &day.Items[itemIndex]
			item.Time = strings.TrimSpace(item.Time)
			if item.Time == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].time is required", dayIndex, itemIndex)
			}
			item.Type = strings.TrimSpace(item.Type)
			if item.Type == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].type is required", dayIndex, itemIndex)
			}
			item.Name = strings.TrimSpace(item.Name)
			if item.Name == "" {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].name is required", dayIndex, itemIndex)
			}
			item.Note = strings.TrimSpace(item.Note)
			if err := budget.NormalizeEstimatedCost(item.EstimatedCost, budget.SourceManual); err != nil {
				return nil, apperrs.NewInvalidInput("itinerary.days[%d].items[%d].estimatedCost: %s", dayIndex, itemIndex, err.Error())
			}
			if err := validateAndNormalizePlaceRef(item.Place, "itinerary.days[%d].items[%d].place", dayIndex, itemIndex); err != nil {
				return nil, err
			}
			if err := validateAndNormalizePlaceEnrichment(item.PlaceEnrichment, "itinerary.days[%d].items[%d].placeEnrichment", dayIndex, itemIndex); err != nil {
				return nil, err
			}
			if err := validateAndNormalizePriceEnrichment(item.PriceEnrichment, "itinerary.days[%d].items[%d].priceEnrichment", dayIndex, itemIndex); err != nil {
				return nil, err
			}
			if err := validateAndNormalizeAvailabilityCheck(item.AvailabilityCheck, "itinerary.days[%d].items[%d].availabilityCheck", dayIndex, itemIndex); err != nil {
				return nil, err
			}
		}
	}

	normalized, err := json.Marshal(itinerary)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizeRegenerationInstruction(raw string) (string, error) {
	instruction := strings.TrimSpace(raw)
	if len(instruction) > maxInstructionLength {
		return "", apperrs.NewInvalidInput("instruction must be at most %d characters", maxInstructionLength)
	}
	return instruction, nil
}

func currentItineraryAndDayIndex(t *entity.Trip, dayNumber int) (aggregate.Itinerary, int, error) {
	if t == nil || len(t.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(t.Itinerary)), "null") {
		return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
	}

	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(t.Itinerary, &itinerary); err != nil {
		return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
	}
	if err := validateCurrentItinerary(itinerary); err != nil {
		return aggregate.Itinerary{}, -1, err
	}

	for index := range itinerary.Days {
		if itinerary.Days[index].Day == dayNumber {
			return itinerary, index, nil
		}
	}

	return aggregate.Itinerary{}, -1, currentItineraryInvalidError()
}

func validateCurrentItinerary(itinerary aggregate.Itinerary) error {
	if len(itinerary.Days) == 0 || len(itinerary.Days) > maxItineraryDays {
		return currentItineraryInvalidError()
	}

	seenDays := make(map[int]struct{}, len(itinerary.Days))
	for _, day := range itinerary.Days {
		if day.Day < 1 {
			return currentItineraryInvalidError()
		}
		if _, exists := seenDays[day.Day]; exists {
			return currentItineraryInvalidError()
		}
		seenDays[day.Day] = struct{}{}

		if strings.TrimSpace(day.Title) == "" {
			return currentItineraryInvalidError()
		}
		if len(day.Items) == 0 || len(day.Items) > maxItineraryItemsPerDay {
			return currentItineraryInvalidError()
		}
		for _, item := range day.Items {
			if strings.TrimSpace(item.Time) == "" ||
				strings.TrimSpace(item.Type) == "" ||
				strings.TrimSpace(item.Name) == "" {
				return currentItineraryInvalidError()
			}
			if item.EstimatedCost != nil && item.EstimatedCost.Amount != nil && *item.EstimatedCost.Amount < 0 {
				return currentItineraryInvalidError()
			}
			if err := validateAndNormalizePlaceRef(item.Place, "place"); err != nil {
				return currentItineraryInvalidError()
			}
			if err := validateAndNormalizePlaceEnrichment(item.PlaceEnrichment, "placeEnrichment"); err != nil {
				return currentItineraryInvalidError()
			}
			if err := validateAndNormalizePriceEnrichment(item.PriceEnrichment, "priceEnrichment"); err != nil {
				return currentItineraryInvalidError()
			}
		}
	}

	return nil
}

func normalizeReplacementDay(day *aggregate.ItineraryDay, dayNumber int) (aggregate.ItineraryDay, error) {
	if day == nil {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day is required")
	}

	normalized := *day
	normalized.Day = dayNumber
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day title is required")
	}
	if len(normalized.Items) == 0 || len(normalized.Items) > maxItineraryItemsPerDay {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("replacement day item count is invalid")
	}
	for index := range normalized.Items {
		item, err := normalizeReplacementItem(&normalized.Items[index])
		if err != nil {
			return aggregate.ItineraryDay{}, err
		}
		normalized.Items[index] = item
	}

	return normalized, nil
}

func normalizeReplacementItem(item *aggregate.ItineraryItem) (aggregate.ItineraryItem, error) {
	if item == nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item is required")
	}

	normalized := *item
	normalized.Time = strings.TrimSpace(normalized.Time)
	normalized.Type = strings.TrimSpace(normalized.Type)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Note = strings.TrimSpace(normalized.Note)
	if normalized.Time == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item time is required")
	}
	if normalized.Type == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item type is required")
	}
	if normalized.Name == "" {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item name is required")
	}
	if err := budget.NormalizeEstimatedCost(normalized.EstimatedCost, budget.SourceAI); err != nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item estimatedCost is invalid: %s", err.Error())
	}
	if err := validateAndNormalizePlaceRef(normalized.Place, "replacement item place"); err != nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item place is invalid")
	}
	if err := validateAndNormalizePlaceEnrichment(normalized.PlaceEnrichment, "replacement item placeEnrichment"); err != nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item placeEnrichment is invalid")
	}
	if err := validateAndNormalizePriceEnrichment(normalized.PriceEnrichment, "replacement item priceEnrichment"); err != nil {
		return aggregate.ItineraryItem{}, apperrs.NewDependencyError("replacement item priceEnrichment is invalid")
	}

	return normalized, nil
}

func validateAndNormalizePlaceRef(place *aggregate.PlaceRef, path string, args ...any) error {
	if place == nil {
		return nil
	}

	label := path
	if len(args) > 0 {
		label = fmt.Sprintf(path, args...)
	}

	place.Provider = strings.TrimSpace(place.Provider)
	if place.Provider == "" {
		return apperrs.NewInvalidInput("%s.provider is required", label)
	}

	place.ProviderPlaceID = strings.TrimSpace(place.ProviderPlaceID)
	if place.ProviderPlaceID == "" {
		return apperrs.NewInvalidInput("%s.providerPlaceId is required", label)
	}

	place.Name = strings.TrimSpace(place.Name)
	if place.Name == "" {
		return apperrs.NewInvalidInput("%s.name is required", label)
	}

	place.Address = strings.TrimSpace(place.Address)
	if place.Address == "" {
		return apperrs.NewInvalidInput("%s.address is required", label)
	}

	if place.Latitude != nil && (*place.Latitude < -90 || *place.Latitude > 90) {
		return apperrs.NewInvalidInput("%s.latitude must be between -90 and 90", label)
	}
	if place.Longitude != nil && (*place.Longitude < -180 || *place.Longitude > 180) {
		return apperrs.NewInvalidInput("%s.longitude must be between -180 and 180", label)
	}
	if place.Rating != nil && (*place.Rating < 0 || *place.Rating > 5) {
		return apperrs.NewInvalidInput("%s.rating must be between 0 and 5", label)
	}
	if place.RatingCount != nil && *place.RatingCount < 0 {
		return apperrs.NewInvalidInput("%s.ratingCount must be >= 0", label)
	}

	place.MapURL = strings.TrimSpace(place.MapURL)
	if len(place.MapURL) > maxPlaceURLLength {
		return apperrs.NewInvalidInput("%s.mapUrl must be at most %d characters", label, maxPlaceURLLength)
	}

	place.Category = strings.TrimSpace(place.Category)
	place.Website = strings.TrimSpace(place.Website)
	if len(place.Website) > maxPlaceURLLength {
		return apperrs.NewInvalidInput("%s.website must be at most %d characters", label, maxPlaceURLLength)
	}

	if err := validateAndNormalizeOpeningHours(place, label); err != nil {
		return err
	}

	return nil
}

func validateAndNormalizeOpeningHours(place *aggregate.PlaceRef, label string) error {
	for index := range place.OpeningHours {
		interval := &place.OpeningHours[index]
		interval.Open = strings.TrimSpace(interval.Open)
		interval.Close = strings.TrimSpace(interval.Close)

		if interval.DayOfWeek < 1 || interval.DayOfWeek > 7 {
			return apperrs.NewInvalidInput("%s.openingHours[%d].dayOfWeek must be between 1 and 7", label, index)
		}

		openMinutes, ok := parseHHMM(interval.Open)
		if !ok {
			return apperrs.NewInvalidInput("%s.openingHours[%d].open must be in HH:mm format", label, index)
		}

		closeMinutes, ok := parseHHMM(interval.Close)
		if !ok {
			return apperrs.NewInvalidInput("%s.openingHours[%d].close must be in HH:mm format", label, index)
		}

		if openMinutes >= closeMinutes {
			return apperrs.NewInvalidInput("%s.openingHours[%d].open must be before close", label, index)
		}
	}

	return nil
}

func validateAndNormalizePlaceEnrichment(meta *aggregate.PlaceEnrichmentMeta, path string, args ...any) error {
	if meta == nil {
		return nil
	}

	label := path
	if len(args) > 0 {
		label = fmt.Sprintf(path, args...)
	}

	meta.Status = strings.TrimSpace(meta.Status)
	switch meta.Status {
	case placeenrichment.StatusMatched, placeenrichment.StatusNoMatch, placeenrichment.StatusSkipped, placeenrichment.StatusFailed:
	default:
		return apperrs.NewInvalidInput("%s.status must be one of matched, no_match, skipped, failed", label)
	}

	meta.ReviewStatus = strings.TrimSpace(meta.ReviewStatus)
	switch meta.ReviewStatus {
	case "", placeenrichment.ReviewStatusPending, placeenrichment.ReviewStatusAccepted, placeenrichment.ReviewStatusChanged, placeenrichment.ReviewStatusRemoved:
	default:
		return apperrs.NewInvalidInput("%s.reviewStatus must be one of pending, accepted, changed, removed", label)
	}

	if meta.Confidence < 0 || meta.Confidence > 1 {
		return apperrs.NewInvalidInput("%s.confidence must be between 0 and 1", label)
	}

	meta.Query = strings.TrimSpace(meta.Query)
	if len(meta.Query) > maxPlaceEnrichmentQuery {
		return apperrs.NewInvalidInput("%s.query must be at most %d characters", label, maxPlaceEnrichmentQuery)
	}

	meta.Provider = strings.TrimSpace(meta.Provider)
	if len(meta.Provider) > maxPlaceEnrichmentProvider {
		return apperrs.NewInvalidInput("%s.provider must be at most %d characters", label, maxPlaceEnrichmentProvider)
	}

	meta.MatchedAt = strings.TrimSpace(meta.MatchedAt)
	meta.Reason = strings.TrimSpace(meta.Reason)
	if len(meta.Reason) > maxPlaceEnrichmentReason {
		return apperrs.NewInvalidInput("%s.reason must be at most %d characters", label, maxPlaceEnrichmentReason)
	}

	return nil
}

func validateAndNormalizePriceEnrichment(meta *aggregate.PriceEnrichmentMeta, path string, args ...any) error {
	if meta == nil {
		return nil
	}

	label := path
	if len(args) > 0 {
		label = fmt.Sprintf(path, args...)
	}

	meta.Status = strings.TrimSpace(meta.Status)
	switch meta.Status {
	case priceenrichment.StatusMatched, priceenrichment.StatusNoMatch, priceenrichment.StatusSkipped, priceenrichment.StatusFailed:
	default:
		return apperrs.NewInvalidInput("%s.status must be one of matched, no_match, skipped, failed", label)
	}

	meta.ReviewStatus = strings.TrimSpace(meta.ReviewStatus)
	switch meta.ReviewStatus {
	case "", priceenrichment.ReviewStatusPending, priceenrichment.ReviewStatusAccepted, priceenrichment.ReviewStatusChanged, priceenrichment.ReviewStatusRemoved:
	default:
		return apperrs.NewInvalidInput("%s.reviewStatus must be one of pending, accepted, changed, removed", label)
	}
	if meta.MatchConfidence < 0 || meta.MatchConfidence > 1 {
		return apperrs.NewInvalidInput("%s.matchConfidence must be between 0 and 1", label)
	}
	meta.Provider = strings.TrimSpace(meta.Provider)
	if len(meta.Provider) > maxPriceEnrichmentProvider {
		return apperrs.NewInvalidInput("%s.provider must be at most %d characters", label, maxPriceEnrichmentProvider)
	}
	meta.PriceType = strings.TrimSpace(meta.PriceType)
	meta.UpdatedAt = strings.TrimSpace(meta.UpdatedAt)
	meta.Reason = strings.TrimSpace(meta.Reason)
	if len(meta.Reason) > maxPriceEnrichmentReason {
		return apperrs.NewInvalidInput("%s.reason must be at most %d characters", label, maxPriceEnrichmentReason)
	}
	return nil
}

// availabilityCheckStatuses are the availability statuses accepted from the web
// when a user applies a provider availability result to an item.
const maxAvailabilityCheckProvider = 40

func validateAndNormalizeAvailabilityCheck(meta *aggregate.AvailabilityCheckMeta, path string, args ...any) error {
	if meta == nil {
		return nil
	}

	label := path
	if len(args) > 0 {
		label = fmt.Sprintf(path, args...)
	}

	meta.Provider = strings.TrimSpace(meta.Provider)
	if len(meta.Provider) > maxAvailabilityCheckProvider {
		return apperrs.NewInvalidInput("%s.provider must be at most %d characters", label, maxAvailabilityCheckProvider)
	}
	meta.Status = strings.TrimSpace(strings.ToLower(meta.Status))
	switch meta.Status {
	case "", "available", "limited", "unavailable", "unknown":
	default:
		return apperrs.NewInvalidInput("%s.status must be one of available, limited, unavailable, unknown", label)
	}
	if meta.MatchConfidence < 0 || meta.MatchConfidence > 1 {
		return apperrs.NewInvalidInput("%s.matchConfidence must be between 0 and 1", label)
	}
	meta.CheckedAt = strings.TrimSpace(meta.CheckedAt)
	meta.SelectedOptionID = strings.TrimSpace(meta.SelectedOptionID)
	if len(meta.SelectedOptionID) > 200 {
		return apperrs.NewInvalidInput("%s.selectedOptionId must be at most 200 characters", label)
	}
	return nil
}

func parseHHMM(value string) (int, bool) {
	if len(value) != len("15:04") || value[2] != ':' {
		return 0, false
	}
	if !asciiDigit(value[0]) || !asciiDigit(value[1]) ||
		!asciiDigit(value[3]) || !asciiDigit(value[4]) {
		return 0, false
	}

	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func asciiDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func (s *Service) saveRegeneratedItinerary(
	ctx context.Context,
	tripID, ownerUserID, actorUserID uuid.UUID,
	itinerary aggregate.Itinerary,
	expectedItineraryRevision int,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, error) {
	raw, err := json.Marshal(itinerary)
	if err != nil {
		return nil, err
	}
	return s.saveItineraryWithVersion(
		ctx,
		tripID,
		ownerUserID,
		actorUserID,
		raw,
		expectedItineraryRevision,
		source,
		metadata,
	)
}

func (s *Service) saveItineraryWithVersion(
	ctx context.Context,
	tripID, ownerUserID, actorUserID uuid.UUID,
	itinerary json.RawMessage,
	expectedItineraryRevision int,
	source entity.ItineraryVersionSource,
	metadata map[string]any,
) (*entity.Trip, error) {
	updated, _, err := s.repo.UpdateItineraryAndCreateVersion(
		ctx,
		tripID,
		ownerUserID,
		actorUserID,
		itinerary,
		entity.StatusCompleted,
		expectedItineraryRevision,
		source,
		metadata,
	)
	if err != nil {
		return updated, err
	}
	// Any itinerary write (manual edit, day/item regeneration, version restore,
	// generation completion) is a material change: if the workspace trip was
	// approved or pending, move it back to draft. Best-effort and post-commit —
	// this never fails the itinerary save. This is the single choke point for
	// itinerary writes, so it also covers the async generation-job worker path.
	s.ResetApprovalIfApproved(ctx, tripID, actorUserID, "Itinerary changed ("+string(source)+")")
	return updated, nil
}

func requireExpectedItineraryRevision(expected *int) (int, error) {
	if expected == nil {
		return 0, apperrs.ErrExpectedItineraryRevisionRequired
	}
	if *expected < 0 {
		return 0, apperrs.NewInvalidInput("expectedItineraryRevision must be >= 0")
	}
	return *expected, nil
}

func checkCurrentItineraryRevision(expected, current int) error {
	if expected != current {
		return apperrs.NewItineraryConflict(current)
	}
	return nil
}

func isItineraryConflict(err error) bool {
	var conflict *apperrs.ItineraryConflictError
	return errors.As(err, &conflict)
}

func contextWithActor(ctx context.Context, actorUserID uuid.UUID) context.Context {
	if user, ok := auth.UserFromContext(ctx); ok && user.ID == actorUserID {
		return ctx
	}
	return auth.WithUser(ctx, auth.AuthenticatedUser{ID: actorUserID})
}

func tripOwnerID(t *entity.Trip) (uuid.UUID, error) {
	if t == nil || t.UserID == nil || *t.UserID == uuid.Nil {
		return uuid.Nil, domainerrs.ErrNotFound
	}
	return *t.UserID, nil
}

func currentItineraryInvalidError() error {
	return apperrs.NewInvalidInput("current itinerary is invalid")
}

func regenerateLogFields(action string, tripID, userID uuid.UUID, dayNumber int, itemIndex *int, instruction string) []zap.Field {
	fields := []zap.Field{
		zap.String("action", action),
		zap.String("tripId", tripID.String()),
		zap.String("userId", userID.String()),
		zap.Int("dayNumber", dayNumber),
		zap.Bool("instructionPresent", instruction != ""),
	}
	if itemIndex != nil {
		fields = append(fields, zap.Int("itemIndex", *itemIndex))
	}
	return fields
}

func (s *Service) logRegenerationSuccess(message string, fields []zap.Field, started time.Time, userContextLoaded bool) {
	s.log.Info(message,
		append(fields,
			zap.Bool("userContextLoaded", userContextLoaded),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
			zap.Bool("success", true),
		)...,
	)
}

func (s *Service) logRegenerationFailure(message string, fields []zap.Field, started time.Time, userContextLoaded bool, err error) {
	s.log.Warn(message,
		append(fields,
			zap.Bool("userContextLoaded", userContextLoaded),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
			zap.Bool("success", false),
			zap.Error(err),
		)...,
	)
}

func (s *Service) loadUserContext(ctx context.Context, user auth.AuthenticatedUser, tripID uuid.UUID) (usercontext.UserContext, error) {
	fields := []zap.Field{
		zap.Bool("userContextEnabled", s.userContextEnabled),
		zap.Bool("userContextFailOpen", s.userContextFailOpen),
		zap.String("userId", user.ID.String()),
		zap.String("tripId", tripID.String()),
	}

	if !s.userContextEnabled {
		s.log.Info("user context disabled",
			append(fields,
				zap.Bool("userContextLoaded", false),
				zap.String("userContextErrorType", ""),
			)...,
		)
		return usercontext.UserContext{}, nil
	}

	if s.userContextProvider == nil {
		err := userContextError(usercontext.ErrorTypeService, "user context provider is not configured")
		return s.handleUserContextError(err, fields)
	}

	accessToken, ok := auth.AccessTokenFromContext(ctx)
	if !ok {
		err := userContextError(usercontext.ErrorTypeAuth, "access token missing from request context")
		return s.handleUserContextError(err, fields)
	}

	loaded, err := s.userContextProvider.GetUserContext(ctx, accessToken)
	if err != nil {
		return s.handleUserContextError(err, fields)
	}
	if loaded == nil {
		loaded = &usercontext.UserContext{}
	}

	contextLoaded := loaded.Profile != nil || loaded.Preferences != nil
	s.log.Info("user context loaded",
		append(fields,
			zap.Bool("userContextLoaded", contextLoaded),
			zap.String("userContextErrorType", ""),
		)...,
	)

	return *loaded, nil
}

func (s *Service) handleUserContextError(err error, fields []zap.Field) (usercontext.UserContext, error) {
	errorType := classifyUserContextError(err)
	logFields := append(fields,
		zap.Bool("userContextLoaded", false),
		zap.String("userContextErrorType", errorType),
		zap.Error(err),
	)

	if s.userContextFailOpen {
		s.log.Warn("failed to load user context; continuing without personalization", logFields...)
		return usercontext.UserContext{}, nil
	}

	s.log.Warn("failed to load user context; generation blocked", logFields...)
	if limitErr, ok := providerlimit.As(err); ok {
		return usercontext.UserContext{}, limitErr
	}
	return usercontext.UserContext{}, apperrs.NewDependencyError("failed to load user preferences")
}

func classifyUserContextError(err error) string {
	var userContextErr *usercontext.Error
	if err != nil && errors.As(err, &userContextErr) {
		return string(userContextErr.Type)
	}
	return string(usercontext.ErrorTypeService)
}

func userContextError(errorType usercontext.ErrorType, message string) error {
	return &usercontext.Error{Type: errorType, Message: message}
}

func (s *Service) loadWeatherContext(ctx context.Context, trip entity.Trip, tripID uuid.UUID) (*weathercontext.WeatherForecast, error) {
	fields := []zap.Field{
		zap.Bool("weatherContextEnabled", s.weatherContextEnabled),
		zap.Bool("weatherContextFailOpen", s.weatherContextFailOpen),
		zap.String("tripId", tripID.String()),
		zap.String("destination", trip.Destination),
		zap.Int32("days", trip.Days),
	}

	if !s.weatherContextEnabled {
		s.log.Info("weather context disabled",
			append(fields,
				zap.Bool("weatherContextLoaded", false),
			)...,
		)
		return nil, nil
	}
	if trip.StartDate == nil {
		s.log.Debug("weather context skipped: missing trip start date",
			append(fields,
				zap.Bool("weatherContextLoaded", false),
			)...,
		)
		return nil, nil
	}
	if strings.TrimSpace(trip.Destination) == "" || trip.Days < 1 {
		s.log.Debug("weather context skipped: incomplete trip fields",
			append(fields,
				zap.Bool("weatherContextLoaded", false),
			)...,
		)
		return nil, nil
	}
	if s.weatherContextProvider == nil {
		return s.handleWeatherContextError(errors.New("weather context provider is not configured"), fields)
	}

	startDate := trip.StartDate.Format("2006-01-02")
	forecast, err := s.weatherContextProvider.GetForecast(ctx, trip.Destination, startDate, int(trip.Days))
	if err != nil {
		return s.handleWeatherContextError(err, fields)
	}

	loaded := forecast != nil && len(forecast.Days) > 0
	s.log.Info("weather context loaded",
		append(fields,
			zap.Bool("weatherContextLoaded", loaded),
		)...,
	)
	return forecast, nil
}

func (s *Service) handleWeatherContextError(err error, fields []zap.Field) (*weathercontext.WeatherForecast, error) {
	logFields := append(fields,
		zap.Bool("weatherContextLoaded", false),
		zap.Error(err),
	)

	if s.weatherContextFailOpen {
		s.log.Warn("failed to load weather context; continuing without weather", logFields...)
		return nil, nil
	}

	s.log.Warn("failed to load weather context; generation blocked", logFields...)
	if limitErr, ok := providerlimit.As(err); ok {
		return nil, limitErr
	}
	return nil, apperrs.NewDependencyError("failed to load weather forecast")
}

func (s *Service) enrichGeneratedItinerary(ctx context.Context, tripID uuid.UUID, trip entity.Trip, itinerary *aggregate.Itinerary, userContext usercontext.UserContext) (*aggregate.Itinerary, error) {
	if itinerary == nil {
		return itinerary, nil
	}
	normalizeGeneratedCosts(itinerary)
	result, err := s.enrichItinerary(ctx, tripID, trip, *itinerary, "generate")
	if err != nil {
		return nil, err
	}
	return s.enrichItineraryPrices(ctx, tripID, trip, *result, userContext, "generate")
}

// normalizeGeneratedCosts repairs AI-generated item cost estimates in place.
// Generated output defaults source to "ai"; an estimate that cannot be repaired
// (negative amount or malformed currency) is dropped rather than failing the
// whole generation.
func normalizeGeneratedCosts(itinerary *aggregate.Itinerary) {
	if itinerary == nil {
		return
	}
	for dayIndex := range itinerary.Days {
		items := itinerary.Days[dayIndex].Items
		for itemIndex := range items {
			cost := items[itemIndex].EstimatedCost
			if cost == nil {
				continue
			}
			if err := budget.NormalizeEstimatedCost(cost, budget.SourceAI); err != nil {
				items[itemIndex].EstimatedCost = nil
			}
		}
	}
}

func (s *Service) enrichReplacementDay(ctx context.Context, tripID uuid.UUID, trip entity.Trip, day aggregate.ItineraryDay, userContext usercontext.UserContext) (aggregate.ItineraryDay, error) {
	itinerary := aggregate.Itinerary{
		Destination: trip.Destination,
		Days:        []aggregate.ItineraryDay{day},
	}
	enriched, err := s.enrichItinerary(ctx, tripID, trip, itinerary, "regenerate_day")
	if err != nil {
		return aggregate.ItineraryDay{}, err
	}
	enriched, err = s.enrichItineraryPrices(ctx, tripID, trip, *enriched, userContext, "regenerate_day")
	if err != nil {
		return aggregate.ItineraryDay{}, err
	}
	if enriched == nil || len(enriched.Days) != 1 {
		return day, nil
	}
	return enriched.Days[0], nil
}

func (s *Service) enrichReplacementItem(ctx context.Context, tripID uuid.UUID, trip entity.Trip, dayNumber int, item aggregate.ItineraryItem, userContext usercontext.UserContext) (aggregate.ItineraryItem, error) {
	itinerary := aggregate.Itinerary{
		Destination: trip.Destination,
		Days: []aggregate.ItineraryDay{{
			Day:   dayNumber,
			Title: "Replacement day",
			Items: []aggregate.ItineraryItem{item},
		}},
	}
	enriched, err := s.enrichItinerary(ctx, tripID, trip, itinerary, "regenerate_item")
	if err != nil {
		return aggregate.ItineraryItem{}, err
	}
	enriched, err = s.enrichItineraryPrices(ctx, tripID, trip, *enriched, userContext, "regenerate_item")
	if err != nil {
		return aggregate.ItineraryItem{}, err
	}
	if enriched == nil || len(enriched.Days) != 1 || len(enriched.Days[0].Items) != 1 {
		return item, nil
	}
	return enriched.Days[0].Items[0], nil
}

func (s *Service) enrichItinerary(ctx context.Context, tripID uuid.UUID, trip entity.Trip, itinerary aggregate.Itinerary, source string) (*aggregate.Itinerary, error) {
	started := time.Now()
	fields := []zap.Field{
		zap.String("action", "place_enrichment"),
		zap.String("tripId", tripID.String()),
		zap.String("destination", trip.Destination),
		zap.String("source", source),
		zap.Bool("enabled", s.placeEnrichmentEnabled),
		zap.Bool("failOpen", s.placeEnrichmentFailOpen),
	}

	if !s.placeEnrichmentEnabled {
		s.log.Info("place enrichment disabled",
			append(fields,
				zap.Int64("durationMs", time.Since(started).Milliseconds()),
			)...,
		)
		return &itinerary, nil
	}
	if s.placeEnrichmentProvider == nil {
		err := errors.New("place enrichment provider is not configured")
		return s.handlePlaceEnrichmentError(err, fields, started, itinerary)
	}

	result, err := s.placeEnrichmentProvider.EnrichItinerary(ctx, placeenrichment.EnrichItineraryInput{
		Destination: trip.Destination,
		Itinerary:   itinerary,
	})
	if err != nil {
		return s.handlePlaceEnrichmentError(err, fields, started, itinerary)
	}
	if result == nil {
		err := errors.New("place enrichment returned no result")
		return s.handlePlaceEnrichmentError(err, fields, started, itinerary)
	}

	s.log.Info("place enrichment completed",
		append(fields,
			zap.Int("attempted", result.Stats.Attempted),
			zap.Int("matched", result.Stats.Matched),
			zap.Int("noMatch", result.Stats.NoMatch),
			zap.Int("skipped", result.Stats.Skipped),
			zap.Int("failed", result.Stats.Failed),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
		)...,
	)
	return &result.Itinerary, nil
}

func (s *Service) handlePlaceEnrichmentError(err error, fields []zap.Field, started time.Time, original aggregate.Itinerary) (*aggregate.Itinerary, error) {
	logFields := append(fields,
		zap.Int64("durationMs", time.Since(started).Milliseconds()),
		zap.Error(err),
	)
	if s.placeEnrichmentFailOpen {
		s.log.Warn("failed to enrich itinerary places; continuing without enrichment", logFields...)
		return &original, nil
	}

	s.log.Warn("failed to enrich itinerary places; generation blocked", logFields...)
	if limitErr, ok := providerlimit.As(err); ok {
		return nil, limitErr
	}
	return nil, apperrs.NewDependencyError("failed to enrich itinerary places")
}

func (s *Service) enrichItineraryPrices(ctx context.Context, tripID uuid.UUID, trip entity.Trip, itinerary aggregate.Itinerary, userContext usercontext.UserContext, source string) (*aggregate.Itinerary, error) {
	started := time.Now()
	fields := []zap.Field{
		zap.String("action", "price_enrichment"),
		zap.String("tripId", tripID.String()),
		zap.String("destination", trip.Destination),
		zap.String("source", source),
		zap.Bool("enabled", s.priceEnrichmentEnabled),
		zap.Bool("failOpen", s.priceEnrichmentFailOpen),
	}

	if !s.priceEnrichmentEnabled {
		s.log.Info("price enrichment disabled",
			append(fields,
				zap.Int64("durationMs", time.Since(started).Milliseconds()),
			)...,
		)
		return &itinerary, nil
	}
	if s.priceEnrichmentProvider == nil {
		err := errors.New("price enrichment provider is not configured")
		return s.handlePriceEnrichmentError(err, fields, started, itinerary)
	}

	preferredCurrency := ""
	if userContext.Profile != nil {
		preferredCurrency = userContext.Profile.PreferredCurrency
	}
	result, err := s.priceEnrichmentProvider.EnrichItinerary(ctx, priceenrichment.EnrichItineraryInput{
		Destination:           trip.Destination,
		BudgetCurrency:        trip.BudgetCurrency,
		UserPreferredCurrency: preferredCurrency,
		StartDate:             trip.StartDate,
		Itinerary:             itinerary,
	})
	if err != nil {
		return s.handlePriceEnrichmentError(err, fields, started, itinerary)
	}
	if result == nil {
		err := errors.New("price enrichment returned no result")
		return s.handlePriceEnrichmentError(err, fields, started, itinerary)
	}

	s.log.Info("price enrichment completed",
		append(fields,
			zap.Int("candidates", result.Stats.Candidates),
			zap.Int("matched", result.Stats.Matched),
			zap.Int("noMatch", result.Stats.NoMatch),
			zap.Int("skipped", result.Stats.Skipped),
			zap.Int("failed", result.Stats.Failed),
			zap.Int("overwritten", result.Stats.Overwritten),
			zap.Int("notOverwrittenExistingCost", result.Stats.NotOverwrittenExistingCost),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
		)...,
	)
	return &result.Itinerary, nil
}

func (s *Service) handlePriceEnrichmentError(err error, fields []zap.Field, started time.Time, original aggregate.Itinerary) (*aggregate.Itinerary, error) {
	logFields := append(fields,
		zap.Int64("durationMs", time.Since(started).Milliseconds()),
		zap.Error(err),
	)
	if s.priceEnrichmentFailOpen {
		s.log.Warn("failed to enrich itinerary prices; continuing without price enrichment", logFields...)
		return &original, nil
	}

	s.log.Warn("failed to enrich itinerary prices; generation blocked", logFields...)
	if limitErr, ok := providerlimit.As(err); ok {
		return nil, limitErr
	}
	return nil, apperrs.NewDependencyError("failed to enrich itinerary prices")
}

func (s *Service) markFailed(ctx context.Context, id, userID uuid.UUID) {
	if _, err := s.repo.UpdateStatusByUserID(ctx, id, userID, entity.StatusFailed); err != nil {
		s.log.Error("failed to mark trip as FAILED",
			zap.String("trip_id", id.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
	}
}
