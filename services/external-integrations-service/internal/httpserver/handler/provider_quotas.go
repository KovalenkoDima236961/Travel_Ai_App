package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// recentBlockWindow is how long after a block a provider is reported as
// rate_limited_recently.
const recentBlockWindow = 10 * time.Minute

// historyDays is the number of days returned by the provider detail view.
const historyDays = 7

// Provider quota status values surfaced to the Ops Dashboard.
const (
	quotaStatusHealthy       = "healthy"
	quotaStatusNearingQuota  = "nearing_quota"
	quotaStatusQuotaExceeded = "quota_exceeded"
	quotaStatusRateLimited   = "rate_limited_recently"
	quotaStatusDisabled      = "disabled"
	quotaStatusUnknown       = "unknown"
)

// ProviderQuotaOpsHandler exposes provider usage and quota/rate-limit status to
// the Ops Dashboard. It never exposes API keys or provider credentials.
type ProviderQuotaOpsHandler struct {
	cfg   *config.Config
	guard *providerlimits.Guard
	log   *zap.Logger
}

func NewProviderQuotaOpsHandler(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) *ProviderQuotaOpsHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ProviderQuotaOpsHandler{cfg: cfg, guard: guard, log: log}
}

func (h *ProviderQuotaOpsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/ops/providers/quotas", h.List)
	r.Get("/ops/providers/quotas/{provider}", h.Detail)
	r.Post("/ops/providers/quotas/{provider}/reset-dev", h.ResetDev)
}

type ProviderQuotaOperationUsage struct {
	Operation      string     `json:"operation"`
	UsedToday      int64      `json:"usedToday"`
	BlockedToday   int64      `json:"blockedToday"`
	FallbackToday  int64      `json:"fallbackToday"`
	LastAllowedAt  *time.Time `json:"lastAllowedAt,omitempty"`
	LastBlockedAt  *time.Time `json:"lastBlockedAt,omitempty"`
	LastFallbackAt *time.Time `json:"lastFallbackAt,omitempty"`
}

type ProviderQuotaSummary struct {
	Provider           string                        `json:"provider"`
	Category           string                        `json:"category"`
	Enabled            bool                          `json:"enabled"`
	RateLimitPerMinute int                           `json:"rateLimitPerMinute"`
	DailyQuota         int64                         `json:"dailyQuota"`
	UsedToday          int64                         `json:"usedToday"`
	RemainingToday     int64                         `json:"remainingToday"`
	BlockedToday       int64                         `json:"blockedToday"`
	FallbackToday      int64                         `json:"fallbackToday"`
	Status             string                        `json:"status"`
	LastBlockedAt      *time.Time                    `json:"lastBlockedAt,omitempty"`
	LastFallbackAt     *time.Time                    `json:"lastFallbackAt,omitempty"`
	Operations         []ProviderQuotaOperationUsage `json:"operations"`
}

type ProviderQuotasResponse struct {
	Date         string                 `json:"date"`
	Enabled      bool                   `json:"enabled"`
	Providers    []ProviderQuotaSummary `json:"providers"`
	ResetAllowed bool                   `json:"resetAllowed"`
}

type ProviderQuotaDayUsage struct {
	Date          string `json:"date"`
	UsedCount     int64  `json:"usedCount"`
	BlockedCount  int64  `json:"blockedCount"`
	FallbackCount int64  `json:"fallbackCount"`
}

type ProviderQuotaDetailResponse struct {
	Date         string                  `json:"date"`
	Enabled      bool                    `json:"enabled"`
	ResetAllowed bool                    `json:"resetAllowed"`
	Provider     ProviderQuotaSummary    `json:"provider"`
	History      []ProviderQuotaDayUsage `json:"history"`
}

