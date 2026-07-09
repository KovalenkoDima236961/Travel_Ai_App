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
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/templateadaptation"
)

const (
	maxAdaptationInterests           = 20
	maxAdaptationInterestLength      = 40
	maxAdaptationAvoid               = 20
	maxAdaptationAvoidLength         = 80
	maxAdaptationSpecialInstructions = 1000
	maxAdaptationDurationDays        = 30
	maxAdaptationTravelers           = 50
)

var adaptationValidPaces = map[string]struct{}{
	"relaxed":   {},
	"balanced":  {},
	"intensive": {},
}

// PrepareTemplateAdaptation validates the request, enforces template and target
// workspace permissions, and creates the DRAFT trip the adaptation job will fill
// in. It returns the draft trip plus the JSON job payload for the worker. The
// generation job itself is created and dispatched by the generationjobs service.
func (s *Service) PrepareTemplateAdaptation(
	ctx context.Context,
	templateID uuid.UUID,
	in appdto.CreateTemplateAdaptationInput,
) (*entity.Trip, json.RawMessage, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	repo, err := s.templateRepo()
	if err != nil {
		return nil, nil, err
	}
	template, err := repo.GetTripTemplateByID(ctx, templateID)
	if err != nil {
		return nil, nil, err
	}
	access, err := s.templateAccess(ctx, template, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if !access.CanUse {
		return nil, nil, apperrs.ErrForbidden
	}
	if template.Status != entity.TripTemplateStatusActive {
		return nil, nil, apperrs.NewInvalidInput("archived templates cannot be adapted")
	}

	normalized, err := normalizeCreateTemplateAdaptationInput(in, template)
	if err != nil {
		return nil, nil, err
	}
	if normalized.WorkspaceID != nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *normalized.WorkspaceID); err != nil {
			return nil, nil, err
		}
	}

	created, err := s.repo.Create(ctx, &entity.Trip{
		UserID:         &user.ID,
		WorkspaceID:    normalized.WorkspaceID,
		Destination:    normalized.Destination,
		StartDate:      parseTemplateDate(normalized.StartDate),
		Days:           int32(normalized.DurationDays),
		BudgetAmount:   normalized.BudgetAmount,
		BudgetCurrency: normalized.BudgetCurrency,
		Travelers:      *normalized.Travelers,
		Interests:      normalized.Interests,
		Pace:           normalized.Pace,
		Status:         entity.StatusDraft,
	})
	if err != nil {
		return nil, nil, err
	}

	var budgetPtr *templateadaptation.Money
	if normalized.BudgetAmount != nil {
		budgetPtr = &templateadaptation.Money{Amount: *normalized.BudgetAmount, Currency: normalized.BudgetCurrency}
	}
	payload := templateadaptation.JobPayload{
		TemplateID:              template.ID,
		TemplateTitle:           template.Title,
		WorkspaceID:             normalized.WorkspaceID,
		Title:                   normalized.Title,
		Destination:             normalized.Destination,
		StartDate:               normalized.StartDate,
		DurationDays:            normalized.DurationDays,
		Budget:                  budgetPtr,
		Travelers:               int(*normalized.Travelers),
		Pace:                    normalized.Pace,
		Interests:               normalized.Interests,
		Avoid:                   normalized.Avoid,
		SpecialInstructions:     normalized.SpecialInstructions,
		FallbackToDeterministic: normalized.FallbackToDeterministic,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal template adaptation payload: %w", err)
	}

	s.log.Info("template adaptation draft trip created",
		zap.String("trip_id", created.ID.String()),
		zap.String("template_id", template.ID.String()),
		zap.String("user_id", user.ID.String()),
		zap.String("destination", normalized.Destination),
	)
	return created, raw, nil
}

