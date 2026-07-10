package aggregate

import "strings"

const (
	TransportModeWalk            = "walk"
	TransportModeCar             = "car"
	TransportModeRentalCar       = "rental_car"
	TransportModeTrain           = "train"
	TransportModeBus             = "bus"
	TransportModeFlight          = "flight"
	TransportModeBoat            = "boat"
	TransportModeFerry           = "ferry"
	TransportModeBike            = "bike"
	TransportModePublicTransport = "public_transport"
	TransportModeHiking          = "hiking"
	TransportModeOther           = "other"
)

// SupportedTransportModes is the closed v1 vocabulary used by trip routes,
// transfer itinerary items, policy checks, AI prompts, and the web UI.
var SupportedTransportModes = map[string]struct{}{
	TransportModeWalk:            {},
	TransportModeCar:             {},
	TransportModeRentalCar:       {},
	TransportModeTrain:           {},
	TransportModeBus:             {},
	TransportModeFlight:          {},
	TransportModeBoat:            {},
	TransportModeFerry:           {},
	TransportModeBike:            {},
	TransportModePublicTransport: {},
	TransportModeHiking:          {},
	TransportModeOther:           {},
}

var SupportedTripStyles = map[string]struct{}{
	"city_break":     {},
	"road_trip":      {},
	"train_trip":     {},
	"backpacking":    {},
	"camping":        {},
	"hiking":         {},
	"island_hopping": {},
	"nature":         {},
	"beach":          {},
	"food":           {},
	"culture":        {},
	"adventure":      {},
	"family":         {},
	"romantic":       {},
	"low_budget":     {},
	"luxury":         {},
	"hidden_gem":     {},
}

var SupportedAccommodationHints = map[string]struct{}{
	"hotel":      {},
	"hostel":     {},
	"apartment":  {},
	"guesthouse": {},
	"campsite":   {},
	"cabin":      {},
	"campervan":  {},
	"home":       {},
	"other":      {},
	"unknown":    {},
}

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type RoutePlace struct {
	Name        string       `json:"name"`
	Country     string       `json:"country,omitempty"`
	Coordinates *Coordinates `json:"coordinates,omitempty"`
}

type TripRoute struct {
	Origin         *RoutePlace      `json:"origin,omitempty"`
	ReturnToOrigin bool             `json:"returnToOrigin"`
	Stops          []RouteStop      `json:"stops"`
	Legs           []RouteLeg       `json:"legs,omitempty"`
	Preferences    RoutePreferences `json:"preferences"`
}

type RouteStop struct {
	ID                string       `json:"id"`
	Destination       string       `json:"destination"`
	City              string       `json:"city,omitempty"`
	Country           string       `json:"country,omitempty"`
	ArrivalDate       string       `json:"arrivalDate,omitempty"`
	DepartureDate     string       `json:"departureDate,omitempty"`
	Nights            *int         `json:"nights,omitempty"`
	Coordinates       *Coordinates `json:"coordinates,omitempty"`
	AccommodationHint string       `json:"accommodationHint,omitempty"`
	Notes             *string      `json:"notes,omitempty"`
}

type RouteLeg struct {
	ID                       string         `json:"id"`
	FromStopID               string         `json:"fromStopId"`
	ToStopID                 string         `json:"toStopId"`
	FromName                 string         `json:"fromName,omitempty"`
	ToName                   string         `json:"toName,omitempty"`
	Mode                     string         `json:"mode"`
	DepartureDate            string         `json:"departureDate,omitempty"`
	EstimatedDurationMinutes *int           `json:"estimatedDurationMinutes,omitempty"`
	EstimatedDistanceKm      *float64       `json:"estimatedDistanceKm,omitempty"`
	EstimatedCost            *EstimatedCost `json:"estimatedCost,omitempty"`
	Notes                    string         `json:"notes,omitempty"`
	ProviderMetadata         map[string]any `json:"providerMetadata,omitempty"`
	Warnings                 []string       `json:"warnings,omitempty"`
}

type RoutePreferences struct {
	PreferredModes         []string `json:"preferredModes,omitempty"`
	AvoidModes             []string `json:"avoidModes,omitempty"`
	CarAvailable           bool     `json:"carAvailable"`
	MaxTransferHoursPerDay *int     `json:"maxTransferHoursPerDay,omitempty"`
	TripStyles             []string `json:"tripStyles,omitempty"`
}

type TransferDetails struct {
	LegID                    string         `json:"legId,omitempty"`
	From                     string         `json:"from"`
	To                       string         `json:"to"`
	Mode                     string         `json:"mode"`
	EstimatedDurationMinutes *int           `json:"estimatedDurationMinutes,omitempty"`
	EstimatedDistanceKm      *float64       `json:"estimatedDistanceKm,omitempty"`
	EstimatedCost            *EstimatedCost `json:"estimatedCost,omitempty"`
	BookingRequired          bool           `json:"bookingRequired"`
	Notes                    string         `json:"notes,omitempty"`
	Warnings                 []string       `json:"warnings,omitempty"`
}

// PublicRoute returns a sanitized route snapshot for public trip shares and
// exports. It drops private notes and provider metadata while preserving the
// visible route plan and estimates.
func PublicRoute(route *TripRoute) *TripRoute {
	if route == nil {
		return nil
	}
	out := *route
	if route.Origin != nil {
		origin := *route.Origin
		if route.Origin.Coordinates != nil {
			coords := *route.Origin.Coordinates
			origin.Coordinates = &coords
		}
		out.Origin = &origin
	}
	out.Stops = make([]RouteStop, 0, len(route.Stops))
	for _, stop := range route.Stops {
		clean := stop
		clean.Notes = nil
		if stop.Coordinates != nil {
			coords := *stop.Coordinates
			clean.Coordinates = &coords
		}
		out.Stops = append(out.Stops, clean)
	}
	out.Legs = make([]RouteLeg, 0, len(route.Legs))
	for _, leg := range route.Legs {
		clean := leg
		clean.Notes = ""
		clean.ProviderMetadata = nil
		if leg.EstimatedCost != nil {
			cost := *leg.EstimatedCost
			clean.EstimatedCost = &cost
		}
		out.Legs = append(out.Legs, clean)
	}
	out.Preferences.AvoidModes = nil
	return &out
}

func NormalizeRouteToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}
