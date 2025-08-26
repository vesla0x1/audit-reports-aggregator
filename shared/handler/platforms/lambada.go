package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shared/handler"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// LambdaAdapter adapts worker handlers to AWS Lambda runtime with SQS event support
type LambdaAdapter struct {
	handler *handler.Handler
	config  *LambdaConfig
}

// LambdaConfig contains Lambda-specific configuration
type LambdaConfig struct {
	// MaxConcurrency for parallel message processing
	MaxConcurrency int
	// ProcessingTimeout for individual messages
	ProcessingTimeout time.Duration
	// EnablePartialBatchFailure allows reporting individual message failures
	EnablePartialBatchFailure bool
	// AutoBase64Decode handles base64 encoded SQS message bodies
	AutoBase64Decode bool
}

// NewLambdaAdapter creates a new Lambda adapter
func NewLambdaAdapter(h *handler.Handler, config *LambdaConfig) *LambdaAdapter {
	if config == nil {
		config = DefaultLambdaConfig()
	}
	return &LambdaAdapter{
		handler: h,
		config:  config,
	}
}

// DefaultLambdaConfig returns default Lambda configuration
func DefaultLambdaConfig() *LambdaConfig {
	return &LambdaConfig{
		MaxConcurrency:            10,
		ProcessingTimeout:         30 * time.Second,
		EnablePartialBatchFailure: true,
		AutoBase64Decode:          true,
	}
}

// Start begins the Lambda runtime handler
func (a *LambdaAdapter) Start() {
	lambda.Start(a.HandleEvent)
}

// HandleEvent is the main Lambda handler that routes different event types
func (a *LambdaAdapter) HandleEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try to parse as SQS event first
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err == nil && len(sqsEvent.Records) > 0 {
		return a.handleSQSEvent(ctx, sqsEvent)
	}

	return nil, fmt.Errorf("unsupported event type")
}

// handleSQSEvent processes SQS events with support for batch processing
func (a *LambdaAdapter) handleSQSEvent(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	response := events.SQSEventResponse{
		BatchItemFailures: []events.SQSBatchItemFailure{},
	}

	// Process each message
	for _, record := range event.Records {
		if err := a.processSQSMessage(ctx, record); err != nil {
			// If partial batch failure is enabled, track failed messages
			if a.config.EnablePartialBatchFailure {
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{
						ItemIdentifier: record.MessageId,
					})
			} else {
				// Without partial batch failure, fail the entire batch
				return response, err
			}
		}
	}

	return response, nil
}

// processSQSMessage processes a single SQS message
func (a *LambdaAdapter) processSQSMessage(ctx context.Context, record events.SQSMessage) error {
	// Build request from SQS message
	request := a.buildRequestFromSQS(record)

	// Add timeout if configured
	if a.config.ProcessingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.ProcessingTimeout)
		defer cancel()
	}

	// Process through handler
	response, err := a.handler.Handle(ctx, request)
	if err != nil {
		return fmt.Errorf("handler error: %w", err)
	}

	// Check if processing was successful
	if !response.Success {
		if response.Error != nil && response.Error.Retryable {
			// Return error to trigger retry via SQS
			return fmt.Errorf("retryable error: %s", response.Error.Message)
		}
		// Non-retryable errors are logged but not returned
		// This prevents the message from being retried
	}

	return nil
}

// buildRequestFromSQS converts SQS message to handler.Request
func (a *LambdaAdapter) buildRequestFromSQS(record events.SQSMessage) handler.Request {
	// Extract metadata from message attributes
	metadata := make(map[string]string)
	for key, attr := range record.MessageAttributes {
		if attr.StringValue != nil {
			metadata[key] = *attr.StringValue
		}
	}

	// Add SQS-specific metadata
	metadata["sqs_message_id"] = record.MessageId
	metadata["sqs_receipt_handle"] = record.ReceiptHandle
	metadata["sqs_event_source"] = record.EventSource

	// Parse message body
	var payload json.RawMessage
	body := record.Body

	// Try to parse as JSON
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		// If not JSON, wrap as string
		wrappedBody, _ := json.Marshal(body)
		payload = wrappedBody
	}

	// Extract request type from attributes or default
	requestType := "sqs_message"
	if msgType, ok := metadata["type"]; ok {
		requestType = msgType
	}

	// Extract request ID or use message ID
	requestID := record.MessageId
	if id, ok := metadata["request_id"]; ok {
		requestID = id
	}

	return handler.Request{
		ID:        requestID,
		Source:    "sqs",
		Type:      requestType,
		Payload:   payload,
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}
}
