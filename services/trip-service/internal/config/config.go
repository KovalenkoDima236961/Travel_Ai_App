package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/validation"
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
	Workspaces         WorkspacesConfig         `yaml:"workspaces"`
	PublicSharing      PublicSharingConfig      `yaml:"public_sharing"`
	Notifications      NotificationsConfig      `yaml:"notifications"`
	Presence           PresenceConfig           `yaml:"presence"`
	ActivityStream     ActivityStreamConfig     `yaml:"activity_stream"`
	EditLocks          EditLocksConfig          `yaml:"edit_locks"`
	GenerationJobs     GenerationJobsConfig     `yaml:"generation_jobs"`
	CalendarSync       CalendarSyncConfig       `yaml:"calendar_sync"`
	BudgetConversion   BudgetConversionConfig   `yaml:"budget_conversion"`
	TransportSearch    TransportSearchConfig    `yaml:"transport_search"`
	Receipts           ReceiptsConfig           `yaml:"receipts"`
	Ops                OpsConfig                `yaml:"ops"`
	TripDiscovery      TripDiscoveryConfig      `yaml:"trip_discovery"`
	TripHealth         TripHealthConfig         `yaml:"trip_health"`
	BudgetConfidence   BudgetConfidenceConfig   `yaml:"budget_confidence"`
	Verification       VerificationConfig       `yaml:"verification"`
	SummaryCache       SummaryCacheConfig       `yaml:"summary_cache"`
	Search             SearchConfig             `yaml:"search"`
	AIValidation       AIValidationConfig       `yaml:"ai_validation"`
	AIObservability    AIObservabilityConfig    `yaml:"ai_observability"`
	Copilot            CopilotConfig            `yaml:"copilot"`
	TripRecap          TripRecapConfig          `yaml:"trip_recap"`
	TripLibrary        TripLibraryConfig        `yaml:"trip_library"`
}

// TripLibraryConfig controls private archive and historical-library behavior.
// Archiving is always user initiated; this setting never enables auto-archive.
type TripLibraryConfig struct {
	Enabled                         bool `yaml:"enabled" env:"TRIP_LIBRARY_ENABLED" env-default:"true"`
	ReadyHealthScoreThreshold       int  `yaml:"ready_health_score_threshold" env:"TRIP_READY_HEALTH_SCORE_THRESHOLD" env-default:"80" validate:"min=1,max=100"`
	ReadyVerificationScoreThreshold int  `yaml:"ready_verification_score_threshold" env:"TRIP_READY_VERIFICATION_SCORE_THRESHOLD" env-default:"75" validate:"min=1,max=100"`
}

// TripRecapConfig controls private, post-trip recap generation. The feature
// is deliberately fail-open only to a deterministic local recap, never to an
// unvalidated AI response.
type TripRecapConfig struct {
	Enabled                   bool `yaml:"enabled" env:"TRIP_RECAP_ENABLED" env-default:"true"`
	AIEnabled                 bool `yaml:"ai_enabled" env:"TRIP_RECAP_AI_ENABLED" env-default:"true"`
	FailOpenWithDeterministic bool `yaml:"fail_open_with_deterministic" env:"TRIP_RECAP_FAIL_OPEN_WITH_DETERMINISTIC" env-default:"true"`
	TimeoutSeconds            int  `yaml:"timeout_seconds" env:"TRIP_RECAP_TIMEOUT_SECONDS" env-default:"30" validate:"min=1,max=120"`
	MaxSourceChars            int  `yaml:"max_source_chars" env:"TRIP_RECAP_MAX_SOURCE_CHARS" env-default:"16000" validate:"min=1000,max=50000"`
}

