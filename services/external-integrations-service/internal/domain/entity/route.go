package entity

import "strings"

const (
	RouteModeWalk            = "walk"
	RouteModeCar             = "car"
	RouteModeRentalCar       = "rental_car"
	RouteModeTrain           = "train"
	RouteModeBus             = "bus"
	RouteModeFlight          = "flight"
	RouteModeBoat            = "boat"
	RouteModeFerry           = "ferry"
	RouteModeBike            = "bike"
	RouteModePublicTransport = "public_transport"
	RouteModeHiking          = "hiking"
	RouteModeOther           = "other"
)

var SupportedRouteModes = map[string]struct{}{
	RouteModeWalk:            {},
	RouteModeCar:             {},
	RouteModeRentalCar:       {},
	RouteModeTrain:           {},
	RouteModeBus:             {},
	RouteModeFlight:          {},
	RouteModeBoat:            {},
	RouteModeFerry:           {},
	RouteModeBike:            {},
	RouteModePublicTransport: {},
	RouteModeHiking:          {},
	RouteModeOther:           {},
}

// RouteStop is a single ordered waypoint in a route-estimate request.
type RouteStop struct {
	Name      string   `json:"name"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
}

// RouteEstimateRequest is the canonical input for a route estimation. Stops are
// visited in the order they are provided.
type RouteEstimateRequest struct {
	Mode     string      `json:"mode"`
	Stops    []RouteStop `json:"stops"`
	From     *RouteStop  `json:"from,omitempty"`
	To       *RouteStop  `json:"to,omitempty"`
	Date     string      `json:"date,omitempty"`
	Currency string      `json:"currency,omitempty"`
	Warnings []string    `json:"-"`
}

// RouteSegment is the estimate for a single consecutive stop pair.
type RouteSegment struct {
	FromName                 string         `json:"fromName"`
	ToName                   string         `json:"toName"`
	DistanceKm               float64        `json:"distanceKm"`
	EstimatedDistanceKm      float64        `json:"estimatedDistanceKm,omitempty"`
	DurationMinutes          int            `json:"durationMinutes"`
	EstimatedDurationMinutes int            `json:"estimatedDurationMinutes,omitempty"`
	EstimatedCost            *EstimatedCost `json:"estimatedCost,omitempty"`
}

// EstimatedCost is a coarse, non-booking price estimate for a route transfer.
type EstimatedCost struct {
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Category   string  `json:"category"`
	Confidence string  `json:"confidence"`
	Source     string  `json:"source"`
	Note       string  `json:"note,omitempty"`
}

// RouteEstimate is the canonical route-estimation result returned by provider
// adapters and exposed by the HTTP API. Totals are the sum of the segments.
//
// RouteGeometry and FallbackUsed are optional and omitted when empty, so the
// response shape is unchanged for the default mock provider and for existing
// clients. RouteGeometry carries the provider-specific encoded path (e.g. the
// OpenRouteService polyline) when available; FallbackUsed is true when a real
// provider failed and the mock provider answered instead.
type RouteEstimate struct {
	Mode                     string         `json:"mode"`
	Provider                 string         `json:"provider"`
	DistanceKm               float64        `json:"distanceKm"`
	EstimatedDistanceKm      float64        `json:"estimatedDistanceKm,omitempty"`
	DurationMinutes          int            `json:"durationMinutes"`
	EstimatedDurationMinutes int            `json:"estimatedDurationMinutes,omitempty"`
	EstimatedCost            *EstimatedCost `json:"estimatedCost,omitempty"`
	Segments                 []RouteSegment `json:"segments"`
	RouteGeometry            any            `json:"routeGeometry,omitempty"`
	FallbackUsed             bool           `json:"fallbackUsed,omitempty"`
	Warnings                 []string       `json:"warnings,omitempty"`
}

// NormalizeRouteMode maps legacy provider-specific modes into the v1 transport
// mode vocabulary while keeping all new multi-modal modes explicit.
func NormalizeRouteMode(value string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	mode = strings.ReplaceAll(mode, "-", "_")
	mode = strings.ReplaceAll(mode, " ", "_")
	switch mode {
	case "walking":
		return RouteModeWalk
	case "driving":
		return RouteModeCar
	case "cycling":
		return RouteModeBike
	case "public_transportation", "transit":
		return RouteModePublicTransport
	default:
		return mode
	}
}

// NormalizeRouteEstimateRequest folds the newer from/to request shape into the
// legacy ordered stops list and normalises coordinate aliases.
func NormalizeRouteEstimateRequest(req RouteEstimateRequest) RouteEstimateRequest {
	req.Mode = NormalizeRouteMode(req.Mode)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	if len(req.Stops) == 0 && req.From != nil && req.To != nil {
		req.Stops = []RouteStop{*req.From, *req.To}
	}
	for i := range req.Stops {
		req.Stops[i] = NormalizeRouteStop(req.Stops[i])
	}
	return req
}

func NormalizeRouteStop(stop RouteStop) RouteStop {
	stop.Name = strings.TrimSpace(stop.Name)
	if stop.Lat != nil {
		stop.Latitude = *stop.Lat
	}
	if stop.Lng != nil {
		stop.Longitude = *stop.Lng
	}
	return stop
}
