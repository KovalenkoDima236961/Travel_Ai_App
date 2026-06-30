package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// Config is the root application configuration. It is loaded from a YAML file
// (path passed via the -config flag) with environment-variable overrides, then
// validated using the project's validation package.
type Config struct {
	Env                string                   `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production"`
	HTTPServer         HTTPServer               `yaml:"http_server"`
	Auth               AuthConfig               `yaml:"auth"`
	CORS               CORSConfig               `yaml:"cors"`
	Postgres           postgres.Config          `yaml:"postgres"`
	ItineraryGenerator ItineraryGeneratorConfig `yaml:"itinerary_generator"`
	UserContext        UserContextConfig        `yaml:"user_context"`
	WeatherContext     WeatherContextConfig     `yaml:"weather_context"`
	PlaceEnrichment    PlaceEnrichmentConfig    `yaml:"place_enrichment"`
	UserLookup         UserLookupConfig         `yaml:"user_lookup"`
	PublicSharing      PublicSharingConfig      `yaml:"public_sharing"`
	Notifications      NotificationsConfig      `yaml:"notifications"`
	Presence           PresenceConfig           `yaml:"presence"`
	ActivityStream     ActivityStreamConfig     `yaml:"activity_stream"`
	EditLocks          EditLocksConfig          `yaml:"edit_locks"`
	GenerationJobs     GenerationJobsConfig     `yaml:"generation_jobs"`
	CalendarSync       CalendarSyncConfig       `yaml:"calendar_sync"`
	BudgetConversion   BudgetConversionConfig   `yaml:"budget_conversion"`
}