// CopilotConfig controls the private, advisory trip copilot. It deliberately
// has no mutation/tool settings: v1 only returns validated navigation actions.
type CopilotConfig struct {
	Enabled              bool   `yaml:"enabled" env:"TRIP_COPILOT_ENABLED" env-default:"true"`
	Mode                 string `yaml:"mode" env:"TRIP_COPILOT_MODE" env-default:"mock" validate:"oneof=mock ai"`
	FailOpen             bool   `yaml:"fail_open" env:"TRIP_COPILOT_FAIL_OPEN" env-default:"false"`
	MaxMessageChars      int    `yaml:"max_message_chars" env:"TRIP_COPILOT_MAX_MESSAGE_CHARS" env-default:"2000" validate:"min=1,max=10000"`
	MaxContextChars      int    `yaml:"max_context_chars" env:"TRIP_COPILOT_MAX_CONTEXT_CHARS" env-default:"12000" validate:"min=1000,max=50000"`
	TimeoutSeconds       int    `yaml:"timeout_seconds" env:"TRIP_COPILOT_TIMEOUT_SECONDS" env-default:"20" validate:"min=1,max=120"`
	StoreHistory         bool   `yaml:"store_history" env:"TRIP_COPILOT_STORE_HISTORY" env-default:"false"`
	HistoryRetentionDays int    `yaml:"history_retention_days" env:"TRIP_COPILOT_HISTORY_RETENTION_DAYS" env-default:"7" validate:"min=1,max=365"`
	PublicShareEnabled   bool   `yaml:"public_share_enabled" env:"TRIP_COPILOT_PUBLIC_SHARE_ENABLED" env-default:"false"`
	RateLimitPerMinute   int    `yaml:"rate_limit_per_minute" env:"TRIP_COPILOT_RATE_LIMIT_PER_MINUTE" env-default:"20" validate:"min=1,max=1000"`
}

type SearchConfig struct {
	Enabled             bool `yaml:"enabled" env:"SEARCH_ENABLED" env-default:"true"`
	DefaultLimit        int  `yaml:"default_limit" env:"SEARCH_DEFAULT_LIMIT" env-default:"20" validate:"min=1,max=50"`
	MaxLimit            int  `yaml:"max_limit" env:"SEARCH_MAX_LIMIT" env-default:"50" validate:"min=1,max=50"`
	PerCategoryLimit    int  `yaml:"per_category_limit" env:"SEARCH_PER_CATEGORY_LIMIT" env-default:"5" validate:"min=1,max=20"`
	MinQueryLength      int  `yaml:"min_query_length" env:"SEARCH_MIN_QUERY_LENGTH" env-default:"2" validate:"min=1,max=20"`
	QueryTimeoutSeconds int  `yaml:"query_timeout_seconds" env:"SEARCH_QUERY_TIMEOUT_SECONDS" env-default:"3" validate:"min=1,max=30"`
}

// SummaryCacheConfig controls the small, process-local cache used by private,
// deterministic trip summaries. Cache keys include the viewer and trip
// revision/update timestamp so responses are never shared across users.
type SummaryCacheConfig struct {
	Enabled                bool `yaml:"enabled" env:"SUMMARY_CACHE_ENABLED" env-default:"true"`
	TTLSeconds             int  `yaml:"ttl_seconds" env:"SUMMARY_CACHE_TTL_SECONDS" env-default:"30" validate:"min=1,max=300"`
	MaxItems               int  `yaml:"max_items" env:"SUMMARY_CACHE_MAX_ITEMS" env-default:"1000" validate:"min=1,max=10000"`
	EndpointTimeoutSeconds int  `yaml:"endpoint_timeout_seconds" env:"SUMMARY_ENDPOINT_TIMEOUT_SECONDS" env-default:"8" validate:"min=1,max=30"`
}

