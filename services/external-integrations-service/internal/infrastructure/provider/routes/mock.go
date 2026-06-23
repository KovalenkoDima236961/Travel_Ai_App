package routes

import (
	"context"
	"math"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const (
	// providerName is the stable identifier echoed back in responses and logs.
	providerName = "mock"

	// earthRadiusKm matches the Haversine implementation used by the Web App
	// fallback so server and client estimates stay comparable.
	earthRadiusKm = 6371.0

	// walkingSpeedKmPerHour is the flat pace assumed for walking estimates.
	walkingSpeedKmPerHour = 5.0

	// walkingRouteFactor inflates straight-line Haversine distance to roughly
	// approximate a real walking route. This is deliberately a rough constant;
	// it is not real routing.
	walkingRouteFactor = 1.25
)

// MockRouteProvider produces deterministic route estimates from Haversine
// distances. It performs no network calls and never reaches a third-party API.
type MockRouteProvider struct{}

func NewMockRouteProvider() *MockRouteProvider {
	return &MockRouteProvider{}
}

// EstimateRoute walks the stops in order, estimating each consecutive pair as a
// Haversine distance scaled by walkingRouteFactor. Totals are derived from the
// rounded segment values so the response is internally consistent (the total
// always equals the sum of the segments the caller sees).
func (p *MockRouteProvider) EstimateRoute(_ context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	segments := make([]entity.RouteSegment, 0, len(req.Stops)-1)
	var totalDistanceKm float64
	var totalDurationMinutes int

	for i := 1; i < len(req.Stops); i++ {
		from := req.Stops[i-1]
		to := req.Stops[i]

		distanceKm := round2(haversineDistanceKm(from, to) * walkingRouteFactor)
		durationMinutes := durationMinutesForDistance(distanceKm)

		segments = append(segments, entity.RouteSegment{
			FromName:        from.Name,
			ToName:          to.Name,
			DistanceKm:      distanceKm,
			DurationMinutes: durationMinutes,
		})

		totalDistanceKm += distanceKm
		totalDurationMinutes += durationMinutes
	}

	return &entity.RouteEstimate{
		Mode:            mode,
		Provider:        providerName,
		DistanceKm:      round2(totalDistanceKm),
		DurationMinutes: totalDurationMinutes,
		Segments:        segments,
	}, nil
}

// haversineDistanceKm is the great-circle distance between two stops in km.
func haversineDistanceKm(a, b entity.RouteStop) float64 {
	latDelta := toRadians(b.Latitude - a.Latitude)
	lonDelta := toRadians(b.Longitude - a.Longitude)

	sinLat := math.Sin(latDelta / 2)
	sinLon := math.Sin(lonDelta / 2)

	h := sinLat*sinLat +
		math.Cos(toRadians(a.Latitude))*math.Cos(toRadians(b.Latitude))*sinLon*sinLon

	centralAngle := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))

	return earthRadiusKm * centralAngle
}

// durationMinutesForDistance converts a distance into walking minutes at a flat
// 5 km/h pace, rounded to the nearest whole minute.
func durationMinutesForDistance(distanceKm float64) int {
	if distanceKm <= 0 {
		return 0
	}
	return int(math.Round((distanceKm / walkingSpeedKmPerHour) * 60))
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// round2 rounds to two decimal places. Rounding each segment and then summing
// keeps the response total exactly equal to the sum of the segment distances.
func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
