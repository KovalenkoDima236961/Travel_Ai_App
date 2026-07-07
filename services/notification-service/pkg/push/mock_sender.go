package push

import (
	"context"

	"go.uber.org/zap"
)

// MockSender records/logs accepted push attempts without contacting external
// push services. It is intended for unit tests and local smoke plumbing.
type MockSender struct {
	log *zap.Logger
}

// NewMockSender constructs a no-network push sender.
func NewMockSender(log *zap.Logger) *MockSender {
	if log == nil {
		log = zap.NewNop()
	}
	return &MockSender{log: log}
}

// Send reports success for one push notification.
func (s *MockSender) Send(_ context.Context, subscription PushSubscription, payload PushPayload) (*PushSendResult, error) {
	s.log.Info("push_send_success",
		zap.String("provider", "mock"),
		zap.String("userId", subscription.UserID.String()),
		zap.String("notificationType", payload.Type),
		zap.String("category", payload.Category),
		zap.String("subscriptionId", subscription.ID.String()),
		zap.String("endpointHash", EndpointHash(subscription.Endpoint)),
		zap.Int("statusCode", 202),
	)
	return &PushSendResult{StatusCode: 202}, nil
}
