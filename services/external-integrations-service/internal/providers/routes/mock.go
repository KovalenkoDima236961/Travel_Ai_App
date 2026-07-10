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

	// Per-mode flat pace assumptions for duration estimates. These are rough
	// constants, not real routing: they keep mock estimates plausible and, just
	// as importantly, keep the mock usable as a fallback for the driving and
	// cycling modes a real provider (ORS) supports.
	walkingSpeedKmPerHour         = 5.0
	bikeSpeedKmPerHour            = 15.0
	hikingSpeedKmPerHour          = 3.5
	carSpeedKmPerHour             = 80.0
	busSpeedKmPerHour             = 60.0
	trainSpeedKmPerHour           = 100.0
	flightSpeedKmPerHour          = 700.0
	boatSpeedKmPerHour            = 35.0
	publicTransportSpeedKmPerHour = 35.0
	otherSpeedKmPerHour           = 50.0

	// routeDistanceFactor inflates straight-line Haversine distance to roughly
	// approximate a real route. This is deliberately a rough constant; it is not
	// real routing.
	routeDistanceFactor = 1.25
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
	req = entity.NormalizeRouteEstimateRequest(req)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))

	segments := make([]entity.RouteSegment, 0, len(req.Stops)-1)
	var totalDistanceKm float64
	var totalDurationMinutes int
	var totalCost float64

	for i := 1; i < len(req.Stops); i++ {
		from := req.Stops[i-1]
		to := req.Stops[i]

		distanceKm := round2(haversineDistanceKm(from, to) * routeDistanceFactor)
		durationMinutes := durationMinutesForMode(distanceKm, mode)
		cost := estimatedCostForMode(distanceKm, mode, req.Currency)

		segments = append(segments, entity.RouteSegment{
			FromName:                 from.Name,
			ToName:                   to.Name,
			DistanceKm:               distanceKm,
			EstimatedDistanceKm:      distanceKm,
			DurationMinutes:          durationMinutes,
			EstimatedDurationMinutes: durationMinutes,
			EstimatedCost:            cost,
		})

		totalDistanceKm += distanceKm
		totalDurationMinutes += durationMinutes
		totalCost += cost.Amount
	}
	totalDistanceKm = round2(totalDistanceKm)

	return &entity.RouteEstimate{
		Mode:                     mode,
		Provider:                 providerName,
		DistanceKm:               totalDistanceKm,
		EstimatedDistanceKm:      totalDistanceKm,
		DurationMinutes:          totalDurationMinutes,
		EstimatedDurationMinutes: totalDurationMinutes,
		EstimatedCost: &entity.EstimatedCost{
			Amount:     round2(totalCost),
			Currency:   req.Currency,
			Category:   "transport",
			Confidence: "low",
			Source:     "mock",
			Note:       "Estimated transfer cost; no live schedules or ticket prices are included.",
		},
		Segments: segments,
		Warnings: warningsForMode(mode),
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

func durationMinutesForMode(distanceKm float64, mode string) int {
	switch mode {
	case entity.RouteModeCar, entity.RouteModeRentalCar:
		return durationMinutesForDistance(distanceKm, carSpeedKmPerHour) + 20
	case entity.RouteModeBus:
		return durationMinutesForDistance(distanceKm, busSpeedKmPerHour) + 30
	case entity.RouteModeTrain:
		return durationMinutesForDistance(distanceKm, trainSpeedKmPerHour) + 20
	case entity.RouteModeFlight:
		return 180 + durationMinutesForDistance(distanceKm, flightSpeedKmPerHour)
	case entity.RouteModeBoat, entity.RouteModeFerry:
		return durationMinutesForDistance(distanceKm, boatSpeedKmPerHour) + 30
	case entity.RouteModeBike:
		return durationMinutesForDistance(distanceKm, bikeSpeedKmPerHour)
	case entity.RouteModeHiking:
		return durationMinutesForDistance(distanceKm, hikingSpeedKmPerHour)
	case entity.RouteModePublicTransport:
		return durationMinutesForDistance(distanceKm, publicTransportSpeedKmPerHour) + 30
	case entity.RouteModeOther:
		return durationMinutesForDistance(distanceKm, otherSpeedKmPerHour)
	default:
		return durationMinutesForDistance(distanceKm, walkingSpeedKmPerHour)
	}
}

// durationMinutesForDistance converts a distance into minutes at the given flat
// pace, rounded to the nearest whole minute.
func durationMinutesForDistance(distanceKm, speedKmPerHour float64) int {
	if distanceKm <= 0 || speedKmPerHour <= 0 {
		return 0
	}
	return int(math.Round((distanceKm / speedKmPerHour) * 60))
}

func estimatedCostForMode(distanceKm float64, mode, currency string) *entity.EstimatedCost {
	amount := 0.0
	switch mode {
	case entity.RouteModeCar:
		amount = distanceKm * 0.18
	case entity.RouteModeRentalCar:
		amount = distanceKm*0.18 + 45
	case entity.RouteModeBus:
		amount = distanceKm * 0.08
	case entity.RouteModeTrain:
		amount = distanceKm * 0.12
	case entity.RouteModeFlight:
		amount = math.Max(50, distanceKm*0.15)
	case entity.RouteModeBoat, entity.RouteModeFerry:
		amount = distanceKm * 0.20
	case entity.RouteModePublicTransport:
		amount = distanceKm * 0.10
	case entity.RouteModeBike, entity.RouteModeHiking, entity.RouteModeWalk:
		amount = 0
	default:
		amount = distanceKm * 0.10
	}
	return &entity.EstimatedCost{
		Amount:     round2(amount),
		Currency:   currency,
		Category:   "transport",
		Confidence: "low",
		Source:     "mock",
	}
}

func warningsForMode(mode string) []string {
	switch mode {
	case entity.RouteModeTrain, entity.RouteModeBus, entity.RouteModeFlight, entity.RouteModeBoat, entity.RouteModeFerry, entity.RouteModePublicTransport:
		return []string{"This is an estimate, not a live schedule or ticket price."}
	case entity.RouteModeHiking:
		return []string{"Hiking estimates are approximate and do not include terrain, weather, permits, or technical navigation."}
	case entity.RouteModeRentalCar:
		return []string{"Rental car costs are approximate and do not include checkout, deposits, insurance, or availability."}
	default:
		return nil
	}
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// round2 rounds to two decimal places. Rounding each segment and then summing
// keeps the response total exactly equal to the sum of the segment distances.
func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
