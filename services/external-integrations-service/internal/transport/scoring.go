package transport

import (
	"crypto/sha1"
	"encoding/hex"
	"math"
	"sort"
	"strings"
)

const earthRadiusKm = 6371.0

func haversineDistanceKm(origin, destination Location) (float64, bool) {
	if origin.Lat == nil || origin.Lng == nil || destination.Lat == nil || destination.Lng == nil {
		return 0, false
	}
	lat1, lng1 := *origin.Lat, *origin.Lng
	lat2, lng2 := *destination.Lat, *destination.Lng
	if !validLatLng(lat1, lng1) || !validLatLng(lat2, lng2) {
		return 0, false
	}
	latDelta := toRadians(lat2 - lat1)
	lngDelta := toRadians(lng2 - lng1)
	sinLat := math.Sin(latDelta / 2)
	sinLng := math.Sin(lngDelta / 2)
	h := sinLat*sinLat + math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*sinLng*sinLng
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h)), true
}

func heuristicDistanceKm(origin, destination Location) float64 {
	seed := stableHash(normalizeText(origin.Name + ":" + origin.Country + ":" + destination.Name + ":" + destination.Country))
	base := 40 + float64(seed%360)
	text := normalizeText(origin.Name + " " + destination.Name + " " + origin.Country + " " + destination.Country)
	switch {
	case strings.Contains(text, "island") || strings.Contains(text, "capri"):
		base = 35 + float64(seed%80)
	case strings.Contains(text, "barcelona") || strings.Contains(text, "london") || strings.Contains(text, "paris"):
		base = 450 + float64(seed%900)
	}
	return round2(base)
}

func transportDistanceKm(req TransportSearchRequest, mode string) float64 {
	distance, ok := haversineDistanceKm(req.Origin, req.Destination)
	if !ok || distance <= 0 {
		return heuristicDistanceKm(req.Origin, req.Destination)
	}
	factor := 1.15
	switch mode {
	case ModeWalk, ModeBike, ModeHiking, ModeCar, ModeRentalCar, ModeBus:
		factor = 1.25
	case ModeTrain, ModePublicTransport:
		factor = 1.18
	case ModeFlight, ModeFerry, ModeBoat:
		factor = 1.0
	}
	return round2(distance * factor)
}

func durationMinutes(distanceKm, speedKmh float64, bufferMinutes int) int {
	if distanceKm <= 0 || speedKmh <= 0 {
		return bufferMinutes
	}
	return int(math.Round(distanceKm/speedKmh*60)) + bufferMinutes
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func validLatLng(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func stableHash(value string) int {
	sum := sha1.Sum([]byte(value))
	out := 0
	for _, b := range sum[:4] {
		out = out*256 + int(b)
	}
	if out < 0 {
		return -out
	}
	return out
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func sortOptions(options []TransportOption) {
	sort.SliceStable(options, func(i, j int) bool {
		a, b := options[i], options[j]
		aScore, bScore := optionScore(a), optionScore(b)
		if aScore != bScore {
			return aScore < bScore
		}
		if a.DurationMinutes != b.DurationMinutes {
			return a.DurationMinutes < b.DurationMinutes
		}
		return a.ID < b.ID
	})
}

func optionScore(option TransportOption) float64 {
	score := float64(option.DurationMinutes) + float64(option.Transfers*20)
	if option.EstimatedPrice != nil {
		score += option.EstimatedPrice.Amount * 1.5
	}
	switch option.Confidence {
	case ConfidenceHigh:
		score -= 30
	case ConfidenceLow:
		score += 30
	}
	return score
}
