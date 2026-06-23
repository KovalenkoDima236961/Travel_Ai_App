package service

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// PlaceProvider is implemented by each external place-data provider.
type PlaceProvider interface {
	SearchPlaces(ctx context.Context, query string, destination string) ([]entity.Place, error)
	GetPlaceDetails(ctx context.Context, providerPlaceID string) (*entity.Place, error)
}

// PlacesService contains place-search use cases over the configured provider.
type PlacesService struct {
	provider PlaceProvider
	log      *zap.Logger
}

func New(provider PlaceProvider, log *zap.Logger) *PlacesService {
	if log == nil {
		log = zap.NewNop()
	}
	return &PlacesService{provider: provider, log: log}
}

func (s *PlacesService) Search(ctx context.Context, query, destination string) ([]entity.Place, error) {
	query = strings.TrimSpace(query)
	destination = strings.TrimSpace(destination)
	s.log.Debug("searching places",
		zap.Int("query_length", len(query)),
		zap.Bool("destination_present", destination != ""),
	)
	return s.provider.SearchPlaces(ctx, query, destination)
}

func (s *PlacesService) Details(ctx context.Context, placeID string) (*entity.Place, error) {
	placeID = strings.TrimSpace(placeID)
	s.log.Debug("loading place details", zap.Bool("place_id_present", placeID != ""))
	return s.provider.GetPlaceDetails(ctx, placeID)
}
