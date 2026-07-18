package verification

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

var sectionWeights = map[Scope]int{
	ScopeTransport:     20,
	ScopePlace:         10,
	ScopeOpeningHours:  10,
	ScopePrice:         10,
	ScopeAvailability:  10,
	ScopeWeather:       10,
	ScopeRouteEstimate: 10,
	ScopeCalendarSync:  5,
	ScopeAccommodation: 10,
}

func Evaluate(in Input) Response {
	cfg := normalizeConfig(in.Config)
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tripID := uuidFromTrip(in.Trip)
	sections := []Section{
		evaluateTransport(in.Trip, now, cfg),
		evaluatePlaces(in.Itinerary, tripID, now, cfg),
		evaluateOpeningHours(in.Itinerary, tripID),
		evaluatePrices(in.Itinerary, tripID, now, cfg),
		evaluateAvailability(in.Itinerary, tripID, now, cfg),
		evaluateWeather(in.Trip, now, cfg),
		evaluateRouteEstimates(in.Trip, now, cfg),
		evaluateCalendar(in.Calendar, now, cfg, tripID),
		evaluateAccommodation(in.Trip, tripID),
	}
	for i := range sections {
		sections[i] = finalizeSection(sections[i])
	}
	sections = limitSectionDetails(sections, cfg.MaxDetails)

	all := make([]Detail, 0)
	summary := Summary{}
	for _, section := range sections {
		for _, detail := range section.Details {
			all = append(all, detail)
			countStatus(&summary, detail.Status)
		}
	}
	sortIssues(all)
	score := scoreSections(sections)
	return Response{
		TripID:             tripID,
		Score:              score,
		Level:              levelForScore(score),
		Summary:            summary,
		Sections:           sections,
		TopIssues:          topIssues(all, 8),
		RecommendedActions: recommendedActions(all, 6),
		ComputedAt:         now,
	}
}

func limitSectionDetails(sections []Section, maxDetails int) []Section {
	if maxDetails <= 0 {
		return sections
	}
	remaining := maxDetails
	for index := range sections {
		if remaining == 0 {
			sections[index].Details = []Detail{}
			continue
		}
		if len(sections[index].Details) > remaining {
			sections[index].Details = sections[index].Details[:remaining]
		}
		remaining -= len(sections[index].Details)
	}
	return sections
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if cfg == (Config{}) {
		return defaults
	}
	if cfg.WeatherStaleHoursNearTrip <= 0 {
		cfg.WeatherStaleHoursNearTrip = defaults.WeatherStaleHoursNearTrip
	}
	if cfg.WeatherStaleHoursFarTrip <= 0 {
		cfg.WeatherStaleHoursFarTrip = defaults.WeatherStaleHoursFarTrip
	}
	if cfg.TransportStaleDays <= 0 {
		cfg.TransportStaleDays = defaults.TransportStaleDays
	}
	if cfg.AvailabilityStaleHours <= 0 {
		cfg.AvailabilityStaleHours = defaults.AvailabilityStaleHours
	}
	if cfg.PriceStaleDays <= 0 {
		cfg.PriceStaleDays = defaults.PriceStaleDays
	}
	if cfg.PlaceStaleDays <= 0 {
		cfg.PlaceStaleDays = defaults.PlaceStaleDays
	}
	if cfg.RouteEstimateStaleDays <= 0 {
		cfg.RouteEstimateStaleDays = defaults.RouteEstimateStaleDays
	}
	if cfg.CalendarSyncStaleDays <= 0 {
		cfg.CalendarSyncStaleDays = defaults.CalendarSyncStaleDays
	}
	if cfg.NearTripDays <= 0 {
		cfg.NearTripDays = defaults.NearTripDays
	}
	if cfg.MaxDetails <= 0 {
		cfg.MaxDetails = defaults.MaxDetails
	}
	if cfg.PlaceMinConfidence <= 0 {
		cfg.PlaceMinConfidence = defaults.PlaceMinConfidence
	}
	return cfg
}

