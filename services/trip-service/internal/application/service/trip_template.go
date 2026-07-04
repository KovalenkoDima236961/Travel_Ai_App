package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	defaultTemplateLimit = 50
	maxTemplateLimit     = 100
	maxTemplateTags      = 20
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

type tripTemplateRepository interface {
	CreateTripTemplate(ctx context.Context, t *entity.TripTemplate) (*entity.TripTemplate, error)
	GetTripTemplateByID(ctx context.Context, id uuid.UUID) (*entity.TripTemplate, error)
	ListTripTemplates(ctx context.Context, userID uuid.UUID, workspaceIDs []uuid.UUID, in appdto.ListTripTemplatesInput) ([]entity.TripTemplate, error)
	UpdateTripTemplateMetadata(ctx context.Context, t *entity.TripTemplate) (*entity.TripTemplate, error)
	ArchiveTripTemplate(ctx context.Context, id, actorUserID uuid.UUID) (*entity.TripTemplate, error)
}

func (s *Service) templateRepo() (tripTemplateRepository, error) {
	repo, ok := s.repo.(tripTemplateRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip templates are not configured")
	}
	return repo, nil
}

func (s *Service) SaveTripAsTemplate(ctx context.Context, tripID uuid.UUID, in appdto.SaveTripAsTemplateInput) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}

	sourceTrip, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeSaveTemplateInput(in)
	if err != nil {
		return nil, err
	}
	if normalized.Visibility == entity.TripTemplateVisibilityWorkspace {
		if normalized.WorkspaceID == nil {
			return nil, apperrs.NewInvalidInput("workspaceId is required for workspace templates")
		}
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return nil, err
		}
	} else if normalized.WorkspaceID != nil {
		return nil, apperrs.NewInvalidInput("workspaceId must be omitted for private templates")
	}

	templateJSON, durationDays, estimatedAmount, estimatedCurrency, err := buildTemplateJSON(*sourceTrip, normalized)
	if err != nil {
		return nil, err
	}
	sourceTripID := sourceTrip.ID
	template := &entity.TripTemplate{
		ID:                     uuid.New(),
		WorkspaceID:            normalized.WorkspaceID,
		CreatedByUserID:        user.ID,
		SourceTripID:           &sourceTripID,
		Title:                  normalized.Title,
		Description:            normalized.Description,
		DestinationHint:        normalized.DestinationHint,
		DurationDays:           durationDays,
		DefaultCurrency:        normalized.DefaultCurrency,
		Visibility:             normalized.Visibility,
		TemplateJSON:           templateJSON,
		Tags:                   normalized.Tags,
		EstimatedTotalAmount:   estimatedAmount,
		EstimatedTotalCurrency: estimatedCurrency,
		Status:                 entity.TripTemplateStatusActive,
	}

	created, err := repo.CreateTripTemplate(ctx, template)
	if err != nil {
		return nil, err
	}
	access, _ := s.templateAccess(ctx, created, user.ID)
	s.log.Info("trip template created",
		zap.String("template_id", created.ID.String()),
		zap.String("trip_id", tripID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("visibility", string(created.Visibility)),
	)
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTemplateCreated,
		EntityType:  activityEntityType(activity.EntityTripTemplate),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"templateId":    created.ID.String(),
			"templateTitle": created.Title,
			"visibility":    string(created.Visibility),
		},
	})

	return &appdto.TripTemplateWithAccess{Template: *created, Access: access}, nil
}

func (s *Service) ListTripTemplates(ctx context.Context, in appdto.ListTripTemplatesInput) ([]appdto.TripTemplateWithAccess, int, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, 0, 0, err
	}
	normalized, err := normalizeListTemplatesInput(in)
	if err != nil {
		return nil, 0, 0, err
	}
	if normalized.WorkspaceID != nil {
		if _, err := s.workspaceRole(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return nil, 0, 0, err
		}
	}

	workspaceIDs, err := s.accessibleWorkspaceIDs(ctx, user.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	templates, err := repo.ListTripTemplates(ctx, user.ID, workspaceIDs, normalized)
	if err != nil {
		return nil, 0, 0, err
	}

	items := make([]appdto.TripTemplateWithAccess, 0, len(templates))
	accessCache := make(map[uuid.UUID]appdto.TripTemplateAccess)
	for i := range templates {
		access, err := s.templateAccessCached(ctx, &templates[i], user.ID, accessCache)
		if err != nil {
			return nil, 0, 0, err
		}
		items = append(items, appdto.TripTemplateWithAccess{Template: templates[i], Access: access})
	}
	return items, normalized.Limit, normalized.Offset, nil
}