// AIObservabilityConfig keeps persisted generation traces privacy-safe. Raw
// prompts are never a supported storage mode; optional snapshots are redacted.
type AIObservabilityConfig struct {
	Enabled                   bool `yaml:"enabled" env:"AI_OBSERVABILITY_ENABLED" env-default:"true"`
	TraceEventsEnabled        bool `yaml:"trace_events_enabled" env:"AI_OBSERVABILITY_TRACE_EVENTS_ENABLED" env-default:"true"`
	StoreRedactedPrompts      bool `yaml:"store_redacted_prompts" env:"AI_OBSERVABILITY_STORE_REDACTED_PROMPTS" env-default:"false"`
	StoreRedactedResponses    bool `yaml:"store_redacted_responses" env:"AI_OBSERVABILITY_STORE_REDACTED_RESPONSES" env-default:"false"`
	MaxPromptSnapshotChars    int  `yaml:"max_prompt_snapshot_chars" env:"AI_OBSERVABILITY_MAX_PROMPT_SNAPSHOT_CHARS" env-default:"12000" validate:"min=1,max=50000"`
	RetentionDays             int  `yaml:"retention_days" env:"AI_OBSERVABILITY_RETENTION_DAYS" env-default:"30" validate:"min=1,max=365"`
	FailOpen                  bool `yaml:"fail_open" env:"AI_OBSERVABILITY_FAIL_OPEN" env-default:"true"`
	DebugLocalOnly            bool `yaml:"debug_local_only" env:"AI_OBSERVABILITY_DEBUG_LOCAL_ONLY" env-default:"true"`
	PromptLoggingEnabled      bool `yaml:"prompt_logging_enabled" env:"AI_PROMPT_LOGGING_ENABLED" env-default:"false"`
	PromptLoggingRedactedOnly bool `yaml:"prompt_logging_redacted_only" env:"AI_PROMPT_LOGGING_REDACTED_ONLY" env-default:"true"`
	RedactionEnabled          bool `yaml:"redaction_enabled" env:"AI_OBSERVABILITY_REDACTION_ENABLED" env-default:"true"`
}

type AIValidationConfig struct {
	Enabled                    bool `yaml:"enabled" env:"AI_VALIDATION_ENABLED" env-default:"true"`
	RepairEnabled              bool `yaml:"repair_enabled" env:"AI_VALIDATION_REPAIR_ENABLED" env-default:"true"`
	MaxRepairAttempts          int  `yaml:"max_repair_attempts" env:"AI_VALIDATION_MAX_REPAIR_ATTEMPTS" env-default:"2" validate:"min=0,max=5"`
	BlockOnSchemaErrors        bool `yaml:"block_on_schema_errors" env:"AI_VALIDATION_BLOCK_ON_SCHEMA_ERRORS" env-default:"true"`
	BlockOnPolicyBlockers      bool `yaml:"block_on_policy_blockers" env:"AI_VALIDATION_BLOCK_ON_POLICY_BLOCKERS" env-default:"true"`
	BlockOnCriticalRouteErrors bool `yaml:"block_on_critical_route_errors" env:"AI_VALIDATION_BLOCK_ON_CRITICAL_ROUTE_ERRORS" env-default:"true"`
	BlockOnBudgetErrors        bool `yaml:"block_on_budget_errors" env:"AI_VALIDATION_BLOCK_ON_BUDGET_ERRORS" env-default:"true"`
	FailOpen                   bool `yaml:"fail_open" env:"AI_VALIDATION_FAIL_OPEN" env-default:"false"`
}

type TripHealthConfig struct {
	Enabled                         bool    `yaml:"enabled" env:"TRIP_HEALTH_ENABLED" env-default:"true"`
	CacheTTLSeconds                 int     `yaml:"cache_ttl_seconds" env:"TRIP_HEALTH_CACHE_TTL_SECONDS" env-default:"60" validate:"min=0,max=3600"`
	IncludeDebug                    bool    `yaml:"include_debug" env:"TRIP_HEALTH_INCLUDE_DEBUG" env-default:"false"`
	LargeExpenseReceiptThreshold    float64 `yaml:"large_expense_receipt_threshold" env:"TRIP_HEALTH_LARGE_EXPENSE_RECEIPT_THRESHOLD" env-default:"100" validate:"min=0"`
	DefaultMaxWalkingKmPerDay       float64 `yaml:"default_max_walking_km_per_day" env:"TRIP_HEALTH_DEFAULT_MAX_WALKING_KM_PER_DAY" env-default:"12" validate:"min=1,max=100"`
	DefaultMaxTransferMinutesPerDay int     `yaml:"default_max_transfer_minutes_per_day" env:"TRIP_HEALTH_DEFAULT_MAX_TRANSFER_MINUTES_PER_DAY" env-default:"480" validate:"min=30,max=2880"`
}

