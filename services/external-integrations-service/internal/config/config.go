package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

const (
	PlaceProviderMock                     = "mock"
	PlaceProviderFoursquare               = "foursquare"
	RouteProviderMock                     = "mock"
	RouteProviderORS                      = "ors"
	WeatherProviderMock                   = "mock"
	WeatherProviderOpenWeather            = "openweathermap"
	ExchangeRateProviderMock              = "mock"
	ExchangeRateProviderHost              = "exchangerate_host"
	ExchangeRateProviderOpenExchangeRates = "openexchangerates"
	ExchangeRateProviderAPI               = "exchangerate_api"
	PriceProviderMock                     = "mock"
	PriceProviderAPI                      = "api"
	CalendarProviderGoogle                = "google"
	CalendarProviderMock                  = "mock"

	DefaultDevelopmentJWTSecret     = "change-me-in-development"
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
	DefaultDevelopmentCalendarKey   = "dev-calendar-token-key-32-bytes!"
	MinProductionJWTSecretLength    = 32
	MinProductionTokenLength        = 32
	MinProductionDBPassword         = 16
)

// Config is the root application configuration. It is loaded from a YAML file
// with environment-variable overrides, then validated during bootstrap.
type Config struct {
	Env                  string                     `yaml:"env" env:"APP_ENV" env-default:"local" validate:"required,oneof=local staging production development test"`
	HTTPServer           HTTPServer                 `yaml:"http_server" validate:"required"`
	Postgres             postgres.Config            `yaml:"postgres" validate:"required"`
	Auth                 AuthConfig                 `yaml:"auth" validate:"required"`
	Internal             InternalConfig             `yaml:"internal" validate:"required"`
	CORS                 CORSConfig                 `yaml:"cors" validate:"required"`
	PlaceProvider        PlaceProviderConfig        `yaml:"place_provider" validate:"required"`
	RouteProvider        RouteProviderConfig        `yaml:"route_provider" validate:"required"`
	WeatherProvider      WeatherProviderConfig      `yaml:"weather_provider" validate:"required"`
	ExchangeRateProvider ExchangeRateProviderConfig `yaml:"exchange_rate_provider" validate:"required"`
	PriceProvider        PriceProviderConfig        `yaml:"price_provider" validate:"required"`
	Calendar             CalendarConfig             `yaml:"calendar" validate:"required"`
	Ops                  OpsConfig                  `yaml:"ops"`
	ProviderLimits       ProviderLimitsConfig       `yaml:"provider_limits"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDR" env-default:":8084" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// CORSConfig controls browser access to External Integrations Service.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,DELETE,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

type AuthConfig struct {
	JWTAccessSecret string `yaml:"jwt_access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development"`
	HeaderName      string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
}

type InternalConfig struct {
	ServiceToken string `yaml:"service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token" validate:"required"`
}

// PlaceProviderConfig selects the place provider adapter.
type PlaceProviderConfig struct {
	Provider                 string `yaml:"provider" env:"PLACE_PROVIDER" env-default:"mock"`
	FallbackToMock           bool   `yaml:"fallback_to_mock" env:"PLACE_PROVIDER_FALLBACK_TO_MOCK" env-default:"true"`
	FoursquareAPIKey         string `yaml:"foursquare_api_key" env:"FOURSQUARE_API_KEY"`
	FoursquareBaseURL        string `yaml:"foursquare_base_url" env:"FOURSQUARE_BASE_URL" env-default:"https://api.foursquare.com/v3"`
	FoursquareTimeoutSeconds int    `yaml:"foursquare_timeout_seconds" env:"FOURSQUARE_TIMEOUT_SECONDS" env-default:"8"`
	GooglePlacesAPIKey       string `yaml:"google_places_api_key" env:"GOOGLE_PLACES_API_KEY"`
	MapboxAPIKey             string `yaml:"mapbox_api_key" env:"MAPBOX_API_KEY"`
}

// RouteProviderConfig selects the route-estimation provider adapter. The mock
// provider remains the default and fallback; the ORS fields configure the real
// OpenRouteService provider, which is opt-in via ROUTE_PROVIDER=ors.
type RouteProviderConfig struct {
	Provider          string `yaml:"provider" env:"ROUTE_PROVIDER" env-default:"mock"`
	FallbackToMock    bool   `yaml:"fallback_to_mock" env:"ROUTE_PROVIDER_FALLBACK_TO_MOCK" env-default:"true"`
	TimeoutSeconds    int    `yaml:"timeout_seconds" env:"ROUTE_PROVIDER_TIMEOUT_SECONDS" env-default:"8"`
	ORSAPIKey         string `yaml:"ors_api_key" env:"ORS_API_KEY"`
	ORSBaseURL        string `yaml:"ors_base_url" env:"ORS_BASE_URL" env-default:"https://api.openrouteservice.org"`
	ORSProfileWalking string `yaml:"ors_profile_walking" env:"ORS_PROFILE_WALKING" env-default:"foot-walking"`
	ORSProfileDriving string `yaml:"ors_profile_driving" env:"ORS_PROFILE_DRIVING" env-default:"driving-car"`
	ORSProfileCycling string `yaml:"ors_profile_cycling" env:"ORS_PROFILE_CYCLING" env-default:"cycling-regular"`
	CacheEnabled      bool   `yaml:"cache_enabled" env:"ROUTE_CACHE_ENABLED" env-default:"true"`
	CacheTTLSeconds   int    `yaml:"cache_ttl_seconds" env:"ROUTE_CACHE_TTL_SECONDS" env-default:"21600"`
	// Documented for future real providers; unused in v1.
	OSRMBaseURL       string `yaml:"osrm_base_url" env:"OSRM_BASE_URL"`
	MapboxAccessToken string `yaml:"mapbox_access_token" env:"MAPBOX_ACCESS_TOKEN"`
	GoogleMapsAPIKey  string `yaml:"google_maps_api_key" env:"GOOGLE_MAPS_API_KEY"`
}

// WeatherProviderConfig selects the weather provider adapter. The mock provider
// remains the default and fallback; the OpenWeather fields configure the real
// OpenWeatherMap provider, which is opt-in via WEATHER_PROVIDER=openweathermap.
type WeatherProviderConfig struct {
	Provider           string `yaml:"provider" env:"WEATHER_PROVIDER" env-default:"mock"`
	FallbackToMock     bool   `yaml:"fallback_to_mock" env:"WEATHER_PROVIDER_FALLBACK_TO_MOCK" env-default:"true"`
	TimeoutSeconds     int    `yaml:"timeout_seconds" env:"WEATHER_PROVIDER_TIMEOUT_SECONDS" env-default:"8"`
	OpenWeatherAPIKey  string `yaml:"openweather_api_key" env:"OPENWEATHER_API_KEY"`
	OpenWeatherBaseURL string `yaml:"openweather_base_url" env:"OPENWEATHER_BASE_URL" env-default:"https://api.openweathermap.org"`
	OpenWeatherUnits   string `yaml:"openweather_units" env:"OPENWEATHER_UNITS" env-default:"metric"`
	CacheEnabled       bool   `yaml:"cache_enabled" env:"WEATHER_CACHE_ENABLED" env-default:"true"`
	CacheTTLSeconds    int    `yaml:"cache_ttl_seconds" env:"WEATHER_CACHE_TTL_SECONDS" env-default:"3600"`
	// Documented for future real providers; unused in v1.
	OpenMeteoBaseURL string `yaml:"open_meteo_base_url" env:"OPEN_METEO_BASE_URL"`
	WeatherAPIKey    string `yaml:"weather_api_key" env:"WEATHER_API_KEY"`
}

// ExchangeRateProviderConfig selects the exchange-rate provider adapter. The
// mock provider remains the default and fallback for local development.
type ExchangeRateProviderConfig struct {
	Provider        string `yaml:"provider" env:"EXCHANGE_RATE_PROVIDER" env-default:"mock"`
	FallbackToMock  bool   `yaml:"fallback_to_mock" env:"EXCHANGE_RATE_PROVIDER_FALLBACK_TO_MOCK" env-default:"true"`
	TimeoutSeconds  int    `yaml:"timeout_seconds" env:"EXCHANGE_RATE_PROVIDER_TIMEOUT_SECONDS" env-default:"8"`
	BaseURL         string `yaml:"base_url" env:"EXCHANGE_RATE_BASE_URL"`
	APIKey          string `yaml:"api_key" env:"EXCHANGE_RATE_API_KEY"`
	CacheEnabled    bool   `yaml:"cache_enabled" env:"EXCHANGE_RATE_CACHE_ENABLED" env-default:"true"`
	CacheTTLSeconds int    `yaml:"cache_ttl_seconds" env:"EXCHANGE_RATE_CACHE_TTL_SECONDS" env-default:"21600"`
}

// PriceProviderConfig selects the attraction/ticket price provider adapter.
// Mock is the v1 default. The API fields are placeholders for future real
// providers and are intentionally server-side only.
type PriceProviderConfig struct {
	Provider        string `yaml:"provider" env:"PRICE_PROVIDER" env-default:"mock"`
	FallbackToMock  bool   `yaml:"fallback_to_mock" env:"PRICE_PROVIDER_FALLBACK_TO_MOCK" env-default:"true"`
	TimeoutSeconds  int    `yaml:"timeout_seconds" env:"PRICE_PROVIDER_TIMEOUT_SECONDS" env-default:"8"`
	CacheEnabled    bool   `yaml:"cache_enabled" env:"PRICE_CACHE_ENABLED" env-default:"true"`
	CacheTTLSeconds int    `yaml:"cache_ttl_seconds" env:"PRICE_CACHE_TTL_SECONDS" env-default:"86400"`
	DefaultCurrency string `yaml:"default_currency" env:"PRICE_ENRICHMENT_DEFAULT_CURRENCY" env-default:"EUR"`
	BaseURL         string `yaml:"base_url" env:"PRICE_API_BASE_URL"`
	APIKey          string `yaml:"api_key" env:"PRICE_API_KEY"`
}

type CalendarConfig struct {
	Enabled            bool   `yaml:"enabled" env:"GOOGLE_CALENDAR_ENABLED" env-default:"true"`
	Provider           string `yaml:"provider" env:"CALENDAR_PROVIDER" env-default:"mock"`
	GoogleClientID     string `yaml:"google_client_id" env:"GOOGLE_OAUTH_CLIENT_ID"`
	GoogleClientSecret string `yaml:"google_client_secret" env:"GOOGLE_OAUTH_CLIENT_SECRET"`
	GoogleRedirectURL  string `yaml:"google_redirect_url" env:"GOOGLE_OAUTH_REDIRECT_URL" env-default:"http://localhost:8084/calendar/google/callback"`
	GoogleScopes       string `yaml:"google_scopes" env:"GOOGLE_CALENDAR_SCOPES" env-default:"https://www.googleapis.com/auth/calendar.events"`
	EncryptionKey      string `yaml:"encryption_key" env:"CALENDAR_TOKEN_ENCRYPTION_KEY" env-default:"dev-calendar-token-key-32-bytes!"`
	OAuthStateTTL      int    `yaml:"oauth_state_ttl_seconds" env:"CALENDAR_OAUTH_STATE_TTL_SECONDS" env-default:"600" validate:"min=60"`
	PublicWebBaseURL   string `yaml:"public_web_base_url" env:"PUBLIC_WEB_BASE_URL" env-default:"http://localhost:3000"`
	DefaultTimeZone    string `yaml:"default_time_zone" env:"DEFAULT_CALENDAR_TIMEZONE" env-default:"Europe/Bratislava"`
	GoogleAuthURL      string `yaml:"google_auth_url" env:"GOOGLE_OAUTH_AUTH_URL" env-default:"https://accounts.google.com/o/oauth2/v2/auth"`
	GoogleTokenURL     string `yaml:"google_token_url" env:"GOOGLE_OAUTH_TOKEN_URL" env-default:"https://oauth2.googleapis.com/token"`
	GoogleUserInfoURL  string `yaml:"google_user_info_url" env:"GOOGLE_USERINFO_URL" env-default:"https://www.googleapis.com/oauth2/v2/userinfo"`
	GoogleCalendarAPI  string `yaml:"google_calendar_api" env:"GOOGLE_CALENDAR_API_URL" env-default:"https://www.googleapis.com/calendar/v3"`
	MockAccountEmail   string `yaml:"mock_account_email" env:"MOCK_GOOGLE_ACCOUNT_EMAIL" env-default:"mock-calendar@example.local"`
	MockEventLinkBase  string `yaml:"mock_event_link_base" env:"MOCK_GOOGLE_EVENT_LINK_BASE" env-default:"http://localhost:3000/mock-calendar/events"`
}

type OpsConfig struct {
	DashboardEnabled     bool   `yaml:"dashboard_enabled" env:"OPS_DASHBOARD_ENABLED" env-default:"false"`
	AdminEmails          string `yaml:"admin_emails" env:"OPS_ADMIN_EMAILS"`
	InternalServiceToken string `yaml:"internal_service_token" env:"OPS_INTERNAL_SERVICE_TOKEN"`
}

// ProviderLimitsConfig configures the central per-provider rate-limit and
// daily-quota guard. Enforcement is off by default so local development stays
// permissive; staging/production should set PROVIDER_LIMITS_ENABLED=true.
//
// Rate limits are per minute; a value of 0 means unlimited. Daily quotas are a
// per-provider cap across all of that provider's operations; a value of 0 means
// unlimited. Both must be non-negative. Each provider call costs 1 unit in v1.
type ProviderLimitsConfig struct {
	Enabled       bool   `yaml:"enabled" env:"PROVIDER_LIMITS_ENABLED" env-default:"false"`
	FailOpen      bool   `yaml:"fail_open" env:"PROVIDER_LIMITS_FAIL_OPEN" env-default:"true"`
	Timezone      string `yaml:"timezone" env:"PROVIDER_LIMITS_TIMEZONE" env-default:"UTC"`
	RateMaxWaitMS int    `yaml:"rate_max_wait_ms" env:"PROVIDER_RATE_LIMIT_MAX_WAIT_MS" env-default:"0"`

	FoursquareRatePerMinute int   `yaml:"foursquare_rate_per_minute" env:"FOURSQUARE_RATE_LIMIT_PER_MINUTE" env-default:"50"`
	FoursquareBurst         int   `yaml:"foursquare_burst" env:"FOURSQUARE_RATE_LIMIT_BURST" env-default:"10"`
	FoursquareDailyQuota    int64 `yaml:"foursquare_daily_quota" env:"FOURSQUARE_DAILY_QUOTA" env-default:"900"`

	ORSRatePerMinute int   `yaml:"ors_rate_per_minute" env:"ORS_RATE_LIMIT_PER_MINUTE" env-default:"30"`
	ORSBurst         int   `yaml:"ors_burst" env:"ORS_RATE_LIMIT_BURST" env-default:"5"`
	ORSDailyQuota    int64 `yaml:"ors_daily_quota" env:"ORS_DAILY_QUOTA" env-default:"1500"`

	OpenWeatherRatePerMinute int   `yaml:"openweather_rate_per_minute" env:"OPENWEATHER_RATE_LIMIT_PER_MINUTE" env-default:"60"`
	OpenWeatherBurst         int   `yaml:"openweather_burst" env:"OPENWEATHER_RATE_LIMIT_BURST" env-default:"10"`
	OpenWeatherDailyQuota    int64 `yaml:"openweather_daily_quota" env:"OPENWEATHER_DAILY_QUOTA" env-default:"1000"`

	GoogleCalendarRatePerMinute int   `yaml:"google_calendar_rate_per_minute" env:"GOOGLE_CALENDAR_RATE_LIMIT_PER_MINUTE" env-default:"30"`
	GoogleCalendarBurst         int   `yaml:"google_calendar_burst" env:"GOOGLE_CALENDAR_RATE_LIMIT_BURST" env-default:"5"`
	GoogleCalendarDailyQuota    int64 `yaml:"google_calendar_daily_quota" env:"GOOGLE_CALENDAR_DAILY_QUOTA" env-default:"1000"`

	ExchangeRateRatePerMinute int   `yaml:"exchange_rate_rate_per_minute" env:"EXCHANGE_RATE_LIMIT_PER_MINUTE" env-default:"30"`
	ExchangeRateBurst         int   `yaml:"exchange_rate_burst" env:"EXCHANGE_RATE_LIMIT_BURST" env-default:"5"`
	ExchangeRateDailyQuota    int64 `yaml:"exchange_rate_daily_quota" env:"EXCHANGE_RATE_DAILY_QUOTA" env-default:"1000"`

	PriceRatePerMinute int   `yaml:"price_rate_per_minute" env:"PRICE_PROVIDER_RATE_LIMIT_PER_MINUTE" env-default:"60"`
	PriceBurst         int   `yaml:"price_burst" env:"PRICE_PROVIDER_RATE_LIMIT_BURST" env-default:"10"`
	PriceDailyQuota    int64 `yaml:"price_daily_quota" env:"PRICE_PROVIDER_DAILY_QUOTA" env-default:"1000"`
}

func (c CalendarConfig) StateTTL() time.Duration {
	return time.Duration(c.OAuthStateTTL) * time.Second
}

func (c CalendarConfig) Scopes() []string {
	parts := strings.Split(c.GoogleScopes, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, field := range strings.Fields(part) {
			if trimmed := strings.TrimSpace(field); trimmed != "" {
				scopes = append(scopes, trimmed)
			}
		}
	}
	return scopes
}

// MustLoad loads and validates the configuration, panicking on any error.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	return cfg
}

// Load reads configuration from the given YAML path, or environment only when
// path is empty, and validates it.
func Load(path string) (*Config, error) {
	var cfg Config

	if path != "" {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("read config file %q: %w", path, err)
		}
	} else if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env config: %w", err)
	}

	cfg.applyDefaults()

	v := validator.New()
	if err := v.Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if err := cfg.validateSecrets(); err != nil {
		return nil, err
	}

	cfg.PlaceProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.PlaceProvider.Provider))
	if cfg.PlaceProvider.Provider == "" {
		return nil, fmt.Errorf("PLACE_PROVIDER is required")
	}

	cfg.PlaceProvider.GooglePlacesAPIKey = strings.TrimSpace(cfg.PlaceProvider.GooglePlacesAPIKey)
	cfg.PlaceProvider.MapboxAPIKey = strings.TrimSpace(cfg.PlaceProvider.MapboxAPIKey)
	cfg.PlaceProvider.FoursquareAPIKey = strings.TrimSpace(cfg.PlaceProvider.FoursquareAPIKey)
	cfg.PlaceProvider.FoursquareBaseURL = strings.TrimRight(strings.TrimSpace(cfg.PlaceProvider.FoursquareBaseURL), "/")
	if cfg.PlaceProvider.FoursquareBaseURL == "" {
		cfg.PlaceProvider.FoursquareBaseURL = "https://api.foursquare.com/v3"
	}
	if cfg.PlaceProvider.FoursquareTimeoutSeconds <= 0 {
		cfg.PlaceProvider.FoursquareTimeoutSeconds = 8
	}

	cfg.RouteProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.RouteProvider.Provider))
	if cfg.RouteProvider.Provider == "" {
		cfg.RouteProvider.Provider = RouteProviderMock
	}

	cfg.RouteProvider.ORSAPIKey = strings.TrimSpace(cfg.RouteProvider.ORSAPIKey)
	cfg.RouteProvider.ORSBaseURL = strings.TrimRight(strings.TrimSpace(cfg.RouteProvider.ORSBaseURL), "/")
	if cfg.RouteProvider.ORSBaseURL == "" {
		cfg.RouteProvider.ORSBaseURL = "https://api.openrouteservice.org"
	}
	cfg.RouteProvider.ORSProfileWalking = strings.TrimSpace(cfg.RouteProvider.ORSProfileWalking)
	if cfg.RouteProvider.ORSProfileWalking == "" {
		cfg.RouteProvider.ORSProfileWalking = "foot-walking"
	}
	cfg.RouteProvider.ORSProfileDriving = strings.TrimSpace(cfg.RouteProvider.ORSProfileDriving)
	if cfg.RouteProvider.ORSProfileDriving == "" {
		cfg.RouteProvider.ORSProfileDriving = "driving-car"
	}
	cfg.RouteProvider.ORSProfileCycling = strings.TrimSpace(cfg.RouteProvider.ORSProfileCycling)
	if cfg.RouteProvider.ORSProfileCycling == "" {
		cfg.RouteProvider.ORSProfileCycling = "cycling-regular"
	}
	if cfg.RouteProvider.TimeoutSeconds <= 0 {
		cfg.RouteProvider.TimeoutSeconds = 8
	}
	if cfg.RouteProvider.CacheTTLSeconds <= 0 {
		cfg.RouteProvider.CacheTTLSeconds = 21600
	}
	cfg.RouteProvider.OSRMBaseURL = strings.TrimSpace(cfg.RouteProvider.OSRMBaseURL)
	cfg.RouteProvider.MapboxAccessToken = strings.TrimSpace(cfg.RouteProvider.MapboxAccessToken)
	cfg.RouteProvider.GoogleMapsAPIKey = strings.TrimSpace(cfg.RouteProvider.GoogleMapsAPIKey)

	cfg.WeatherProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.WeatherProvider.Provider))
	if cfg.WeatherProvider.Provider == "" {
		cfg.WeatherProvider.Provider = WeatherProviderMock
	}
	cfg.WeatherProvider.OpenWeatherAPIKey = strings.TrimSpace(cfg.WeatherProvider.OpenWeatherAPIKey)
	cfg.WeatherProvider.OpenWeatherBaseURL = strings.TrimRight(strings.TrimSpace(cfg.WeatherProvider.OpenWeatherBaseURL), "/")
	if cfg.WeatherProvider.OpenWeatherBaseURL == "" {
		cfg.WeatherProvider.OpenWeatherBaseURL = "https://api.openweathermap.org"
	}
	cfg.WeatherProvider.OpenWeatherUnits = strings.ToLower(strings.TrimSpace(cfg.WeatherProvider.OpenWeatherUnits))
	if cfg.WeatherProvider.OpenWeatherUnits == "" {
		cfg.WeatherProvider.OpenWeatherUnits = "metric"
	}
	if cfg.WeatherProvider.TimeoutSeconds <= 0 {
		cfg.WeatherProvider.TimeoutSeconds = 8
	}
	if cfg.WeatherProvider.CacheTTLSeconds <= 0 {
		cfg.WeatherProvider.CacheTTLSeconds = 3600
	}
	cfg.WeatherProvider.OpenMeteoBaseURL = strings.TrimSpace(cfg.WeatherProvider.OpenMeteoBaseURL)
	cfg.WeatherProvider.WeatherAPIKey = strings.TrimSpace(cfg.WeatherProvider.WeatherAPIKey)

	cfg.ExchangeRateProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.ExchangeRateProvider.Provider))
	if cfg.ExchangeRateProvider.Provider == "" {
		cfg.ExchangeRateProvider.Provider = ExchangeRateProviderMock
	}
	if cfg.ExchangeRateProvider.TimeoutSeconds <= 0 {
		cfg.ExchangeRateProvider.TimeoutSeconds = 8
	}
	cfg.ExchangeRateProvider.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.ExchangeRateProvider.BaseURL), "/")
	cfg.ExchangeRateProvider.APIKey = strings.TrimSpace(cfg.ExchangeRateProvider.APIKey)
	if cfg.ExchangeRateProvider.CacheTTLSeconds <= 0 {
		cfg.ExchangeRateProvider.CacheTTLSeconds = 21600
	}

	cfg.PriceProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.PriceProvider.Provider))
	if cfg.PriceProvider.Provider == "" {
		cfg.PriceProvider.Provider = PriceProviderMock
	}
	if cfg.PriceProvider.TimeoutSeconds <= 0 {
		cfg.PriceProvider.TimeoutSeconds = 8
	}
	if cfg.PriceProvider.CacheTTLSeconds <= 0 {
		cfg.PriceProvider.CacheTTLSeconds = 86400
	}
	cfg.PriceProvider.DefaultCurrency = strings.ToUpper(strings.TrimSpace(cfg.PriceProvider.DefaultCurrency))
	if cfg.PriceProvider.DefaultCurrency == "" {
		cfg.PriceProvider.DefaultCurrency = "EUR"
	}
	cfg.PriceProvider.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.PriceProvider.BaseURL), "/")
	cfg.PriceProvider.APIKey = strings.TrimSpace(cfg.PriceProvider.APIKey)

	cfg.Calendar.Provider = strings.ToLower(strings.TrimSpace(cfg.Calendar.Provider))
	if cfg.Calendar.Provider == "" {
		cfg.Calendar.Provider = CalendarProviderMock
	}
	if cfg.Calendar.Provider != CalendarProviderGoogle && cfg.Calendar.Provider != CalendarProviderMock {
		return nil, fmt.Errorf("CALENDAR_PROVIDER must be google or mock")
	}
	cfg.Calendar.GoogleClientID = strings.TrimSpace(cfg.Calendar.GoogleClientID)
	cfg.Calendar.GoogleClientSecret = strings.TrimSpace(cfg.Calendar.GoogleClientSecret)
	cfg.Calendar.GoogleRedirectURL = strings.TrimSpace(cfg.Calendar.GoogleRedirectURL)
	cfg.Calendar.GoogleScopes = strings.TrimSpace(cfg.Calendar.GoogleScopes)
	cfg.Calendar.PublicWebBaseURL = strings.TrimRight(strings.TrimSpace(cfg.Calendar.PublicWebBaseURL), "/")
	cfg.Calendar.DefaultTimeZone = strings.TrimSpace(cfg.Calendar.DefaultTimeZone)
	cfg.Calendar.GoogleAuthURL = strings.TrimSpace(cfg.Calendar.GoogleAuthURL)
	cfg.Calendar.GoogleTokenURL = strings.TrimSpace(cfg.Calendar.GoogleTokenURL)
	cfg.Calendar.GoogleUserInfoURL = strings.TrimSpace(cfg.Calendar.GoogleUserInfoURL)
	cfg.Calendar.GoogleCalendarAPI = strings.TrimRight(strings.TrimSpace(cfg.Calendar.GoogleCalendarAPI), "/")
	cfg.Calendar.MockAccountEmail = strings.TrimSpace(cfg.Calendar.MockAccountEmail)
	cfg.Calendar.MockEventLinkBase = strings.TrimRight(strings.TrimSpace(cfg.Calendar.MockEventLinkBase), "/")
	if cfg.Calendar.Enabled && cfg.Calendar.Provider == CalendarProviderGoogle {
		if cfg.Calendar.GoogleClientID == "" {
			return nil, fmt.Errorf("GOOGLE_OAUTH_CLIENT_ID is required when CALENDAR_PROVIDER=google")
		}
		if cfg.Calendar.GoogleClientSecret == "" {
			return nil, fmt.Errorf("GOOGLE_OAUTH_CLIENT_SECRET is required when CALENDAR_PROVIDER=google")
		}
		if err := validateHTTPURL("GOOGLE_OAUTH_REDIRECT_URL", cfg.Calendar.GoogleRedirectURL, cfg.IsProduction()); err != nil {
			return nil, err
		}
	}
	if cfg.Calendar.Enabled && len(cfg.Calendar.Scopes()) == 0 {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_SCOPES is required when calendar is enabled")
	}
	if err := cfg.validateProviders(); err != nil {
		return nil, err
	}
	if err := cfg.validatePostgres(); err != nil {
		return nil, err
	}
	if err := cfg.validateCORS(); err != nil {
		return nil, err
	}
	if err := cfg.validatePublicURLs(); err != nil {
		return nil, err
	}
	if err := cfg.validateProviderLimits(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) IsStrictEnv() bool { return c.Env == "staging" || c.Env == "production" }

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && isLocalEnv(c.Env) {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,DELETE,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}

func (c *Config) validateSecrets() error {
	jwtSecret := strings.TrimSpace(c.Auth.JWTAccessSecret)
	if jwtSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.IsStrictEnv() && isUnsafeSecret(jwtSecret, DefaultDevelopmentJWTSecret) {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use a development default in %s", c.Env)
	}
	if c.IsStrictEnv() && len(jwtSecret) < MinProductionJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in %s", MinProductionJWTSecretLength, c.Env)
	}
	c.Auth.JWTAccessSecret = jwtSecret

	internalToken := strings.TrimSpace(c.Internal.ServiceToken)
	if internalToken == "" {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if c.IsStrictEnv() && isUnsafeSecret(internalToken, DefaultDevelopmentInternalToken) {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must not use a development default in %s", c.Env)
	}
	if c.IsStrictEnv() && len(internalToken) < MinProductionTokenLength {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must be at least %d characters in %s", MinProductionTokenLength, c.Env)
	}
	c.Internal.ServiceToken = internalToken

	key := strings.TrimSpace(c.Calendar.EncryptionKey)
	if c.Calendar.Enabled {
		switch len([]byte(key)) {
		case 16, 24, 32:
			if c.IsStrictEnv() && isUnsafeSecret(key, DefaultDevelopmentCalendarKey) {
				return fmt.Errorf("CALENDAR_TOKEN_ENCRYPTION_KEY must not use a development default in %s", c.Env)
			}
			c.Calendar.EncryptionKey = key
		default:
			return fmt.Errorf("CALENDAR_TOKEN_ENCRYPTION_KEY must be 16, 24, or 32 bytes")
		}
	}
	return nil
}

func (c *Config) validateProviders() error {
	switch c.PlaceProvider.Provider {
	case PlaceProviderMock:
	case PlaceProviderFoursquare:
		if c.PlaceProvider.FoursquareAPIKey == "" {
			return fmt.Errorf("FOURSQUARE_API_KEY is required when PLACE_PROVIDER=foursquare")
		}
	default:
		return fmt.Errorf("PLACE_PROVIDER must be mock or foursquare")
	}

	switch c.RouteProvider.Provider {
	case RouteProviderMock:
	case RouteProviderORS:
		if c.RouteProvider.ORSAPIKey == "" {
			return fmt.Errorf("ORS_API_KEY is required when ROUTE_PROVIDER=ors")
		}
	default:
		return fmt.Errorf("ROUTE_PROVIDER must be mock or ors")
	}

	switch c.WeatherProvider.Provider {
	case WeatherProviderMock:
	case WeatherProviderOpenWeather:
		if c.WeatherProvider.OpenWeatherAPIKey == "" {
			return fmt.Errorf("OPENWEATHER_API_KEY is required when WEATHER_PROVIDER=openweathermap")
		}
	default:
		return fmt.Errorf("WEATHER_PROVIDER must be mock or openweathermap")
	}

	switch c.ExchangeRateProvider.Provider {
	case ExchangeRateProviderMock:
	case ExchangeRateProviderHost, ExchangeRateProviderOpenExchangeRates, ExchangeRateProviderAPI:
		if c.ExchangeRateProvider.APIKey == "" {
			return fmt.Errorf("EXCHANGE_RATE_API_KEY is required when EXCHANGE_RATE_PROVIDER=%s", c.ExchangeRateProvider.Provider)
		}
	default:
		return fmt.Errorf("EXCHANGE_RATE_PROVIDER must be mock, exchangerate_host, openexchangerates, or exchangerate_api")
	}

	switch c.PriceProvider.Provider {
	case PriceProviderMock:
	case PriceProviderAPI:
		if c.PriceProvider.APIKey == "" {
			return fmt.Errorf("PRICE_API_KEY is required when PRICE_PROVIDER=api")
		}
	default:
		return fmt.Errorf("PRICE_PROVIDER must be mock or api")
	}
	return nil
}

func (c *Config) validatePostgres() error {
	password := strings.TrimSpace(c.Postgres.Password)
	if password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if c.IsStrictEnv() {
		if isUnsafeSecret(password, "postgres") {
			return fmt.Errorf("POSTGRES_PASSWORD must not use a development default in %s", c.Env)
		}
		if len(password) < MinProductionDBPassword {
			return fmt.Errorf("POSTGRES_PASSWORD must be at least %d characters in %s", MinProductionDBPassword, c.Env)
		}
	}
	c.Postgres.Password = password
	return nil
}

func (c *Config) validateCORS() error {
	origins := strings.TrimSpace(c.CORS.AllowedOrigins)
	if origins == "" {
		if c.IsStrictEnv() {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS is required in %s", c.Env)
		}
		return nil
	}
	if c.IsStrictEnv() && origins == "*" {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be wildcard in %s", c.Env)
	}
	if c.IsStrictEnv() {
		for _, origin := range strings.Split(origins, ",") {
			if err := validateHTTPURL("CORS_ALLOWED_ORIGINS", origin, false); err != nil {
				return err
			}
			if c.IsProduction() && isLocalhostURL(origin) {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not use localhost in production")
			}
		}
	}
	c.CORS.AllowedOrigins = origins
	return nil
}

func (c *Config) validatePublicURLs() error {
	if c.Calendar.Enabled {
		if err := validateHTTPURL("PUBLIC_WEB_BASE_URL", c.Calendar.PublicWebBaseURL, c.IsProduction()); err != nil {
			return err
		}
		if c.IsProduction() && isLocalhostURL(c.Calendar.PublicWebBaseURL) {
			return fmt.Errorf("PUBLIC_WEB_BASE_URL must not use localhost in production")
		}
	}
	return nil
}

// validateProviderLimits normalizes and validates the provider limit config.
// Negative limits are always invalid. A rate limit or daily quota of 0 means
// unlimited (documented). The timezone must be resolvable.
func (c *Config) validateProviderLimits() error {
	pl := &c.ProviderLimits
	pl.Timezone = strings.TrimSpace(pl.Timezone)
	if pl.Timezone == "" {
		pl.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(pl.Timezone); err != nil {
		return fmt.Errorf("PROVIDER_LIMITS_TIMEZONE %q is not a valid timezone: %w", pl.Timezone, err)
	}
	if pl.RateMaxWaitMS < 0 {
		return fmt.Errorf("PROVIDER_RATE_LIMIT_MAX_WAIT_MS must not be negative")
	}

	checks := []struct {
		name  string
		rate  int
		burst int
		quota int64
	}{
		{"FOURSQUARE", pl.FoursquareRatePerMinute, pl.FoursquareBurst, pl.FoursquareDailyQuota},
		{"ORS", pl.ORSRatePerMinute, pl.ORSBurst, pl.ORSDailyQuota},
		{"OPENWEATHER", pl.OpenWeatherRatePerMinute, pl.OpenWeatherBurst, pl.OpenWeatherDailyQuota},
		{"GOOGLE_CALENDAR", pl.GoogleCalendarRatePerMinute, pl.GoogleCalendarBurst, pl.GoogleCalendarDailyQuota},
		{"EXCHANGE_RATE", pl.ExchangeRateRatePerMinute, pl.ExchangeRateBurst, pl.ExchangeRateDailyQuota},
		{"PRICE_PROVIDER", pl.PriceRatePerMinute, pl.PriceBurst, pl.PriceDailyQuota},
	}
	for _, check := range checks {
		if check.rate < 0 {
			return fmt.Errorf("%s_RATE_LIMIT_PER_MINUTE must not be negative", check.name)
		}
		if check.burst < 0 {
			return fmt.Errorf("%s_RATE_LIMIT_BURST must not be negative", check.name)
		}
		if check.quota < 0 {
			return fmt.Errorf("%s_DAILY_QUOTA must not be negative", check.name)
		}
	}
	return nil
}

func validateHTTPURL(name, value string, requireHTTPS bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", name)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid http/https URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", name)
	}
	if requireHTTPS && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", name)
	}
	return nil
}

func isLocalhostURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isLocalEnv(env string) bool {
	return env == "local" || env == "development" || env == "test"
}

func isUnsafeSecret(value string, additional ...string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	disallowed := []string{"secret", "password", "dev", "changeme", "change-me", "guest", "admin"}
	disallowed = append(disallowed, additional...)
	for _, item := range disallowed {
		if normalized == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
