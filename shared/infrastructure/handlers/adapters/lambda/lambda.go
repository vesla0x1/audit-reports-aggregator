package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"shared/config"
	"shared/domain/handler"
)

type Adapter struct {
	handler handler.Handler
	config  *config.LambdaConfig
}

func NewAdapter(h handler.Handler, cfg *config.LambdaConfig) *Adapter {
	return &Adapter{
		handler: h,
		config:  cfg,
	}
}

func (a *Adapter) Start() error {
	a.handler.Logger().Info("Starting Lambda adapter")
	a.handler.Metrics().IncrementCounter("lambda.starts", nil)

	lambda.Start(a.handleEvent)
	return nil
}

func (a *Adapter) handleEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	logger := a.handler.Logger()
	metrics := a.handler.Metrics()

	startTime := time.Now()
	logger.Info("Lambda invoked", "event_size", len(event))
	metrics.IncrementCounter("lambda.invocations", nil)

	// Try to parse as SQS event
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err == nil && len(sqsEvent.Records) > 0 {
		logger.Info("Processing SQS batch",
			"batch_size", len(sqsEvent.Records),
			"source", sqsEvent.Records[0].EventSource)

		metrics.IncrementCounter("lambda.invocations.sqs", nil)
		metrics.RecordHistogram("lambda.batch_size", float64(len(sqsEvent.Records)), nil)

		result, err := a.handleSQSEvent(ctx, sqsEvent)

		duration := time.Since(startTime)
		metrics.RecordHistogram("lambda.duration", float64(duration.Milliseconds()), map[string]string{
			"event_type": "sqs",
		})

		return result, err
	}

	// Try direct handler.Request (for testing)
	var req handler.Request
	if err := json.Unmarshal(event, &req); err == nil && req.ID != "" {
		logger.Info("Processing direct request", "request_id", req.ID)
		metrics.IncrementCounter("lambda.invocations.direct", nil)

		result, err := a.handler.Handle(ctx, req)

		duration := time.Since(startTime)
		metrics.RecordHistogram("lambda.duration", float64(duration.Milliseconds()), map[string]string{
			"event_type": "direct",
		})

		return result, err
	}

	logger.Error("Unsupported event type")
	metrics.IncrementCounter("lambda.invocations.unsupported", nil)
	return nil, fmt.Errorf("unsupported event type")
}

func (a *Adapter) handleSQSEvent(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	logger := a.handler.Logger()
	metrics := a.handler.Metrics()

	response := events.SQSEventResponse{
		BatchItemFailures: []events.SQSBatchItemFailure{},
	}

	successCount := 0
	failureCount := 0

	for i, record := range event.Records {
		logger.Info("Processing SQS message",
			"message_id", record.MessageId,
			"position", i+1,
			"total", len(event.Records))

		req := a.sqsMessageToRequest(record)

		// Apply timeout if configured
		reqCtx := ctx
		if a.config.Timeout > 0 {
			var cancel context.CancelFunc
			reqCtx, cancel = context.WithTimeout(ctx, a.config.Timeout)
			defer cancel()
		}

		resp, err := a.handler.Handle(reqCtx, req)

		if err != nil || !resp.Success {
			failureCount++

			logger.Error("Message processing failed",
				"message_id", record.MessageId,
				"error", err,
				"response_success", resp.Success)

			if a.config.EnablePartialBatchFailure {
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{
						ItemIdentifier: record.MessageId,
					})
			} else {
				metrics.IncrementCounter("lambda.batch.failed", map[string]string{
					"reason": "single_message_failure",
				})
				return response, err
			}
		} else {
			successCount++
		}
	}

	// Log batch summary
	logger.Info("SQS batch processing complete",
		"total_messages", len(event.Records),
		"success_count", successCount,
		"failure_count", failureCount,
		"partial_batch_enabled", a.config.EnablePartialBatchFailure)

	metrics.RecordHistogram("lambda.batch.success_count", float64(successCount), nil)
	metrics.RecordHistogram("lambda.batch.failure_count", float64(failureCount), nil)

	if failureCount == 0 {
		metrics.IncrementCounter("lambda.batch.complete_success", nil)
	} else if successCount == 0 {
		metrics.IncrementCounter("lambda.batch.complete_failure", nil)
	} else {
		metrics.IncrementCounter("lambda.batch.partial_failure", nil)
	}

	return response, nil
}

func (a *Adapter) sqsMessageToRequest(record events.SQSMessage) handler.Request {
	metadata := make(map[string]string)
	for key, attr := range record.MessageAttributes {
		if attr.StringValue != nil {
			metadata[key] = *attr.StringValue
		}
	}

	metadata["sqs_message_id"] = record.MessageId
	metadata["sqs_receipt_handle"] = record.ReceiptHandle

	var payload json.RawMessage
	if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
		payload, _ = json.Marshal(record.Body)
	}

	return handler.Request{
		ID:        record.MessageId,
		Source:    "sqs",
		Type:      metadata["type"],
		Payload:   payload,
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	}
}
