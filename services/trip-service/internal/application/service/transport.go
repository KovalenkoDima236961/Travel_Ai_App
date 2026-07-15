package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/providerlimit"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/transportclient"
)

func (s *Service) SearchRouteLegTransportOptions(
	ctx context.Context,
	tripID uuid.UUID,
	legID string,
	in appdto.SearchRouteLegTransportInput,
) (*transportclient.TransportSearchResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if !s.transportSearchEnabled || s.transportSearchProvider == nil {
		return nil, apperrs.NewDependencyError("transport_search_unavailable")
	}
	leg, origin, destination, err := routeLegSearchContext(trip.Route, legID)
	if err != nil {
		return nil, err
	}
	req := transportclient.TransportSearchRequest{
		Origin:         origin,
		Destination:    destination,
		Date:           defaultTransportDate(in.Date, leg.DepartureDate, trip.StartDate),
		Time:           strings.TrimSpace(in.Time),
		TimePreference: strings.TrimSpace(in.TimePreference),
		Travelers:      defaultTransportTravelers(in.Travelers, trip.Travelers),
		Modes:          defaultTransportModes(in.Modes, leg.Mode, trip.Route),
		Currency:       defaultTransportCurrency(in.Currency, trip.BudgetCurrency),
		Locale:         "en",
		Constraints: transportclient.SearchConstraints{
			MaxDurationMinutes: in.Constraints.MaxDurationMinutes,
			MaxPriceAmount:     in.Constraints.MaxPriceAmount,
			AvoidFlights:       in.Constraints.AvoidFlights,
			PreferredModes:     normalizeTransportModes(in.Constraints.PreferredModes),
			AccessibilityNotes: in.Constraints.AccessibilityNotes,
		},
	}
	if strings.TrimSpace(req.Date) == "" {
		return nil, apperrs.NewInvalidInput("date is required")
	}
	result, err := s.transportSearchProvider.SearchTransportOptions(ctx, req)
	if err != nil {
		if s.transportSearchFailOpen {
			return &transportclient.TransportSearchResponse{
				Options: []transportclient.TransportOption{},
				Summary: transportclient.SearchSummary{
					Origin:        req.Origin.Name,
					Destination:   req.Destination.Name,
					Date:          req.Date,
					SearchedModes: req.Modes,
					Provider:      "unavailable",
					Warnings:      []string{"Transport search is temporarily unavailable."},
				},
			}, nil
		}
		if _, ok := providerlimit.As(err); ok {
			return nil, err
		}
		return nil, apperrs.NewDependencyError("transport_search_failed")
	}
	return result, nil
}

func (s *Service) AttachRouteLegTransportOption(
	ctx context.Context,
	tripID uuid.UUID,
	legID string,
	in appdto.AttachRouteLegTransportOptionInput,
) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if len(current.Itinerary) > 0 {
		expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
		if err != nil {
			return nil, err
		}
		if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
			return nil, err
		}
	}
	if current.Route == nil {
		return nil, domainerrs.ErrNotFound
	}
	route := cloneRoute(current.Route)
	legIndex, err := findRouteLegIndex(route, legID)
	if err != nil {
		return nil, err
	}
	option := in.Option
	option.SelectedAt = time.Now().UTC().Format(time.RFC3339)
	option.SelectedByUserID = user.ID.String()
	if err := validateAndNormalizeSelectedTransportOption(&option, legIndex); err != nil {
		return nil, err
	}
	route.Legs[legIndex].SelectedTransportOption = &option
	if in.UpdateLegMode {
		route.Legs[legIndex].Mode = option.Mode
	}
	if option.DurationMinutes > 0 {
		duration := option.DurationMinutes
		route.Legs[legIndex].EstimatedDurationMinutes = &duration
	}
	if option.EstimatedPrice != nil {
		amount := option.EstimatedPrice.Amount
		route.Legs[legIndex].EstimatedCost = &aggregate.EstimatedCost{
			Amount:     &amount,
			Currency:   option.EstimatedPrice.Currency,
			Category:   budget.CategoryTransport,
			Confidence: option.Confidence,
			Source:     budget.SourceProvider,
			Note:       "Estimated from selected transport option (" + option.Provider + ").",
		}
	}
	if err := validateAndNormalizeRoute(route, dateString(current.StartDate), current.Days); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateTripRoute(ctx, tripID, ownerID, route, normalizeTripType(current.TripType, route))
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTransportOptionAttached,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"routeLegId": legID,
			"mode":       option.Mode,
			"provider":   option.Provider,
			"operator":   option.OperatorName,
		},
	})
	if len(current.Itinerary) > 0 {
		s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Route transport option changed")
	}
	return updated, nil
}

func (s *Service) RemoveRouteLegTransportOption(
	ctx context.Context,
	tripID uuid.UUID,
	legID string,
	in appdto.RemoveRouteLegTransportOptionInput,
) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if len(current.Itinerary) > 0 {
		expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
		if err != nil {
			return nil, err
		}
		if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
			return nil, err
		}
	}
	if current.Route == nil {
		return nil, domainerrs.ErrNotFound
	}
	route := cloneRoute(current.Route)
	legIndex, err := findRouteLegIndex(route, legID)
	if err != nil {
		return nil, err
	}
	previous := route.Legs[legIndex].SelectedTransportOption
	route.Legs[legIndex].SelectedTransportOption = nil
	if err := validateAndNormalizeRoute(route, dateString(current.StartDate), current.Days); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateTripRoute(ctx, tripID, ownerID, route, normalizeTripType(current.TripType, route))
	if err != nil {
		return nil, err
	}
	metadata := map[string]any{"routeLegId": legID}
	if previous != nil {
		metadata["mode"] = previous.Mode
		metadata["provider"] = previous.Provider
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTransportOptionRemoved,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata:    metadata,
	})
	if len(current.Itinerary) > 0 {
		s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Route transport option removed")
	}
	return updated, nil
}

