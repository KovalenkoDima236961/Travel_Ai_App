package knowledge

import "errors"

// Knowledge errors map onto the documented API error codes. Handlers translate
// these into responses; they never surface provider or database internals.
var (
	// ErrProviderUnavailable -> knowledge_provider_unavailable
	ErrProviderUnavailable = errors.New("knowledge provider unavailable")
	// ErrProviderRateLimited -> knowledge_provider_rate_limited
	ErrProviderRateLimited = errors.New("knowledge provider rate limited")
	// ErrLicenseMissing -> knowledge_license_missing
	ErrLicenseMissing = errors.New("knowledge source license or attribution missing")
	// ErrObservationInvalid -> knowledge_observation_invalid
	ErrObservationInvalid = errors.New("knowledge observation invalid")
	// ErrDuplicateGroupNotFound -> knowledge_duplicate_group_not_found
	ErrDuplicateGroupNotFound = errors.New("knowledge duplicate group not found")
	// ErrMergeConflict -> knowledge_merge_conflict
	ErrMergeConflict = errors.New("knowledge merge conflict")
	// ErrPlaceRejected -> knowledge_place_rejected
	ErrPlaceRejected = errors.New("knowledge place is rejected")
	// ErrQualityTooLow -> knowledge_quality_too_low
	ErrQualityTooLow = errors.New("knowledge quality below threshold")
)

// ErrorCode returns the stable API error code for a knowledge error.
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrProviderUnavailable):
		return "knowledge_provider_unavailable"
	case errors.Is(err, ErrProviderRateLimited):
		return "knowledge_provider_rate_limited"
	case errors.Is(err, ErrLicenseMissing):
		return "knowledge_license_missing"
	case errors.Is(err, ErrObservationInvalid):
		return "knowledge_observation_invalid"
	case errors.Is(err, ErrDuplicateGroupNotFound):
		return "knowledge_duplicate_group_not_found"
	case errors.Is(err, ErrMergeConflict):
		return "knowledge_merge_conflict"
	case errors.Is(err, ErrPlaceRejected):
		return "knowledge_place_rejected"
	case errors.Is(err, ErrQualityTooLow):
		return "knowledge_quality_too_low"
	default:
		return ""
	}
}
