package transport

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const mockWarning = "Mock transport option. Verify schedule before booking."
const noLiveWarning = "No live availability guarantee."

type MockProvider struct {
	maxOptionsPerMode int
}

func NewMockProvider(maxOptionsPerMode int) *MockProvider {
	if maxOptionsPerMode <= 0 || maxOptionsPerMode > 5 {
		maxOptionsPerMode = 3
	}
	return &MockProvider{maxOptionsPerMode: maxOptionsPerMode}
}

func (p *MockProvider) SearchTransportOptions(_ context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	req.Modes = normalizeModes(req.Modes)
	if len(req.Modes) == 0 {
		req.Modes = []string{ModeTrain, ModeBus, ModeCar}
	}

	options := make([]TransportOption, 0, len(req.Modes)*p.maxOptionsPerMode)
	for _, mode := range req.Modes {
		options = append(options, p.optionsForMode(req, mode)...)
	}
	options = filterOptions(options, req.Constraints)
	sortOptions(options)

	return TransportSearchResponse{
		Options: options,
		Summary: SearchSummary{
			Origin:        displayLocationName(req.Origin),
			Destination:   displayLocationName(req.Destination),
			Date:          req.Date,
			SearchedModes: append([]string(nil), req.Modes...),
			Provider:      ProviderMock,
			Warnings:      []string{},
		},
	}, nil
}

func (p *MockProvider) optionsForMode(req TransportSearchRequest, mode string) []TransportOption {
	distanceKm := transportDistanceKm(req, mode)
	if mode == ModeFlight && distanceKm <= 250 && !modeRequested(req, ModeFlight) {
		return nil
	}
	if (mode == ModeFerry || mode == ModeBoat) && !modeRequested(req, mode) && !looksWaterRoute(req) {
		return nil
	}

	assumptions := assumptionsForMode(mode, distanceKm)
	if assumptions.speedKmh <= 0 {
		return nil
	}
	count := assumptions.count
	if count > p.maxOptionsPerMode {
		count = p.maxOptionsPerMode
	}
	out := make([]TransportOption, 0, count)
	for i := 0; i < count; i++ {
		duration := durationMinutes(distanceKm, assumptions.speedKmh+float64(i*7), assumptions.bufferMinutes+i*5)
		departure, arrival := optionTimes(req, mode, i, duration)
		minAmount, maxAmount := priceRangeForMode(mode, distanceKm, req.Travelers, i)
		var estimatedPrice *Money
		var priceRange *MoneyRange
		if maxAmount > 0 || minAmount > 0 {
			estimated := round2((minAmount + maxAmount) / 2)
			estimatedPrice = &Money{Amount: estimated, Currency: req.Currency}
			priceRange = &MoneyRange{
				Min: Money{Amount: round2(minAmount), Currency: req.Currency},
				Max: Money{Amount: round2(maxAmount), Currency: req.Currency},
			}
		}
		option := TransportOption{
			ID:              fmt.Sprintf("mock-%s-%d", strings.ReplaceAll(mode, "_", "-"), i+1),
			Mode:            mode,
			Provider:        ProviderMock,
			OperatorName:    operatorNameForMode(mode, i),
			ServiceName:     serviceNameForMode(mode),
			OriginName:      originNameForMode(req, mode),
			DestinationName: destinationNameForMode(req, mode),
			DepartureDate:   departure.Format("2006-01-02"),
			DepartureTime:   departure.Format("15:04"),
			ArrivalDate:     arrival.Format("2006-01-02"),
			ArrivalTime:     arrival.Format("15:04"),
			DurationMinutes: duration,
			Transfers:       transfersForMode(mode, distanceKm, i),
			EstimatedPrice:  estimatedPrice,
			PriceRange:      priceRange,
			Status:          StatusUnknown,
			Confidence:      confidenceForMockMode(mode, req),
			Warnings:        []string{mockWarning, noLiveWarning},
			Metadata: map[string]any{
				"distanceKm":   distanceKm,
				"generatedBy":  "mock_provider",
				"optionNumber": i + 1,
			},
		}
		if mode == ModeFlight {
			baggage := "Baggage rules vary by airline and fare. Verify before booking."
			option.BaggageNotes = &baggage
		}
		out = append(out, option)
	}
	return out
}

type modeAssumptions struct {
	speedKmh      float64
	bufferMinutes int
	count         int
}

