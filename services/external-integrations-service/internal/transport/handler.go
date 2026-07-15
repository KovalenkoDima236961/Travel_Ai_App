package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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
	r.Post("/transport/search", h.Search)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req TransportSearchRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrorValidationFailed, "request body must be valid JSON")
		return
	}
	normalizeSearchRequest(&req, h.defaultCurrency)
	if message, ok := validateSearchRequest(req); !ok {
		writeError(w, http.StatusBadRequest, ErrorValidationFailed, message)
		return
	}

	result, err := h.svc.SearchTransportOptions(r.Context(), req)
	if err != nil {
		if writeTransportProviderLimitError(w, err) {
			return
		}
		var providerErr *ProviderError
		if errors.As(err, &providerErr) && providerErr.Kind == providerErrorMalformed {
			writeError(w, http.StatusBadGateway, ErrorMalformedResponse, "transport provider returned an invalid response")
			return
		}
		h.log.Warn("transport search failed", zap.String("origin", req.Origin.Name), zap.String("destination", req.Destination.Name), zap.Error(err))
		writeError(w, http.StatusServiceUnavailable, ErrorProviderUnavailable, "transport provider unavailable")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}

func writeTransportProviderLimitError(w http.ResponseWriter, err error) bool {
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
