package service

import (
	"context"
	"fmt"
	"net/url"
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
)

const (
	maxRouteStops              = 20
	defaultMaxTransferHoursDay = 8
)

func (s *Service) GetTripRoute(ctx context.Context, tripID uuid.UUID) (*aggregate.TripRoute, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	return trip.Route, nil
}

func (s *Service) UpdateTripRoute(ctx context.Context, tripID uuid.UUID, in appdto.UpdateTripRouteInput) (*entity.Trip, error) {
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
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	tripType := normalizeTripType("", in.Route)
	if in.Route != nil {
		if err := validateAndNormalizeRoute(in.Route, dateString(current.StartDate), current.Days); err != nil {
			return nil, err
		}
		if len(in.Route.Stops) > 1 {
			tripType = entity.TripTypeMultiDestination
		}
	} else {
		tripType = entity.TripTypeSingleDestination
	}

	updated, err := s.repo.UpdateTripRoute(ctx, tripID, ownerID, in.Route, tripType)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventRouteUpdated,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"tripType":        updated.TripType,
			"stopCount":       routeStopCount(updated.Route),
			"staleItinerary":  len(current.Itinerary) > 0,
			"routeWasPresent": current.Route != nil,
		},
	})

	if len(current.Itinerary) > 0 {
		s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Route changed")
	}
	return updated, nil
}

func normalizeTripType(value string, route *aggregate.TripRoute) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case entity.TripTypeMultiDestination:
		return entity.TripTypeMultiDestination
	case entity.TripTypeSingleDestination:
		return entity.TripTypeSingleDestination
	default:
		if route != nil && len(route.Stops) > 1 {
			return entity.TripTypeMultiDestination
		}
		return entity.TripTypeSingleDestination
	}
}

