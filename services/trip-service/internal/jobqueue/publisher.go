package jobqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
)

type RabbitMQPublisher struct {
	cfg     Config
	log     *zap.Logger
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex
}

func NewRabbitMQPublisher(ctx context.Context, cfg Config, log *zap.Logger) (*RabbitMQPublisher, error) {
	cfg = NormalizeConfig(cfg)
	if cfg.URL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if log == nil {
		log = zap.NewNop()
	}

	conn, err := dialWithRetry(ctx, cfg.URL, 10, 500*time.Millisecond)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := DeclareTopology(ch, cfg); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &RabbitMQPublisher{
		cfg:     cfg,
		log:     log,
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *RabbitMQPublisher) PublishGenerationJob(ctx context.Context, msg generationjobs.QueueMessage) error {
	return p.publish(ctx, p.cfg.Exchange, p.cfg.RoutingKey, msg, 1, generationjobs.SourceTripService)
}

func (p *RabbitMQPublisher) PublishRetry(ctx context.Context, msg generationjobs.QueueMessage, attempt int) error {
	return p.publish(ctx, p.cfg.Exchange, p.cfg.RetryRoutingKey, msg, attempt, "worker-service")
}

func (p *RabbitMQPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	if p.channel != nil {
		err = p.channel.Close()
	}
	if p.conn != nil {
		if closeErr := p.conn.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (p *RabbitMQPublisher) IsReady() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn != nil && !p.conn.IsClosed() && p.channel != nil && !p.channel.IsClosed()
}

func (p *RabbitMQPublisher) publish(
	ctx context.Context,
	exchange string,
	routingKey string,
	msg generationjobs.QueueMessage,
	attempt int,
	sourceService string,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel == nil || p.channel.IsClosed() {
		return fmt.Errorf("rabbitmq channel is closed")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal generation job message: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, p.cfg.PublishTimeout)
	defer cancel()

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	err = p.channel.PublishWithContext(
		publishCtx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  generationjobs.ContentTypeJSON,
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.MessageID.String(),
			Type:         generationjobs.MessageTypeTripGenerationJob,
			Timestamp:    time.Now().UTC(),
			Headers: amqp.Table{
				generationjobs.HeaderAttempts:      int32(attempt),
				generationjobs.HeaderSourceService: sourceService,
				generationjobs.HeaderMessageType:   generationjobs.MessageTypeTripGenerationJob,
			},
			Body: body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish generation job: %w", err)
	}

	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("rabbitmq negatively acknowledged publish")
		}
	case <-publishCtx.Done():
		return fmt.Errorf("wait for rabbitmq publish confirm: %w", publishCtx.Err())
	}

	p.log.Info("generation job message published",
		zap.String("job_id", msg.JobID.String()),
		zap.String("trip_id", msg.TripID.String()),
		zap.String("job_type", string(msg.JobType)),
		zap.String("message_id", msg.MessageID.String()),
		zap.String("routing_key", routingKey),
		zap.Int("attempt", attempt),
	)
	return nil
}

func DeclareTopology(ch *amqp.Channel, cfg Config) error {
	cfg = NormalizeConfig(cfg)

	if err := ch.ExchangeDeclare(cfg.Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare rabbitmq exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(cfg.DLX, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare rabbitmq dead-letter exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(cfg.DeadLetterQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare rabbitmq dead-letter queue: %w", err)
	}
	if err := ch.QueueBind(
		cfg.DeadLetterQueueName,
		cfg.DeadLetterRoutingKey,
		cfg.DLX,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind rabbitmq dead-letter queue: %w", err)
	}

	retryDelayMs := int32(cfg.RetryDelay / time.Millisecond)
	if retryDelayMs < 1 {
		retryDelayMs = int32((10 * time.Second) / time.Millisecond)
	}
	if _, err := ch.QueueDeclare(
		cfg.RetryQueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             retryDelayMs,
			"x-dead-letter-exchange":    cfg.Exchange,
			"x-dead-letter-routing-key": cfg.RoutingKey,
		},
	); err != nil {
		return fmt.Errorf("declare rabbitmq retry queue: %w", err)
	}
	if err := ch.QueueBind(cfg.RetryQueueName, cfg.RetryRoutingKey, cfg.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind rabbitmq retry queue: %w", err)
	}

	if _, err := ch.QueueDeclare(
		cfg.QueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    cfg.DLX,
			"x-dead-letter-routing-key": cfg.DeadLetterRoutingKey,
		},
	); err != nil {
		return fmt.Errorf("declare rabbitmq generation queue: %w", err)
	}
	if err := ch.QueueBind(cfg.QueueName, cfg.RoutingKey, cfg.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind rabbitmq generation queue: %w", err)
	}

	return nil
}

func dialWithRetry(ctx context.Context, url string, attempts int, delay time.Duration) (*amqp.Connection, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("connect rabbitmq: %w", ctx.Err())
		case <-timer.C:
		}
		if delay < 5*time.Second {
			delay *= 2
		}
	}
	return nil, fmt.Errorf("connect rabbitmq: %w", lastErr)
}