// AdaptTemplateForActor is the worker entry point: it loads the template and
// draft trip, calls the AI adapter, validates and saves the adapted itinerary,
// records the version/activity, and returns the trip plus the adaptation summary
// JSON. On AI failure it can fall back to a deterministic template copy.
func (s *Service) AdaptTemplateForActor(
	ctx context.Context,
	tripID, actorUserID uuid.UUID,
	expectedRevision int,
	requestPayload json.RawMessage,
) (*entity.Trip, json.RawMessage, error) {
	ctx = contextWithActor(ctx, actorUserID)
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	payload := templateadaptation.DecodeJobPayload(requestPayload)

	repo, err := s.templateRepo()
	if err != nil {
		return nil, nil, err
	}
	template, err := repo.GetTripTemplateByID(ctx, payload.TemplateID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, nil, templateadaptation.NewError(templateadaptation.ErrorTemplateNotFound, "The template no longer exists.", err)
		}
		return nil, nil, err
	}
	access, err := s.templateAccess(ctx, template, actorUserID)
	if err != nil || !access.CanUse {
		return nil, nil, templateadaptation.NewError(templateadaptation.ErrorTemplateAccessDenied, "You no longer have access to this template.", err)
	}

	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, tripID)
	if err != nil {
		return nil, nil, err
	}

	if _, err := s.repo.UpdateStatusByUserID(ctx, tripID, ownerID, entity.StatusProcessing); err != nil {
		return nil, nil, err
	}

	adaptTemplate := buildAdaptationTemplate(template)
	input := templateadaptation.AdaptInput{
		TripID:   tripID,
		Template: adaptTemplate,
		Target: templateadaptation.Target{
			Destination:  payload.Destination,
			StartDate:    payload.StartDate,
			DurationDays: payload.DurationDays,
			Budget:       payload.Budget,
			Travelers:    payload.Travelers,
			Pace:         payload.Pace,
			Interests:    payload.Interests,
			Avoid:        payload.Avoid,
		},
		Constraints: templateadaptation.Constraints{
			PreserveStructure:       true,
			AdaptCosts:              true,
			PreserveMealStructure:   true,
			PreserveActivityDensity: true,
			SpecialInstructions:     payload.SpecialInstructions,
		},
		UserProfile:                userContext.Profile,
		UserPreferences:            userContext.Preferences,
		WorkspacePolicyConstraints: s.workspacePolicyAIConstraints(ctx, current),
	}

	var itinerary *aggregate.Itinerary
	var summary templateadaptation.Summary

	result, aiErr := s.generator.AdaptTemplate(ctx, input)
	// Validate the AI itinerary up front so structurally-invalid AI output can
	// also trigger the deterministic fallback (not just AI transport failures).
	failureCode := templateadaptation.ErrorAIAdaptationFailed
	if aiErr == nil {
		if raw, mErr := json.Marshal(&result.Itinerary); mErr != nil {
			aiErr = mErr
		} else if _, vErr := validateAndNormalizeItinerary(raw); vErr != nil {
			aiErr = vErr
			failureCode = templateadaptation.ErrorValidationFailed
		}
	}
	if aiErr != nil {
		s.log.Warn("ai template adaptation failed",
			zap.String("trip_id", tripID.String()),
			zap.String("template_id", template.ID.String()),
			zap.String("failure_code", failureCode),
			zap.Bool("fallback_enabled", payload.FallbackToDeterministic),
			zap.Error(aiErr),
		)
		if !payload.FallbackToDeterministic {
			s.markFailed(ctx, tripID, ownerID)
			return nil, nil, templateadaptation.NewError(failureCode, safeAdaptationFailureMessage(failureCode), aiErr)
		}
		fallbackItinerary, fbErr := s.deterministicFallbackItinerary(template, payload)
		if fbErr != nil {
			s.markFailed(ctx, tripID, ownerID)
			return nil, nil, templateadaptation.NewError(templateadaptation.ErrorDeterministicFallbackFailed, "AI adaptation and deterministic fallback both failed.", fbErr)
		}
		itinerary = fallbackItinerary
		summary = templateadaptation.Summary{
			SourceDurationDays: adaptTemplate.DurationDays,
			TargetDurationDays: payload.DurationDays,
			PreservedStructure: true,
			ChangedDestination: true,
			FallbackUsed:       true,
			FallbackReason:     failureCode,
			MajorChanges:       []string{fmt.Sprintf("Created a deterministic copy of %q for %s.", template.Title, payload.Destination)},
			Warnings:           []string{"AI adaptation failed; created a deterministic template copy instead. Review and adjust it for the destination."},
		}
	} else {
		itinerary = &result.Itinerary
		summary = result.Summary
	}

	itinerary, err = s.enrichGeneratedItinerary(ctx, tripID, *current, itinerary, userContext)
	if err != nil {
		s.markFailed(ctx, tripID, ownerID)
		return nil, nil, templateadaptation.NewError(templateadaptation.ErrorProviderEnrichmentFailed, "Adapted itinerary enrichment failed.", err)
	}

	raw, err := json.Marshal(itinerary)
	if err != nil {
		s.markFailed(ctx, tripID, ownerID)
		return nil, nil, err
	}
	normalizedRaw, err := validateAndNormalizeItinerary(raw)
	if err != nil {
		s.markFailed(ctx, tripID, ownerID)
		return nil, nil, templateadaptation.NewError(templateadaptation.ErrorValidationFailed, err.Error(), err)
	}

	updated, err := s.saveItineraryWithVersion(
		ctx,
		tripID,
		ownerID,
		user.ID,
		normalizedRaw,
		expectedRevision,
		entity.ItineraryVersionSourceCreatedFromTemplateAI,
		map[string]any{
			"source":        "ai_template_adaptation",
			"templateId":    template.ID.String(),
			"templateTitle": template.Title,
			"fallbackUsed":  summary.FallbackUsed,
		},
	)
	if err != nil {
		if !isItineraryConflict(err) {
			s.markFailed(ctx, tripID, ownerID)
		}
		return nil, nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripCreatedFromAITemplateAdaptation,
		EntityType:  activityEntityType(activity.EntityTripTemplate),
		EntityID:    activityEntityID(template.ID),
		Metadata: map[string]any{
			"templateId":         template.ID.String(),
			"templateTitle":      template.Title,
			"targetDestination":  payload.Destination,
			"targetDurationDays": payload.DurationDays,
			"fallbackUsed":       summary.FallbackUsed,
		},
	})

	destination := tripDestination(current)
	s.notifyTripBroadcast(ctx, current, user.ID,
		notifications.TypeItineraryGenerated,
		"Trip created from template",
		fmt.Sprintf("A trip for %s was created by adapting the template %q.", destination, template.Title),
		notifications.EntityItinerary, activityEntityID(tripID),
		map[string]any{"tripId": tripID.String(), "destination": destination, "fallbackUsed": summary.FallbackUsed})

	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return updated, nil, nil
	}
	return updated, summaryJSON, nil
}