func (s *Service) GetTripTemplate(ctx context.Context, templateID uuid.UUID) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}
	template, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	access, err := s.templateAccess(ctx, template, user.ID)
	if err != nil {
		return nil, err
	}
	return &appdto.TripTemplateWithAccess{Template: *template, Access: access}, nil
}

func (s *Service) UpdateTripTemplateMetadata(ctx context.Context, templateID uuid.UUID, in appdto.UpdateTripTemplateInput) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}
	current, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	access, err := s.templateAccess(ctx, current, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanEdit {
		return nil, apperrs.ErrForbidden
	}
	updatedEntity, err := applyTemplateMetadataUpdate(*current, in)
	if err != nil {
		return nil, err
	}
	updated, err := repo.UpdateTripTemplateMetadata(ctx, &updatedEntity)
	if err != nil {
		return nil, err
	}
	updatedAccess, _ := s.templateAccess(ctx, updated, user.ID)
	return &appdto.TripTemplateWithAccess{Template: *updated, Access: updatedAccess}, nil
}

func (s *Service) ArchiveTripTemplate(ctx context.Context, templateID uuid.UUID) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}
	current, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	access, err := s.templateAccess(ctx, current, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanArchive {
		return nil, apperrs.ErrForbidden
	}
	archived, err := repo.ArchiveTripTemplate(ctx, templateID, user.ID)
	if err != nil {
		return nil, err
	}
	if archived.SourceTripID != nil {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      *archived.SourceTripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventTemplateArchived,
			EntityType:  activityEntityType(activity.EntityTripTemplate),
			EntityID:    activityEntityID(archived.ID),
			Metadata: map[string]any{
				"templateId":    archived.ID.String(),
				"templateTitle": archived.Title,
				"visibility":    string(archived.Visibility),
			},
		})
	}
	updatedAccess, _ := s.templateAccess(ctx, archived, user.ID)
	return &appdto.TripTemplateWithAccess{Template: *archived, Access: updatedAccess}, nil
}

func (s *Service) DuplicateTripTemplate(ctx context.Context, templateID uuid.UUID, in appdto.DuplicateTripTemplateInput) (*appdto.TripTemplateWithAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}
	source, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	access, err := s.templateAccess(ctx, source, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanDuplicate {
		return nil, apperrs.ErrForbidden
	}
	normalized, err := normalizeDuplicateTemplateInput(in, source.Title)
	if err != nil {
		return nil, err
	}
	if normalized.Visibility == entity.TripTemplateVisibilityWorkspace {
		if normalized.WorkspaceID == nil {
			return nil, apperrs.NewInvalidInput("workspaceId is required for workspace templates")
		}
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return nil, err
		}
	}

	duplicate := *source
	duplicate.ID = uuid.New()
	duplicate.WorkspaceID = normalized.WorkspaceID
	duplicate.CreatedByUserID = user.ID
	duplicate.Title = normalized.Title
	duplicate.Visibility = normalized.Visibility
	duplicate.Status = entity.TripTemplateStatusActive
	duplicate.ArchivedAt = nil
	duplicate.ArchivedByUserID = nil

	created, err := repo.CreateTripTemplate(ctx, &duplicate)
	if err != nil {
		return nil, err
	}
	createdAccess, _ := s.templateAccess(ctx, created, user.ID)
	return &appdto.TripTemplateWithAccess{Template: *created, Access: createdAccess}, nil
}

