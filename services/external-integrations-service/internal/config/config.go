package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

const (
	PlaceProviderMock          = "mock"
	PlaceProviderFoursquare    = "foursquare"
	RouteProviderMock          = "mock"
	RouteProviderORS           = "ors"
	WeatherProviderMock        = "mock"
	WeatherProviderOpenWeather = "openweathermap"
	CalendarProviderGoogle     = "google"
	CalendarProviderMock       = "mock"

	DefaultDevelopmentJWTSecret     = "change-me-in-development"
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
)

// Config is the root application configuration. It is loaded from a YAML file
// with environment-variable overrides, then validated during bootstrap.
type Config struct {
	Env             string                `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production test"`
	HTTPServer      HTTPServer            `yaml:"http_server" validate:"required"`
	Postgres        postgres.Config       `yaml:"postgres" validate:"required"`
	Auth            AuthConfig            `yaml:"auth" validate:"required"`
	Internal        InternalConfig        `yaml:"internal" validate:"required"`
	CORS            CORSConfig            `yaml:"cors" validate:"required"`
	PlaceProvider   PlaceProviderConfig   `yaml:"place_provider" validate:"required"`
	RouteProvider   RouteProviderConfig   `yaml:"route_provider" validate:"required"`
	WeatherProvider WeatherProviderConfig `yaml:"weather_provider" validate:"required"`
	Calendar        CalendarConfig        `yaml:"calendar" validate:"required"`
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
	}
	if cfg.Calendar.Enabled && len(cfg.Calendar.Scopes()) == 0 {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_SCOPES is required when calendar is enabled")
	}

	return &cfg, nil
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
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
	if c.IsProduction() && jwtSecret == DefaultDevelopmentJWTSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use the development default in production")
	}
	c.Auth.JWTAccessSecret = jwtSecret

	internalToken := strings.TrimSpace(c.Internal.ServiceToken)
	if internalToken == "" {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if c.IsProduction() && internalToken == DefaultDevelopmentInternalToken {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must not use the development default in production")
	}
	c.Internal.ServiceToken = internalToken

	key := strings.TrimSpace(c.Calendar.EncryptionKey)
	if c.Calendar.Enabled {
		switch len([]byte(key)) {
		case 16, 24, 32:
			c.Calendar.EncryptionKey = key
		default:
			return fmt.Errorf("CALENDAR_TOKEN_ENCRYPTION_KEY must be 16, 24, or 32 bytes")
		}
	}
	return nil
}
