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
)

// ManagementConfig configures a RabbitMQ Management API client.
type ManagementConfig struct {
	URL      string
	User     string
	Password string
	AMQPURL  string
	Client   *http.Client
}

// ManagementClient is a small RabbitMQ Management API client for queue status,
// message inspection, and raw publish operations.
type ManagementClient struct {
	base   string
	user   string
	pass   string
	vhost  string
	client *http.Client
}

// QueueStatus is the normalized status of one RabbitMQ queue.
type QueueStatus struct {
	Name            string   `json:"name"`
	MessagesReady   int      `json:"messagesReady"`
	MessagesUnacked int      `json:"messagesUnacked"`
	Consumers       int      `json:"consumers"`
	PublishRate     *float64 `json:"publishRate,omitempty"`
	DeliverRate     *float64 `json:"deliverRate,omitempty"`
}

// Message is a raw RabbitMQ Management API message payload.
type Message struct {
	Payload    string         `json:"payload"`
	Properties map[string]any `json:"properties"`
}

type queueResponse struct {
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

// NewManagementClient constructs a RabbitMQ Management API client.
func NewManagementClient(cfg ManagementConfig) (*ManagementClient, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if base == "" {
		return nil, fmt.Errorf("RABBITMQ_MANAGEMENT_URL is required")
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("RABBITMQ_MANAGEMENT_URL must be a valid http/https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("RABBITMQ_MANAGEMENT_URL must use http or https")
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &ManagementClient{
		base:   base,
		user:   cfg.User,
		pass:   cfg.Password,
		vhost:  vhostFromAMQPURL(cfg.AMQPURL),
		client: client,
	}, nil
}

// QueueStatuses loads status for all provided queue names.
func (c *ManagementClient) QueueStatuses(ctx context.Context, names []string) ([]QueueStatus, error) {
	out := make([]QueueStatus, 0, len(names))
	for _, name := range names {
		status, err := c.QueueStatus(ctx, name)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

// QueueStatus loads status for one queue.
func (c *ManagementClient) QueueStatus(ctx context.Context, name string) (QueueStatus, error) {
	var body queueResponse
	path := "/api/queues/" + url.PathEscape(c.vhost) + "/" + url.PathEscape(name)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &body); err != nil {
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

// GetMessages fetches messages from a queue using RabbitMQ Management API
// ackmode semantics.
func (c *ManagementClient) GetMessages(ctx context.Context, queue string, count int, ackmode string) ([]Message, error) {
	body := map[string]any{
		"count":    count,
		"ackmode":  ackmode,
		"encoding": "auto",
		"truncate": 5000,
	}
	var out []Message
	path := "/api/queues/" + url.PathEscape(c.vhost) + "/" + url.PathEscape(queue) + "/get"
	if err := c.doJSON(ctx, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PublishRaw republishes a raw message payload with the provided properties.
func (c *ManagementClient) PublishRaw(ctx context.Context, exchange, routingKey string, msg Message) error {
	body := map[string]any{
		"properties":       msg.Properties,
		"routing_key":      routingKey,
		"payload":          msg.Payload,
		"payload_encoding": "string",
	}
	var out struct {
		Routed bool `json:"routed"`
	}
	path := "/api/exchanges/" + url.PathEscape(c.vhost) + "/" + url.PathEscape(exchange) + "/publish"
	if err := c.doJSON(ctx, http.MethodPost, path, body, &out); err != nil {
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
	req.SetBasicAuth(c.user, c.pass)
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

func vhostFromAMQPURL(amqpURL string) string {
	vhost := "/"
	parsed, err := url.Parse(amqpURL)
	if err != nil {
		return vhost
	}
	if parsed.Scheme == "" && parsed.Host == "" {
		return vhost
	}
	path := strings.TrimSpace(parsed.Path)
	if path == "" || path == "/" {
		return vhost
	}
	unescaped, err := url.PathUnescape(strings.TrimPrefix(path, "/"))
	if err != nil {
		return strings.TrimPrefix(path, "/")
	}
	return unescaped
}
