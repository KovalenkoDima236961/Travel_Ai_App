package email

import (
	"context"

	"go.uber.org/zap"
)

// MockSender is the default, local-dev email sender. It never contacts an
// external mail server: it validates the message and logs safe metadata only
// (masked recipient, subject, provider). It never logs the body at info level
// or any secret.
type MockSender struct {
	log *zap.Logger
}

// NewMockSender constructs the mock sender.
func NewMockSender(log *zap.Logger) *MockSender {
	if log == nil {
		log = zap.NewNop()
	}
	return &MockSender{log: log}
}

// Send validates the message and logs that it "would" have been sent. The body
// is only emitted at debug level so info-level logs never carry email content.
func (s *MockSender) Send(_ context.Context, msg EmailMessage) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	s.log.Info("email send (mock)",
		zap.String("provider", ProviderMock),
		zap.String("to", MaskEmail(msg.ToEmail)),
		zap.String("subject", msg.Subject),
	)
	// Body text is potentially user-influenced; keep it out of info logs.
	s.log.Debug("email body (mock)",
		zap.String("to", MaskEmail(msg.ToEmail)),
		zap.String("textBody", msg.TextBody),
	)
	return nil
}