// List handles GET /ops/providers/quotas.
func (h *ProviderQuotaOpsHandler) List(w http.ResponseWriter, r *http.Request) {
	date, ok := h.parseDate(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
		return
	}

	usage, err := h.guard.Store().ListUsageByDate(r.Context(), date)
	if err != nil {
		h.log.Warn("failed to list provider quotas", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load provider quotas")
		return
	}
	byOperation := indexUsage(usage)

	summaries := make([]ProviderQuotaSummary, 0)
	for _, limit := range h.guard.Limits() {
		summaries = append(summaries, h.buildSummary(limit, byOperation))
	}

	writeJSON(w, http.StatusOK, ProviderQuotasResponse{
		Date:         date.Format("2006-01-02"),
		Enabled:      h.guard.Enabled(),
		Providers:    summaries,
		ResetAllowed: !h.cfg.IsProduction(),
	})
}

// Detail handles GET /ops/providers/quotas/{provider}.
func (h *ProviderQuotaOpsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	providerName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	limit, ok := h.limitForProvider(providerName)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown provider")
		return
	}

	today := h.guard.Today()
	todayUsage, err := h.guard.Store().ListUsageByDate(r.Context(), today)
	if err != nil {
		h.log.Warn("failed to load provider quota detail", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load provider quota detail")
		return
	}
	summary := h.buildSummary(limit, indexUsage(todayUsage))

	from := today.AddDate(0, 0, -(historyDays - 1))
	rows, err := h.guard.Store().ListUsageByProvider(r.Context(), providerName, from, today)
	if err != nil {
		h.log.Warn("failed to load provider quota history", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to load provider quota history")
		return
	}

	writeJSON(w, http.StatusOK, ProviderQuotaDetailResponse{
		Date:         today.Format("2006-01-02"),
		Enabled:      h.guard.Enabled(),
		ResetAllowed: !h.cfg.IsProduction(),
		Provider:     summary,
		History:      buildHistory(rows),
	})
}

// ResetDev handles POST /ops/providers/quotas/{provider}/reset-dev. It is
// forbidden in production.
func (h *ProviderQuotaOpsHandler) ResetDev(w http.ResponseWriter, r *http.Request) {
	if h.cfg.IsProduction() {
		writeError(w, http.StatusForbidden, "reset is not allowed in production")
		return
	}
	providerName := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	if _, ok := h.limitForProvider(providerName); !ok {
		writeError(w, http.StatusNotFound, "unknown provider")
		return
	}
	today := h.guard.Today()
	if err := h.guard.Store().ResetProviderForDate(r.Context(), providerName, today); err != nil {
		h.log.Warn("failed to reset provider quota", zap.String("provider", providerName), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to reset provider quota")
		return
	}
	h.log.Info("provider quota reset (dev)",
		zap.String("provider", providerName),
		zap.String("date", today.Format("2006-01-02")),
	)
	writeJSON(w, http.StatusOK, map[string]any{
		"reset":    true,
		"provider": providerName,
		"date":     today.Format("2006-01-02"),
	})
}

func (h *ProviderQuotaOpsHandler) buildSummary(limit providerlimits.ProviderLimit, byOperation map[string]providerlimits.OperationUsage) ProviderQuotaSummary {
	summary := ProviderQuotaSummary{
		Provider:           limit.Provider,
		Category:           limit.Category,
		Enabled:            h.guard.Enabled(),
		RateLimitPerMinute: limit.RatePerMinute,
		DailyQuota:         limit.DailyQuota,
		Operations:         make([]ProviderQuotaOperationUsage, 0),
	}
	for _, op := range providerlimits.OperationsForCategory(limit.Category) {
		row, ok := byOperation[op]
		if !ok {
			continue
		}
		summary.UsedToday += row.UsedCount
		summary.BlockedToday += row.BlockedCount
		summary.FallbackToday += row.FallbackCount
		summary.LastBlockedAt = latest(summary.LastBlockedAt, row.LastBlockedAt)
		summary.LastFallbackAt = latest(summary.LastFallbackAt, row.LastFallbackAt)
		summary.Operations = append(summary.Operations, ProviderQuotaOperationUsage{
			Operation:      op,
			UsedToday:      row.UsedCount,
			BlockedToday:   row.BlockedCount,
			FallbackToday:  row.FallbackCount,
			LastAllowedAt:  row.LastAllowedAt,
			LastBlockedAt:  row.LastBlockedAt,
			LastFallbackAt: row.LastFallbackAt,
		})
	}
	summary.RemainingToday = remainingQuota(limit.DailyQuota, summary.UsedToday)
	summary.Status = h.status(limit, summary)
	return summary
}

func (h *ProviderQuotaOpsHandler) status(limit providerlimits.ProviderLimit, summary ProviderQuotaSummary) string {
	if !h.guard.Enabled() {
		return quotaStatusDisabled
	}
	if limit.DailyQuota > 0 && summary.UsedToday >= limit.DailyQuota {
		return quotaStatusQuotaExceeded
	}
	if summary.LastBlockedAt != nil && time.Since(*summary.LastBlockedAt) <= recentBlockWindow {
		return quotaStatusRateLimited
	}
	if limit.DailyQuota > 0 && summary.UsedToday*100 >= limit.DailyQuota*80 {
		return quotaStatusNearingQuota
	}
	if summary.UsedToday > 0 || summary.BlockedToday > 0 {
		return quotaStatusHealthy
	}
	return quotaStatusUnknown
}

func (h *ProviderQuotaOpsHandler) parseDate(r *http.Request) (time.Time, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("date"))
	if raw == "" {
		return h.guard.Today(), true
	}
	parsed, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC().Truncate(24 * time.Hour), true
}

func (h *ProviderQuotaOpsHandler) limitForProvider(provider string) (providerlimits.ProviderLimit, bool) {
	for _, limit := range h.guard.Limits() {
		if strings.EqualFold(limit.Provider, provider) {
			return limit, true
		}
	}
	return providerlimits.ProviderLimit{}, false
}

func indexUsage(rows []providerlimits.OperationUsage) map[string]providerlimits.OperationUsage {
	out := make(map[string]providerlimits.OperationUsage, len(rows))
	for _, row := range rows {
		out[row.Operation] = row
	}
	return out
}

func buildHistory(rows []providerlimits.OperationUsage) []ProviderQuotaDayUsage {
	byDate := map[string]*ProviderQuotaDayUsage{}
	order := make([]string, 0)
	for _, row := range rows {
		key := row.UsageDate.UTC().Format("2006-01-02")
		day, ok := byDate[key]
		if !ok {
			day = &ProviderQuotaDayUsage{Date: key}
			byDate[key] = day
			order = append(order, key)
		}
		day.UsedCount += row.UsedCount
		day.BlockedCount += row.BlockedCount
		day.FallbackCount += row.FallbackCount
	}
	out := make([]ProviderQuotaDayUsage, 0, len(order))
	for _, key := range order {
		out = append(out, *byDate[key])
	}
	return out
}

func remainingQuota(quota, used int64) int64 {
	if quota <= 0 {
		return 0
	}
	if used >= quota {
		return 0
	}
	return quota - used
}

func latest(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if b.After(*a) {
		return b
	}
	return a
}
