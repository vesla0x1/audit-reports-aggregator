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
	lambda.Start(a.handleEvent)
	return nil
}

func (a *Adapter) handleEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try to parse as SQS event
	var sqsEvent events.SQSEvent
	if err := json.Unmarshal(event, &sqsEvent); err == nil && len(sqsEvent.Records) > 0 {
		return a.handleSQSEvent(ctx, sqsEvent)
	}

	// Try direct handler.Request (for testing)
	var req handler.Request
	if err := json.Unmarshal(event, &req); err == nil && req.ID != "" {
		return a.handler.Handle(ctx, req)
	}

	return nil, fmt.Errorf("unsupported event type")
}

func (a *Adapter) handleSQSEvent(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	response := events.SQSEventResponse{
		BatchItemFailures: []events.SQSBatchItemFailure{},
	}

	for _, record := range event.Records {
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
			if a.config.EnablePartialBatchFailure {
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{
						ItemIdentifier: record.MessageId,
					})
			} else {
				return response, err
			}
		}
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
