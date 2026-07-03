package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
)

type ManagementConfig struct {
	URL      string
	User     string
	Password string
	AMQPURL  string
	Queue    jobqueue.Config
}

type ManagementClient struct {
	cfg    ManagementConfig
	base   string
	vhost  string
	client *http.Client
}

type QueueStatus struct {
	Name            string   `json:"name"`
	MessagesReady   int      `json:"messagesReady"`
	MessagesUnacked int      `json:"messagesUnacked"`
	Consumers       int      `json:"consumers"`
	PublishRate     *float64 `json:"publishRate,omitempty"`
	DeliverRate     *float64 `json:"deliverRate,omitempty"`
}

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

type rabbitQueueResponse struct {
	Name            string `json:"name"`
	MessagesReady   int    `json:"messages_ready"`
	MessagesUnacked int    `json:"messages_unacknowledged"`
	Consumers       int    `json:"consumers"`
	MessageStats    struct {
		PublishDetails struct {
			Rate float64 `json:"rate"`
		} `json:"publish_details"`
		DeliverGetDetails struct {
			Rate float64 `json:"rate"`
		} `json:"deliver_get_details"`
	} `json:"message_stats"`
}

type rabbitGetMessage struct {
	Payload    string         `json:"payload"`
	Properties map[string]any `json:"properties"`
}

func NewManagementClient(cfg ManagementConfig) (*ManagementClient, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if base == "" {
		return nil, fmt.Errorf("RABBITMQ_MANAGEMENT_URL is required")
	}
	vhost := "/"
	if parsed, err := url.Parse(cfg.AMQPURL); err == nil {
		if strings.TrimSpace(parsed.Path) != "" && parsed.Path != "/" {
			vhost = strings.TrimPrefix(parsed.Path, "/")
		}
	}
	return &ManagementClient{
		cfg:    cfg,
		base:   base,
		vhost:  vhost,
		client: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (c *ManagementClient) QueueStatuses(ctx context.Context) ([]QueueStatus, error) {
	names := []string{
		c.cfg.Queue.QueueName,
		c.cfg.Queue.RetryQueueName,
		c.cfg.Queue.DeadLetterQueueName,
	}
	out := make([]QueueStatus, 0, len(names))
	for _, name := range names {
		status, err := c.queueStatus(ctx, name)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

func (c *ManagementClient) ListDLQMessages(ctx context.Context, limit int) ([]DLQMessage, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	raw, err := c.getMessages(ctx, c.cfg.Queue.DeadLetterQueueName, limit, "ack_requeue_true")
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

func (c *ManagementClient) queueStatus(ctx context.Context, name string) (QueueStatus, error) {
	var body rabbitQueueResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/queues/"+url.PathEscape(c.vhost)+"/"+url.PathEscape(name), nil, &body); err != nil {
		return QueueStatus{}, err
	}
	publishRate := body.MessageStats.PublishDetails.Rate
	deliverRate := body.MessageStats.DeliverGetDetails.Rate
	return QueueStatus{
		Name:            body.Name,
		MessagesReady:   body.MessagesReady,
		MessagesUnacked: body.MessagesUnacked,
		Consumers:       body.Consumers,
		PublishRate:     &publishRate,
		DeliverRate:     &deliverRate,
	}, nil
}

func (c *ManagementClient) moveDLQMessage(ctx context.Context, messageID, action, reason string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return fmt.Errorf("messageId is required")
	}
	messages, err := c.getMessages(ctx, c.cfg.Queue.DeadLetterQueueName, 100, "ack_requeue_false")
	if err != nil {
		return err
	}
	found := false
	for _, msg := range messages {
		sanitized := sanitizeDLQMessage(msg)
		if sanitized.MessageID == messageID {
			found = true
			if action == "requeue" {
				if err := c.publishRaw(ctx, c.cfg.Queue.Exchange, c.cfg.Queue.RoutingKey, msg, reason); err != nil {
					return err
				}
			}
			continue
		}
		if err := c.publishRaw(ctx, c.cfg.Queue.DLX, c.cfg.Queue.DeadLetterRoutingKey, msg, "preserve_unmatched"); err != nil {
			return err
		}
	}
	if !found {
		return ErrMessageNotFound
	}
	return nil
}

func (c *ManagementClient) getMessages(ctx context.Context, queue string, count int, ackmode string) ([]rabbitGetMessage, error) {
	body := map[string]any{
		"count":    count,
		"ackmode":  ackmode,
		"encoding": "auto",
		"truncate": 5000,
	}
	var out []rabbitGetMessage
	if err := c.doJSON(ctx, http.MethodPost, "/api/queues/"+url.PathEscape(c.vhost)+"/"+url.PathEscape(queue)+"/get", body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ManagementClient) publishRaw(ctx context.Context, exchange, routingKey string, msg rabbitGetMessage, opsReason string) error {
	props := cloneMap(msg.Properties)
	headers := cloneMap(anyMap(props["headers"]))
	headers["x-requeued-by-ops"] = true
	if strings.TrimSpace(opsReason) != "" {
		headers["x-ops-reason"] = truncate(opsReason, 200)
	}
	props["headers"] = headers
	body := map[string]any{
		"properties":       props,
		"routing_key":      routingKey,
		"payload":          msg.Payload,
		"payload_encoding": "string",
	}
	var out struct {
		Routed bool `json:"routed"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/exchanges/"+url.PathEscape(c.vhost)+"/"+url.PathEscape(exchange)+"/publish", body, &out); err != nil {
		return err
	}
	if !out.Routed {
		return fmt.Errorf("rabbitmq publish was not routed")
	}
	return nil
}

func (c *ManagementClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.User, c.cfg.Password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("rabbitmq management API returned HTTP %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
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
