package availability

import (
	"math"
	"strings"
)

// ticketmasterSupportedTypes is the set of itinerary item types Ticketmaster can
// plausibly answer for (ticketed live events). Anything outside this set is
// short-circuited to unknown without consuming provider quota.
var ticketmasterSupportedTypes = map[string]bool{
	"event":       true,
	"concert":     true,
	"gig":         true,
	"music":       true,
	"sports":      true,
	"sport":       true,
	"match":       true,
	"game":        true,
	"theatre":     true,
	"theater":     true,
	"show":        true,
	"musical":     true,
	"opera":       true,
	"festival":    true,
	"performance": true,
}

// ticketmasterSupportsItem reports whether the item type is one Ticketmaster can
// serve. When the type is empty we allow the call: some itineraries omit a type
// and the name alone ("Jazz concert at Blue Note") is enough to search.
func ticketmasterSupportsItem(item AvailabilityItem) bool {
	itemType := normalizeKey(item.Type)
	if itemType == "" {
		return true
	}
	return ticketmasterSupportedTypes[itemType]
}

// ticketmasterClassification maps an item type to a Discovery API segment name
// (classificationName) to narrow the search. An empty result means no segment
// filter is applied.
func ticketmasterClassification(itemType string) string {
	switch normalizeKey(itemType) {
	case "concert", "gig", "music":
		return "Music"
	case "sports", "sport", "match", "game":
		return "Sports"
	case "theatre", "theater", "show", "musical", "opera", "performance":
		return "Arts & Theatre"
	default:
		return ""
	}
}

// matchScore is a deterministic confidence score for how well a discovered event
// matches the requested item, broken down by signal for observability/tests.
type matchScore struct {
	total       float64
	titleScore  float64
	venueScore  float64
	cityScore   float64
	dateScore   float64
	typeScore   float64
	geoScore    float64
	matchedName string
}

// scoreTicketmasterEvent computes a deterministic 0..1 match confidence. The
// weights follow the task's recommended budget: title 0.35, venue 0.20, city
// 0.15, date 0.15, category 0.10, coordinate proximity 0.05.
func scoreTicketmasterEvent(req AvailabilitySearchRequest, event tmEvent) matchScore {
	score := matchScore{matchedName: strings.TrimSpace(event.Name)}

	queryName := firstNonEmpty(req.Item.Name, req.Item.PlaceName())
	score.titleScore = tokenCoverage(queryName, event.Name) * 0.35

	venue := firstVenue(event)
	if req.Item.Place != nil && strings.TrimSpace(req.Item.Place.Name) != "" && venue != nil {
		score.venueScore = tokenCoverage(req.Item.Place.Name, venue.Name) * 0.20
	}

	if venue != nil {
		cityMatch := tokenCoverage(req.Destination, venue.City.Name)
		if cityMatch == 0 {
			cityMatch = tokenCoverage(req.Destination, venue.Address.Line1)
		}
		score.cityScore = cityMatch * 0.15
	}

	if strings.TrimSpace(req.Date) != "" && event.Dates.Start.LocalDate == strings.TrimSpace(req.Date) {
		score.dateScore = 0.15
	}

	if segment := ticketmasterClassification(req.Item.Type); segment != "" {
		if eventSegment := strings.ToLower(strings.TrimSpace(eventSegmentName(event))); eventSegment != "" {
			if strings.EqualFold(eventSegment, segment) || strings.Contains(strings.ToLower(segment), eventSegment) {
				score.typeScore = 0.10
			}
		}
	}

	if req.Item.Place != nil && req.Item.Place.Latitude != nil && req.Item.Place.Longitude != nil && venue != nil {
		if lat, lng, ok := venueCoordinates(venue); ok {
			distanceKm := haversineKm(*req.Item.Place.Latitude, *req.Item.Place.Longitude, lat, lng)
			if distanceKm <= 2 {
				// Linear falloff: same point ~0.05, 2km ~0.0.
				score.geoScore = 0.05 * (1 - distanceKm/2)
			}
		}
	}

	score.total = clampConfidence(score.titleScore + score.venueScore + score.cityScore + score.dateScore + score.typeScore + score.geoScore)
	return score
}

// tokenCoverage returns the fraction of query tokens that appear in the
// candidate, ignoring very short and common filler tokens. It is symmetric-ish
// enough for name/venue matching and is fully deterministic.
func tokenCoverage(query, candidate string) float64 {
	queryTokens := significantTokens(query)
	if len(queryTokens) == 0 {
		return 0
	}
	candidateSet := make(map[string]struct{})
	for _, token := range significantTokens(candidate) {
		candidateSet[token] = struct{}{}
	}
	if len(candidateSet) == 0 {
		return 0
	}
	matched := 0
	for _, token := range queryTokens {
		if _, ok := candidateSet[token]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(queryTokens))
}

var tokenStopwords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "and": true, "at": true,
	"in": true, "on": true, "to": true, "for": true, "with": true, "tour": true,
	"visit": true, "see": true, "ticket": true, "tickets": true,
}

func significantTokens(value string) []string {
	fields := strings.Fields(normalizeText(value))
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 3 || tokenStopwords[field] {
			continue
		}
		tokens = append(tokens, field)
	}
	// If everything was filtered (very short name), fall back to raw fields so we
	// still have something to compare.
	if len(tokens) == 0 {
		return fields
	}
	return tokens
}

func firstVenue(event tmEvent) *tmVenue {
	if len(event.Embedded.Venues) == 0 {
		return nil
	}
	return &event.Embedded.Venues[0]
}

func eventSegmentName(event tmEvent) string {
	for _, classification := range event.Classifications {
		if name := strings.TrimSpace(classification.Segment.Name); name != "" {
			return name
		}
	}
	return ""
}

func venueCoordinates(venue *tmVenue) (float64, float64, bool) {
	lat, okLat := parseCoordinate(venue.Location.Latitude)
	lng, okLng := parseCoordinate(venue.Location.Longitude)
	if !okLat || !okLng {
		return 0, 0, false
	}
	return lat, lng, true
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*100) / 100
}

// haversineKm returns the great-circle distance between two coordinates in km.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := degToRad(lat2 - lat1)
	dLon := degToRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func degToRad(deg float64) float64 { return deg * math.Pi / 180 }
