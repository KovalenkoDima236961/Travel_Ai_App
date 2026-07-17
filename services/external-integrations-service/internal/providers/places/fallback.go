package places

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

type fallbackPlaceProvider struct {
	providerName         string
	fallbackProviderName string
	primary              service.PlaceProvider
	fallback             service.PlaceProvider
	log                  *zap.Logger
}

func newFallbackPlaceProvider(
	providerName string,
	primary service.PlaceProvider,
	fallback service.PlaceProvider,
	log *zap.Logger,
) service.PlaceProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackPlaceProvider{
		providerName:         providerName,
		fallbackProviderName: "mock",
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackPlaceProvider) SearchPlaces(ctx context.Context, query string, destination string) ([]entity.Place, error) {
	var items []entity.Place
	var err error
	if extobs.ProviderCircuitAllows(p.providerName, "place_search") {
		items, err = p.primary.SearchPlaces(ctx, query, destination)
		if err == nil {
			extobs.RecordProviderCircuitSuccess(p.providerName, "place_search")
			return items, nil
		}
		extobs.RecordProviderCircuitFailure(p.providerName, "place_search")
	} else {
		err = errors.New("place provider cooldown active")
	}

	p.log.Warn("place provider fallback used",
		zap.String("action", "place_search"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackItems, fallbackErr := p.fallback.SearchPlaces(ctx, query, destination)
	if fallbackErr != nil {
		p.log.Warn("place provider fallback failed",
			zap.String("action", "place_search"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	return fallbackItems, nil
}

func (p *fallbackPlaceProvider) GetPlaceDetails(ctx context.Context, providerPlaceID string) (*entity.Place, error) {
	var place *entity.Place
	var err error
	if extobs.ProviderCircuitAllows(p.providerName, "place_details") {
		place, err = p.primary.GetPlaceDetails(ctx, providerPlaceID)
		if err == nil {
			extobs.RecordProviderCircuitSuccess(p.providerName, "place_details")
			return place, nil
		}
		extobs.RecordProviderCircuitFailure(p.providerName, "place_details")
	} else {
		err = errors.New("place provider cooldown active")
	}

	p.log.Warn("place provider fallback used",
		zap.String("action", "place_details"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackPlace, fallbackErr := p.fallback.GetPlaceDetails(ctx, providerPlaceID)
	if fallbackErr != nil {
		p.log.Warn("place provider fallback failed",
			zap.String("action", "place_details"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	if fallbackPlace == nil {
		return nil, err
	}
	return fallbackPlace, nil
}
