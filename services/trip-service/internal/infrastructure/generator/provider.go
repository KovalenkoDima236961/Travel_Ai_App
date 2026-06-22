package generator

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
)

const (
	defaultCurrency = "EUR"
	defaultPace     = "balanced"

	generatorModeMock = "mock"
	generatorModeHTTP = "http"
)

// NewItineraryGenerator selects the configured itinerary generator adapter.
func NewItineraryGenerator(cfg *config.Config, logger *zap.Logger) (application.ItineraryGenerator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.ItineraryGenerator.Mode))
	if mode == "" {
		mode = generatorModeMock
	}

	switch mode {
	case generatorModeMock:
		return NewMockItineraryGenerator(logger), nil
	case generatorModeHTTP:
		timeoutSeconds := cfg.ItineraryGenerator.AIPlanningTimeoutSeconds
		if timeoutSeconds <= 0 {
			return nil, fmt.Errorf("AI_PLANNING_TIMEOUT_SECONDS must be greater than 0")
		}

		client := &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		}
		return NewAIPlanningHTTPGenerator(cfg.ItineraryGenerator.AIPlanningServiceURL, client, logger)
	default:
		return nil, fmt.Errorf("unknown ITINERARY_GENERATOR_MODE %q (valid values: mock, http)", cfg.ItineraryGenerator.Mode)
	}
}