func validateAndNormalizeRoute(route *aggregate.TripRoute, startDate string, days int32) error {
	if route == nil {
		return nil
	}
	if route.Origin != nil {
		route.Origin.Name = strings.TrimSpace(route.Origin.Name)
		route.Origin.Country = strings.TrimSpace(route.Origin.Country)
		if route.Origin.Coordinates != nil && !validCoordinates(route.Origin.Coordinates) {
			return apperrs.NewInvalidInput("route.origin.coordinates must contain valid lat/lng")
		}
	}
	if len(route.Stops) < 1 || len(route.Stops) > maxRouteStops {
		return apperrs.NewInvalidInput("route.stops must contain between 1 and %d stops", maxRouteStops)
	}

	stopIDs := map[string]struct{}{"origin": {}}
	for index := range route.Stops {
		stop := &route.Stops[index]
		stop.ID = strings.TrimSpace(stop.ID)
		if stop.ID == "" {
			stop.ID = fmt.Sprintf("stop_%d", index+1)
		}
		if _, exists := stopIDs[stop.ID]; exists {
			return apperrs.NewInvalidInput("route.stops[%d].id must be unique", index)
		}
		stopIDs[stop.ID] = struct{}{}
		stop.Destination = strings.TrimSpace(stop.Destination)
		if stop.Destination == "" {
			return apperrs.NewInvalidInput("route.stops[%d].destination is required", index)
		}
		stop.City = strings.TrimSpace(stop.City)
		stop.Country = strings.TrimSpace(stop.Country)
		stop.ArrivalDate = strings.TrimSpace(stop.ArrivalDate)
		stop.DepartureDate = strings.TrimSpace(stop.DepartureDate)
		if stop.Nights != nil && *stop.Nights < 0 {
			return apperrs.NewInvalidInput("route.stops[%d].nights must be >= 0", index)
		}
		if err := validateRouteStopDates(stop, index); err != nil {
			return err
		}
		if stop.Coordinates != nil && !validCoordinates(stop.Coordinates) {
			return apperrs.NewInvalidInput("route.stops[%d].coordinates must contain valid lat/lng", index)
		}
		stop.AccommodationHint = aggregate.NormalizeRouteToken(stop.AccommodationHint)
		if stop.AccommodationHint == "" {
			stop.AccommodationHint = "unknown"
		}
		if _, ok := aggregate.SupportedAccommodationHints[stop.AccommodationHint]; !ok {
			return apperrs.NewInvalidInput("route.stops[%d].accommodationHint is unsupported", index)
		}
		if stop.Notes != nil {
			trimmed := strings.TrimSpace(*stop.Notes)
			if trimmed == "" {
				stop.Notes = nil
			} else {
				stop.Notes = &trimmed
			}
		}
	}

	for index := range route.Legs {
		leg := &route.Legs[index]
		leg.ID = strings.TrimSpace(leg.ID)
		if leg.ID == "" {
			leg.ID = fmt.Sprintf("leg_%d", index+1)
		}
		leg.FromStopID = strings.TrimSpace(leg.FromStopID)
		leg.ToStopID = strings.TrimSpace(leg.ToStopID)
		if _, ok := stopIDs[leg.FromStopID]; !ok {
			return apperrs.NewInvalidInput("route.legs[%d].fromStopId must reference origin or a route stop", index)
		}
		if _, ok := stopIDs[leg.ToStopID]; !ok {
			return apperrs.NewInvalidInput("route.legs[%d].toStopId must reference a route stop", index)
		}
		leg.Mode = aggregate.NormalizeRouteToken(leg.Mode)
		if _, ok := aggregate.SupportedTransportModes[leg.Mode]; !ok {
			return apperrs.NewInvalidInput("route.legs[%d].mode is unsupported", index)
		}
		leg.FromName = strings.TrimSpace(leg.FromName)
		leg.ToName = strings.TrimSpace(leg.ToName)
		leg.DepartureDate = strings.TrimSpace(leg.DepartureDate)
		if leg.DepartureDate != "" {
			if _, err := time.Parse("2006-01-02", leg.DepartureDate); err != nil {
				return apperrs.NewInvalidInput("route.legs[%d].departureDate must be in YYYY-MM-DD format", index)
			}
		}
		if leg.EstimatedDurationMinutes != nil && *leg.EstimatedDurationMinutes < 0 {
			return apperrs.NewInvalidInput("route.legs[%d].estimatedDurationMinutes must be >= 0", index)
		}
		if leg.EstimatedDistanceKm != nil && *leg.EstimatedDistanceKm < 0 {
			return apperrs.NewInvalidInput("route.legs[%d].estimatedDistanceKm must be >= 0", index)
		}
		if err := budget.NormalizeEstimatedCost(leg.EstimatedCost, budget.SourceAI); err != nil {
			return apperrs.NewInvalidInput("route.legs[%d].estimatedCost: %s", index, err.Error())
		}
		if leg.EstimatedCost != nil && leg.EstimatedCost.Category == "" {
			leg.EstimatedCost.Category = "transport"
		}
		if err := validateAndNormalizeSelectedTransportOption(leg.SelectedTransportOption, index); err != nil {
			return err
		}
		leg.Notes = strings.TrimSpace(leg.Notes)
	}

	prefs := &route.Preferences
	var err error
	prefs.PreferredModes, err = normalizeSupportedTransportList(prefs.PreferredModes, "route.preferences.preferredModes")
	if err != nil {
		return err
	}
	prefs.AvoidModes, err = normalizeSupportedTransportList(prefs.AvoidModes, "route.preferences.avoidModes")
	if err != nil {
		return err
	}
	if prefs.MaxTransferHoursPerDay != nil {
		if *prefs.MaxTransferHoursPerDay < 1 || *prefs.MaxTransferHoursPerDay > 24 {
			return apperrs.NewInvalidInput("route.preferences.maxTransferHoursPerDay must be between 1 and 24")
		}
	} else {
		value := defaultMaxTransferHoursDay
		prefs.MaxTransferHoursPerDay = &value
	}
	styles := make([]string, 0, len(prefs.TripStyles))
	seenStyles := map[string]struct{}{}
	for _, raw := range prefs.TripStyles {
		style := aggregate.NormalizeRouteToken(raw)
		if style == "" {
			continue
		}
		if _, ok := aggregate.SupportedTripStyles[style]; !ok {
			return apperrs.NewInvalidInput("route.preferences.tripStyles contains unsupported style %q", raw)
		}
		if _, exists := seenStyles[style]; exists {
			continue
		}
		seenStyles[style] = struct{}{}
		styles = append(styles, style)
	}
	prefs.TripStyles = styles
	return validateRouteDatesAgainstTrip(route, startDate, days)
}