func evaluateTransport(trip *entity.Trip, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopeTransport, Details: []Detail{}}
	if trip == nil || trip.Route == nil || len(trip.Route.Legs) == 0 {
		section.Details = append(section.Details, naDetail(ScopeTransport, "trip", "transport", "Transport selection is not required for this trip."))
		return section
	}
	for _, leg := range trip.Route.Legs {
		title := nonEmpty(leg.FromName+" → "+leg.ToName, "Route transport")
		if leg.SelectedTransportOption == nil {
			if simpleLocalMode(leg.Mode) {
				section.Details = append(section.Details, naDetail(ScopeTransport, "route_leg", leg.ID, title+" is a local route."))
				continue
			}
			section.Details = append(section.Details, detail(ScopeTransport, "route_leg", leg.ID, StatusMissing, SourceUnknown, "", nil, nil, nil, title, "No selected transport option is attached to this route leg.", SeverityHigh, actionFor(ScopeTransport, trip.ID, leg.ID), map[string]any{"mode": leg.Mode}))
			continue
		}
		option := leg.SelectedTransportOption
		check := transportCheck(trip.CreationMetadata, leg.ID)
		provider := nonEmpty(metadataString(check, "provider"), option.Provider)
		checked := firstTime(parseTime(option.CheckedAt), parseTime(option.SelectedAt), metadataTime(leg.ProviderMetadata, "checkedAt"))
		checked = firstTime(metadataTime(check, "checkedAt"), checked)
		expires := expiry(checked, time.Duration(cfg.TransportStaleDays)*24*time.Hour)
		fallback := option.FallbackUsed || metadataBool(check, "fallbackUsed")
		source := sourceForProvider(provider, fallback)
		status := StatusVerified
		message := "Selected transport is provider-backed and recently checked."
		severity := SeverityInfo
		if provider == "" {
			status, source, message, severity = StatusEstimated, SourceUnknown, "Selected transport has no provider verification metadata.", SeverityWarning
		} else if isMockProvider(provider) || fallback {
			status, message, severity = StatusEstimated, "Selected transport uses mock or fallback data.", SeverityWarning
		} else if isUnavailable(option.Status) {
			status, message, severity = StatusUnavailable, "The selected transport provider reported this option unavailable.", SeverityCritical
		} else if isStale(now, expires) {
			status, message, severity = StatusStale, "Selected transport has not been checked recently.", SeverityHigh
		} else if isLowConfidence(option.Confidence) || len(option.Warnings) > 0 {
			status, message, severity = StatusNeedsReview, "Selected transport has provider warnings or low confidence.", SeverityWarning
		} else if check != nil && !metadataBool(check, "selectedOptionAvailable") {
			status, message, severity = StatusNeedsReview, "The current provider search did not return the selected transport option.", SeverityWarning
		}
		section.Details = append(section.Details, detail(ScopeTransport, "route_leg", leg.ID, status, source, provider, checked, expires, confidenceFromString(option.Confidence), title, message, severity, actionFor(ScopeTransport, trip.ID, leg.ID), map[string]any{"mode": leg.Mode, "fallbackUsed": fallback}))
	}
	return section
}

func evaluatePlaces(itinerary aggregate.Itinerary, tripID uuid.UUID, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopePlace, Details: []Detail{}}
	for dayIndex, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if !placeCandidate(item) {
				continue
			}
			id := itemID(day, dayIndex, itemIndex)
			if item.Place == nil {
				section.Details = append(section.Details, detail(ScopePlace, "itinerary_item", id, StatusMissing, SourceUnknown, "", nil, nil, nil, item.Name, "No provider place is attached to this itinerary item.", SeverityWarning, actionFor(ScopePlace, tripID, id), nil))
				continue
			}
			provider := nonEmpty(item.Place.Provider, metaProvider(item.PlaceEnrichment))
			confidence := metaConfidence(item.PlaceEnrichment)
			checked := parseMetaTime(item.PlaceEnrichment, "matched")
			expires := expiry(checked, time.Duration(cfg.PlaceStaleDays)*24*time.Hour)
			status, source, message, severity := StatusVerified, sourceForProvider(provider, false), "Place match is provider-backed.", SeverityInfo
			if isMockProvider(provider) {
				status, source, message, severity = StatusEstimated, SourceMock, "Place match uses mock data.", SeverityWarning
			} else if item.PlaceEnrichment == nil {
				status, source, message, severity = StatusNeedsReview, SourceManual, "Place was attached without match-confidence metadata.", SeverityWarning
			} else if strings.EqualFold(item.PlaceEnrichment.Status, "failed") {
				status, message, severity = StatusFailed, "Place verification failed.", SeverityWarning
			} else if confidence != nil && *confidence < cfg.PlaceMinConfidence {
				status, message, severity = StatusNeedsReview, "Place match confidence is low.", SeverityWarning
			} else if isStale(now, expires) {
				status, message, severity = StatusStale, "Place details have not been refreshed recently.", SeverityWarning
			}
			section.Details = append(section.Details, detail(ScopePlace, "itinerary_item", id, status, source, provider, checked, expires, confidence, item.Name, message, severity, actionFor(ScopePlace, tripID, id), nil))
		}
	}
	if len(section.Details) == 0 {
		section.Details = append(section.Details, naDetail(ScopePlace, "trip", "places", "No itinerary places need verification."))
	}
	return section
}

