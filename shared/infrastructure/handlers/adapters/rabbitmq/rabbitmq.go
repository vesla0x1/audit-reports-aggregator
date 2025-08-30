package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"

	"shared/config"
	"shared/domain/handler"
)

// Adapter handles RabbitMQ consumer runtime integration (simplified)
type Adapter struct {
	handler handler.Handler
	config  *config.RabbitMQConfig
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewAdapter creates a new RabbitMQ adapter with minimal config
func NewAdapter(h handler.Handler, cfg *config.RabbitMQConfig) *Adapter {
	return &Adapter{
		handler: h,
		config:  cfg,
	}
}

// Start begins consuming messages from RabbitMQ
func (a *Adapter) Start() error {
	// Connect
	conn, err := amqp.Dial(a.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	a.conn = conn

	// Open channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}
	a.channel = ch

	// Set QoS
	if a.config.PrefetchCount > 0 {
		if err := ch.Qos(a.config.PrefetchCount, 0, false); err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	// Declare queue (idempotent - creates if doesn't exist)
	q, err := ch.QueueDeclare(
		a.config.Queue, // name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer tag (auto-generated)
		false,  // auto-ack (we'll ack manually)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to consume: %w", err)
	}

	a.handler.Logger().Info("RabbitMQ consumer started",
		"queue", a.config.Queue,
		"prefetch", a.config.PrefetchCount)

	// Process messages
	for msg := range msgs {
		a.processMessage(msg)
	}

	return nil
}

// processMessage handles a single message
func (a *Adapter) processMessage(msg amqp.Delivery) {
	startTime := time.Now()

	// Create context with timeout
	ctx := context.Background()
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// Convert to handler.Request
	req := handler.Request{
		ID:        msg.MessageId,
		Source:    "rabbitmq",
		Type:      a.extractType(msg),
		Payload:   json.RawMessage(msg.Body),
		Metadata:  a.buildMetadata(msg),
		Timestamp: msg.Timestamp,
	}

	// Set defaults
	if req.ID == "" {
		req.ID = fmt.Sprintf("rmq-%d", msg.DeliveryTag)
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// Process
	resp, err := a.handler.Handle(ctx, req)

	// Handle result
	if err == nil && resp.Success {
		// Success - acknowledge
		if err := msg.Ack(false); err != nil {
			a.handler.Logger().Error("Failed to ack message",
				"id", req.ID,
				"error", err)
		}
		a.handler.Logger().Info("Message processed",
			"id", req.ID,
			"duration_ms", time.Since(startTime).Milliseconds())
		a.handler.Metrics().IncrementCounter("rabbitmq.success", nil)
	} else {
		// Failure - reject and requeue if not already redelivered
		requeue := !msg.Redelivered
		if err := msg.Nack(false, requeue); err != nil {
			a.handler.Logger().Error("Failed to nack message",
				"id", req.ID,
				"error", err)
		}
		a.handler.Logger().Error("Message processing failed",
			"id", req.ID,
			"error", err,
			"requeued", requeue)
		a.handler.Metrics().IncrementCounter("rabbitmq.failure", nil)
	}

	// Record duration
	a.handler.Metrics().RecordHistogram("rabbitmq.duration_ms",
		float64(time.Since(startTime).Milliseconds()), nil)
}

// Stop gracefully shuts down the consumer
func (a *Adapter) Stop(ctx context.Context) error {
	if a.channel != nil {
		a.channel.Close()
	}
	if a.conn != nil {
		a.conn.Close()
	}
	a.handler.Logger().Info("RabbitMQ consumer stopped")
	return nil
}

// extractType gets message type from headers or routing key
func (a *Adapter) extractType(msg amqp.Delivery) string {
	// Check headers first
	if t, ok := msg.Headers["type"]; ok {
		return fmt.Sprintf("%v", t)
	}
	// Use routing key if available
	if msg.RoutingKey != "" {
		return msg.RoutingKey
	}
	return "message"
}

// buildMetadata creates metadata from message properties
func (a *Adapter) buildMetadata(msg amqp.Delivery) map[string]string {
	meta := make(map[string]string)

	if msg.RoutingKey != "" {
		meta["routing_key"] = msg.RoutingKey
	}
	if msg.Exchange != "" {
		meta["exchange"] = msg.Exchange
	}
	if msg.CorrelationId != "" {
		meta["correlation_id"] = msg.CorrelationId
	}
	if msg.ReplyTo != "" {
		meta["reply_to"] = msg.ReplyTo
	}
	meta["redelivered"] = fmt.Sprintf("%v", msg.Redelivered)

	// Add custom headers
	for k, v := range msg.Headers {
		meta[fmt.Sprintf("header_%s", k)] = fmt.Sprintf("%v", v)
	}

	return meta
}
