package routes

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

func TestNewUnsupportedRouteProviderReturnsError(t *testing.T) {
	_, err := New(&config.Config{RouteProvider: config.RouteProviderConfig{Provider: "osrm"}}, zap.NewNop())
	if err == nil {
		t.Fatal("expected unsupported route provider error")
	}
	if !strings.Contains(err.Error(), "unsupported ROUTE_PROVIDER") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockRouteProviderIsDeterministic(t *testing.T) {
	provider := NewMockRouteProvider()
	req := entity.RouteEstimateRequest{
		Mode: "walking",
		Stops: []entity.RouteStop{
			{Name: "Colosseum", Latitude: 41.8902, Longitude: 12.4922},
			{Name: "Trevi Fountain", Latitude: 41.9009, Longitude: 12.4833},
		},
	}

	first, err := provider.EstimateRoute(context.Background(), req)
	if err != nil {
		t.Fatalf("first estimate: %v", err)
	}
	second, err := provider.EstimateRoute(context.Background(), req)
	if err != nil {
		t.Fatalf("second estimate: %v", err)
	}

	if first.DistanceKm != second.DistanceKm || first.DurationMinutes != second.DurationMinutes {
		t.Fatalf("expected deterministic estimate, got %+v vs %+v", first, second)
	}
	if first.DistanceKm <= 0 || first.DurationMinutes <= 0 {
		t.Fatalf("expected positive estimate, got %+v", first)
	}
}
