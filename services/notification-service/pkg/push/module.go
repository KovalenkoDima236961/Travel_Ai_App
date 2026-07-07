package push

import (
	"strings"

	"go.uber.org/zap"
)

// NewSender selects the runtime sender. Disabled push uses MockSender only when
// tests instantiate it directly; production wiring skips push fan-out entirely
// when Config.Enabled is false.
func NewSender(cfg Config, log *zap.Logger) (PushSender, error) {
	if !cfg.Enabled {
		return NewMockSender(log), nil
	}
	if strings.TrimSpace(cfg.VAPIDPublicKey) == "" || strings.TrimSpace(cfg.VAPIDPrivateKey) == "" {
		return NewMockSender(log), nil
	}
	return NewWebPushSender(cfg, log)
}
