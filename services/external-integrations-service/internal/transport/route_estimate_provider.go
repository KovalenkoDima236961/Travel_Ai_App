package transport

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

type RouteEstimateProvider struct {
	routeProvider     appservice.RouteProvider
	mock              *MockProvider
	fallbackToMock    bool
	maxOptionsPerMode int
	log               *zap.Logger
}

func NewRouteEstimateProvider(
	routeProvider appservice.RouteProvider,
	mock *MockProvider,
	fallbackToMock bool,
	maxOptionsPerMode int,
	log *zap.Logger,
) *RouteEstimateProvider {
	if log == nil {
		log = zap.NewNop()
	}
	if mock == nil {
		mock = NewMockProvider(maxOptionsPerMode)
	}
	if maxOptionsPerMode <= 0 {
		maxOptionsPerMode = 3
	}
	return &RouteEstimateProvider{
		routeProvider:     routeProvider,
		mock:              mock,
		fallbackToMock:    fallbackToMock,
		maxOptionsPerMode: maxOptionsPerMode,
		log:               log,
	}
}

func (p *RouteEstimateProvider) SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	if p.routeProvider == nil {
		if p.fallbackToMock {
			result, err := p.mock.SearchTransportOptions(ctx, req)
			result.Summary.FallbackUsed = true
			return result, err
		}
		return TransportSearchResponse{}, &ProviderError{Provider: ProviderRouteEstimate, Kind: providerErrorConfiguration}
	}

	options := make([]TransportOption, 0, len(req.Modes))
	fallbackModes := make([]string, 0)
	fallbackUsed := false
	for _, mode := range req.Modes {
		if !routeEstimateMode(mode) {
			fallbackModes = append(fallbackModes, mode)
			continue
		}
		option, err := p.optionFromRouteEstimate(ctx, req, mode)
		if err != nil {
			if !p.fallbackToMock {
				return TransportSearchResponse{}, err
			}
			fallbackUsed = true
			fallbackModes = append(fallbackModes, mode)
			p.log.Warn("route estimate transport fallback used",
				zap.String("mode", mode),
				zap.String("errorType", providerErrorKind(err)),
				zap.Error(err),
			)
			continue
		}
		options = append(options, option)
	}

	if len(fallbackModes) > 0 {
		fallbackReq := req
		fallbackReq.Modes = fallbackModes
		mockResult, err := p.mock.SearchTransportOptions(ctx, fallbackReq)
		if err != nil {
			return TransportSearchResponse{}, err
		}
		fallbackUsed = true
		options = append(options, mockResult.Options...)
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
			Provider:      ProviderRouteEstimate,
			FallbackUsed:  fallbackUsed,
			Warnings:      warningsForFallback(fallbackUsed),
		},
	}, nil
}

func (p *RouteEstimateProvider) optionFromRouteEstimate(ctx context.Context, req TransportSearchRequest, mode string) (TransportOption, error) {
	if req.Origin.Lat == nil || req.Origin.Lng == nil || req.Destination.Lat == nil || req.Destination.Lng == nil {
		return TransportOption{}, &ProviderError{Provider: ProviderRouteEstimate, Kind: providerErrorUnavailable}
	}
	estimate, err := p.routeProvider.EstimateRoute(ctx, entity.RouteEstimateRequest{
		Mode:     mode,
		Currency: req.Currency,
		Stops: []entity.RouteStop{
			routeStopFromLocation(req.Origin),
			routeStopFromLocation(req.Destination),
		},
		Date: req.Date,
	})
	if err != nil {
		return TransportOption{}, &ProviderError{Provider: ProviderRouteEstimate, Kind: providerErrorUnavailable, Err: err}
	}
	if estimate == nil || estimate.DurationMinutes <= 0 {
		return TransportOption{}, &ProviderError{Provider: ProviderRouteEstimate, Kind: providerErrorMalformed}
	}
	departure, arrival := optionTimes(req, mode, 0, estimate.DurationMinutes)
	option := TransportOption{
		ID:              fmt.Sprintf("route-estimate-%s-%s", strings.ReplaceAll(mode, "_", "-"), shortHash(req.Origin.Name+req.Destination.Name+req.Date+req.Time+mode)),
		Mode:            mode,
		Provider:        ProviderRouteEstimate,
		OperatorName:    operatorNameForMode(mode, 0),
		ServiceName:     serviceNameForMode(mode),
		OriginName:      displayLocationName(req.Origin),
		DestinationName: displayLocationName(req.Destination),
		DepartureDate:   departure.Format("2006-01-02"),
		DepartureTime:   departure.Format("15:04"),
		ArrivalDate:     arrival.Format("2006-01-02"),
		ArrivalTime:     arrival.Format("15:04"),
		DurationMinutes: estimate.DurationMinutes,
		Transfers:       transfersForMode(mode, estimate.DistanceKm, 0),
		Status:          StatusUnknown,
		Confidence:      ConfidenceHigh,
		Warnings:        append([]string(nil), estimate.Warnings...),
		Metadata: map[string]any{
			"distanceKm":      estimate.DistanceKm,
			"generatedBy":     "route_estimate_provider",
			"routeProvider":   estimate.Provider,
			"routeFallback":   estimate.FallbackUsed,
			"estimatedByMode": estimate.Mode,
		},
	}
	if len(option.Warnings) == 0 {
		option.Warnings = []string{"Route estimate only. Verify schedules and prices before booking."}
	}
	if estimate.EstimatedCost != nil && estimate.EstimatedCost.Amount >= 0 {
		amount := round2(estimate.EstimatedCost.Amount)
		option.EstimatedPrice = &Money{Amount: amount, Currency: req.Currency}
		option.PriceRange = &MoneyRange{
			Min: Money{Amount: amount, Currency: req.Currency},
			Max: Money{Amount: amount, Currency: req.Currency},
		}
	}
	return option, nil
}

func routeEstimateMode(mode string) bool {
	switch mode {
	case ModeCar, ModeRentalCar, ModeWalk, ModeBike, ModeHiking, ModePublicTransport:
		return true
	default:
		return false
	}
}

func routeStopFromLocation(location Location) entity.RouteStop {
	stop := entity.RouteStop{Name: displayLocationName(location)}
	if location.Lat != nil {
		stop.Latitude = *location.Lat
	}
	if location.Lng != nil {
		stop.Longitude = *location.Lng
	}
	return stop
}

func warningsForFallback(fallbackUsed bool) []string {
	if fallbackUsed {
		return []string{"Some transport options used mock fallback estimates."}
	}
	return []string{}
}