func assumptionsForMode(mode string, distanceKm float64) modeAssumptions {
	switch mode {
	case ModeTrain:
		return modeAssumptions{speedKmh: 105, bufferMinutes: 20, count: 3}
	case ModeBus:
		return modeAssumptions{speedKmh: 72, bufferMinutes: 15, count: 3}
	case ModeCar:
		return modeAssumptions{speedKmh: 84, bufferMinutes: 10, count: 1}
	case ModeRentalCar:
		return modeAssumptions{speedKmh: 80, bufferMinutes: 30, count: 1}
	case ModeFlight:
		if distanceKm < 80 {
			distanceKm = 80
		}
		return modeAssumptions{speedKmh: 700, bufferMinutes: 180, count: 2}
	case ModeFerry:
		return modeAssumptions{speedKmh: 35, bufferMinutes: 20, count: 3}
	case ModeBoat:
		return modeAssumptions{speedKmh: 30, bufferMinutes: 25, count: 2}
	case ModeWalk:
		return modeAssumptions{speedKmh: 5, bufferMinutes: 0, count: 1}
	case ModeBike:
		return modeAssumptions{speedKmh: 15, bufferMinutes: 5, count: 1}
	case ModeHiking:
		return modeAssumptions{speedKmh: 3.5, bufferMinutes: 0, count: 1}
	case ModePublicTransport:
		return modeAssumptions{speedKmh: 38, bufferMinutes: 20, count: 2}
	case ModeOther:
		return modeAssumptions{speedKmh: 45, bufferMinutes: 20, count: 1}
	default:
		return modeAssumptions{}
	}
}

func priceRangeForMode(mode string, distanceKm float64, travelers int, optionIndex int) (float64, float64) {
	if travelers < 1 {
		travelers = 1
	}
	multiplier := 1 + float64(optionIndex)*0.08
	perPerson := float64(travelers)
	switch mode {
	case ModeTrain:
		return distanceKm * 0.10 * perPerson * multiplier, distanceKm * 0.25 * perPerson * multiplier
	case ModeBus:
		return distanceKm * 0.06 * perPerson * multiplier, distanceKm * 0.18 * perPerson * multiplier
	case ModeCar:
		return distanceKm * 0.18 * multiplier, distanceKm * 0.35 * multiplier
	case ModeRentalCar:
		return (distanceKm*0.18 + 45) * multiplier, (distanceKm*0.35 + 90) * multiplier
	case ModeFlight:
		minimum := math.Max(40, distanceKm*0.07)
		maximum := math.Max(90, distanceKm*0.18)
		return minimum * perPerson * multiplier, math.Min(220*perPerson*multiplier, maximum*perPerson*multiplier+80)
	case ModeFerry:
		return math.Max(10, distanceKm*0.20) * perPerson * multiplier, math.Max(25, distanceKm*0.50) * perPerson * multiplier
	case ModeBoat:
		return math.Max(12, distanceKm*0.22) * perPerson * multiplier, math.Max(35, distanceKm*0.65) * perPerson * multiplier
	case ModePublicTransport:
		return distanceKm * 0.08 * perPerson * multiplier, distanceKm * 0.16 * perPerson * multiplier
	case ModeBike:
		if distanceKm > 20 {
			return 10 * perPerson, 25 * perPerson
		}
		return 0, 0
	case ModeWalk, ModeHiking:
		return 0, 0
	default:
		return distanceKm * 0.08 * perPerson, distanceKm * 0.18 * perPerson
	}
}

func optionTimes(req TransportSearchRequest, mode string, optionIndex int, duration int) (time.Time, time.Time) {
	timeValue := req.Time
	if timeValue == "" {
		timeValue = "09:00"
	}
	base, err := time.Parse("2006-01-02 15:04", req.Date+" "+timeValue)
	if err != nil {
		base, _ = time.Parse("2006-01-02 15:04", req.Date+" 09:00")
	}
	offset := optionIndex * 30
	switch mode {
	case ModeFlight:
		offset = optionIndex * 120
	case ModeFerry, ModeBoat:
		offset = optionIndex * 45
	case ModeWalk, ModeBike, ModeHiking, ModeCar, ModeRentalCar:
		offset = optionIndex * 15
	}
	if req.TimePreference == TimePreferenceArriveBefore {
		arrival := base.Add(-time.Duration(offset) * time.Minute)
		return arrival.Add(-time.Duration(duration) * time.Minute), arrival
	}
	departure := base.Add(time.Duration(offset+deterministicMinuteOffset(req, mode, optionIndex)) * time.Minute)
	return departure, departure.Add(time.Duration(duration) * time.Minute)
}

