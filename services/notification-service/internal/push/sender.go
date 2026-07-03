package push

import "context"

// PushSender sends one payload to one browser push subscription.
type PushSender interface {
	Send(ctx context.Context, subscription PushSubscription, payload PushPayload) (*PushSendResult, error)
}
