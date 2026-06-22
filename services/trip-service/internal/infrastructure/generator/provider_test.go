package generator

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
)

func TestNewItineraryGenerator_ModeMockSelectsMock(t *testing.T) {
	got, err := NewItineraryGenerator(&config.Config{
		ItineraryGenerator: config.ItineraryGeneratorConfig{Mode: "mock"},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got.(*MockItineraryGenerator); !ok {
		t.Fatalf("expected *MockItineraryGenerator, got %T", got)
	}
}

func TestNewItineraryGenerator_EmptyModeDefaultsToMock(t *testing.T) {
	got, err := NewItineraryGenerator(&config.Config{}, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got.(*MockItineraryGenerator); !ok {
		t.Fatalf("expected *MockItineraryGenerator, got %T", got)
	}
}

func TestNewItineraryGenerator_ModeHTTPSelectsHTTPGenerator(t *testing.T) {
	got, err := NewItineraryGenerator(&config.Config{
		ItineraryGenerator: config.ItineraryGeneratorConfig{
			Mode:                     "http",
			AIPlanningServiceURL:     "http://ai-planning-service:8000",
			AIPlanningTimeoutSeconds: 10,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got.(*AIPlanningHTTPGenerator); !ok {
		t.Fatalf("expected *AIPlanningHTTPGenerator, got %T", got)
	}
}

func TestNewItineraryGenerator_UnknownModeReturnsError(t *testing.T) {
	_, err := NewItineraryGenerator(&config.Config{
		ItineraryGenerator: config.ItineraryGeneratorConfig{Mode: "bogus"},
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown ITINERARY_GENERATOR_MODE") {
		t.Fatalf("expected unknown mode error, got %v", err)
	}
}

func TestNewItineraryGenerator_MissingAIPlanningURLReturnsError(t *testing.T) {
	_, err := NewItineraryGenerator(&config.Config{
		ItineraryGenerator: config.ItineraryGeneratorConfig{
			Mode:                     "http",
			AIPlanningTimeoutSeconds: 10,
		},
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "AI_PLANNING_SERVICE_URL is required") {
		t.Fatalf("expected missing URL error, got %v", err)
	}
}

func TestNewItineraryGenerator_InvalidAIPlanningURLReturnsError(t *testing.T) {
	_, err := NewItineraryGenerator(&config.Config{
		ItineraryGenerator: config.ItineraryGeneratorConfig{
			Mode:                     "http",
			AIPlanningServiceURL:     "://bad-url",
			AIPlanningTimeoutSeconds: 10,
		},
	}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid AI_PLANNING_SERVICE_URL") {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
}