func evaluateOpeningHours(itinerary aggregate.Itinerary, tripID uuid.UUID) Section {
	section := Section{Scope: ScopeOpeningHours, Details: []Detail{}}
	for dayIndex, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if !placeCandidate(item) || item.Place == nil {
				continue
			}
			id := itemID(day, dayIndex, itemIndex)
			if len(item.Place.OpeningHours) == 0 {
				section.Details = append(section.Details, detail(ScopeOpeningHours, "itinerary_item", id, StatusNeedsReview, SourceUnknown, item.Place.Provider, nil, nil, nil, item.Name, "Opening hours are unknown for this place.", SeverityWarning, actionFor(ScopeOpeningHours, tripID, id), nil))
				continue
			}
			if scheduledOutsideOpeningHours(day.Date, item.Time, item.Place.OpeningHours) {
				section.Details = append(section.Details, detail(ScopeOpeningHours, "itinerary_item", id, StatusUnavailable, SourceProvider, item.Place.Provider, nil, nil, nil, item.Name, "This item appears to be scheduled outside the listed opening hours.", SeverityHigh, actionFor(ScopeOpeningHours, tripID, id), nil))
				continue
			}
			section.Details = append(section.Details, detail(ScopeOpeningHours, "itinerary_item", id, StatusVerified, SourceProvider, item.Place.Provider, nil, nil, nil, item.Name, "Opening hours are available for the planned visit.", SeverityInfo, actionFor(ScopeOpeningHours, tripID, id), nil))
		}
	}
	if len(section.Details) == 0 {
		section.Details = append(section.Details, naDetail(ScopeOpeningHours, "trip", "opening-hours", "No time-sensitive place visits need opening-hours checks."))
	}
	return section
}

func evaluatePrices(itinerary aggregate.Itinerary, tripID uuid.UUID, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopePrice, Details: []Detail{}}
	for dayIndex, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if !priceRelevant(item) {
				continue
			}
			id := itemID(day, dayIndex, itemIndex)
			if item.EstimatedCost == nil || item.EstimatedCost.Amount == nil {
				section.Details = append(section.Details, detail(ScopePrice, "itinerary_item", id, StatusMissing, SourceUnknown, "", nil, nil, nil, item.Name, "This ticketed or paid item has no price estimate.", SeverityWarning, actionFor(ScopePrice, tripID, id), nil))
				continue
			}
			provider := priceMetaProvider(item.PriceEnrichment)
			source := sourceForCost(item.EstimatedCost, provider)
			checked := parsePriceTime(item.PriceEnrichment)
			expires := expiry(checked, time.Duration(cfg.PriceStaleDays)*24*time.Hour)
			status, message, severity := StatusVerified, "Price is backed by a trusted source.", SeverityInfo
			if source == SourceAI || source == SourceMock || source == SourceFallback || source == SourceHeuristic || source == SourceUnknown {
				status, message, severity = StatusEstimated, "This price is an estimate, not a booking confirmation.", SeverityWarning
			} else if source == SourceProvider && isStale(now, expires) {
				status, message, severity = StatusStale, "Provider price has not been refreshed recently.", SeverityWarning
			} else if item.PriceEnrichment != nil && strings.EqualFold(item.PriceEnrichment.Status, "failed") {
				status, message, severity = StatusFailed, "Price verification failed.", SeverityWarning
			}
			section.Details = append(section.Details, detail(ScopePrice, "itinerary_item", id, status, source, provider, checked, expires, metaPriceConfidence(item.PriceEnrichment), item.Name, message, severity, actionFor(ScopePrice, tripID, id), nil))
		}
	}
	if len(section.Details) == 0 {
		section.Details = append(section.Details, naDetail(ScopePrice, "trip", "prices", "No ticketed or paid item prices need verification."))
	}
	return section
}

