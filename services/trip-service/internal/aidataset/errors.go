package aidataset

import "errors"

var (
	ErrConsentRequired     = errors.New("ai dataset consent required")
	ErrConsentRevoked      = errors.New("ai dataset consent revoked")
	ErrSanitizationFailed  = errors.New("ai dataset sanitization failed")
	ErrQualityTooLow       = errors.New("ai dataset quality too low")
	ErrDatasetDuplicate    = errors.New("ai dataset duplicate")
	ErrVersionExists       = errors.New("ai dataset version exists")
	ErrVersionNotReady     = errors.New("ai dataset version is not ready")
	ErrExportDisabled      = errors.New("ai dataset export disabled")
	ErrExportFailed        = errors.New("ai dataset export failed")
	ErrPrivateDataDetected = errors.New("ai dataset private data detected")
	ErrLicenseNotAllowed   = errors.New("ai dataset license not allowed")
	ErrInvalidReviewStatus = errors.New("invalid dataset review status")
	ErrNoEligibleExamples  = errors.New("no eligible dataset examples")
)
