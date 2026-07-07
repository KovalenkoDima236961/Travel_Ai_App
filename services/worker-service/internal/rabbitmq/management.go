package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
	rabbitmqinfra "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/pkg/rabbitmq"
)

type ManagementConfig struct {
	URL      string
	User     string
	Password string
	AMQPURL  string
	Queue    jobqueue.Config
}

type ManagementClient struct {
	queue  jobqueue.Config
	client *rabbitmqinfra.ManagementClient
}

type QueueStatus = rabbitmqinfra.QueueStatus

type DLQMessage struct {
	MessageID      string         `json:"messageId"`
	JobID          string         `json:"jobId,omitempty"`
	TripID         string         `json:"tripId,omitempty"`
	JobType        string         `json:"jobType,omitempty"`
	Attempts       int            `json:"attempts"`
	Reason         string         `json:"reason,omitempty"`
	CorrelationID  string         `json:"correlationId,omitempty"`
	CreatedAt      *time.Time     `json:"createdAt,omitempty"`
	DeadLetteredAt *time.Time     `json:"deadLetteredAt,omitempty"`
	PayloadPreview map[string]any `json:"payloadPreview,omitempty"`
}

type rabbitGetMessage = rabbitmqinfra.Message

func NewManagementClient(cfg ManagementConfig) (*ManagementClient, error) {
	client, err := rabbitmqinfra.NewManagementClient(rabbitmqinfra.ManagementConfig{
		URL:      cfg.URL,
		User:     cfg.User,
		Password: cfg.Password,
		AMQPURL:  cfg.AMQPURL,
	})
	if err != nil {
		return nil, err
	}
	return &ManagementClient{
		queue:  cfg.Queue,
		client: client,
	}, nil
}

func (c *ManagementClient) QueueStatuses(ctx context.Context) ([]QueueStatus, error) {
	names := []string{
		c.queue.QueueName,
		c.queue.RetryQueueName,
		c.queue.DeadLetterQueueName,
	}
	return c.client.QueueStatuses(ctx, names)
}

func (c *ManagementClient) ListDLQMessages(ctx context.Context, limit int) ([]DLQMessage, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	raw, err := c.getMessages(ctx, c.queue.DeadLetterQueueName, limit, "ack_requeue_true")
	if err != nil {
		return nil, err
	}
	out := make([]DLQMessage, 0, len(raw))
	for _, msg := range raw {
		out = append(out, sanitizeDLQMessage(msg))
	}
	return out, nil
}

func (c *ManagementClient) RequeueDLQMessage(ctx context.Context, messageID string, reason string) error {
	return c.moveDLQMessage(ctx, messageID, "requeue", reason)
}

func (c *ManagementClient) DiscardDLQMessage(ctx context.Context, messageID string, reason string) error {
	return c.moveDLQMessage(ctx, messageID, "discard", reason)
}

func (c *ManagementClient) moveDLQMessage(ctx context.Context, messageID, action, reason string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return fmt.Errorf("messageId is required")
	}
	messages, err := c.getMessages(ctx, c.queue.DeadLetterQueueName, 100, "ack_requeue_false")
	if err != nil {
		return err
	}
	found := false
	for _, msg := range messages {
		sanitized := sanitizeDLQMessage(msg)
		if sanitized.MessageID == messageID {
			found = true
			if action == "requeue" {
				if err := c.publishRaw(ctx, c.queue.Exchange, c.queue.RoutingKey, msg, reason); err != nil {
					return err
				}
			}
			continue
		}
		if err := c.publishRaw(ctx, c.queue.DLX, c.queue.DeadLetterRoutingKey, msg, "preserve_unmatched"); err != nil {
			return err
		}
	}
	if !found {
		return ErrMessageNotFound
	}
	return nil
}

func (c *ManagementClient) getMessages(ctx context.Context, queue string, count int, ackmode string) ([]rabbitGetMessage, error) {
	return c.client.GetMessages(ctx, queue, count, ackmode)
}

func (c *ManagementClient) publishRaw(ctx context.Context, exchange, routingKey string, msg rabbitGetMessage, opsReason string) error {
	props := cloneMap(msg.Properties)
	headers := cloneMap(anyMap(props["headers"]))
	headers["x-requeued-by-ops"] = true
	if strings.TrimSpace(opsReason) != "" {
		headers["x-ops-reason"] = truncate(opsReason, 200)
	}
	props["headers"] = headers
	msg.Properties = props
	return c.client.PublishRaw(ctx, exchange, routingKey, msg)
}

func sanitizeDLQMessage(msg rabbitGetMessage) DLQMessage {
	var payload generationjobs.QueueMessage
	_ = json.Unmarshal([]byte(msg.Payload), &payload)
	props := msg.Properties
	messageID := stringValue(props["message_id"])
	if messageID == "" {
		messageID = payload.MessageID.String()
	}
	headers := anyMap(props["headers"])
	attempts := intValue(headers[generationjobs.HeaderAttempts])
	reason := firstNonEmpty(stringValue(headers["x-death-reason"]), stringValue(headers["x-first-death-reason"]))
	createdAt := payload.CreatedAt
	var created *time.Time
	if !createdAt.IsZero() {
		created = &createdAt
	}
	return DLQMessage{
		MessageID:     messageID,
		JobID:         payload.JobID.String(),
		TripID:        payload.TripID.String(),
		JobType:       string(payload.JobType),
		Attempts:      attempts,
		Reason:        reason,
		CorrelationID: payload.CorrelationID,
		CreatedAt:     created,
		PayloadPreview: map[string]any{
			"jobId":   payload.JobID.String(),
			"jobType": string(payload.JobType),
		},
	}
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func anyMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if out, ok := v.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
