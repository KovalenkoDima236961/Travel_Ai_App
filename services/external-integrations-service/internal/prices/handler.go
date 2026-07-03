package prices

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

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
	r.Post("/prices/estimate", h.Estimate)
}

func (h *Handler) Estimate(w http.ResponseWriter, r *http.Request) {
	var input PriceEstimateInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if message, ok := h.normalizeAndValidate(&input); !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	result, err := h.svc.EstimatePrice(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrUnsupportedCurrency) {
			writeError(w, http.StatusBadRequest, "unsupported_currency")
			return
		}
		if writeProviderLimitError(w, err) {
			return
		}
		h.log.Warn("price estimate failed", zap.String("destination", input.Destination), zap.Error(err))
		writeError(w, http.StatusBadGateway, "price_provider_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) normalizeAndValidate(input *PriceEstimateInput) (string, bool) {
	input.Destination = strings.TrimSpace(input.Destination)
	if input.Destination == "" {
		return "destination is required", false
	}
	input.Currency = normalizeCurrency(input.Currency)
	if input.Currency == "" {
		input.Currency = h.defaultCurrency
	}
	if !currencyPattern.MatchString(input.Currency) {
		return "invalid currency", false
	}
	input.Date = strings.TrimSpace(input.Date)
	if input.Date != "" {
		if _, err := time.Parse("2006-01-02", input.Date); err != nil {
			return "invalid date", false
		}
	}
	if input.Place == nil {
		return "place is required", false
	}
	input.Place.Provider = strings.TrimSpace(input.Place.Provider)
	input.Place.ProviderPlaceID = strings.TrimSpace(input.Place.ProviderPlaceID)
	input.Place.Name = strings.TrimSpace(input.Place.Name)
	input.Place.Address = strings.TrimSpace(input.Place.Address)
	input.Place.Category = strings.TrimSpace(input.Place.Category)
	if input.Place.Name == "" {
		return "place.name is required", false
	}
	if input.Place.Latitude != nil && (*input.Place.Latitude < -90 || *input.Place.Latitude > 90) {
		return "place.lat must be between -90 and 90", false
	}
	if input.Place.Longitude != nil && (*input.Place.Longitude < -180 || *input.Place.Longitude > 180) {
		return "place.lng must be between -180 and 180", false
	}
	if input.Place.Rating != nil && (*input.Place.Rating < 0 || *input.Place.Rating > 5) {
		return "place.rating must be between 0 and 5", false
	}
	if input.ItemContext != nil {
		input.ItemContext.Name = strings.TrimSpace(input.ItemContext.Name)
		input.ItemContext.Type = strings.TrimSpace(input.ItemContext.Type)
		input.ItemContext.Description = strings.TrimSpace(input.ItemContext.Description)
	}
	return "", true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeProviderLimitError writes a controlled provider-limit response and returns
// true when err is a provider limit error. It exposes only safe fields.
func writeProviderLimitError(w http.ResponseWriter, err error) bool {
	var limitErr *providerlimits.LimitError
	if !errors.As(err, &limitErr) {
		return false
	}
	status := http.StatusTooManyRequests
	if limitErr.Code == providerlimits.CodeLimitsUnavailable {
		status = http.StatusServiceUnavailable
	}
	if limitErr.RetryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(limitErr.RetryAfterSeconds))
	}
	writeJSON(w, status, map[string]any{
		"error":             limitErr.Code,
		"message":           limitErr.Message,
		"provider":          limitErr.Provider,
		"operation":         limitErr.Operation,
		"retryAfterSeconds": limitErr.RetryAfterSeconds,
	})
	return true
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizedCategory(value string) string {
	value = normalizeKey(value)
	return strings.ReplaceAll(value, "-", "_")
}

func normalizeKey(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), "_")
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", " ", "-", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}
