package availability

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

type Handler struct {
	svc             *Service
	log             *zap.Logger
	defaultCurrency string
}

func NewHandler(svc *Service, log *zap.Logger, defaultCurrency string) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	defaultCurrency = normalizeCurrency(defaultCurrency)
	if defaultCurrency == "" {
		defaultCurrency = "EUR"
	}
	return &Handler{svc: svc, log: log, defaultCurrency: defaultCurrency}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/availability/search", h.Search)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var input AvailabilitySearchRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, ErrorValidationFailed, "request body must be valid JSON")
		return
	}
	if message, ok := h.normalizeAndValidate(&input); !ok {
		writeError(w, http.StatusBadRequest, ErrorValidationFailed, message)
		return
	}

	result, err := h.svc.SearchAvailability(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrUnsupportedCurrency) {
			writeError(w, http.StatusBadRequest, ErrorUnsupportedCurrency, "unsupported currency")
			return
		}
		if writeAvailabilityProviderLimitError(w, err) {
			return
		}
		var providerErr *ProviderError
		if errors.As(err, &providerErr) && providerErr.Kind == providerErrorMalformed {
			writeError(w, http.StatusBadGateway, ErrorMalformedResponse, "availability provider returned an invalid response")
			return
		}
		h.log.Warn("availability search failed", zap.String("destination", input.Destination), zap.Error(err))
		writeError(w, http.StatusServiceUnavailable, ErrorProviderUnavailable, "availability provider unavailable")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) normalizeAndValidate(input *AvailabilitySearchRequest) (string, bool) {
	input.Destination = strings.TrimSpace(input.Destination)
	if input.Destination == "" {
		return "destination is required", false
	}
	if len(input.Destination) > 200 {
		return "destination must be 200 characters or fewer", false
	}
	input.Date = strings.TrimSpace(input.Date)
	if input.Date == "" {
		return "date is required", false
	}
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return "date must be YYYY-MM-DD", false
	}
	input.Currency = normalizeCurrency(input.Currency)
	if input.Currency == "" {
		input.Currency = h.defaultCurrency
	}
	if !currencyPattern.MatchString(input.Currency) {
		return "currency must be a 3-letter uppercase code", false
	}
	input.Item.Name = strings.TrimSpace(input.Item.Name)
	if input.Item.Name == "" {
		return "item.name is required", false
	}
	input.Item.Type = strings.TrimSpace(input.Item.Type)
	input.Item.Description = strings.TrimSpace(input.Item.Description)
	input.Item.StartTime = strings.TrimSpace(input.Item.StartTime)
	if input.Item.Place != nil {
		input.Item.Place.Name = strings.TrimSpace(input.Item.Place.Name)
		input.Item.Place.Address = strings.TrimSpace(input.Item.Place.Address)
		input.Item.Place.Provider = strings.TrimSpace(input.Item.Place.Provider)
		input.Item.Place.ProviderPlaceID = strings.TrimSpace(input.Item.Place.ProviderPlaceID)
		if input.Item.Place.Latitude != nil && (*input.Item.Place.Latitude < -90 || *input.Item.Place.Latitude > 90) {
			return "item.place.lat must be between -90 and 90", false
		}
		if input.Item.Place.Longitude != nil && (*input.Item.Place.Longitude < -180 || *input.Item.Place.Longitude > 180) {
			return "item.place.lng must be between -180 and 180", false
		}
	}
	if input.Item.EstimatedCost != nil {
		input.Item.EstimatedCost.Currency = normalizeCurrency(input.Item.EstimatedCost.Currency)
		input.Item.EstimatedCost.Category = strings.TrimSpace(input.Item.EstimatedCost.Category)
		input.Item.EstimatedCost.Source = strings.TrimSpace(input.Item.EstimatedCost.Source)
		input.Item.EstimatedCost.Confidence = strings.TrimSpace(input.Item.EstimatedCost.Confidence)
		input.Item.EstimatedCost.Note = strings.TrimSpace(input.Item.EstimatedCost.Note)
	}
	if input.Travelers.Adults == 0 {
		input.Travelers.Adults = 1
	}
	if input.Travelers.Adults < 1 || input.Travelers.Adults > 20 {
		return "travelers.adults must be between 1 and 20", false
	}
	if input.Travelers.Children < 0 || input.Travelers.Children > 20 {
		return "travelers.children must be between 0 and 20", false
	}
	return "", true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}

func writeAvailabilityProviderLimitError(w http.ResponseWriter, err error) bool {
	var limitErr *providerlimits.LimitError
	if !errors.As(err, &limitErr) {
		return false
	}
	status := http.StatusTooManyRequests
	code := ErrorRateLimited
	if limitErr.Code == providerlimits.CodeQuotaExceeded {
		code = ErrorQuotaExceeded
	}
	if limitErr.Code == providerlimits.CodeLimitsUnavailable {
		status = http.StatusServiceUnavailable
		code = ErrorProviderUnavailable
	}
	if limitErr.RetryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(limitErr.RetryAfterSeconds))
	}
	writeJSON(w, status, map[string]any{
		"error":             code,
		"message":           limitErr.Message,
		"provider":          limitErr.Provider,
		"operation":         limitErr.Operation,
		"retryAfterSeconds": limitErr.RetryAfterSeconds,
	})
	return true
}