func evaluateAvailability(itinerary aggregate.Itinerary, tripID uuid.UUID, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopeAvailability, Details: []Detail{}}
	for dayIndex, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if !priceRelevant(item) {
				continue
			}
			id := itemID(day, dayIndex, itemIndex)
			meta := item.AvailabilityCheck
			if meta == nil || strings.TrimSpace(meta.Status) == "" {
				section.Details = append(section.Details, detail(ScopeAvailability, "itinerary_item", id, StatusMissing, SourceUnknown, "", nil, nil, nil, item.Name, "Availability has not been checked for this paid activity.", SeverityWarning, actionFor(ScopeAvailability, tripID, id), nil))
				continue
			}
			provider := meta.Provider
			checked := parseTime(meta.CheckedAt)
			expires := expiry(checked, time.Duration(cfg.AvailabilityStaleHours)*time.Hour)
			source := sourceForProvider(provider, meta.FallbackUsed)
			status, message, severity := StatusVerified, "Availability was recently checked with the provider.", SeverityInfo
			if isUnavailable(meta.Status) {
				status, message, severity = StatusUnavailable, "The provider reports this activity unavailable.", SeverityHigh
			} else if isMockProvider(provider) || meta.FallbackUsed {
				status, message, severity = StatusEstimated, "Availability uses mock or fallback data.", SeverityWarning
			} else if isStale(now, expires) {
				status, message, severity = StatusStale, "Availability check has expired.", SeverityWarning
			} else if isLowConfidenceFloat(meta.MatchConfidence) || strings.Contains(strings.ToLower(meta.Status), "limited") || meta.PriceChanged {
				status, message, severity = StatusNeedsReview, "Availability has a warning that needs review.", SeverityWarning
			}
			section.Details = append(section.Details, detail(ScopeAvailability, "itinerary_item", id, status, source, provider, checked, expires, floatPtrIfPositive(meta.MatchConfidence), item.Name, message, severity, actionFor(ScopeAvailability, tripID, id), map[string]any{"fallbackUsed": meta.FallbackUsed, "priceChanged": meta.PriceChanged}))
		}
	}
	if len(section.Details) == 0 {
		section.Details = append(section.Details, naDetail(ScopeAvailability, "trip", "availability", "No ticketed or paid activities need availability checks."))
	}
	return section
}

func evaluateWeather(trip *entity.Trip, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopeWeather, Details: []Detail{}}
	if trip == nil || trip.StartDate == nil || strings.TrimSpace(trip.Destination) == "" {
		section.Details = append(section.Details, naDetail(ScopeWeather, "trip", "weather", "Weather cannot be checked until destination and dates are set."))
		return section
	}
	metadata := verificationMetadata(trip.CreationMetadata, "weather")
	if metadata == nil {
		section.Details = append(section.Details, detail(ScopeWeather, "trip", trip.ID.String(), StatusMissing, SourceUnknown, "", nil, nil, nil, "Weather forecast", "Weather has not been refreshed for this trip.", SeverityWarning, actionFor(ScopeWeather, trip.ID, ""), nil))
		return section
	}
	provider := metadataString(metadata, "provider")
	checked := metadataTime(metadata, "checkedAt")
	nearTrip := trip.StartDate.Before(now.AddDate(0, 0, cfg.NearTripDays+1))
	staleAfter := time.Duration(cfg.WeatherStaleHoursFarTrip) * time.Hour
	if nearTrip {
		staleAfter = time.Duration(cfg.WeatherStaleHoursNearTrip) * time.Hour
	}
	expires := expiry(checked, staleAfter)
	fallback := metadataBool(metadata, "fallbackUsed")
	status, source, message, severity := StatusVerified, sourceForProvider(provider, fallback), "Weather forecast is provider-backed and recently checked.", SeverityInfo
	if isMockProvider(provider) || fallback {
		status, message, severity = StatusEstimated, "Weather forecast uses mock or fallback data.", SeverityWarning
	} else if isStale(now, expires) {
		status, message, severity = StatusStale, "Weather forecast has not been refreshed recently.", SeverityWarning
	}
	section.Details = append(section.Details, detail(ScopeWeather, "trip", trip.ID.String(), status, source, provider, checked, expires, nil, "Weather forecast", message, severity, actionFor(ScopeWeather, trip.ID, ""), map[string]any{"fallbackUsed": fallback}))
	return section
}