func deterministicMinuteOffset(req TransportSearchRequest, mode string, optionIndex int) int {
	if optionIndex > 0 || mode == ModeCar || mode == ModeWalk || mode == ModeBike || mode == ModeHiking {
		return 0
	}
	return stableHash(req.Origin.Name+req.Destination.Name+mode+req.Date) % 20
}

func transfersForMode(mode string, distanceKm float64, optionIndex int) int {
	switch mode {
	case ModeTrain:
		if distanceKm > 240 && optionIndex > 0 {
			return 1
		}
	case ModeBus:
		if distanceKm > 350 && optionIndex > 1 {
			return 1
		}
	case ModeFlight:
		if distanceKm > 900 && optionIndex > 0 {
			return 1
		}
	case ModePublicTransport:
		return 1 + optionIndex
	}
	return 0
}

func confidenceForMockMode(mode string, req TransportSearchRequest) string {
	switch mode {
	case ModeTrain, ModeBus, ModeCar, ModeRentalCar, ModePublicTransport, ModeWalk, ModeBike, ModeHiking:
		return ConfidenceMedium
	case ModeFerry:
		if looksWaterRoute(req) {
			return ConfidenceMedium
		}
		return ConfidenceLow
	default:
		return ConfidenceLow
	}
}

func operatorNameForMode(mode string, optionIndex int) string {
	switch mode {
	case ModeTrain:
		if optionIndex%2 == 0 {
			return "Regional Rail"
		}
		return "InterCity Rail"
	case ModeBus:
		if optionIndex%2 == 0 {
			return "Regional Bus"
		}
		return "Express Coach"
	case ModeCar:
		return "Self drive"
	case ModeRentalCar:
		return "Rental car estimate"
	case ModeFlight:
		return "Mock Air"
	case ModeFerry:
		return "Harbor Ferry"
	case ModeBoat:
		return "Local Boat"
	case ModePublicTransport:
		return "Public transport"
	case ModeWalk:
		return "Walking route"
	case ModeBike:
		return "Bike route"
	case ModeHiking:
		return "Trail estimate"
	default:
		return "Transport estimate"
	}
}

func serviceNameForMode(mode string) string {
	switch mode {
	case ModeRentalCar:
		return "Rental car"
	case ModePublicTransport:
		return "Public transport"
	default:
		parts := strings.Split(mode, "_")
		for i := range parts {
			if parts[i] != "" {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		return strings.Join(parts, " ")
	}
}

func originNameForMode(req TransportSearchRequest, mode string) string {
	return stopNameForMode(req.Origin, mode, true)
}

func destinationNameForMode(req TransportSearchRequest, mode string) string {
	return stopNameForMode(req.Destination, mode, false)
}

func stopNameForMode(location Location, mode string, origin bool) string {
	name := displayLocationName(location)
	switch mode {
	case ModeTrain:
		if origin {
			return name + " main station"
		}
		return name + " central station"
	case ModeBus:
		return name + " bus station"
	case ModeFlight:
		return name + " airport"
	case ModeFerry, ModeBoat:
		return name + " port"
	default:
		return name
	}
}

func displayLocationName(location Location) string {
	if strings.TrimSpace(location.Name) != "" {
		return strings.TrimSpace(location.Name)
	}
	if location.Lat != nil && location.Lng != nil {
		return fmt.Sprintf("%.4f, %.4f", *location.Lat, *location.Lng)
	}
	return "Unknown"
}

func modeRequested(req TransportSearchRequest, mode string) bool {
	for _, requested := range req.Modes {
		if requested == mode {
			return true
		}
	}
	return false
}

func looksWaterRoute(req TransportSearchRequest) bool {
	text := normalizeText(req.Origin.Name + " " + req.Destination.Name + " " + req.Origin.Country + " " + req.Destination.Country)
	keywords := []string{"island", "capri", "sorrento", "port", "harbor", "harbour", "ferry", "boat", "bay"}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func filterOptions(options []TransportOption, constraints SearchConstraints) []TransportOption {
	out := make([]TransportOption, 0, len(options))
	for _, option := range options {
		if constraints.AvoidFlights && option.Mode == ModeFlight {
			continue
		}
		if constraints.MaxDurationMinutes != nil && option.DurationMinutes > *constraints.MaxDurationMinutes {
			continue
		}
		if constraints.MaxPriceAmount != nil && option.EstimatedPrice != nil && option.EstimatedPrice.Amount > *constraints.MaxPriceAmount {
			continue
		}
		out = append(out, option)
	}
	return out
}
