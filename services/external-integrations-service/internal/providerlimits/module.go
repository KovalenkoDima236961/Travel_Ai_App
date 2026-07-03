package providerlimits

import (
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

// New builds the provider-limit Guard from service configuration. The active
// provider for each category is read from the provider config; mock providers
// are treated as unlimited (rate 0, quota 0) but their usage is still tracked
// when limits are enabled.
func New(cfg *config.Config, db *storage.DB, log *zap.Logger) *Guard {
	if log == nil {
		log = zap.NewNop()
	}
	loc, err := time.LoadLocation(cfg.ProviderLimits.Timezone)
	if err != nil {
		log.Warn("invalid provider limits timezone, falling back to UTC",
			zap.String("timezone", cfg.ProviderLimits.Timezone),
			zap.Error(err),
		)
		loc = time.UTC
	}

	return NewGuard(GuardParams{
		Enabled:  cfg.ProviderLimits.Enabled,
		FailOpen: cfg.ProviderLimits.FailOpen,
		Location: loc,
		MaxWait:  time.Duration(cfg.ProviderLimits.RateMaxWaitMS) * time.Millisecond,
		Limiter:  NewLimiter(),
		Store:    NewPostgresStore(db),
		Limits:   resolveLimits(cfg),
		Logger:   log,
	})
}

// resolveLimits maps the active provider of each category to its configured
// limits. When the active provider is a mock, the limit is unlimited.
func resolveLimits(cfg *config.Config) []ProviderLimit {
	pl := cfg.ProviderLimits
	return []ProviderLimit{
		limitFor(CategoryPlaces, cfg.PlaceProvider.Provider, config.PlaceProviderFoursquare,
			pl.FoursquareRatePerMinute, pl.FoursquareBurst, pl.FoursquareDailyQuota),
		limitFor(CategoryRoutes, cfg.RouteProvider.Provider, config.RouteProviderORS,
			pl.ORSRatePerMinute, pl.ORSBurst, pl.ORSDailyQuota),
		limitFor(CategoryWeather, cfg.WeatherProvider.Provider, config.WeatherProviderOpenWeather,
			pl.OpenWeatherRatePerMinute, pl.OpenWeatherBurst, pl.OpenWeatherDailyQuota),
		limitFor(CategoryCalendar, cfg.Calendar.Provider, config.CalendarProviderGoogle,
			pl.GoogleCalendarRatePerMinute, pl.GoogleCalendarBurst, pl.GoogleCalendarDailyQuota),
		exchangeRateLimit(cfg),
		limitFor(CategoryPrice, cfg.PriceProvider.Provider, config.PriceProviderAPI,
			pl.PriceRatePerMinute, pl.PriceBurst, pl.PriceDailyQuota),
	}
}

// limitFor returns the real-provider limit when the active provider is the given
// realProvider, otherwise an unlimited limit for the active (mock) provider.
func limitFor(category, active, realProvider string, rate, burst int, quota int64) ProviderLimit {
	if active == realProvider {
		return ProviderLimit{Category: category, Provider: active, RatePerMinute: rate, Burst: burst, DailyQuota: quota}
	}
	return ProviderLimit{Category: category, Provider: active}
}

// exchangeRateLimit resolves the exchange-rate limit. Any non-mock provider is a
// real provider that shares the EXCHANGE_RATE_* limits.
func exchangeRateLimit(cfg *config.Config) ProviderLimit {
	active := cfg.ExchangeRateProvider.Provider
	pl := cfg.ProviderLimits
	if active != config.ExchangeRateProviderMock && active != "" {
		return ProviderLimit{
			Category:      CategoryExchangeRate,
			Provider:      active,
			RatePerMinute: pl.ExchangeRateRatePerMinute,
			Burst:         pl.ExchangeRateBurst,
			DailyQuota:    pl.ExchangeRateDailyQuota,
		}
	}
	return ProviderLimit{Category: CategoryExchangeRate, Provider: active}
}