type BudgetConfidenceConfig struct {
	Enabled                         bool    `yaml:"enabled" env:"BUDGET_CONFIDENCE_ENABLED" env-default:"true"`
	CacheTTLSeconds                 int     `yaml:"cache_ttl_seconds" env:"BUDGET_CONFIDENCE_CACHE_TTL_SECONDS" env-default:"60" validate:"min=0,max=3600"`
	FailOpen                        bool    `yaml:"fail_open" env:"BUDGET_CONFIDENCE_FAIL_OPEN" env-default:"true"`
	LargeExpenseReceiptThreshold    float64 `yaml:"large_expense_receipt_threshold" env:"BUDGET_CONFIDENCE_LARGE_EXPENSE_RECEIPT_THRESHOLD" env-default:"100" validate:"min=0"`
	ActualSpendHighThresholdPercent float64 `yaml:"actual_spend_high_threshold_percent" env:"BUDGET_CONFIDENCE_ACTUAL_SPEND_HIGH_THRESHOLD_PERCENT" env-default:"80" validate:"min=1,max=1000"`
	PlannedActualGapWarningPercent  float64 `yaml:"planned_actual_gap_warning_percent" env:"BUDGET_CONFIDENCE_PLANNED_ACTUAL_GAP_WARNING_PERCENT" env-default:"20" validate:"min=1,max=1000"`
	PlannedActualGapHighPercent     float64 `yaml:"planned_actual_gap_high_percent" env:"BUDGET_CONFIDENCE_PLANNED_ACTUAL_GAP_HIGH_PERCENT" env-default:"40" validate:"min=1,max=1000"`
}

// VerificationConfig controls the advisory real-world data evaluator. It
// never enables background provider polling; provider calls remain explicit
// user actions and reuse the configured integration clients.
type VerificationConfig struct {
	Enabled                   bool    `yaml:"enabled" env:"VERIFICATION_ENABLED" env-default:"true"`
	CacheEnabled              bool    `yaml:"cache_enabled" env:"VERIFICATION_CACHE_ENABLED" env-default:"true"`
	CacheTTLSeconds           int     `yaml:"cache_ttl_seconds" env:"VERIFICATION_CACHE_TTL_SECONDS" env-default:"60" validate:"min=1,max=300"`
	WeatherStaleHoursNearTrip int     `yaml:"weather_stale_hours_near_trip" env:"VERIFICATION_WEATHER_STALE_HOURS_NEAR_TRIP" env-default:"12" validate:"min=1,max=168"`
	WeatherStaleHoursFarTrip  int     `yaml:"weather_stale_hours_far_trip" env:"VERIFICATION_WEATHER_STALE_HOURS_FAR_TRIP" env-default:"24" validate:"min=1,max=336"`
	TransportStaleDays        int     `yaml:"transport_stale_days" env:"VERIFICATION_TRANSPORT_STALE_DAYS" env-default:"7" validate:"min=1,max=90"`
	AvailabilityStaleHours    int     `yaml:"availability_stale_hours" env:"VERIFICATION_AVAILABILITY_STALE_HOURS" env-default:"48" validate:"min=1,max=336"`
	PriceStaleDays            int     `yaml:"price_stale_days" env:"VERIFICATION_PRICE_STALE_DAYS" env-default:"7" validate:"min=1,max=90"`
	PlaceStaleDays            int     `yaml:"place_stale_days" env:"VERIFICATION_PLACE_STALE_DAYS" env-default:"30" validate:"min=1,max=365"`
	RouteEstimateStaleDays    int     `yaml:"route_estimate_stale_days" env:"VERIFICATION_ROUTE_ESTIMATE_STALE_DAYS" env-default:"14" validate:"min=1,max=180"`
	CalendarSyncStaleDays     int     `yaml:"calendar_sync_stale_days" env:"VERIFICATION_CALENDAR_SYNC_STALE_DAYS" env-default:"7" validate:"min=1,max=90"`
	NearTripDays              int     `yaml:"near_trip_days" env:"VERIFICATION_NEAR_TRIP_DAYS" env-default:"7" validate:"min=0,max=90"`
	MaxDetails                int     `yaml:"max_details" env:"VERIFICATION_MAX_DETAILS" env-default:"100" validate:"min=1,max=500"`
	PlaceMinConfidence        float64 `yaml:"place_min_confidence" env:"VERIFICATION_PLACE_MIN_CONFIDENCE" env-default:"0.75" validate:"min=0,max=1"`
}