// NotificationsConfig controls synchronous in-app notification fan-out to the
// Notification Service after successful collaboration/comment/itinerary actions.
// When disabled, Trip Service makes no calls. FailOpen keeps a notification
// failure from breaking the originating action (the recommended v1 default).
type NotificationsConfig struct {
	Enabled                  bool   `yaml:"enabled" env:"NOTIFICATIONS_ENABLED" env-default:"true"`
	FailOpen                 bool   `yaml:"fail_open" env:"NOTIFICATIONS_FAIL_OPEN" env-default:"true"`
	NotificationServiceURL   string `yaml:"notification_service_url" env:"NOTIFICATION_SERVICE_URL" env-default:"http://notification-service:8086"`
	NotificationServiceToken string `yaml:"notification_service_token" env:"NOTIFICATION_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds           int    `yaml:"timeout_seconds" env:"NOTIFICATION_SERVICE_TIMEOUT_SECONDS" env-default:"3" validate:"min=1"`
}

// PresenceConfig controls instance-local real-time trip presence.
type PresenceConfig struct {
	Enabled                      bool `yaml:"enabled" env:"TRIP_PRESENCE_ENABLED" env-default:"true"`
	HeartbeatSeconds             int  `yaml:"heartbeat_seconds" env:"TRIP_PRESENCE_HEARTBEAT_SECONDS" env-default:"25" validate:"min=1"`
	StaleAfterSeconds            int  `yaml:"stale_after_seconds" env:"TRIP_PRESENCE_STALE_AFTER_SECONDS" env-default:"60" validate:"min=1"`
	MaxConnectionsPerUserPerTrip int  `yaml:"max_connections_per_user_per_trip" env:"TRIP_PRESENCE_MAX_CONNECTIONS_PER_USER_PER_TRIP" env-default:"5" validate:"min=1"`
	SendFullSnapshot             bool `yaml:"send_full_snapshot" env:"TRIP_PRESENCE_SEND_FULL_SNAPSHOT" env-default:"true"`
}

// ActivityStreamConfig controls instance-local real-time activity fan-out.
type ActivityStreamConfig struct {
	Enabled                      bool `yaml:"enabled" env:"TRIP_ACTIVITY_STREAM_ENABLED" env-default:"true"`
	HeartbeatSeconds             int  `yaml:"heartbeat_seconds" env:"TRIP_ACTIVITY_STREAM_HEARTBEAT_SECONDS" env-default:"25" validate:"min=1"`
	WriteTimeoutSeconds          int  `yaml:"write_timeout_seconds" env:"TRIP_ACTIVITY_STREAM_WRITE_TIMEOUT_SECONDS" env-default:"10" validate:"min=1"`
	MaxConnectionsPerUserPerTrip int  `yaml:"max_connections_per_user_per_trip" env:"TRIP_ACTIVITY_STREAM_MAX_CONNECTIONS_PER_USER_PER_TRIP" env-default:"5" validate:"min=1"`
	ClientBufferSize             int  `yaml:"client_buffer_size" env:"TRIP_ACTIVITY_STREAM_CLIENT_BUFFER_SIZE" env-default:"20" validate:"min=1"`
}

// EditLocksConfig controls instance-local advisory itinerary edit locks.
type EditLocksConfig struct {
	Enabled             bool `yaml:"enabled" env:"TRIP_EDIT_LOCKS_ENABLED" env-default:"true"`
	TTLSeconds          int  `yaml:"ttl_seconds" env:"TRIP_EDIT_LOCK_TTL_SECONDS" env-default:"180" validate:"min=1"`
	RenewSeconds        int  `yaml:"renew_seconds" env:"TRIP_EDIT_LOCK_RENEW_SECONDS" env-default:"45" validate:"min=1"`
	StaleCleanupSeconds int  `yaml:"stale_cleanup_seconds" env:"TRIP_EDIT_LOCK_STALE_CLEANUP_SECONDS" env-default:"30" validate:"min=1"`
}

type GenerationJobsConfig struct {
	Enabled                   bool `yaml:"enabled" env:"GENERATION_JOBS_ENABLED" env-default:"true"`
	WorkerEnabled             bool `yaml:"worker_enabled" env:"GENERATION_JOB_WORKER_ENABLED" env-default:"true"`
	WorkerPollIntervalSeconds int  `yaml:"worker_poll_interval_seconds" env:"GENERATION_JOB_WORKER_POLL_INTERVAL_SECONDS" env-default:"2" validate:"min=1"`
	WorkerMaxConcurrent       int  `yaml:"worker_max_concurrent" env:"GENERATION_JOB_WORKER_MAX_CONCURRENT" env-default:"1" validate:"min=1"`
	MaxRunningSeconds         int  `yaml:"max_running_seconds" env:"GENERATION_JOB_MAX_RUNNING_SECONDS" env-default:"600" validate:"min=1"`
	FailOpenNotifications     bool `yaml:"fail_open_notifications" env:"GENERATION_JOB_FAIL_OPEN_NOTIFICATIONS" env-default:"true"`
}

type CalendarSyncConfig struct {
	Enabled                        bool   `yaml:"enabled" env:"CALENDAR_SYNC_ENABLED" env-default:"true"`
	ExternalIntegrationsServiceURL string `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	InternalServiceToken           string `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds                 int    `yaml:"timeout_seconds" env:"CALENDAR_SYNC_TIMEOUT_SECONDS" env-default:"30" validate:"min=1"`
	DefaultTimeZone                string `yaml:"default_time_zone" env:"DEFAULT_CALENDAR_TIMEZONE" env-default:"Europe/Bratislava"`
}

type BudgetConversionConfig struct {
	Enabled                        bool   `yaml:"enabled" env:"BUDGET_CONVERSION_ENABLED" env-default:"true"`
	FailOpen                       bool   `yaml:"fail_open" env:"BUDGET_CONVERSION_FAIL_OPEN" env-default:"true"`
	ExternalIntegrationsServiceURL string `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	InternalServiceToken           string `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds                 int    `yaml:"timeout_seconds" env:"EXCHANGE_RATE_CLIENT_TIMEOUT_SECONDS" env-default:"8" validate:"min=1"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDRESS" env-default:":8080" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"150s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// AuthConfig controls local JWT validation for protected trip endpoints.
type AuthConfig struct {
	Required        bool   `yaml:"required" env:"AUTH_REQUIRED" env-default:"true"`
	JWTAccessSecret string `yaml:"jwt_access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development" validate:"required"`
	HeaderName      string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
	DevUserID       string `yaml:"dev_user_id" env:"DEV_USER_ID" env-default:"00000000-0000-0000-0000-000000000001" validate:"required,uuid"`
}

// CORSConfig controls browser access to the Trip Service API.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// ItineraryGeneratorConfig selects the itinerary generation adapter.
type ItineraryGeneratorConfig struct {
	Mode                     string `yaml:"mode" env:"ITINERARY_GENERATOR_MODE" env-default:"mock"`
	AIPlanningServiceURL     string `yaml:"ai_planning_service_url" env:"AI_PLANNING_SERVICE_URL" env-default:"http://ai-planning-service:8000"`
	AIPlanningTimeoutSeconds int    `yaml:"ai_planning_timeout_seconds" env:"AI_PLANNING_TIMEOUT_SECONDS" env-default:"120" validate:"min=1"`
}

// UserContextConfig controls optional profile/preferences loading from User
// Service before itinerary generation.
type UserContextConfig struct {
	Enabled        bool   `yaml:"enabled" env:"USER_CONTEXT_ENABLED" env-default:"true"`
	UserServiceURL string `yaml:"user_service_url" env:"USER_SERVICE_URL" env-default:"http://user-service:8083"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"USER_CONTEXT_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
	FailOpen       bool   `yaml:"fail_open" env:"USER_CONTEXT_FAIL_OPEN" env-default:"true"`
}

// WeatherContextConfig controls optional weather forecast loading from External
// Integrations Service before itinerary generation.
type WeatherContextConfig struct {
	Enabled                        bool   `yaml:"enabled" env:"WEATHER_CONTEXT_ENABLED" env-default:"true"`
	ExternalIntegrationsServiceURL string `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	TimeoutSeconds                 int    `yaml:"timeout_seconds" env:"WEATHER_CONTEXT_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
	FailOpen                       bool   `yaml:"fail_open" env:"WEATHER_CONTEXT_FAIL_OPEN" env-default:"true"`
}

// PlaceEnrichmentConfig controls optional automatic place matching after AI
// itinerary generation.
type PlaceEnrichmentConfig struct {
	Enabled                        bool    `yaml:"enabled" env:"PLACE_ENRICHMENT_ENABLED" env-default:"true"`
	ExternalIntegrationsServiceURL string  `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	FailOpen                       bool    `yaml:"fail_open" env:"PLACE_ENRICHMENT_FAIL_OPEN" env-default:"true"`
	TimeoutSeconds                 int     `yaml:"timeout_seconds" env:"PLACE_ENRICHMENT_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
	MinConfidence                  float64 `yaml:"min_confidence" env:"PLACE_ENRICHMENT_MIN_CONFIDENCE" env-default:"0.75" validate:"min=0,max=1"`
	MaxItems                       int     `yaml:"max_items" env:"PLACE_ENRICHMENT_MAX_ITEMS" env-default:"20" validate:"min=1"`
	OverwriteExisting              bool    `yaml:"overwrite_existing" env:"PLACE_ENRICHMENT_OVERWRITE_EXISTING" env-default:"false"`
}

// UserLookupConfig controls exact-email registered-user lookup for trip invites.
// The endpoint is internal to the compose network in v1.
type UserLookupConfig struct {
	AuthServiceURL string `yaml:"auth_service_url" env:"AUTH_SERVICE_URL" env-default:"http://auth-service:8081"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"USER_LOOKUP_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
}

// PublicSharingConfig controls read-only public trip share links.
type PublicSharingConfig struct {
	Enabled                     bool   `yaml:"enabled" env:"PUBLIC_SHARING_ENABLED" env-default:"true"`
	PublicWebBaseURL            string `yaml:"public_web_base_url" env:"PUBLIC_WEB_BASE_URL" env-default:"http://localhost:3000"`
	ShareTokenBytes             int    `yaml:"share_token_bytes" env:"SHARE_TOKEN_BYTES" env-default:"32" validate:"min=32,max=128"`
	PublicShareAccessSecret     string `yaml:"public_share_access_secret" env:"PUBLIC_SHARE_ACCESS_SECRET" env-default:"dev-public-share-secret-change-me" validate:"required"`
	PublicShareAccessTTLMinutes int    `yaml:"public_share_access_ttl_minutes" env:"PUBLIC_SHARE_ACCESS_TTL_MINUTES" env-default:"60" validate:"min=1"`
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// PresenceHeartbeatInterval returns the configured presence heartbeat period.
func (c *Config) PresenceHeartbeatInterval() time.Duration {
	return time.Duration(c.Presence.HeartbeatSeconds) * time.Second
}

// PresenceStaleAfter returns the configured stale-session threshold.
func (c *Config) PresenceStaleAfter() time.Duration {
	return time.Duration(c.Presence.StaleAfterSeconds) * time.Second
}

func (c *Config) ActivityStreamHeartbeatInterval() time.Duration {
	return time.Duration(c.ActivityStream.HeartbeatSeconds) * time.Second
}

func (c *Config) ActivityStreamWriteTimeout() time.Duration {
	return time.Duration(c.ActivityStream.WriteTimeoutSeconds) * time.Second
}

func (c *Config) EditLockTTL() time.Duration {
	return time.Duration(c.EditLocks.TTLSeconds) * time.Second
}

func (c *Config) EditLockRenewalInterval() time.Duration {
	return time.Duration(c.EditLocks.RenewSeconds) * time.Second
}

func (c *Config) EditLockCleanupInterval() time.Duration {
	return time.Duration(c.EditLocks.StaleCleanupSeconds) * time.Second
}

func (c *Config) GenerationJobWorkerPollInterval() time.Duration {
	return time.Duration(c.GenerationJobs.WorkerPollIntervalSeconds) * time.Second
}

func (c *Config) GenerationJobMaxRunning() time.Duration {
	return time.Duration(c.GenerationJobs.MaxRunningSeconds) * time.Second
}

// MustLoad loads and validates the configuration, panicking on any error.
// It is intended for use during application bootstrap.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	return cfg
}

// Load reads configuration from the given YAML path (or environment only when
// path is empty) and validates it.
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

	validator, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}
	if err := validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}
