package entity

// RouteStop is a single ordered waypoint in a route-estimate request.
type RouteStop struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RouteEstimateRequest is the canonical input for a route estimation. Stops are
// visited in the order they are provided.
type RouteEstimateRequest struct {
	Mode  string      `json:"mode"`
	Stops []RouteStop `json:"stops"`
}

// RouteSegment is the estimate for a single consecutive stop pair.
type RouteSegment struct {
	FromName        string  `json:"fromName"`
	ToName          string  `json:"toName"`
	DistanceKm      float64 `json:"distanceKm"`
	DurationMinutes int     `json:"durationMinutes"`
}

// RouteEstimate is the canonical route-estimation result returned by provider
// adapters and exposed by the HTTP API. Totals are the sum of the segments.
type RouteEstimate struct {
	Mode            string         `json:"mode"`
	Provider        string         `json:"provider"`
	DistanceKm      float64        `json:"distanceKm"`
	DurationMinutes int            `json:"durationMinutes"`
	Segments        []RouteSegment `json:"segments"`
}