type ReceiptsConfig struct {
	StorageProvider          string `yaml:"storage_provider" env:"RECEIPT_STORAGE_PROVIDER" env-default:"local" validate:"oneof=local"`
	LocalDir                 string `yaml:"local_dir" env:"RECEIPT_STORAGE_LOCAL_DIR" env-default:"./data/receipts"`
	MaxFileSizeMB            int    `yaml:"max_file_size_mb" env:"RECEIPT_MAX_FILE_SIZE_MB" env-default:"10" validate:"min=1,max=50"`
	AllowedMIMETypes         string `yaml:"allowed_mime_types" env:"RECEIPT_ALLOWED_MIME_TYPES" env-default:"image/jpeg,image/png,image/webp,application/pdf"`
	UploadMaxBytes           int64  `yaml:"upload_max_bytes" env:"RECEIPT_UPLOAD_MAX_BYTES" env-default:"10485760" validate:"min=1,max=52428800"`
	UploadAllowedMIME        string `yaml:"upload_allowed_mime" env:"RECEIPT_UPLOAD_ALLOWED_MIME"`
	UploadAllowedExt         string `yaml:"upload_allowed_ext" env:"RECEIPT_UPLOAD_ALLOWED_EXT" env-default:".jpg,.jpeg,.png,.webp,.pdf"`
	FileScanningEnabled      bool   `yaml:"file_scanning_enabled" env:"FILE_SCANNING_ENABLED" env-default:"false"`
	FileScanningFailOpen     bool   `yaml:"file_scanning_fail_open" env:"FILE_SCANNING_FAIL_OPEN" env-default:"false"`
	UploadRateLimitPerMinute int    `yaml:"upload_rate_limit_per_minute" env:"RECEIPT_UPLOAD_RATE_LIMIT_PER_MINUTE" env-default:"20" validate:"min=1,max=10000"`
	OCREnabled               bool   `yaml:"ocr_enabled" env:"RECEIPT_OCR_ENABLED" env-default:"true"`
	OCRProvider              string `yaml:"ocr_provider" env:"RECEIPT_OCR_PROVIDER" env-default:"mock" validate:"oneof=mock local"`
	OCRTimeoutSeconds        int    `yaml:"ocr_timeout_seconds" env:"RECEIPT_OCR_TIMEOUT_SECONDS" env-default:"30" validate:"min=1,max=300"`
	OCRFailOpen              bool   `yaml:"ocr_fail_open" env:"RECEIPT_OCR_FAIL_OPEN" env-default:"true"`
	OCRStoreRawText          bool   `yaml:"ocr_store_raw_text" env:"RECEIPT_OCR_STORE_RAW_TEXT" env-default:"true"`
}

type TripDiscoveryConfig struct {
	Enabled                bool `yaml:"enabled" env:"TRIP_DISCOVERY_ENABLED" env-default:"true"`
	AITimeoutSeconds       int  `yaml:"ai_timeout_seconds" env:"TRIP_DISCOVERY_AI_TIMEOUT_SECONDS" env-default:"120" validate:"min=1"`
	MaxPreviousTrips       int  `yaml:"max_previous_trips" env:"TRIP_DISCOVERY_MAX_PREVIOUS_TRIPS" env-default:"15" validate:"min=1,max=20"`
	DefaultSuggestionCount int  `yaml:"default_suggestion_count" env:"TRIP_DISCOVERY_DEFAULT_SUGGESTION_COUNT" env-default:"5" validate:"min=3,max=5"`
}

