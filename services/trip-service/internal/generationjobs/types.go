package generationjobs

const (
	ErrorItineraryConflict   = "itinerary_conflict"
	ErrorTripNotFound        = "trip_not_found"
	ErrorPermissionDenied    = "permission_denied"
	ErrorValidationFailed    = "validation_failed"
	ErrorAIGeneration        = "ai_generation_failed"
	ErrorNoOptimizationFound = "no_optimization_found"
	ErrorEnrichment          = "enrichment_failed"
	ErrorCancelled           = "cancelled"
	ErrorWorkerRestarted     = "worker_restarted"
	ErrorWorkerInterrupted   = "worker_interrupted"
	ErrorJobDispatchFailed   = "job_dispatch_failed"
	ErrorUnknown             = "unknown_error"

	// Provider rate-limit / quota error codes surfaced by External Integrations
	// Service. Rate limits and store-unavailable are transient (retryable);
	// an exhausted daily quota is terminal until the next day (Ops can retry).
	ErrorProviderRateLimited       = "provider_rate_limited"
	ErrorProviderQuotaExceeded     = "provider_quota_exceeded"
	ErrorProviderLimitsUnavailable = "provider_limits_unavailable"
)

const maxInstructionLength = 2000
const defaultLimit = 20
const maxLimit = 100
