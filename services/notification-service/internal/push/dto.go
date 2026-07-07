package push

import (
	"github.com/google/uuid"
)

const (
	MaxEndpointLength    = 2048
	MaxKeyLength         = 512
	MaxAuthLength        = 256
	MaxUserAgentLength   = 500
	MaxBrowserLength     = 100
	MaxDeviceLabelLength = 100
)

// SubscribeInput is one browser Push API subscription registered by a user.
type SubscribeInput struct {
	UserID      uuid.UUID
	Endpoint    string
	P256DH      string
	Auth        string
	UserAgent   *string
	Browser     *string
	DeviceLabel *string
}

// PublicKeyResult is returned to the browser before subscription.
type PublicKeyResult struct {
	Enabled   bool
	PublicKey *string
}

// StatusResult reports push state for the current user/device settings UI.
type StatusResult struct {
	Enabled             bool
	ActiveSubscriptions int
}

// BatchResult summarises push fan-out for an internal notification batch.
type BatchResult struct {
	Attempted                      int `json:"attempted"`
	Sent                           int `json:"sent"`
	Skipped                        int `json:"skipped"`
	SkippedByPreference            int `json:"skippedByPreference"`
	Failed                         int `json:"failed"`
	SubscriptionsDisabled          int `json:"subscriptionsDisabled"`
	SubscriptionsDisabledAsGone    int `json:"subscriptionsDisabledAsGone,omitempty"`
	SubscriptionsDisabledAsInvalid int `json:"subscriptionsDisabledAsInvalid,omitempty"`
}

// Config configures the notification-service push orchestration.
type Config struct {
	Enabled         bool
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	FailOpen        bool
}