func (s *Service) CreateTripFromTemplate(ctx context.Context, templateID uuid.UUID, in appdto.CreateTripFromTemplateInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, err
	}
	template, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	access, err := s.templateAccess(ctx, template, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanUse {
		return nil, apperrs.ErrForbidden
	}
	if template.Status != entity.TripTemplateStatusActive {
		return nil, apperrs.NewInvalidInput("archived templates cannot be used")
	}

	normalized, err := normalizeCreateTripFromTemplateInput(in, template)
	if err != nil {
		return nil, err
	}
	if normalized.WorkspaceID != nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return nil, err
		}
	}
	itinerary, err := instantiateTemplateItinerary(template, normalized, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(itinerary)
	if err != nil {
		return nil, fmt.Errorf("marshal template itinerary: %w", err)
	}
	normalizedRaw, err := validateAndNormalizeItinerary(raw)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, &entity.Trip{
		UserID:         &user.ID,
		WorkspaceID:    normalized.WorkspaceID,
		Destination:    normalized.Destination,
		StartDate:      parseTemplateDate(normalized.StartDate),
		Days:           template.DurationDays,
		BudgetAmount:   normalized.BudgetAmount,
		BudgetCurrency: normalized.BudgetCurrency,
		Travelers:      *normalized.Travelers,
		Interests:      []string{},
		Pace:           normalized.Pace,
		Status:         entity.StatusDraft,
	})
	if err != nil {
		return nil, err
	}
	updated, err := s.saveItineraryWithVersion(
		ctx,
		created.ID,
		user.ID,
		user.ID,
		normalizedRaw,
		created.ItineraryRevision,
		entity.ItineraryVersionSourceCreatedFromTemplate,
		map[string]any{
			"source":         "created_from_template",
			"templateId":     template.ID.String(),
			"templateTitle":  template.Title,
			"requestedTitle": normalized.Title,
		},
	)
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      updated.ID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripCreatedFromTemplate,
		EntityType:  activityEntityType(activity.EntityTripTemplate),
		EntityID:    activityEntityID(template.ID),
		Metadata: map[string]any{
			"templateId":     template.ID.String(),
			"templateTitle":  template.Title,
			"requestedTitle": normalized.Title,
		},
	})
	return updated, nil
}

func normalizeListTemplatesInput(in appdto.ListTripTemplatesInput) (appdto.ListTripTemplatesInput, error) {
	limit := in.Limit
	if limit == 0 {
		limit = defaultTemplateLimit
	}
	if limit < 1 || limit > maxTemplateLimit {
		return appdto.ListTripTemplatesInput{}, apperrs.NewInvalidInput("limit must be between 1 and %d", maxTemplateLimit)
	}
	if in.Offset < 0 {
		return appdto.ListTripTemplatesInput{}, apperrs.NewInvalidInput("offset must be >= 0")
	}
	status := in.Status
	if status == "" {
		status = entity.TripTemplateStatusActive
	}
	if status != entity.TripTemplateStatusActive && status != entity.TripTemplateStatusArchived {
		return appdto.ListTripTemplatesInput{}, apperrs.NewInvalidInput("invalid status")
	}
	switch in.Visibility {
	case "", entity.TripTemplateVisibilityPrivate, entity.TripTemplateVisibilityWorkspace:
	default:
		return appdto.ListTripTemplatesInput{}, apperrs.NewInvalidInput("invalid visibility")
	}
	tag := normalizeTemplateTag(in.Tag)
	if in.Tag != "" && tag == "" {
		return appdto.ListTripTemplatesInput{}, apperrs.NewInvalidInput("invalid tag")
	}
	in.Limit = limit
	in.Status = status
	in.Tag = tag
	in.Query = strings.TrimSpace(in.Query)
	return in, nil
}

func normalizeSaveTemplateInput(in appdto.SaveTripAsTemplateInput) (appdto.SaveTripAsTemplateInput, error) {
	title, err := validateTemplateTitle(in.Title)
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	description, err := normalizeOptionalText(in.Description, 1000, "description")
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	destinationHint, err := normalizeOptionalText(in.DestinationHint, 200, "destinationHint")
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	currency, err := normalizeOptionalCurrency(in.DefaultCurrency, "defaultCurrency")
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	tags, err := normalizeTemplateTags(in.Tags)
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	if in.Visibility != entity.TripTemplateVisibilityPrivate && in.Visibility != entity.TripTemplateVisibilityWorkspace {
		return appdto.SaveTripAsTemplateInput{}, apperrs.NewInvalidInput("visibility must be private or workspace")
	}
	in.Title = title
	in.Description = description
	in.DestinationHint = destinationHint
	in.DefaultCurrency = currency
	in.Tags = tags
	return in, nil
}

