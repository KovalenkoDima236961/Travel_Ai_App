package service

import (
	"bytes"
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
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
)

var ErrVerificationDisabled = errors.New("verification is disabled")

// GetTripVerification evaluates persisted route, place, price, availability,
// weather, calendar, and accommodation metadata. It is private and advisory:
// a verified result means recently checked data, never a booking guarantee.
func (s *Service) GetTripVerification(
	ctx context.Context,
	tripID uuid.UUID,
) (verification.Response, error) {
	started := time.Now()
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return verification.Response{}, err
	}
	if !s.verificationConfig.Enabled {
		return verification.Response{}, ErrVerificationDisabled
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return verification.Response{}, err
	}
	cacheKey := summaryCacheKey("verification", trip, user.ID, access.Role())
	if cached, ok := s.verificationCache.get("verification", cacheKey); ok {
		if response, valid := cached.(verification.Response); valid {
			tripobs.RecordVerificationRead("cache_hit", time.Since(started))
			return response, nil
		}
	}
	response := s.verificationForTrip(ctx, trip)
	s.verificationCache.set("verification", cacheKey, response)
	tripobs.RecordVerificationRead("computed", time.Since(started))
	tripobs.RecordVerificationComputed(response)
	s.log.Info("trip verification computed",
		zap.String("trip_id", trip.ID.String()),
		zap.Int("score", response.Score),
		zap.String("level", string(response.Level)),
		zap.Int("issue_count", len(response.TopIssues)),
	)
	return response, nil
}

func (s *Service) verificationForTrip(ctx context.Context, trip *entity.Trip) verification.Response {
	var calendar *verification.CalendarState
	var itinerary aggregate.Itinerary
	if trip != nil && s.calendarSyncEnabled {
		status, err := s.GetGoogleCalendarSyncStatus(ctx, trip.ID)
		if err != nil {
			s.warn("verification: calendar status unavailable", zap.String("trip_id", trip.ID.String()), zap.Error(err))
		} else if status != nil {
			calendar = &verification.CalendarState{
				Connected:                status.Connected,
				Synced:                   status.Synced,
				LastSyncedAt:             status.LastSyncedAt,
				SyncedItineraryRevision:  status.SyncedItineraryRevision,
				CurrentItineraryRevision: status.CurrentItineraryRevision,
				OutOfDate:                status.OutOfDate,
				Provider:                 status.Provider,
			}
		}
	}
	if trip != nil {
		itinerary = parseItineraryLenient(trip.Itinerary)
	}
	return verification.Evaluate(verification.Input{
		Trip:      trip,
		Itinerary: itinerary,
		Calendar:  calendar,
		Now:       time.Now().UTC(),
		Config:    s.verificationConfig,
	})
}

// RunTripVerificationAction performs only explicit user-requested refreshes.
// It neither books nor purchases anything. Viewer access is read-only, so
// provider refreshes and metadata writes require an owner or editor.
func (s *Service) RunTripVerificationAction(
	ctx context.Context,
	tripID uuid.UUID,
	in verification.ActionRequest,
) (out verification.ActionResult, err error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return verification.ActionResult{}, err
	}
	if !s.verificationConfig.Enabled {
		return verification.ActionResult{}, ErrVerificationDisabled
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return verification.ActionResult{}, err
	}
	actionType := strings.TrimSpace(in.ActionType)
	if !verificationActionAllowed(actionType) {
		return verification.ActionResult{}, apperrs.NewInvalidInput("unsupported verification action")
	}
	defer func() {
		result := "completed"
		if err != nil || out.Status == "failed" {
			result = "failed"
		}
		tripobs.RecordVerificationAction(actionType, result)
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      trip.ID,
			ActorUserID: &user.ID,
			EventType:   activity.EventVerificationRefreshed,
			EntityType:  activityEntityType(activity.EntityTrip),
			EntityID:    activityEntityID(trip.ID),
			Metadata:    map[string]any{"actionType": actionType, "result": result},
		})
	}()
	if !access.CanEdit() {
		return verification.ActionResult{}, apperrs.ErrForbidden
	}
	s.log.Info("verification action requested",
		zap.String("trip_id", tripID.String()),
		zap.String("action_type", actionType),
		zap.String("scope", string(in.Scope)),
		zap.String("entity_type", strings.TrimSpace(in.EntityType)),
	)

	switch actionType {
	case "refresh_weather":
		return s.refreshVerificationWeather(ctx, trip, user.ID)
	case "recheck_transport":
		return s.recheckVerificationTransport(ctx, trip, user.ID, strings.TrimSpace(in.EntityID))
	case "refresh_place_details":
		return s.refreshVerificationPlaces(ctx, trip, user.ID)
	case "refresh_price":
		return s.refreshVerificationPrices(ctx, trip, user.ID)
	case "update_calendar_sync":
		return s.refreshVerificationCalendar(ctx, trip)
	case "review_opening_hours", "add_accommodation", "attach_place", "open_route", "open_budget", "open_itinerary_item":
		return verification.ActionResult{
			Status:              "completed",
			Message:             "Open the linked trip section to review or update this detail. No booking or purchase was made.",
			UpdatedVerification: s.verificationForTrip(ctx, trip),
		}, nil
	case "check_availability":
		// Availability search already has a per-item, user-driven flow in the
		// web app. Do not fabricate a provider check when no item search input
		// (date, quantity, and provider) was supplied to this endpoint.
		return verification.ActionResult{
			Status:              "failed",
			Message:             "Open the itinerary item to run its availability check. No provider availability was claimed.",
			UpdatedVerification: s.verificationForTrip(ctx, trip),
		}, nil
	default:
		return verification.ActionResult{}, apperrs.NewInvalidInput("unsupported verification action")
	}
}

