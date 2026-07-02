package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
)

type Consumer struct {
	cfg         jobqueue.Config
	prefetch    int
	maxAttempts int
	processor   *generationjobs.Worker
	publisher   *jobqueue.RabbitMQPublisher
	log         *zap.Logger

	conn    *amqp.Connection
	channel *amqp.Channel
	ready   bool
}

func NewConsumer(
	cfg jobqueue.Config,
	prefetch int,
	maxAttempts int,
	processor *generationjobs.Worker,
	publisher *jobqueue.RabbitMQPublisher,
	log *zap.Logger,
) *Consumer {
	if prefetch < 1 {
		prefetch = 1
	}
	if maxAttempts < 1 {
		maxAttempts = 3
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Consumer{
		cfg:         jobqueue.NormalizeConfig(cfg),
		prefetch:    prefetch,
		maxAttempts: maxAttempts,
		processor:   processor,
		publisher:   publisher,
		log:         log,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	conn, err := dialWithRetry(ctx, c.cfg.URL, 10, 500*time.Millisecond)
	if err != nil {
		return err
	}
	c.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open rabbitmq channel: %w", err)
	}
	c.channel = ch

	if err := jobqueue.DeclareTopology(ch, c.cfg); err != nil {
		return err
	}
	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("set rabbitmq qos: %w", err)
	}

	deliveries, err := ch.Consume(
		c.cfg.QueueName,
		"worker-service",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume rabbitmq queue: %w", err)
	}

	c.ready = true
	c.log.Info("rabbitmq consumer started",
		zap.String("queue", c.cfg.QueueName),
		zap.Int("prefetch", c.prefetch),
	)

	defer func() {
		c.ready = false
		_ = ch.Close()
		_ = conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("rabbitmq deliveries channel closed")
			}
			c.handleDelivery(ctx, delivery)
		}
	}
}

func (c *Consumer) Close() error {
	c.ready = false
	var err error
	if c.channel != nil {
		err = c.channel.Close()
	}
	if c.conn != nil {
		if closeErr := c.conn.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (c *Consumer) Ready() bool {
	return c.ready &&
		c.conn != nil &&
		!c.conn.IsClosed() &&
		c.channel != nil &&
		!c.channel.IsClosed() &&
		c.publisher != nil &&
		c.publisher.IsReady()
}

func (c *Consumer) handleDelivery(ctx context.Context, delivery amqp.Delivery) {
	startedAt := time.Now()
	attempt := readAttempt(delivery.Headers)
	if attempt < 1 {
		attempt = 1
	}
	msg, err := decodeMessage(delivery)
	if err != nil {
		c.log.Warn("invalid generation job message rejected",
			zap.Error(err),
			zap.Int("attempt", attempt),
		)
		_ = delivery.Nack(false, false)
		return
	}

	logFields := []zap.Field{
		zap.String("job_id", msg.JobID.String()),
		zap.String("trip_id", msg.TripID.String()),
		zap.String("job_type", string(msg.JobType)),
		zap.String("message_id", msg.MessageID.String()),
		zap.Int("attempt", attempt),
	}

	result, err := c.processor.ProcessJobByID(ctx, msg.JobID, false)
	if err != nil {
		c.log.Warn("generation job processing failed before terminal state", append(logFields, zap.Error(err))...)
		c.retryOrRequeue(ctx, delivery, msg, attempt, "worker_processing_failed", err.Error())
		return
	}

	switch result.Status {
	case generationjobs.ProcessStatusSkipped:
		_ = delivery.Ack(false)
		c.log.Info("generation job message acknowledged as duplicate", logFields...)
	case generationjobs.ProcessStatusCompleted:
		_ = delivery.Ack(false)
		c.log.Info("generation job message acknowledged after completion",
			append(logFields,
				zap.Duration("duration", time.Since(startedAt)),
			)...,
		)
	case generationjobs.ProcessStatusFailed:
		fields := append(logFields,
			zap.String("error_code", result.ErrorCode),
			zap.String("error_message", result.ErrorMessage),
			zap.Bool("retryable", result.Retryable),
		)
		if result.Retryable && attempt < c.maxAttempts {
			if err := c.processor.ResetRunningJobForRetry(ctx, msg.JobID, result.ErrorCode, result.ErrorMessage); err != nil {
				c.log.Warn("failed to reset generation job for retry", append(fields, zap.Error(err))...)
				_ = delivery.Nack(false, true)
				return
			}
			msg.MessageID = uuid.New()
			msg.CreatedAt = time.Now().UTC()
			if err := c.publisher.PublishRetry(ctx, msg, attempt+1); err != nil {
				c.log.Warn("failed to publish retry generation job message", append(fields, zap.Error(err))...)
				_ = delivery.Nack(false, true)
				return
			}
			_ = delivery.Ack(false)
			c.log.Warn("generation job scheduled for retry", append(fields, zap.Int("next_attempt", attempt+1))...)
			return
		}

		if err := c.processor.FailClaimedJob(ctx, result.Job, result.ErrorCode, result.ErrorMessage); err != nil {
			c.log.Warn("failed to mark generation job terminally failed", append(fields, zap.Error(err))...)
			_ = delivery.Nack(false, true)
			return
		}
		_ = delivery.Nack(false, false)
		c.log.Warn("generation job terminally failed and dead-lettered", fields...)
	default:
		c.log.Warn("unknown generation job process result", append(logFields, zap.String("status", string(result.Status)))...)
		_ = delivery.Nack(false, false)
	}
}

func (c *Consumer) retryOrRequeue(
	ctx context.Context,
	delivery amqp.Delivery,
	msg generationjobs.QueueMessage,
	attempt int,
	code string,
	message string,
) {
	if attempt >= c.maxAttempts {
		_ = delivery.Nack(false, false)
		return
	}
	msg.MessageID = uuid.New()
	msg.CreatedAt = time.Now().UTC()
	if err := c.publisher.PublishRetry(ctx, msg, attempt+1); err != nil {
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
	c.log.Warn("generation job message requeued for retry",
		zap.String("job_id", msg.JobID.String()),
		zap.String("message_id", msg.MessageID.String()),
		zap.Int("next_attempt", attempt+1),
		zap.String("error_code", code),
		zap.String("error_message", message),
	)
}

func decodeMessage(delivery amqp.Delivery) (generationjobs.QueueMessage, error) {
	if delivery.ContentType != "" && delivery.ContentType != generationjobs.ContentTypeJSON {
		return generationjobs.QueueMessage{}, fmt.Errorf("unsupported content type %q", delivery.ContentType)
	}
	var msg generationjobs.QueueMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return generationjobs.QueueMessage{}, fmt.Errorf("decode generation job message: %w", err)
	}
	if err := generationjobs.ValidateQueueMessage(msg); err != nil {
		return generationjobs.QueueMessage{}, err
	}
	return msg, nil
}

func readAttempt(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	switch v := headers[generationjobs.HeaderAttempts].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	default:
		return 0
	}
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