// WorkspacesConfig controls service-to-service checks against User Service for
// workspace membership and role resolution.
type WorkspacesConfig struct {
	Enabled        bool   `yaml:"enabled" env:"WORKSPACES_ENABLED" env-default:"true"`
	UserServiceURL string `yaml:"user_service_url" env:"USER_SERVICE_URL" env-default:"http://user-service:8083"`
	ServiceToken   string `yaml:"service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"WORKSPACE_ACCESS_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
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
	FreeBusyImportEnabled          bool   `yaml:"free_busy_import_enabled" env:"CALENDAR_FREE_BUSY_IMPORT_ENABLED" env-default:"true"`
	FreeBusyImportFailOpen         bool   `yaml:"free_busy_import_fail_open" env:"CALENDAR_FREE_BUSY_IMPORT_FAIL_OPEN" env-default:"false"`
	FreeBusyImportTimeoutSeconds   int    `yaml:"free_busy_import_timeout_seconds" env:"CALENDAR_FREE_BUSY_IMPORT_TIMEOUT_SECONDS" env-default:"12" validate:"min=1"`
}

type BudgetConversionConfig struct {
	Enabled                        bool   `yaml:"enabled" env:"BUDGET_CONVERSION_ENABLED" env-default:"true"`
	FailOpen                       bool   `yaml:"fail_open" env:"BUDGET_CONVERSION_FAIL_OPEN" env-default:"true"`
	ExternalIntegrationsServiceURL string `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	InternalServiceToken           string `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds                 int    `yaml:"timeout_seconds" env:"EXCHANGE_RATE_CLIENT_TIMEOUT_SECONDS" env-default:"8" validate:"min=1"`
}

type TransportSearchConfig struct {
	Enabled                        bool   `yaml:"enabled" env:"TRANSPORT_SEARCH_ENABLED" env-default:"true"`
	FailOpen                       bool   `yaml:"fail_open" env:"TRANSPORT_SEARCH_FAIL_OPEN" env-default:"true"`
	ExternalIntegrationsServiceURL string `yaml:"external_integrations_service_url" env:"EXTERNAL_INTEGRATIONS_SERVICE_URL" env-default:"http://external-integrations-service:8084"`
	InternalServiceToken           string `yaml:"internal_service_token" env:"EXTERNAL_INTEGRATIONS_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds                 int    `yaml:"timeout_seconds" env:"TRANSPORT_SEARCH_TIMEOUT_SECONDS" env-default:"10" validate:"min=1"`
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
	Required              bool   `yaml:"required" env:"AUTH_REQUIRED" env-default:"true"`
	JWTAccessSecret       string `yaml:"jwt_access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development" validate:"required"`
	HeaderName            string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
	DevUserID             string `yaml:"dev_user_id" env:"DEV_USER_ID" env-default:"00000000-0000-0000-0000-000000000001" validate:"required,uuid"`
	InternalServiceToken  string `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	InternalServiceTokens string `yaml:"internal_service_tokens" env:"INTERNAL_SERVICE_TOKENS"`
}

func (c AuthConfig) ActiveInternalServiceTokens() string {
	if tokens := strings.TrimSpace(c.InternalServiceTokens); tokens != "" {
		return tokens
	}
	return c.InternalServiceToken
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
	AuthServiceURL       string `yaml:"auth_service_url" env:"AUTH_SERVICE_URL" env-default:"http://auth-service:8081"`
	InternalServiceToken string `yaml:"internal_service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds       int    `yaml:"timeout_seconds" env:"USER_LOOKUP_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
}