func evaluateRouteEstimates(trip *entity.Trip, now time.Time, cfg Config) Section {
	section := Section{Scope: ScopeRouteEstimate, Details: []Detail{}}
	if trip == nil || trip.Route == nil || len(trip.Route.Legs) == 0 {
		section.Details = append(section.Details, naDetail(ScopeRouteEstimate, "trip", "route-estimates", "No inter-city route estimates are required."))
		return section
	}
	for _, leg := range trip.Route.Legs {
		title := nonEmpty(leg.FromName+" → "+leg.ToName, "Route estimate")
		if leg.EstimatedDurationMinutes == nil && leg.EstimatedDistanceKm == nil {
			section.Details = append(section.Details, detail(ScopeRouteEstimate, "route_leg", leg.ID, StatusMissing, SourceUnknown, "", nil, nil, nil, title, "This route leg has no distance or duration estimate.", SeverityWarning, actionFor(ScopeRouteEstimate, trip.ID, leg.ID), nil))
			continue
		}
		provider := metadataString(leg.ProviderMetadata, "provider")
		checked := metadataTime(leg.ProviderMetadata, "checkedAt")
		fallback := metadataBool(leg.ProviderMetadata, "fallbackUsed")
		if provider == "" && leg.SelectedTransportOption != nil {
			provider, checked = leg.SelectedTransportOption.Provider, firstTime(checked, parseTime(leg.SelectedTransportOption.CheckedAt), parseTime(leg.SelectedTransportOption.SelectedAt))
			fallback = fallback || leg.SelectedTransportOption.FallbackUsed
		}
		expires := expiry(checked, time.Duration(cfg.RouteEstimateStaleDays)*24*time.Hour)
		source := sourceForProvider(provider, fallback)
		status, message, severity := StatusVerified, "Route duration and distance are provider-backed.", SeverityInfo
		if provider == "" || isMockProvider(provider) || fallback {
			status, message, severity = StatusEstimated, "This route estimate is heuristic, mock, or fallback data.", SeverityWarning
		} else if isStale(now, expires) {
			status, message, severity = StatusStale, "Route estimate has not been refreshed recently.", SeverityWarning
		}
		section.Details = append(section.Details, detail(ScopeRouteEstimate, "route_leg", leg.ID, status, source, provider, checked, expires, nil, title, message, severity, actionFor(ScopeRouteEstimate, trip.ID, leg.ID), map[string]any{"fallbackUsed": fallback, "mode": leg.Mode}))
	}
	return section
}

func evaluateCalendar(calendar *CalendarState, now time.Time, cfg Config, tripID uuid.UUID) Section {
	section := Section{Scope: ScopeCalendarSync, Details: []Detail{}}
	if calendar == nil || !calendar.Connected {
		section.Details = append(section.Details, naDetail(ScopeCalendarSync, "trip", "calendar", "No connected calendar requires sync verification."))
		return section
	}
	if !calendar.Synced {
		section.Details = append(section.Details, detail(ScopeCalendarSync, "trip", tripID.String(), StatusMissing, SourceCalendarSync, calendar.Provider, nil, nil, nil, "Calendar sync", "A connected calendar has not been synced to this itinerary.", SeverityWarning, actionFor(ScopeCalendarSync, tripID, ""), nil))
		return section
	}
	expires := expiry(calendar.LastSyncedAt, time.Duration(cfg.CalendarSyncStaleDays)*24*time.Hour)
	status, message, severity := StatusVerified, "Calendar is synced to the current itinerary revision.", SeverityInfo
	if calendar.OutOfDate || calendar.SyncedItineraryRevision < calendar.CurrentItineraryRevision {
		status, message, severity = StatusStale, "Calendar sync is older than the current itinerary revision.", SeverityWarning
	} else if isStale(now, expires) {
		status, message, severity = StatusStale, "Calendar sync has not been refreshed recently.", SeverityWarning
	}
	section.Details = append(section.Details, detail(ScopeCalendarSync, "trip", tripID.String(), status, SourceCalendarSync, calendar.Provider, calendar.LastSyncedAt, expires, nil, "Calendar sync", message, severity, actionFor(ScopeCalendarSync, tripID, ""), nil))
	return section
}

