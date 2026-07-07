package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const (
	maxWeatherDestinationLength = 200
	minForecastDays             = 1
	maxForecastDays             = 30
)

// WeatherHandler wires weather forecast use cases to HTTP.
type WeatherHandler struct {
	svc *appservice.WeatherService
	log *zap.Logger
}

func NewWeatherHandler(svc *appservice.WeatherService, log *zap.Logger) *WeatherHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &WeatherHandler{svc: svc, log: log}
}

// RegisterRoutes mounts weather routes onto the given chi router.
func (h *WeatherHandler) RegisterRoutes(r chi.Router) {
	r.Get("/weather/forecast", h.Forecast)
}

// Forecast handles GET /weather/forecast?destination=&startDate=&days=.
func (h *WeatherHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	req, ok := parseWeatherForecastRequest(w, r)
	if !ok {
		return
	}

	forecast, err := h.svc.GetForecast(r.Context(), req)
	if err != nil {
		if writeProviderLimitError(w, err) {
			return
		}
		// Validation already passed, so any error here is an upstream provider
		// failure. Return a safe, generic provider-unavailable response.
		h.log.Warn("weather forecast failed",
			zap.String("destination", req.Destination),
			zap.Int("days", req.Days),
			zap.Error(err),
		)
		writeError(w, http.StatusBadGateway, "weather_provider_unavailable")
		return
	}

	writeJSON(w, http.StatusOK, forecast)
}

func parseWeatherForecastRequest(w http.ResponseWriter, r *http.Request) (entity.WeatherForecastRequest, bool) {
	query := r.URL.Query()

	destination := strings.TrimSpace(query.Get("destination"))
	if destination == "" {
		writeError(w, http.StatusBadRequest, "destination is required")
		return entity.WeatherForecastRequest{}, false
	}
	if len(destination) > maxWeatherDestinationLength {
		writeError(w, http.StatusBadRequest, "destination must be at most 200 characters")
		return entity.WeatherForecastRequest{}, false
	}

	rawStartDate := strings.TrimSpace(query.Get("startDate"))
	if rawStartDate == "" {
		writeError(w, http.StatusBadRequest, "startDate is required")
		return entity.WeatherForecastRequest{}, false
	}
	startDate, err := time.Parse("2006-01-02", rawStartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "startDate must be in YYYY-MM-DD format")
		return entity.WeatherForecastRequest{}, false
	}

	rawDays := strings.TrimSpace(query.Get("days"))
	if rawDays == "" {
		writeError(w, http.StatusBadRequest, "days is required")
		return entity.WeatherForecastRequest{}, false
	}
	days, err := strconv.Atoi(rawDays)
	if err != nil {
		writeError(w, http.StatusBadRequest, "days must be an integer")
		return entity.WeatherForecastRequest{}, false
	}
	if days < minForecastDays {
		writeError(w, http.StatusBadRequest, "days must be at least 1")
		return entity.WeatherForecastRequest{}, false
	}
	if days > maxForecastDays {
		writeError(w, http.StatusBadRequest, "days must be at most 30")
		return entity.WeatherForecastRequest{}, false
	}

	return entity.WeatherForecastRequest{
		Destination: destination,
		StartDate:   startDate,
		Days:        days,
	}, true
}