func normalizeDuplicateTemplateInput(in appdto.DuplicateTripTemplateInput, sourceTitle string) (appdto.DuplicateTripTemplateInput, error) {
	if strings.TrimSpace(in.Title) == "" {
		in.Title = "Copy of " + sourceTitle
	}
	title, err := validateTemplateTitle(in.Title)
	if err != nil {
		return appdto.DuplicateTripTemplateInput{}, err
	}
	if in.Visibility == "" {
		in.Visibility = entity.TripTemplateVisibilityPrivate
	}
	if in.Visibility != entity.TripTemplateVisibilityPrivate && in.Visibility != entity.TripTemplateVisibilityWorkspace {
		return appdto.DuplicateTripTemplateInput{}, apperrs.NewInvalidInput("visibility must be private or workspace")
	}
	if in.Visibility == entity.TripTemplateVisibilityPrivate {
		in.WorkspaceID = nil
	}
	in.Title = title
	return in, nil
}

func normalizeCreateTripFromTemplateInput(in appdto.CreateTripFromTemplateInput, template *entity.TripTemplate) (appdto.CreateTripFromTemplateInput, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Trip from " + template.Title
	}
	if len([]rune(title)) > 120 {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("title must be at most 120 characters")
	}
	destination := strings.TrimSpace(in.Destination)
	if destination == "" && template.DestinationHint != nil {
		destination = strings.TrimSpace(*template.DestinationHint)
	}
	if destination == "" {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("destination is required")
	}
	if strings.TrimSpace(in.StartDate) == "" {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("startDate is required")
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
	}
	travelers := in.Travelers
	if travelers == nil {
		v := int32(1)
		travelers = &v
	}
	if *travelers < 1 {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("travelers must be at least 1")
	}
	pace := strings.TrimSpace(in.Pace)
	if pace == "" {
		pace = defaultPace
	}
	currencyFallback := ""
	if template.DefaultCurrency != nil {
		currencyFallback = *template.DefaultCurrency
	}
	amount, currency, err := budget.NormalizeBudgetInput(in.BudgetAmount, in.BudgetCurrency, currencyFallback)
	if err != nil {
		return appdto.CreateTripFromTemplateInput{}, apperrs.NewInvalidInput("%s", err.Error())
	}
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(currencyFallback))
	}
	if currency == "" {
		currency = defaultCurrency
	}
	in.Title = title
	in.Destination = destination
	in.BudgetAmount = amount
	in.BudgetCurrency = currency
	in.Travelers = travelers
	in.Pace = pace
	return in, nil
}

func applyTemplateMetadataUpdate(current entity.TripTemplate, in appdto.UpdateTripTemplateInput) (entity.TripTemplate, error) {
	if in.Title != nil {
		title, err := validateTemplateTitle(*in.Title)
		if err != nil {
			return entity.TripTemplate{}, err
		}
		current.Title = title
	}
	var err error
	if in.Description != nil {
		current.Description, err = normalizeOptionalText(in.Description, 1000, "description")
		if err != nil {
			return entity.TripTemplate{}, err
		}
	}
	if in.DestinationHint != nil {
		current.DestinationHint, err = normalizeOptionalText(in.DestinationHint, 200, "destinationHint")
		if err != nil {
			return entity.TripTemplate{}, err
		}
	}
	if in.DefaultCurrency != nil {
		current.DefaultCurrency, err = normalizeOptionalCurrency(in.DefaultCurrency, "defaultCurrency")
		if err != nil {
			return entity.TripTemplate{}, err
		}
	}
	if in.ReplaceTags {
		current.Tags, err = normalizeTemplateTags(in.Tags)
		if err != nil {
			return entity.TripTemplate{}, err
		}
	}
	return current, nil
}