// PublicSharingConfig controls read-only public trip share links.
type PublicSharingConfig struct {
	Enabled                     bool   `yaml:"enabled" env:"PUBLIC_SHARING_ENABLED" env-default:"true"`
	PublicWebBaseURL            string `yaml:"public_web_base_url" env:"PUBLIC_WEB_BASE_URL" env-default:"http://localhost:3000"`
	ShareTokenBytes             int    `yaml:"share_token_bytes" env:"SHARE_TOKEN_BYTES" env-default:"32" validate:"min=32,max=128"`
	PublicShareAccessSecret     string `yaml:"public_share_access_secret" env:"PUBLIC_SHARE_ACCESS_SECRET" env-default:"dev-public-share-secret-change-me" validate:"required"`
	PublicShareAccessTTLMinutes int    `yaml:"public_share_access_ttl_minutes" env:"PUBLIC_SHARE_ACCESS_TTL_MINUTES" env-default:"60" validate:"min=1"`
	UnlockRateLimitPerMinute    int    `yaml:"unlock_rate_limit_per_minute" env:"SHARE_UNLOCK_RATE_LIMIT_PER_MINUTE" env-default:"5" validate:"min=1,max=1000"`
	AccessRateLimitPerMinute    int    `yaml:"access_rate_limit_per_minute" env:"PUBLIC_SHARE_ACCESS_RATE_LIMIT_PER_MINUTE" env-default:"120" validate:"min=1,max=10000"`
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
	// Keep the legacy receipt MIME setting working while allowing the more
	// explicit upload-specific setting to override it.
	if strings.TrimSpace(c.Receipts.UploadAllowedMIME) == "" {
		c.Receipts.UploadAllowedMIME = c.Receipts.AllowedMIMETypes
	}
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
	if err := c.validateAIObservability(); err != nil {
		return err
	}
	if c.IsProduction() && c.Receipts.FileScanningEnabled && c.Receipts.FileScanningFailOpen {
		return fmt.Errorf("FILE_SCANNING_FAIL_OPEN must be false when scanning is enabled in production")
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
		{"AI_PLANNING_SERVICE_URL", c.ItineraryGenerator.AIPlanningServiceURL, strings.TrimSpace(c.ItineraryGenerator.Mode) == "http" || (c.Copilot.Enabled && strings.TrimSpace(c.Copilot.Mode) == "ai"), false},
		{"USER_SERVICE_URL", c.UserContext.UserServiceURL, c.UserContext.Enabled, false},
		{"USER_SERVICE_URL", c.Workspaces.UserServiceURL, c.Workspaces.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.WeatherContext.ExternalIntegrationsServiceURL, c.WeatherContext.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.PlaceEnrichment.ExternalIntegrationsServiceURL, c.PlaceEnrichment.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.PriceEnrichment.ExternalIntegrationsServiceURL, c.PriceEnrichment.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.CalendarSync.ExternalIntegrationsServiceURL, c.CalendarSync.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.BudgetConversion.ExternalIntegrationsServiceURL, c.BudgetConversion.Enabled, false},
		{"EXTERNAL_INTEGRATIONS_SERVICE_URL", c.TransportSearch.ExternalIntegrationsServiceURL, c.TransportSearch.Enabled, false},
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
		{"EXTERNAL_INTEGRATIONS_SERVICE_TOKEN", c.TransportSearch.InternalServiceToken, c.TransportSearch.Enabled},
		{"NOTIFICATION_SERVICE_TOKEN", c.Notifications.NotificationServiceToken, c.Notifications.Enabled},
		{"INTERNAL_SERVICE_TOKEN", c.Workspaces.ServiceToken, c.Workspaces.Enabled},
		{"INTERNAL_SERVICE_TOKEN", c.Auth.InternalServiceToken, true},
	}
	for _, token := range tokens {
		if !token.enabled {
			continue
		}
		if err := validateTokenValue(token.name, token.value, c.Env, c.IsStrictEnv()); err != nil {
			return err
		}
	}
	if strings.TrimSpace(c.Auth.InternalServiceTokens) != "" {
		for _, token := range strings.Split(c.Auth.InternalServiceTokens, ",") {
			if err := validateTokenValue("INTERNAL_SERVICE_TOKENS", token, c.Env, c.IsStrictEnv()); err != nil {
				return err
			}
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

func (c *Config) validateAIObservability() error {
	cfg := c.AIObservability
	if c.IsProduction() && cfg.PromptLoggingEnabled && !cfg.PromptLoggingRedactedOnly {
		return fmt.Errorf("AI_PROMPT_LOGGING_ENABLED requires AI_PROMPT_LOGGING_REDACTED_ONLY=true in production")
	}
	if c.IsProduction() && cfg.StoreRedactedPrompts && !cfg.RedactionEnabled {
		return fmt.Errorf("AI_OBSERVABILITY_STORE_REDACTED_PROMPTS requires AI_OBSERVABILITY_REDACTION_ENABLED=true in production")
	}
	if c.IsProduction() && cfg.DebugLocalOnly && (cfg.StoreRedactedPrompts || cfg.StoreRedactedResponses) {
		return fmt.Errorf("AI observability prompt/response snapshots are local-debug only in production")
	}
	if c.IsStrictEnv() && cfg.RetentionDays > 90 {
		return fmt.Errorf("AI_OBSERVABILITY_RETENTION_DAYS must not exceed 90 in %s", c.Env)
	}
	return nil
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
