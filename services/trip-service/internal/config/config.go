package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

const (
	DefaultDevelopmentJWTSecret         = "change-me-in-development"
	DefaultDevelopmentInternalToken     = "dev-internal-service-token"
	DefaultDevelopmentPublicShareSecret = "dev-public-share-secret-change-me"
	MinProductionJWTSecretLength        = 32
	MinProductionTokenLength            = 32
	MinProductionDBPassword             = 16
)

// Config is the root application configuration. It is loaded from a YAML file
// (path passed via the -config flag) with environment-variable overrides, then
// validated using the project's validation package.
type Config struct {
	Env                string                   `yaml:"env" env:"APP_ENV" env-default:"local" validate:"required,oneof=local staging production development test"`
	HTTPServer         HTTPServer               `yaml:"http_server"`
	Auth               AuthConfig               `yaml:"auth"`
	CORS               CORSConfig               `yaml:"cors"`
	Postgres           postgres.Config          `yaml:"postgres"`
	ItineraryGenerator ItineraryGeneratorConfig `yaml:"itinerary_generator"`
	UserContext        UserContextConfig        `yaml:"user_context"`
	WeatherContext     WeatherContextConfig     `yaml:"weather_context"`
	PlaceEnrichment    PlaceEnrichmentConfig    `yaml:"place_enrichment"`
	PriceEnrichment    PriceEnrichmentConfig    `yaml:"price_enrichment"`
	UserLookup         UserLookupConfig         `yaml:"user_lookup"`
	PublicSharing      PublicSharingConfig      `yaml:"public_sharing"`
	Notifications      NotificationsConfig      `yaml:"notifications"`
	Presence           PresenceConfig           `yaml:"presence"`
	ActivityStream     ActivityStreamConfig     `yaml:"activity_stream"`
	EditLocks          EditLocksConfig          `yaml:"edit_locks"`
	GenerationJobs     GenerationJobsConfig     `yaml:"generation_jobs"`
	CalendarSync       CalendarSyncConfig       `yaml:"calendar_sync"`
	BudgetConversion   BudgetConversionConfig   `yaml:"budget_conversion"`
	Ops                OpsConfig                `yaml:"ops"`
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
	Enabled                   bool   `yaml:"enabled" env:"GENERATION_JOBS_ENABLED" env-default:"true"`
	WorkerEnabled             bool   `yaml:"worker_enabled" env:"GENERATION_JOB_WORKER_ENABLED" env-default:"true"`
	DispatchMode              string `yaml:"dispatch_mode" env:"GENERATION_JOB_DISPATCH_MODE" env-default:"in_process" validate:"oneof=in_process queue"`
	WorkerPollIntervalSeconds int    `yaml:"worker_poll_interval_seconds" env:"GENERATION_JOB_WORKER_POLL_INTERVAL_SECONDS" env-default:"2" validate:"min=1"`
	WorkerMaxConcurrent       int    `yaml:"worker_max_concurrent" env:"GENERATION_JOB_WORKER_MAX_CONCURRENT" env-default:"1" validate:"min=1"`
	MaxRunningSeconds         int    `yaml:"max_running_seconds" env:"GENERATION_JOB_MAX_RUNNING_SECONDS" env-default:"600" validate:"min=1"`
	PublishTimeoutSeconds     int    `yaml:"publish_timeout_seconds" env:"GENERATION_JOB_PUBLISH_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
	PublishFailOpen           bool   `yaml:"publish_fail_open" env:"GENERATION_JOB_PUBLISH_FAIL_OPEN" env-default:"false"`
	RabbitMQURL               string `yaml:"rabbitmq_url" env:"RABBITMQ_URL" env-default:"amqp://guest:guest@rabbitmq:5672/"`
	RabbitMQExchange          string `yaml:"rabbitmq_exchange" env:"RABBITMQ_EXCHANGE" env-default:"trip.jobs.exchange"`
	RabbitMQDLX               string `yaml:"rabbitmq_dlx" env:"RABBITMQ_DLX" env-default:"trip.jobs.dlx"`
	QueueName                 string `yaml:"queue_name" env:"GENERATION_JOBS_QUEUE" env-default:"trip.generation.jobs"`
	RoutingKey                string `yaml:"routing_key" env:"GENERATION_JOBS_ROUTING_KEY" env-default:"trip.generation"`
	DeadLetterQueueName       string `yaml:"dead_letter_queue_name" env:"GENERATION_JOBS_DEAD_LETTER_QUEUE" env-default:"trip.generation.dead_letter"`
	DeadLetterRoutingKey      string `yaml:"dead_letter_routing_key" env:"GENERATION_JOBS_DEAD_LETTER_ROUTING_KEY" env-default:"trip.generation.dead"`
	RetryQueueName            string `yaml:"retry_queue_name" env:"GENERATION_JOBS_RETRY_QUEUE" env-default:"trip.generation.retry"`
	RetryRoutingKey           string `yaml:"retry_routing_key" env:"GENERATION_JOBS_RETRY_ROUTING_KEY" env-default:"trip.generation.retry"`
	RetryDelaySeconds         int    `yaml:"retry_delay_seconds" env:"GENERATION_JOBS_RETRY_DELAY_SECONDS" env-default:"10" validate:"min=1"`
	Prefetch                  int    `yaml:"prefetch" env:"GENERATION_JOBS_PREFETCH" env-default:"1" validate:"min=1"`
	MaxAttempts               int    `yaml:"max_attempts" env:"GENERATION_JOBS_MAX_ATTEMPTS" env-default:"3" validate:"min=1"`
	FailOpenNotifications     bool   `yaml:"fail_open_notifications" env:"GENERATION_JOB_FAIL_OPEN_NOTIFICATIONS" env-default:"true"`
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

// OpsConfig controls the internal allowlisted operations dashboard endpoints.
type OpsConfig struct {
	DashboardEnabled       bool   `yaml:"dashboard_enabled" env:"OPS_DASHBOARD_ENABLED" env-default:"false"`
	AdminEmails            string `yaml:"admin_emails" env:"OPS_ADMIN_EMAILS"`
	InternalServiceToken   string `yaml:"internal_service_token" env:"OPS_INTERNAL_SERVICE_TOKEN"`
	StaleRunningJobSeconds int    `yaml:"stale_running_job_seconds" env:"OPS_STALE_RUNNING_JOB_SECONDS" env-default:"900" validate:"min=1"`
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

// PriceEnrichmentConfig controls optional automatic provider ticket/attraction
// price estimates after generated items have been place-enriched.
type PriceEnrichmentConfig struct {
	Enabled                        bool    `yaml:"enabled" env:"PRICE_ENRICHMENT_ENABLED" env-default:"true"`
	ExternalIntegrationsServiceURL string  `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	InternalServiceToken           string  `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	FailOpen                       bool    `yaml:"fail_open" env:"PRICE_ENRICHMENT_FAIL_OPEN" env-default:"true"`
	TimeoutSeconds                 int     `yaml:"timeout_seconds" env:"PRICE_ENRICHMENT_TIMEOUT_SECONDS" env-default:"8" validate:"min=1"`
	OverwriteAICosts               bool    `yaml:"overwrite_ai_costs" env:"PRICE_ENRICHMENT_OVERWRITE_AI_COSTS" env-default:"false"`
	OverwriteManualCosts           bool    `yaml:"overwrite_manual_costs" env:"PRICE_ENRICHMENT_OVERWRITE_MANUAL_COSTS" env-default:"false"`
	MinMatchConfidence             float64 `yaml:"min_match_confidence" env:"PRICE_ENRICHMENT_MIN_MATCH_CONFIDENCE" env-default:"0.55" validate:"min=0,max=1"`
	MaxItems                       int     `yaml:"max_items" env:"PRICE_ENRICHMENT_MAX_ITEMS" env-default:"30" validate:"min=1"`
	DefaultCurrency                string  `yaml:"default_currency" env:"PRICE_ENRICHMENT_DEFAULT_CURRENCY" env-default:"EUR"`
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

func (c *Config) IsStrictEnv() bool { return c.Env == "staging" || c.Env == "production" }

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

func (c *Config) GenerationJobPublishTimeout() time.Duration {
	return time.Duration(c.GenerationJobs.PublishTimeoutSeconds) * time.Second
}

func (c *Config) OpsStaleRunningJobThreshold() time.Duration {
	return time.Duration(c.Ops.StaleRunningJobSeconds) * time.Second
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
	if err := cfg.validateStrictConfig(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && isLocalEnv(c.Env) {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}

func (c *Config) validateStrictConfig() error {
	if err := c.validatePostgres(); err != nil {
		return err
	}
	if err := c.validateAuth(); err != nil {
		return err
	}
	if err := c.validateCORS(); err != nil {
		return err
	}
	if err := c.validateServiceURLs(); err != nil {
		return err
	}
	if err := c.validatePublicSharing(); err != nil {
		return err
	}
	if err := c.validateGenerationJobs(); err != nil {
		return err
	}
	if err := c.validateInternalTokens(); err != nil {
		return err
	}
	if err := c.validateOps(); err != nil {
		return err
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

func (c *Config) validateAuth() error {
	secret := strings.TrimSpace(c.Auth.JWTAccessSecret)
	if secret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.IsStrictEnv() {
		if isUnsafeSecret(secret, DefaultDevelopmentJWTSecret) {
			return fmt.Errorf("JWT_ACCESS_SECRET must not use a development default in %s", c.Env)
		}
		if len(secret) < MinProductionJWTSecretLength {
			return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in %s", MinProductionJWTSecretLength, c.Env)
		}
	}
	c.Auth.JWTAccessSecret = secret
	if c.IsStrictEnv() && !c.Auth.Required {
		return fmt.Errorf("AUTH_REQUIRED must be true in %s", c.Env)
	}
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

func (c *Config) validateServiceURLs() error {
	checks := []struct {
		name         string
		value        string
		enabled      bool
		requireHTTPS bool
	}{
		{"AI_PLANNING_SERVICE_URL", c.ItineraryGenerator.AIPlanningServiceURL, strings.TrimSpace(c.ItineraryGenerator.Mode) == "http", false},
		{"USER_SERVICE_URL", c.UserContext.UserServiceURL, c.UserContext.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.WeatherContext.ExternalIntegrationsServiceURL, c.WeatherContext.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.PlaceEnrichment.ExternalIntegrationsServiceURL, c.PlaceEnrichment.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.PriceEnrichment.ExternalIntegrationsServiceURL, c.PriceEnrichment.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.CalendarSync.ExternalIntegrationsServiceURL, c.CalendarSync.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.BudgetConversion.ExternalIntegrationsServiceURL, c.BudgetConversion.Enabled, false},
		{"NOTIFICATION_SERVICE_URL", c.Notifications.NotificationServiceURL, c.Notifications.Enabled, false},
		{"AUTH_SERVICE_URL", c.UserLookup.AuthServiceURL, true, false},
		{"PUBLIC_WEB_BASE_URL", c.PublicSharing.PublicWebBaseURL, c.PublicSharing.Enabled, c.IsProduction()},
	}
	for _, check := range checks {
		if !check.enabled {
			continue
		}
		if err := validateHTTPURL(check.name, check.value, check.requireHTTPS); err != nil {
			if c.IsStrictEnv() || check.name == "PUBLIC_WEB_BASE_URL" {
				return err
			}
		}
	}
	publicWeb := strings.TrimRight(strings.TrimSpace(c.PublicSharing.PublicWebBaseURL), "/")
	if c.IsProduction() && c.PublicSharing.Enabled && isLocalhostURL(publicWeb) {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL must not use localhost in production")
	}
	c.PublicSharing.PublicWebBaseURL = publicWeb
	return nil
}

func (c *Config) validatePublicSharing() error {
	if !c.PublicSharing.Enabled {
		return nil
	}
	secret := strings.TrimSpace(c.PublicSharing.PublicShareAccessSecret)
	if secret == "" {
		return fmt.Errorf("PUBLIC_SHARE_ACCESS_SECRET is required when public sharing is enabled")
	}
	if c.IsStrictEnv() {
		if isUnsafeSecret(secret, DefaultDevelopmentPublicShareSecret) {
			return fmt.Errorf("PUBLIC_SHARE_ACCESS_SECRET must not use a development default in %s", c.Env)
		}
		if len(secret) < MinProductionTokenLength {
			return fmt.Errorf("PUBLIC_SHARE_ACCESS_SECRET must be at least %d characters in %s", MinProductionTokenLength, c.Env)
		}
	}
	c.PublicSharing.PublicShareAccessSecret = secret
	return nil
}

func (c *Config) validateGenerationJobs() error {
	if !c.GenerationJobs.Enabled || c.GenerationJobs.DispatchMode != "queue" {
		return nil
	}
	return validateRabbitMQURL("RABBITMQ_URL", c.GenerationJobs.RabbitMQURL, c.IsStrictEnv())
}

func (c *Config) validateInternalTokens() error {
	tokens := []struct {
		name    string
		value   string
		enabled bool
	}{
		{"INTERNAL_SERVICE_TOKEN", c.PriceEnrichment.InternalServiceToken, c.PriceEnrichment.Enabled},
		{"INTERNAL_SERVICE_TOKEN", c.CalendarSync.InternalServiceToken, c.CalendarSync.Enabled},
		{"INTERNAL_SERVICE_TOKEN", c.BudgetConversion.InternalServiceToken, c.BudgetConversion.Enabled},
		{"NOTIFICATION_SERVICE_TOKEN", c.Notifications.NotificationServiceToken, c.Notifications.Enabled},
	}
	for _, token := range tokens {
		if !token.enabled {
			continue
		}
		if err := validateTokenValue(token.name, token.value, c.Env, c.IsStrictEnv()); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateOps() error {
	if !c.Ops.DashboardEnabled {
		return nil
	}
	if c.IsStrictEnv() && strings.TrimSpace(c.Ops.AdminEmails) == "" {
		return fmt.Errorf("OPS_ADMIN_EMAILS is required when OPS_DASHBOARD_ENABLED=true in %s", c.Env)
	}
	return validateTokenValue("OPS_INTERNAL_SERVICE_TOKEN", c.Ops.InternalServiceToken, c.Env, c.IsStrictEnv())
}

func validateTokenValue(name, value, env string, strict bool) error {
	token := strings.TrimSpace(value)
	if token == "" {
		return fmt.Errorf("%s is required", name)
	}
	if strict {
		if isUnsafeSecret(token, DefaultDevelopmentInternalToken) {
			return fmt.Errorf("%s must not use a development default in %s", name, env)
		}
		if len(token) < MinProductionTokenLength {
			return fmt.Errorf("%s must be at least %d characters in %s", name, MinProductionTokenLength, env)
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

func validateRabbitMQURL(name, value string, strict bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", name)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid amqp/amqps URL", name)
	}
	if parsed.Scheme != "amqp" && parsed.Scheme != "amqps" {
		return fmt.Errorf("%s must use amqp or amqps", name)
	}
	if strict && parsed.User != nil {
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		if strings.EqualFold(username, "guest") || isUnsafeSecret(password, "guest") {
			return fmt.Errorf("%s must not use guest credentials in staging or production", name)
		}
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