func validateAndNormalizeSelectedTransportOption(option *aggregate.SelectedTransportOption, legIndex int) error {
	if option == nil {
		return nil
	}
	option.ID = strings.TrimSpace(option.ID)
	if option.ID == "" {
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.id is required", legIndex)
	}
	option.Mode = aggregate.NormalizeRouteToken(option.Mode)
	if _, ok := aggregate.SupportedTransportModes[option.Mode]; !ok {
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.mode is unsupported", legIndex)
	}
	option.Provider = aggregate.NormalizeRouteToken(option.Provider)
	if option.Provider == "" {
		option.Provider = "manual"
	}
	option.OperatorName = strings.TrimSpace(option.OperatorName)
	option.ServiceName = strings.TrimSpace(option.ServiceName)
	option.OriginName = strings.TrimSpace(option.OriginName)
	option.DestinationName = strings.TrimSpace(option.DestinationName)
	option.DepartureDate = strings.TrimSpace(option.DepartureDate)
	option.ArrivalDate = strings.TrimSpace(option.ArrivalDate)
	for _, value := range []struct {
		label string
		raw   string
	}{
		{"departureDate", option.DepartureDate},
		{"arrivalDate", option.ArrivalDate},
	} {
		if value.raw == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value.raw); err != nil {
			return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.%s must be in YYYY-MM-DD format", legIndex, value.label)
		}
	}
	option.DepartureTime = strings.TrimSpace(option.DepartureTime)
	option.ArrivalTime = strings.TrimSpace(option.ArrivalTime)
	for _, value := range []struct {
		label string
		raw   string
	}{
		{"departureTime", option.DepartureTime},
		{"arrivalTime", option.ArrivalTime},
	} {
		if value.raw == "" {
			continue
		}
		if _, err := time.Parse("15:04", value.raw); err != nil {
			return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.%s must be in HH:mm format", legIndex, value.label)
		}
	}
	if option.DurationMinutes < 0 {
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.durationMinutes must be >= 0", legIndex)
	}
	if option.Transfers < 0 {
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.transfers must be >= 0", legIndex)
	}
	option.Status = aggregate.NormalizeRouteToken(option.Status)
	if option.Status == "" {
		option.Status = "unknown"
	}
	switch option.Status {
	case "available", "limited", "unknown", "unavailable":
	default:
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.status is unsupported", legIndex)
	}
	option.Confidence = aggregate.NormalizeRouteToken(option.Confidence)
	if option.Confidence == "" {
		option.Confidence = budget.ConfidenceLow
	}
	switch option.Confidence {
	case budget.ConfidenceLow, budget.ConfidenceMedium, budget.ConfidenceHigh:
	default:
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.confidence is unsupported", legIndex)
	}
	if option.EstimatedPrice != nil {
		option.EstimatedPrice.Currency = strings.ToUpper(strings.TrimSpace(option.EstimatedPrice.Currency))
		if option.EstimatedPrice.Amount < 0 {
			return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.estimatedPrice.amount must be >= 0", legIndex)
		}
		if option.EstimatedPrice.Currency == "" {
			option.EstimatedPrice.Currency = defaultCurrency
		}
		if len(option.EstimatedPrice.Currency) != 3 {
			return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.estimatedPrice.currency must be a 3-letter code", legIndex)
		}
	}
	if err := validateOptionalURL(option.BookingURL, "bookingUrl", legIndex); err != nil {
		return err
	}
	if err := validateOptionalURL(option.ProviderURL, "providerUrl", legIndex); err != nil {
		return err
	}
	option.SelectedAt = strings.TrimSpace(option.SelectedAt)
	option.SelectedByUserID = strings.TrimSpace(option.SelectedByUserID)
	option.Warnings = cleanStringList(option.Warnings)
	return nil
}

