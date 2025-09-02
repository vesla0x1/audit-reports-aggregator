package ports

import (
	"context"
)

// Message represents a message to be published to a queue
type QueueMessage struct {
	// Queue or Topic to publish to
	Target string
	// Message body (will be JSON encoded)
	Body interface{}
}

// Queue defines the interface for message queue operations
type Queue interface {
	// Publish sends a message to the specified queue/topic
	Publish(ctx context.Context, message *QueueMessage) error

	// PublishBatch sends multiple messages to the same target
	PublishBatch(ctx context.Context, messages []*QueueMessage) error
}
