package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

type ProviderOpsHandler struct {
	cfg *config.Config
	log *zap.Logger
}

type ProviderStatusResponse struct {
	Providers []ProviderStatus `json:"providers"`
}

type ProviderStatus struct {
	Name               string     `json:"name"`
	ActiveProvider     string     `json:"activeProvider"`
	Enabled            bool       `json:"enabled"`
	FallbackEnabled    bool       `json:"fallbackEnabled"`
	LastSuccessAt      *time.Time `json:"lastSuccessAt,omitempty"`
	LastFailureAt      *time.Time `json:"lastFailureAt,omitempty"`
	RecentSuccessCount int        `json:"recentSuccessCount"`
	RecentFailureCount int        `json:"recentFailureCount"`
	Status             string     `json:"status"`
	LastErrorCode      string     `json:"lastErrorCode,omitempty"`
}

func NewProviderOpsHandler(cfg *config.Config, log *zap.Logger) *ProviderOpsHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ProviderOpsHandler{cfg: cfg, log: log}
}

func (h *ProviderOpsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ops/providers/status", h.Status)
}

func (h *ProviderOpsHandler) Status(w http.ResponseWriter, _ *http.Request) {
	providers := []ProviderStatus{
		h.provider("places", h.cfg.PlaceProvider.Provider, true, h.cfg.PlaceProvider.FallbackToMock),
		h.provider("route", h.cfg.RouteProvider.Provider, true, h.cfg.RouteProvider.FallbackToMock),
		h.provider("weather", h.cfg.WeatherProvider.Provider, true, h.cfg.WeatherProvider.FallbackToMock),
		h.provider("exchange_rate", h.cfg.ExchangeRateProvider.Provider, true, h.cfg.ExchangeRateProvider.FallbackToMock),
		h.provider("price", h.cfg.PriceProvider.Provider, true, h.cfg.PriceProvider.FallbackToMock),
		h.provider("availability", h.cfg.Availability.Provider, h.cfg.Availability.Enabled, h.cfg.Availability.FallbackToMock),
		h.provider("calendar", h.cfg.Calendar.Provider, h.cfg.Calendar.Enabled, false),
	}
	writeJSON(w, http.StatusOK, ProviderStatusResponse{Providers: providers})
}

func (h *ProviderOpsHandler) provider(name, active string, enabled, fallback bool) ProviderStatus {
	snapshot := extobs.ProviderHealth(active)
	return ProviderStatus{
		Name:               name,
		ActiveProvider:     active,
		Enabled:            enabled,
		FallbackEnabled:    fallback,
		LastSuccessAt:      snapshot.LastSuccessAt,
		LastFailureAt:      snapshot.LastFailureAt,
		RecentSuccessCount: snapshot.RecentSuccessCount,
		RecentFailureCount: snapshot.RecentFailureCount,
		Status:             providerStatus(enabled, snapshot),
		LastErrorCode:      snapshot.LastErrorCode,
	}
}

func providerStatus(enabled bool, snapshot extobs.ProviderHealthSnapshot) string {
	if !enabled {
		return "down"
	}
	if snapshot.RecentFailureCount == 0 && snapshot.RecentSuccessCount == 0 {
		return "unknown"
	}
	if snapshot.RecentFailureCount == 0 {
		return "healthy"
	}
	if snapshot.RecentSuccessCount > 0 {
		return "degraded"
	}
	return "down"
}