func validateOptionalURL(value *string, field string, legIndex int) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		*value = ""
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperrs.NewInvalidInput("route.legs[%d].selectedTransportOption.%s must be an http or https URL", legIndex, field)
	}
	*value = trimmed
	return nil
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeSupportedTransportList(values []string, label string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		mode := aggregate.NormalizeRouteToken(raw)
		if mode == "" {
			continue
		}
		if _, ok := aggregate.SupportedTransportModes[mode]; !ok {
			return nil, apperrs.NewInvalidInput("%s contains unsupported mode %q", label, raw)
		}
		if _, exists := seen[mode]; exists {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out, nil
}

func validateRouteStopDates(stop *aggregate.RouteStop, index int) error {
	var arrival, departure *time.Time
	if stop.ArrivalDate != "" {
		parsed, err := time.Parse("2006-01-02", stop.ArrivalDate)
		if err != nil {
			return apperrs.NewInvalidInput("route.stops[%d].arrivalDate must be in YYYY-MM-DD format", index)
		}
		arrival = &parsed
	}
	if stop.DepartureDate != "" {
		parsed, err := time.Parse("2006-01-02", stop.DepartureDate)
		if err != nil {
			return apperrs.NewInvalidInput("route.stops[%d].departureDate must be in YYYY-MM-DD format", index)
		}
		departure = &parsed
	}
	if arrival != nil && departure != nil && departure.Before(*arrival) {
		return apperrs.NewInvalidInput("route.stops[%d].departureDate must be on or after arrivalDate", index)
	}
	return nil
}

func validateRouteDatesAgainstTrip(route *aggregate.TripRoute, startDate string, days int32) error {
	if strings.TrimSpace(startDate) == "" || days <= 0 {
		return nil
	}
	tripStart, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil
	}
	tripEnd := tripStart.AddDate(0, 0, int(days)-1)
	for index := range route.Stops {
		stop := route.Stops[index]
		for _, value := range []struct {
			label string
			raw   string
		}{
			{"arrivalDate", stop.ArrivalDate},
			{"departureDate", stop.DepartureDate},
		} {
			if value.raw == "" {
				continue
			}
			parsed, _ := time.Parse("2006-01-02", value.raw)
			if parsed.Before(tripStart) || parsed.After(tripEnd.AddDate(0, 0, 1)) {
				return apperrs.NewInvalidInput("route.stops[%d].%s should fit within trip dates", index, value.label)
			}
		}
	}
	return nil
}

func validCoordinates(coords *aggregate.Coordinates) bool {
	return coords != nil &&
		coords.Lat >= -90 && coords.Lat <= 90 &&
		coords.Lng >= -180 && coords.Lng <= 180
}

func deriveRouteDestination(route *aggregate.TripRoute) string {
	if route == nil || len(route.Stops) == 0 {
		return "Multi-destination route"
	}
	country := strings.TrimSpace(route.Stops[0].Country)
	if country != "" {
		allSame := true
		for _, stop := range route.Stops[1:] {
			if !strings.EqualFold(strings.TrimSpace(stop.Country), country) {
				allSame = false
				break
			}
		}
		if allSame {
			return country + " route"
		}
	}
	return strings.TrimSpace(route.Stops[0].Destination) + " route"
}

func dateString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func routeStopCount(route *aggregate.TripRoute) int {
	if route == nil {
		return 0
	}
	return len(route.Stops)
}
