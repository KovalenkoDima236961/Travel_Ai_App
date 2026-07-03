package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	exchangerateprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/exchangerates"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// ExchangeRateHandler wires exchange-rate use cases to HTTP.
type ExchangeRateHandler struct {
	svc *appservice.ExchangeRateService
	log *zap.Logger
}

func NewExchangeRateHandler(svc *appservice.ExchangeRateService, log *zap.Logger) *ExchangeRateHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ExchangeRateHandler{svc: svc, log: log}
}

func (h *ExchangeRateHandler) RegisterRoutes(r chi.Router) {
	r.Get("/exchange-rates/latest", h.Latest)
	r.Get("/exchange-rates/convert", h.Convert)
}

func (h *ExchangeRateHandler) Latest(w http.ResponseWriter, r *http.Request) {
	base, ok := parseCurrencyQuery(w, r, "base")
	if !ok {
		return
	}
	table, err := h.svc.Latest(r.Context(), base)
	if err != nil {
		h.writeExchangeRateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, table)
}

func (h *ExchangeRateHandler) Convert(w http.ResponseWriter, r *http.Request) {
	rawAmount := strings.TrimSpace(r.URL.Query().Get("amount"))
	if rawAmount == "" {
		writeError(w, http.StatusBadRequest, "amount is required")
		return
	}
	amount, err := strconv.ParseFloat(rawAmount, 64)
	if err != nil || amount < 0 {
		writeError(w, http.StatusBadRequest, "invalid amount")
		return
	}
	from, ok := parseCurrencyQuery(w, r, "from")
	if !ok {
		return
	}
	to, ok := parseCurrencyQuery(w, r, "to")
	if !ok {
		return
	}

	result, err := h.svc.Convert(r.Context(), amount, from, to)
	if err != nil {
		h.writeExchangeRateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseCurrencyQuery(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	value := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get(key)))
	if value == "" {
		writeError(w, http.StatusBadRequest, key+" is required")
		return "", false
	}
	if !currencyCodePattern.MatchString(value) {
		writeError(w, http.StatusBadRequest, "invalid currency code")
		return "", false
	}
	return value, true
}

func (h *ExchangeRateHandler) writeExchangeRateError(w http.ResponseWriter, err error) {
	if errors.Is(err, exchangerateprovider.ErrUnsupportedCurrency) {
		writeError(w, http.StatusBadRequest, "unsupported_currency")
		return
	}
	if writeProviderLimitError(w, err) {
		return
	}
	h.log.Warn("exchange rate request failed", zap.Error(err))
	writeError(w, http.StatusBadGateway, "exchange_rate_provider_unavailable")
}
