package email

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// EmailSender delivers a single email. Implementations validate the message
// before sending. A nil error means the message was accepted for delivery (for
// the mock sender, "accepted" means logged and discarded).
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// NewSender selects an EmailSender from the provider name. An unsupported
// provider is a startup error so a misconfiguration fails fast rather than
// silently dropping mail.
func NewSender(cfg Config, log *zap.Logger) (EmailSender, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case ProviderMock, "":
		return NewMockSender(log), nil
	case ProviderSMTP:
		return NewSMTPSender(cfg.SMTP, log)
	default:
		return nil, fmt.Errorf("unsupported EMAIL_PROVIDER %q (want %q or %q)", cfg.Provider, ProviderMock, ProviderSMTP)
	}
}