func evaluateAccommodation(trip *entity.Trip, tripID uuid.UUID) Section {
	section := Section{Scope: ScopeAccommodation, Details: []Detail{}}
	if trip == nil || trip.Days <= 1 {
		section.Details = append(section.Details, naDetail(ScopeAccommodation, "trip", "accommodation", "Accommodation is optional for this trip."))
		return section
	}
	if trip.Accommodation == nil {
		section.Details = append(section.Details, detail(ScopeAccommodation, "trip", tripID.String(), StatusMissing, SourceUnknown, "", nil, nil, nil, "Accommodation", "No accommodation details are saved for this multi-day trip.", SeverityWarning, actionFor(ScopeAccommodation, tripID, ""), nil))
		return section
	}
	accommodation := trip.Accommodation
	if strings.TrimSpace(accommodation.Address) == "" && accommodation.Place == nil {
		section.Details = append(section.Details, detail(ScopeAccommodation, "accommodation", "accommodation", StatusNeedsReview, SourceManual, "", nil, nil, nil, nonEmpty(accommodation.Name, "Accommodation"), "Accommodation is missing an address or place reference.", SeverityWarning, actionFor(ScopeAccommodation, tripID, ""), nil))
		return section
	}
	if strings.TrimSpace(accommodation.CheckInDate) == "" || strings.TrimSpace(accommodation.CheckOutDate) == "" {
		section.Details = append(section.Details, detail(ScopeAccommodation, "accommodation", "accommodation", StatusNeedsReview, SourceManual, "", nil, nil, nil, nonEmpty(accommodation.Name, "Accommodation"), "Accommodation dates need review.", SeverityWarning, actionFor(ScopeAccommodation, tripID, ""), nil))
		return section
	}
	section.Details = append(section.Details, detail(ScopeAccommodation, "accommodation", "accommodation", StatusVerified, SourceManual, "", nil, nil, nil, nonEmpty(accommodation.Name, "Accommodation"), "Accommodation has an address/place and check-in dates.", SeverityInfo, actionFor(ScopeAccommodation, tripID, ""), nil))
	return section
}

func finalizeSection(section Section) Section {
	if len(section.Details) == 0 {
		section.Details = []Detail{naDetail(section.Scope, "trip", string(section.Scope), "No verification data applies.")}
	}
	sum, applicable := 0, 0
	worst := StatusVerified
	for _, detail := range section.Details {
		if detail.Status != StatusNotApplicable {
			sum += statusScore(detail.Status)
			applicable++
		}
		if statusPriority(detail.Status) > statusPriority(worst) {
			worst = detail.Status
		}
	}
	if applicable == 0 {
		section.Score, section.Status = 100, StatusNotApplicable
		return section
	}
	section.Score = sum / applicable
	section.Status = worst
	return section
}

func scoreSections(sections []Section) int {
	weighted, total := 0, 0
	for _, section := range sections {
		if section.Status != StatusNotApplicable {
			weight := sectionWeights[section.Scope]
			weighted += section.Score * weight
			total += weight
		}
	}
	if total == 0 {
		return 100
	}
	return weighted / total
}

func levelForScore(score int) Level {
	switch {
	case score >= 90:
		return LevelReady
	case score >= 75:
		return LevelMostlyReady
	case score >= 50:
		return LevelNeedsReview
	default:
		return LevelNotReady
	}
}
func statusScore(status Status) int {
	switch status {
	case StatusVerified:
		return 100
	case StatusNeedsReview:
		return 65
	case StatusEstimated:
		return 55
	case StatusStale:
		return 45
	case StatusMissing:
		return 25
	case StatusUnavailable:
		return 10
	case StatusFailed:
		return 20
	default:
		return 100
	}
}
func statusPriority(status Status) int {
	switch status {
	case StatusUnavailable:
		return 8
	case StatusMissing:
		return 7
	case StatusStale:
		return 6
	case StatusFailed:
		return 5
	case StatusNeedsReview:
		return 4
	case StatusEstimated:
		return 3
	case StatusNotApplicable:
		return 0
	default:
		return 1
	}
}
func countStatus(summary *Summary, status Status) {
	switch status {
	case StatusVerified:
		summary.VerifiedCount++
	case StatusNeedsReview:
		summary.NeedsReviewCount++
	case StatusEstimated:
		summary.EstimatedCount++
	case StatusStale:
		summary.StaleCount++
	case StatusMissing:
		summary.MissingCount++
	case StatusUnavailable:
		summary.UnavailableCount++
	case StatusFailed:
		summary.FailedCount++
	}
}