func (s *Service) refreshVerificationWeather(ctx context.Context, trip *entity.Trip, actorID uuid.UUID) (verification.ActionResult, error) {
	if trip == nil || trip.StartDate == nil || strings.TrimSpace(trip.Destination) == "" || trip.Days < 1 {
		return verification.ActionResult{}, apperrs.NewInvalidInput("destination, startDate, and days are required to refresh weather")
	}
	if !s.weatherContextEnabled || s.weatherContextProvider == nil {
		return verification.ActionResult{Status: "failed", Message: "Weather refresh is not configured.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	forecast, err := s.weatherContextProvider.GetForecast(ctx, trip.Destination, trip.StartDate.Format("2006-01-02"), int(trip.Days))
	if err != nil {
		return verification.ActionResult{Status: "failed", Message: "Weather could not be refreshed. Try again later.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	if forecast == nil || len(forecast.Days) == 0 {
		return verification.ActionResult{Status: "failed", Message: "Weather provider returned no forecast for this trip.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return verification.ActionResult{}, err
	}
	now := time.Now().UTC()
	metadata := withVerificationValue(trip.CreationMetadata, "weather", map[string]any{
		"provider":     strings.TrimSpace(forecast.Provider),
		"checkedAt":    now.Format(time.RFC3339),
		"fallbackUsed": verificationMockProvider(forecast.Provider),
		"dayCount":     len(forecast.Days),
	})
	updated, err := s.repo.UpdateTripCreationMetadata(ctx, trip.ID, ownerID, metadata)
	if err != nil {
		return verification.ActionResult{}, err
	}
	_ = actorID // actor is included in request logs; metadata stays trip-scoped and non-sensitive.
	return verification.ActionResult{Status: "completed", Message: "Weather refreshed. Forecasts can still change before travel.", UpdatedVerification: s.verificationForTrip(ctx, updated)}, nil
}

func (s *Service) recheckVerificationTransport(ctx context.Context, trip *entity.Trip, actorID uuid.UUID, legID string) (verification.ActionResult, error) {
	if trip == nil || trip.Route == nil {
		return verification.ActionResult{}, apperrs.NewInvalidInput("route transport is not available for this trip")
	}
	if legID == "" {
		for _, leg := range trip.Route.Legs {
			if leg.SelectedTransportOption != nil {
				legID = leg.ID
				break
			}
		}
	}
	if legID == "" {
		return verification.ActionResult{}, apperrs.NewInvalidInput("a route leg with selected transport is required")
	}
	leg, _, _, err := routeLegSearchContext(trip.Route, legID)
	if err != nil {
		return verification.ActionResult{}, err
	}
	if leg.SelectedTransportOption == nil {
		return verification.ActionResult{}, apperrs.NewInvalidInput("selected transport is required for recheck")
	}
	result, err := s.SearchRouteLegTransportOptions(ctx, trip.ID, legID, appdto.SearchRouteLegTransportInput{})
	if err != nil {
		return verification.ActionResult{Status: "failed", Message: "Transport could not be rechecked. Try again later.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	matched := false
	for _, option := range result.Options {
		if option.ID == leg.SelectedTransportOption.ID {
			matched = true
			break
		}
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return verification.ActionResult{}, err
	}
	now := time.Now().UTC()
	metadata := withTransportVerification(trip.CreationMetadata, legID, map[string]any{
		"provider":                result.Summary.Provider,
		"checkedAt":               now.Format(time.RFC3339),
		"fallbackUsed":            result.Summary.FallbackUsed || verificationMockProvider(result.Summary.Provider),
		"selectedOptionAvailable": matched,
	})
	updated, err := s.repo.UpdateTripCreationMetadata(ctx, trip.ID, ownerID, metadata)
	if err != nil {
		return verification.ActionResult{}, err
	}
	_ = actorID
	message := "Transport search refreshed. Review the results before relying on the selected option."
	if !matched {
		message = "Transport search refreshed, but the selected option was not returned. Review alternatives before travel."
	}
	return verification.ActionResult{Status: "completed", Message: message, UpdatedVerification: s.verificationForTrip(ctx, updated)}, nil
}

func (s *Service) refreshVerificationPlaces(ctx context.Context, trip *entity.Trip, actorID uuid.UUID) (verification.ActionResult, error) {
	itinerary := parseItineraryLenient(trip.Itinerary)
	if len(itinerary.Days) == 0 {
		return verification.ActionResult{}, apperrs.NewInvalidInput("itinerary is required to refresh place details")
	}
	updatedItinerary, err := s.enrichItinerary(ctx, trip.ID, *trip, itinerary, "verification")
	if err != nil {
		return verification.ActionResult{Status: "failed", Message: "Place details could not be refreshed. Try again later.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	return s.persistVerificationItinerary(ctx, trip, actorID, updatedItinerary, "Place details refreshed from available providers.")
}

func (s *Service) refreshVerificationPrices(ctx context.Context, trip *entity.Trip, actorID uuid.UUID) (verification.ActionResult, error) {
	itinerary := parseItineraryLenient(trip.Itinerary)
	if len(itinerary.Days) == 0 {
		return verification.ActionResult{}, apperrs.NewInvalidInput("itinerary is required to refresh prices")
	}
	updatedItinerary, err := s.enrichItineraryPrices(ctx, trip.ID, *trip, itinerary, usercontext.UserContext{}, "verification")
	if err != nil {
		return verification.ActionResult{Status: "failed", Message: "Prices could not be refreshed. Try again later.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	return s.persistVerificationItinerary(ctx, trip, actorID, updatedItinerary, "Prices refreshed from available providers. Prices remain estimates unless explicitly confirmed.")
}

func (s *Service) persistVerificationItinerary(ctx context.Context, trip *entity.Trip, actorID uuid.UUID, itinerary *aggregate.Itinerary, message string) (verification.ActionResult, error) {
	if trip == nil || itinerary == nil {
		return verification.ActionResult{}, apperrs.NewInvalidInput("itinerary refresh did not return usable data")
	}
	current := parseItineraryLenient(trip.Itinerary)
	before, _ := json.Marshal(current)
	after, err := json.Marshal(itinerary)
	if err != nil {
		return verification.ActionResult{}, fmt.Errorf("encode verification itinerary: %w", err)
	}
	if bytes.Equal(before, after) {
		return verification.ActionResult{Status: "completed", Message: "No newer provider details were available. Existing verification was kept.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return verification.ActionResult{}, err
	}
	updated, err := s.saveItineraryWithVersion(ctx, trip.ID, ownerID, actorID, after, trip.ItineraryRevision, entity.ItineraryVersionSourceManualEdit, map[string]any{"verificationRefresh": true})
	if err != nil {
		return verification.ActionResult{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{TripID: trip.ID, ActorUserID: &actorID, EventType: activity.EventItineraryUpdated, EntityType: activityEntityType(activity.EntityItinerary), EntityID: activityEntityID(trip.ID), Metadata: map[string]any{"source": "verification_refresh"}})
	return verification.ActionResult{Status: "completed", Message: message, UpdatedVerification: s.verificationForTrip(ctx, updated)}, nil
}

func (s *Service) refreshVerificationCalendar(ctx context.Context, trip *entity.Trip) (verification.ActionResult, error) {
	if trip == nil {
		return verification.ActionResult{}, apperrs.NewInvalidInput("trip is required")
	}
	revision := trip.ItineraryRevision
	result, err := s.SyncTripToGoogleCalendar(ctx, trip.ID, &revision)
	if err != nil {
		return verification.ActionResult{Status: "failed", Message: "Calendar sync could not be updated. Check the calendar connection and try again.", UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
	}
	return verification.ActionResult{Status: "completed", Message: fmt.Sprintf("Calendar sync updated (%d created, %d updated).", result.Created, result.Updated), UpdatedVerification: s.verificationForTrip(ctx, trip)}, nil
}

func verificationActionAllowed(value string) bool {
	switch value {
	case "refresh_weather", "recheck_transport", "check_availability", "refresh_place_details", "refresh_price", "review_opening_hours", "update_calendar_sync", "add_accommodation", "attach_place", "open_route", "open_budget", "open_itinerary_item":
		return true
	default:
		return false
	}
}

func withVerificationValue(metadata map[string]any, key string, value map[string]any) map[string]any {
	out := cloneMetadata(metadata)
	verificationData := cloneAnyMap(out["verification"])
	verificationData[key] = value
	out["verification"] = verificationData
	return out
}

func withTransportVerification(metadata map[string]any, legID string, value map[string]any) map[string]any {
	out := cloneMetadata(metadata)
	verificationData := cloneAnyMap(out["verification"])
	transport := cloneAnyMap(verificationData["transport"])
	transport[legID] = value
	verificationData["transport"] = transport
	out["verification"] = verificationData
	return out
}

func cloneAnyMap(value any) map[string]any {
	in, _ := value.(map[string]any)
	out := make(map[string]any, len(in)+1)
	for key, item := range in {
		out[key] = item
	}
	return out
}

func verificationMockProvider(provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return provider == "mock" || strings.Contains(provider, "mock") || provider == "fallback"
}
