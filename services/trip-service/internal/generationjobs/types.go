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
	ErrorUnknown             = "unknown_error"
)

const maxInstructionLength = 2000
const defaultLimit = 20
const maxLimit = 100