func routeLegSearchContext(route *aggregate.TripRoute, legID string) (*aggregate.RouteLeg, transportclient.Location, transportclient.Location, error) {
	if route == nil {
		return nil, transportclient.Location{}, transportclient.Location{}, domainerrs.ErrNotFound
	}
	legIndex, err := findRouteLegIndex(route, legID)
	if err != nil {
		return nil, transportclient.Location{}, transportclient.Location{}, err
	}
	leg := &route.Legs[legIndex]
	origin, ok := locationForStop(route, leg.FromStopID, leg.FromName)
	if !ok {
		return nil, transportclient.Location{}, transportclient.Location{}, apperrs.NewInvalidInput("route leg origin could not be resolved")
	}
	destination, ok := locationForStop(route, leg.ToStopID, leg.ToName)
	if !ok {
		return nil, transportclient.Location{}, transportclient.Location{}, apperrs.NewInvalidInput("route leg destination could not be resolved")
	}
	return leg, origin, destination, nil
}

func locationForStop(route *aggregate.TripRoute, stopID, fallbackName string) (transportclient.Location, bool) {
	if stopID == "origin" {
		if route.Origin != nil {
			return locationFromRoutePlace(*route.Origin, stopID, fallbackName), true
		}
		if strings.TrimSpace(fallbackName) != "" {
			return transportclient.Location{Name: strings.TrimSpace(fallbackName), StopID: stopID}, true
		}
		return transportclient.Location{}, false
	}
	for _, stop := range route.Stops {
		if stop.ID == stopID {
			name := strings.TrimSpace(stop.City)
			if name == "" {
				name = strings.TrimSpace(stop.Destination)
			}
			if name == "" {
				name = strings.TrimSpace(fallbackName)
			}
			location := transportclient.Location{Name: name, Country: strings.TrimSpace(stop.Country), StopID: stop.ID}
			if stop.Coordinates != nil {
				lat, lng := stop.Coordinates.Lat, stop.Coordinates.Lng
				location.Lat = &lat
				location.Lng = &lng
			}
			return location, strings.TrimSpace(location.Name) != ""
		}
	}
	return transportclient.Location{}, false
}

func locationFromRoutePlace(place aggregate.RoutePlace, stopID, fallbackName string) transportclient.Location {
	name := strings.TrimSpace(place.Name)
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	location := transportclient.Location{Name: name, Country: strings.TrimSpace(place.Country), StopID: stopID}
	if place.Coordinates != nil {
		lat, lng := place.Coordinates.Lat, place.Coordinates.Lng
		location.Lat = &lat
		location.Lng = &lng
	}
	return location
}

func defaultTransportDate(inputDate, legDate string, tripStart *time.Time) string {
	if value := strings.TrimSpace(inputDate); value != "" {
		return value
	}
	if value := strings.TrimSpace(legDate); value != "" {
		return value
	}
	return dateString(tripStart)
}

func defaultTransportTravelers(input int, tripTravelers int32) int {
	if input > 0 {
		return input
	}
	if tripTravelers > 0 {
		return int(tripTravelers)
	}
	return 1
}

func defaultTransportModes(input []string, legMode string, route *aggregate.TripRoute) []string {
	modes := normalizeTransportModes(input)
	if len(modes) > 0 {
		return modes
	}
	if mode := aggregate.NormalizeRouteToken(legMode); mode != "" {
		return []string{mode}
	}
	if route != nil {
		modes = normalizeTransportModes(route.Preferences.PreferredModes)
		if len(modes) > 0 {
			return modes
		}
	}
	return []string{aggregate.TransportModeTrain, aggregate.TransportModeBus, aggregate.TransportModeCar}
}

func defaultTransportCurrency(input, tripCurrency string) string {
	if value := strings.ToUpper(strings.TrimSpace(input)); value != "" {
		return value
	}
	if value := strings.ToUpper(strings.TrimSpace(tripCurrency)); value != "" {
		return value
	}
	return defaultCurrency
}

func normalizeTransportModes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		mode := aggregate.NormalizeRouteToken(raw)
		if mode == "" {
			continue
		}
		if _, ok := aggregate.SupportedTransportModes[mode]; !ok {
			continue
		}
		if _, exists := seen[mode]; exists {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}

func findRouteLegIndex(route *aggregate.TripRoute, legID string) (int, error) {
	if route == nil {
		return -1, domainerrs.ErrNotFound
	}
	normalized := strings.TrimSpace(legID)
	for index := range route.Legs {
		if route.Legs[index].ID == normalized {
			return index, nil
		}
	}
	return -1, domainerrs.ErrNotFound
}

func cloneRoute(route *aggregate.TripRoute) *aggregate.TripRoute {
	if route == nil {
		return nil
	}
	raw, err := json.Marshal(route)
	if err != nil {
		copied := *route
		return &copied
	}
	var out aggregate.TripRoute
	if err := json.Unmarshal(raw, &out); err != nil {
		copied := *route
		return &copied
	}
	return &out
}