func (s *Service) templateAccess(ctx context.Context, template *entity.TripTemplate, userID uuid.UUID) (appdto.TripTemplateAccess, error) {
	if template.Visibility == entity.TripTemplateVisibilityPrivate {
		if template.CreatedByUserID != userID {
			return appdto.TripTemplateAccess{}, domainerrs.ErrNotFound
		}
		active := template.Status == entity.TripTemplateStatusActive
		return appdto.TripTemplateAccess{
			Role:         "owner",
			Source:       "private",
			CanUse:       active,
			CanEdit:      true,
			CanArchive:   active,
			CanDuplicate: true,
		}, nil
	}
	if template.WorkspaceID == nil {
		return appdto.TripTemplateAccess{}, domainerrs.ErrNotFound
	}
	role, err := s.workspaceRole(ctx, userID, *template.WorkspaceID)
	if err != nil {
		return appdto.TripTemplateAccess{}, err
	}
	active := template.Status == entity.TripTemplateStatusActive
	canWrite := role == workspaces.RoleOwner || role == workspaces.RoleAdmin || role == workspaces.RoleMember
	canManageAny := role == workspaces.RoleOwner || role == workspaces.RoleAdmin
	canManageOwn := role == workspaces.RoleMember && template.CreatedByUserID == userID
	return appdto.TripTemplateAccess{
		Role:         string(role),
		Source:       "workspace",
		CanUse:       active && canWrite,
		CanEdit:      canManageAny || canManageOwn,
		CanArchive:   active && (canManageAny || canManageOwn),
		CanDuplicate: active && canWrite,
	}, nil
}

func (s *Service) templateAccessCached(
	ctx context.Context,
	template *entity.TripTemplate,
	userID uuid.UUID,
	cache map[uuid.UUID]appdto.TripTemplateAccess,
) (appdto.TripTemplateAccess, error) {
	if template.Visibility != entity.TripTemplateVisibilityWorkspace || template.WorkspaceID == nil {
		return s.templateAccess(ctx, template, userID)
	}
	if cached, ok := cache[*template.WorkspaceID]; ok {
		active := template.Status == entity.TripTemplateStatusActive
		cached.CanArchive = active && (cached.Role == "owner" || cached.Role == "admin" || (cached.Role == "member" && template.CreatedByUserID == userID))
		cached.CanEdit = cached.Role == "owner" || cached.Role == "admin" || (cached.Role == "member" && template.CreatedByUserID == userID)
		cached.CanUse = active && (cached.Role == "owner" || cached.Role == "admin" || cached.Role == "member")
		cached.CanDuplicate = cached.CanUse
		return cached, nil
	}
	access, err := s.templateAccess(ctx, template, userID)
	if err != nil {
		return appdto.TripTemplateAccess{}, err
	}
	cache[*template.WorkspaceID] = access
	return access, nil
}

func (s *Service) workspaceRole(ctx context.Context, userID, workspaceID uuid.UUID) (workspaces.Role, error) {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return "", apperrs.ErrForbidden
	}
	access, err := s.workspaceProvider.AccessCheck(ctx, userID, workspaceID)
	if err != nil {
		return "", err
	}
	if access == nil || !access.HasAccess || access.WorkspaceArchived {
		return "", apperrs.ErrForbidden
	}
	return access.Role, nil
}

func validateTemplateTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if len([]rune(title)) < 2 || len([]rune(title)) > 120 {
		return "", apperrs.NewInvalidInput("title must be between 2 and 120 characters")
	}
	return title, nil
}

func normalizeOptionalText(value *string, maxLength int, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if len([]rune(trimmed)) > maxLength {
		return nil, apperrs.NewInvalidInput("%s must be at most %d characters", field, maxLength)
	}
	return &trimmed, nil
}

func normalizeOptionalCurrency(value *string, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	currency := strings.ToUpper(strings.TrimSpace(*value))
	if currency == "" {
		return nil, nil
	}
	if !currencyCodePattern.MatchString(currency) {
		return nil, apperrs.NewInvalidInput("%s must be a 3-letter uppercase code", field)
	}
	return &currency, nil
}

func normalizeTemplateTags(tags []string) ([]string, error) {
	if len(tags) > maxTemplateTags {
		return nil, apperrs.NewInvalidInput("tags must contain at most %d items", maxTemplateTags)
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := normalizeTemplateTag(raw)
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > 40 {
			return nil, apperrs.NewInvalidInput("tags must be at most 40 characters")
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out, nil
}

func normalizeTemplateTag(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(value)
	var b strings.Builder
	lastHyphen := false
	for _, ch := range value {
		allowed := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if allowed {
			b.WriteRune(ch)
			lastHyphen = false
			continue
		}
		if ch == '-' && !lastHyphen {
			b.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func parseTemplateDate(raw string) *time.Time {
	parsed, _ := time.Parse("2006-01-02", raw)
	return &parsed
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func templateStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func sortedTemplateDays(days []templateDay) {
	sort.SliceStable(days, func(i, j int) bool {
		return days[i].DayOffset < days[j].DayOffset
	})
}