func topIssues(details []Detail, limit int) []Detail {
	out := make([]Detail, 0, limit)
	for _, item := range details {
		if item.Status != StatusVerified && item.Status != StatusNotApplicable {
			out = append(out, item)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}
func recommendedActions(details []Detail, limit int) []Action {
	out, seen := make([]Action, 0, limit), map[string]struct{}{}
	for _, item := range details {
		if item.Action == nil || item.Status == StatusVerified || item.Status == StatusNotApplicable {
			continue
		}
		if _, ok := seen[item.Action.Type]; ok {
			continue
		}
		seen[item.Action.Type] = struct{}{}
		out = append(out, *item.Action)
		if len(out) == limit {
			break
		}
	}
	return out
}
func sortIssues(items []Detail) {
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := statusPriority(items[i].Status), statusPriority(items[j].Status)
		if pi != pj {
			return pi > pj
		}
		si, sj := severityPriority(items[i].Severity), severityPriority(items[j].Severity)
		if si != sj {
			return si > sj
		}
		return items[i].Title < items[j].Title
	})
}
func severityPriority(value Severity) int {
	switch value {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}

func detail(scope Scope, entityType, entityID string, status Status, source Source, provider string, checked, expires *time.Time, confidence *float64, title, message string, severity Severity, action *Action, metadata map[string]any) Detail {
	return Detail{Scope: scope, EntityType: entityType, EntityID: entityID, Status: status, Source: source, Provider: strings.TrimSpace(provider), CheckedAt: checked, ExpiresAt: expires, Confidence: confidence, Title: nonEmpty(strings.TrimSpace(title), "Trip detail"), Message: message, Severity: severity, Action: action, Metadata: metadata}
}
func naDetail(scope Scope, entityType, entityID, message string) Detail {
	return detail(scope, entityType, entityID, StatusNotApplicable, SourceUnknown, "", nil, nil, nil, string(scope), message, SeverityInfo, nil, nil)
}

func actionFor(scope Scope, tripID uuid.UUID, entityID string) *Action {
	path := fmt.Sprintf("/trips/%s?tab=verification", tripID.String())
	switch scope {
	case ScopeTransport:
		path = fmt.Sprintf("/trips/%s?tab=route&legId=%s&action=verify", tripID.String(), entityID)
		return &Action{Type: "recheck_transport", Label: "Recheck transport", Href: path}
	case ScopeWeather:
		return &Action{Type: "refresh_weather", Label: "Refresh weather", Href: path}
	case ScopePlace:
		return &Action{Type: "refresh_place_details", Label: "Refresh place details", Href: path}
	case ScopeOpeningHours:
		return &Action{Type: "review_opening_hours", Label: "Review opening hours", Href: path}
	case ScopePrice:
		return &Action{Type: "refresh_price", Label: "Refresh price", Href: path}
	case ScopeAvailability:
		return &Action{Type: "check_availability", Label: "Check availability", Href: fmt.Sprintf("/trips/%s?tab=itinerary&itemId=%s&action=availability", tripID.String(), entityID)}
	case ScopeRouteEstimate:
		return &Action{Type: "open_route", Label: "Open route", Href: fmt.Sprintf("/trips/%s?tab=route", tripID.String())}
	case ScopeCalendarSync:
		return &Action{Type: "update_calendar_sync", Label: "Update calendar sync", Href: fmt.Sprintf("/trips/%s?tab=calendar", tripID.String())}
	case ScopeAccommodation:
		return &Action{Type: "add_accommodation", Label: "Add accommodation", Href: fmt.Sprintf("/trips/%s?tab=accommodation", tripID.String())}
	}
	return nil
}

func uuidFromTrip(trip *entity.Trip) uuid.UUID {
	if trip == nil {
		return uuid.Nil
	}
	return trip.ID
}
func itemID(day aggregate.ItineraryDay, dayIndex, itemIndex int) string {
	return fmt.Sprintf("day_%d_item_%d", dayNumber(day, dayIndex), itemIndex)
}
func dayNumber(day aggregate.ItineraryDay, index int) int {
	if day.Day > 0 {
		return day.Day
	}
	return index + 1
}
func firstTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil && !value.IsZero() {
			valueCopy := value.UTC()
			return &valueCopy
		}
	}
	return nil
}
func expiry(checked *time.Time, ttl time.Duration) *time.Time {
	if checked == nil || ttl <= 0 {
		return nil
	}
	value := checked.Add(ttl)
	return &value
}
func isStale(now time.Time, expires *time.Time) bool { return expires == nil || !now.Before(*expires) }
func parseTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}
func metadataTime(metadata map[string]any, key string) *time.Time {
	if metadata == nil {
		return nil
	}
	value, _ := metadata[key].(string)
	return parseTime(value)
}
func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}
func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	value, _ := metadata[key].(bool)
	return value
}
func verificationMetadata(metadata map[string]any, key string) map[string]any {
	root, _ := metadata["verification"].(map[string]any)
	if root == nil {
		return nil
	}
	value, _ := root[key].(map[string]any)
	return value
}
func transportCheck(metadata map[string]any, legID string) map[string]any {
	root := verificationMetadata(metadata, "transport")
	if root == nil {
		return nil
	}
	value, _ := root[legID].(map[string]any)
	return value
}
func sourceForProvider(provider string, fallback bool) Source {
	if fallback {
		return SourceFallback
	}
	if isMockProvider(provider) {
		return SourceMock
	}
	if strings.TrimSpace(provider) == "" {
		return SourceUnknown
	}
	return SourceProvider
}
func sourceForCost(cost *aggregate.EstimatedCost, provider string) Source {
	if cost == nil {
		return SourceUnknown
	}
	switch strings.ToLower(strings.TrimSpace(cost.Source)) {
	case "manual":
		return SourceManual
	case "receipt":
		return SourceReceipt
	case "provider":
		return sourceForProvider(provider, false)
	case "mock":
		return SourceMock
	case "fallback":
		return SourceFallback
	case "heuristic":
		return SourceHeuristic
	case "imported":
		return SourceImported
	case "ai":
		return SourceAI
	default:
		if provider != "" {
			return sourceForProvider(provider, false)
		}
		return SourceUnknown
	}
}
func isMockProvider(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "mock" || strings.Contains(value, "mock") || value == "fallback"
}
func isUnavailable(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "unavailable" || value == "sold_out" || value == "not_available" || value == "none"
}
func isLowConfidence(value string) bool       { return strings.EqualFold(strings.TrimSpace(value), "low") }
func isLowConfidenceFloat(value float64) bool { return value > 0 && value < 0.75 }
func confidenceFromString(value string) *float64 {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return floatPtr(0.9)
	case "medium":
		return floatPtr(0.65)
	case "low":
		return floatPtr(0.35)
	default:
		return nil
	}
}
func floatPtrIfPositive(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	return floatPtr(value)
}
func floatPtr(value float64) *float64 { return &value }
func metaConfidence(meta *aggregate.PlaceEnrichmentMeta) *float64 {
	if meta == nil || meta.Confidence <= 0 {
		return nil
	}
	return floatPtr(meta.Confidence)
}
func metaPriceConfidence(meta *aggregate.PriceEnrichmentMeta) *float64 {
	if meta == nil || meta.MatchConfidence <= 0 {
		return nil
	}
	return floatPtr(meta.MatchConfidence)
}
func metaProvider(meta *aggregate.PlaceEnrichmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.Provider)
}
func priceMetaProvider(meta *aggregate.PriceEnrichmentMeta) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta.Provider)
}
func parseMetaTime(meta *aggregate.PlaceEnrichmentMeta, _ string) *time.Time {
	if meta == nil {
		return nil
	}
	return parseTime(meta.MatchedAt)
}
func parsePriceTime(meta *aggregate.PriceEnrichmentMeta) *time.Time {
	if meta == nil {
		return nil
	}
	return parseTime(meta.UpdatedAt)
}
func nonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
func simpleLocalMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case aggregate.TransportModeWalk, aggregate.TransportModeBike, aggregate.TransportModeHiking:
		return true
	default:
		return false
	}
}
func placeCandidate(item aggregate.ItineraryItem) bool {
	switch strings.ToLower(strings.TrimSpace(item.Type)) {
	case "place", "food", "activity", "museum", "landmark", "restaurant", "cafe", "market", "park", "attraction", "viewpoint":
		return true
	default:
		return false
	}
}
func priceRelevant(item aggregate.ItineraryItem) bool {
	return item.EstimatedCost != nil || item.PriceEnrichment != nil || item.AvailabilityCheck != nil
}
func scheduledOutsideOpeningHours(date, clock string, hours []aggregate.OpeningHoursInterval) bool {
	if strings.TrimSpace(date) == "" || strings.TrimSpace(clock) == "" {
		return false
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	target, ok := minutes(clock)
	if !ok {
		return false
	}
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	matching := false
	for _, interval := range hours {
		if interval.DayOfWeek != weekday {
			continue
		}
		matching = true
		open, validOpen := minutes(interval.Open)
		close, validClose := minutes(interval.Close)
		if validOpen && validClose && target >= open && target <= close {
			return false
		}
	}
	return matching
}
func minutes(value string) (int, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}
