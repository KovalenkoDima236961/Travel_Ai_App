package push

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"go.uber.org/zap"
)

// WebPushSender sends encrypted browser push notifications using VAPID.
type WebPushSender struct {
	cfg Config
	log *zap.Logger
}

// NewWebPushSender constructs a VAPID Web Push sender.
func NewWebPushSender(cfg Config, log *zap.Logger) (*WebPushSender, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if strings.TrimSpace(cfg.VAPIDPublicKey) == "" || strings.TrimSpace(cfg.VAPIDPrivateKey) == "" {
		return nil, fmt.Errorf("web push sender requires VAPID public and private keys")
	}
	if strings.TrimSpace(cfg.Subject) == "" {
		return nil, fmt.Errorf("web push sender requires subject")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.TTLSeconds < 0 {
		cfg.TTLSeconds = 0
	}
	cfg.Urgency = strings.ToLower(strings.TrimSpace(cfg.Urgency))
	if cfg.Urgency == "" {
		cfg.Urgency = "normal"
	}
	return &WebPushSender{cfg: cfg, log: log}, nil
}

// Send JSON-encodes the payload, sends it to the subscription endpoint, and
// classifies permanent failures so callers can disable expired subscriptions.
func (s *WebPushSender) Send(ctx context.Context, subscription PushSubscription, payload PushPayload) (*PushSendResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal push payload: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()

	resp, err := webpush.SendNotificationWithContext(
		sendCtx,
		body,
		&webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				P256dh: subscription.P256DH,
				Auth:   subscription.Auth,
			},
		},
		&webpush.Options{
			Subscriber:      s.cfg.Subject,
			VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
			VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
			TTL:             s.cfg.TTLSeconds,
			Urgency:         webpush.Urgency(s.cfg.Urgency),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("send web push: %w", err)
	}
	defer resp.Body.Close()

	result := &PushSendResult{
		StatusCode:       resp.StatusCode,
		SubscriptionGone: isPermanentSubscriptionFailure(resp.StatusCode),
	}
	if isSuccessStatus(resp.StatusCode) || result.SubscriptionGone {
		return result, nil
	}

	return result, fmt.Errorf("%w: status %d", ErrPushRejected, resp.StatusCode)
}

func isSuccessStatus(status int) bool {
	switch status {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return true
	default:
		return false
	}
}

func isPermanentSubscriptionFailure(status int) bool {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone:
		return true
	default:
		return false
	}
}