func safeAdaptationFailureMessage(code string) string {
	if code == templateadaptation.ErrorValidationFailed {
		return "AI template adaptation produced an invalid itinerary."
	}
	return "AI template adaptation failed."
}

// deterministicFallbackItinerary reuses the existing create-from-template
// instantiation so a failed AI adaptation still yields a usable trip.
func (s *Service) deterministicFallbackItinerary(
	template *entity.TripTemplate,
	payload templateadaptation.JobPayload,
) (*aggregate.Itinerary, error) {
	travelers := int32(payload.Travelers)
	if travelers < 1 {
		travelers = 1
	}
	currency := ""
	if payload.Budget != nil {
		currency = payload.Budget.Currency
	}
	var amount *float64
	if payload.Budget != nil {
		v := payload.Budget.Amount
		amount = &v
	}
	itinerary, err := instantiateTemplateItinerary(template, appdto.CreateTripFromTemplateInput{
		Title:          payload.Title,
		Destination:    payload.Destination,
		StartDate:      payload.StartDate,
		WorkspaceID:    payload.WorkspaceID,
		BudgetAmount:   amount,
		BudgetCurrency: currency,
		Travelers:      &travelers,
		Pace:           payload.Pace,
	}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &itinerary, nil
}

// buildAdaptationTemplate decodes the sanitized template body and maps it to the
// AI request shape, deliberately dropping template metadata and provider place
// identifiers so no private data reaches the model prompt.
func buildAdaptationTemplate(template *entity.TripTemplate) templateadaptation.Template {
	var payload tripTemplateJSON
	if err := json.Unmarshal(template.TemplateJSON, &payload); err != nil {
		return templateadaptation.Template{SchemaVersion: 1, DurationDays: int(template.DurationDays)}
	}
	days := make([]templateadaptation.TemplateDay, 0, len(payload.Days))
	for _, day := range payload.Days {
		items := make([]templateadaptation.TemplateItem, 0, len(day.Items))
		for _, item := range day.Items {
			var place *templateadaptation.TemplatePlace
			if item.Place != nil && strings.TrimSpace(item.Place.Name) != "" {
				place = &templateadaptation.TemplatePlace{
					Name:     strings.TrimSpace(item.Place.Name),
					Category: strings.TrimSpace(item.Place.Category),
				}
			}
			items = append(items, templateadaptation.TemplateItem{
				Name:          strings.TrimSpace(item.Name),
				Type:          strings.TrimSpace(item.Type),
				Description:   strings.TrimSpace(item.Description),
				Time:          strings.TrimSpace(item.Time),
				StartTime:     strings.TrimSpace(item.StartTime),
				EndTime:       strings.TrimSpace(item.EndTime),
				Place:         place,
				EstimatedCost: item.EstimatedCost,
				Notes:         strings.TrimSpace(item.Notes),
			})
		}
		days = append(days, templateadaptation.TemplateDay{
			DayOffset: day.DayOffset,
			Title:     strings.TrimSpace(day.Title),
			Items:     items,
		})
	}
	durationDays := int(payload.DurationDays)
	if durationDays <= 0 {
		durationDays = int(template.DurationDays)
	}
	return templateadaptation.Template{
		SchemaVersion: 1,
		DurationDays:  durationDays,
		Days:          days,
	}
}

func normalizeCreateTemplateAdaptationInput(
	in appdto.CreateTemplateAdaptationInput,
	template *entity.TripTemplate,
) (appdto.CreateTemplateAdaptationInput, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Trip from " + template.Title
	}
	validTitle, err := validateTemplateTitle(title)
	if err != nil {
		return appdto.CreateTemplateAdaptationInput{}, err
	}

	destination := strings.TrimSpace(in.Destination)
	if destination == "" && template.DestinationHint != nil {
		destination = strings.TrimSpace(*template.DestinationHint)
	}
	if len([]rune(destination)) < 2 || len([]rune(destination)) > 120 {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("destination must be between 2 and 120 characters")
	}

	if strings.TrimSpace(in.StartDate) == "" {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("startDate is required")
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("startDate must be in YYYY-MM-DD format")
	}

	duration := in.DurationDays
	if duration == 0 {
		duration = int(template.DurationDays)
	}
	if duration < 1 || duration > maxAdaptationDurationDays {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("durationDays must be between 1 and %d", maxAdaptationDurationDays)
	}

	travelers := in.Travelers
	if travelers == nil {
		v := int32(1)
		travelers = &v
	}
	if *travelers < 1 || *travelers > maxAdaptationTravelers {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("travelers must be between 1 and %d", maxAdaptationTravelers)
	}

	pace := strings.ToLower(strings.TrimSpace(in.Pace))
	if pace == "" {
		pace = defaultPace
	}
	if _, ok := adaptationValidPaces[pace]; !ok {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("pace must be relaxed, balanced, or intensive")
	}

	currencyFallback := ""
	if template.DefaultCurrency != nil {
		currencyFallback = *template.DefaultCurrency
	}
	amount, currency, err := budget.NormalizeBudgetInput(in.BudgetAmount, in.BudgetCurrency, currencyFallback)
	if err != nil {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("%s", err.Error())
	}
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(currencyFallback))
	}
	if currency == "" {
		currency = defaultCurrency
	}

	interests, err := normalizeAdaptationTags(in.Interests, maxAdaptationInterests, maxAdaptationInterestLength, "interests")
	if err != nil {
		return appdto.CreateTemplateAdaptationInput{}, err
	}
	avoid, err := normalizeAdaptationTags(in.Avoid, maxAdaptationAvoid, maxAdaptationAvoidLength, "avoid")
	if err != nil {
		return appdto.CreateTemplateAdaptationInput{}, err
	}

	special := strings.TrimSpace(in.SpecialInstructions)
	if len([]rune(special)) > maxAdaptationSpecialInstructions {
		return appdto.CreateTemplateAdaptationInput{}, apperrs.NewInvalidInput("specialInstructions must be at most %d characters", maxAdaptationSpecialInstructions)
	}

	in.Title = validTitle
	in.Destination = destination
	in.DurationDays = duration
	in.Travelers = travelers
	in.Pace = pace
	in.BudgetAmount = amount
	in.BudgetCurrency = currency
	in.Interests = interests
	in.Avoid = avoid
	in.SpecialInstructions = special
	return in, nil
}

func normalizeAdaptationTags(values []string, maxCount, maxLength int, field string) ([]string, error) {
	if len(values) > maxCount {
		return nil, apperrs.NewInvalidInput("%s must contain at most %d items", field, maxCount)
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > maxLength {
			return nil, apperrs.NewInvalidInput("%s entries must be at most %d characters", field, maxLength)
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out, nil
}
