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

// Adapter handles Lambda runtime integration
type Adapter struct {
	handler handler.Handler
	config  *config.LambdaConfig
}

// EventProcessor handles different event types
type EventProcessor interface {
	Process(ctx context.Context, event json.RawMessage) (interface{}, error)
}

// NewAdapter creates a new Lambda adapter
func NewAdapter(h handler.Handler, cfg *config.LambdaConfig) *Adapter {
	return &Adapter{
		handler: h,
		config:  cfg,
	}
}

// Start begins the Lambda runtime
func (a *Adapter) Start() error {
	a.logStartup()
	lambda.Start(a.handleEvent)
	return nil
}

// handleEvent is the main Lambda entry point
func (a *Adapter) handleEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	invocation := a.trackInvocation(event)
	defer invocation.recordDuration()

	return a.routeEvent(ctx, event)
}

// routeEvent determines event type and routes to appropriate handler
func (a *Adapter) routeEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try SQS event
	if sqsEvent, ok := a.tryParseSQSEvent(event); ok {
		return a.processSQSEvent(ctx, sqsEvent)
	}

	// Try direct request
	if request, ok := a.tryParseDirectRequest(event); ok {
		return a.processDirectRequest(ctx, request)
	}

	// Unsupported event type
	a.recordUnsupportedEvent()
	return nil, fmt.Errorf("unsupported event type")
}

// --- SQS Event Processing ---

// processSQSEvent handles SQS batch events
func (a *Adapter) processSQSEvent(ctx context.Context, event events.SQSEvent) (interface{}, error) {
	batch := newBatchProcessor(a.handler, a.config)

	a.logBatchStart(event)
	a.recordBatchMetrics(event)

	response := batch.process(ctx, event)

	a.logBatchComplete(batch.getStats(), event)
	a.recordBatchResults(batch.getStats())

	return response, batch.getError()
}

// batchProcessor encapsulates batch processing logic
type batchProcessor struct {
	handler  handler.Handler
	config   *config.LambdaConfig
	stats    batchStats
	response events.SQSEventResponse
}

type batchStats struct {
	successCount int
	failureCount int
	totalCount   int
}

func newBatchProcessor(h handler.Handler, cfg *config.LambdaConfig) *batchProcessor {
	return &batchProcessor{
		handler: h,
		config:  cfg,
		response: events.SQSEventResponse{
			BatchItemFailures: []events.SQSBatchItemFailure{},
		},
	}
}

func (b *batchProcessor) process(ctx context.Context, event events.SQSEvent) events.SQSEventResponse {
	b.stats.totalCount = len(event.Records)

	for i, record := range event.Records {
		b.processMessage(ctx, record, i)
	}

	return b.response
}

func (b *batchProcessor) processMessage(ctx context.Context, record events.SQSMessage, index int) {
	b.logMessageStart(record, index)

	request := b.convertToRequest(record)
	reqCtx := b.applyTimeout(ctx)

	resp, err := b.handler.Handle(reqCtx, request)

	if b.isFailure(resp, err) {
		b.handleFailure(record, err, resp)
	} else {
		b.stats.successCount++
	}
}

func (b *batchProcessor) isFailure(resp handler.Response, err error) bool {
	return err != nil || !resp.Success
}

func (b *batchProcessor) handleFailure(record events.SQSMessage, err error, resp handler.Response) {
	b.stats.failureCount++
	b.logMessageFailure(record, err, resp)

	if b.config.EnablePartialBatchFailure {
		b.response.BatchItemFailures = append(b.response.BatchItemFailures,
			events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
	}
}

func (b *batchProcessor) convertToRequest(record events.SQSMessage) handler.Request {
	return handler.Request{
		ID:        record.MessageId,
		Source:    "sqs",
		Type:      extractMessageType(record),
		Payload:   parseMessageBody(record.Body),
		Metadata:  extractMetadata(record),
		Timestamp: time.Now().UTC(),
	}
}

func (b *batchProcessor) applyTimeout(ctx context.Context) context.Context {
	if b.config.Timeout <= 0 {
		return ctx
	}

	reqCtx, _ := context.WithTimeout(ctx, b.config.Timeout)
	return reqCtx
}

func (b *batchProcessor) getStats() batchStats {
	return b.stats
}

func (b *batchProcessor) getError() error {
	if !b.config.EnablePartialBatchFailure && b.stats.failureCount > 0 {
		return fmt.Errorf("batch processing failed: %d/%d messages failed",
			b.stats.failureCount, b.stats.totalCount)
	}
	return nil
}

// --- Direct Request Processing ---

// processDirectRequest handles direct handler requests
func (a *Adapter) processDirectRequest(ctx context.Context, req handler.Request) (interface{}, error) {
	a.logDirectRequest(req)
	a.recordDirectRequestMetric()

	return a.handler.Handle(ctx, req)
}

// --- Parsing Helpers ---

func (a *Adapter) tryParseSQSEvent(event json.RawMessage) (events.SQSEvent, bool) {
	var sqsEvent events.SQSEvent
	err := json.Unmarshal(event, &sqsEvent)
	return sqsEvent, err == nil && len(sqsEvent.Records) > 0
}

func (a *Adapter) tryParseDirectRequest(event json.RawMessage) (handler.Request, bool) {
	var req handler.Request
	err := json.Unmarshal(event, &req)
	return req, err == nil && req.ID != ""
}

func extractMessageType(record events.SQSMessage) string {
	if attr, exists := record.MessageAttributes["type"]; exists && attr.StringValue != nil {
		return *attr.StringValue
	}
	return ""
}

func parseMessageBody(body string) json.RawMessage {
	var payload json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		payload, _ = json.Marshal(body)
	}
	return payload
}

func extractMetadata(record events.SQSMessage) map[string]string {
	metadata := make(map[string]string)

	// Extract message attributes
	for key, attr := range record.MessageAttributes {
		if attr.StringValue != nil {
			metadata[key] = *attr.StringValue
		}
	}

	// Add SQS-specific metadata
	metadata["sqs_message_id"] = record.MessageId
	metadata["sqs_receipt_handle"] = record.ReceiptHandle

	return metadata
}

// --- Logging Helpers ---

func (a *Adapter) logStartup() {
	a.handler.Logger().Info("Starting Lambda adapter")
	a.handler.Metrics().IncrementCounter("lambda.starts", nil)
}

func (a *Adapter) logBatchStart(event events.SQSEvent) {
	a.handler.Logger().Info("Processing SQS batch",
		"batch_size", len(event.Records),
		"source", event.Records[0].EventSource)
}

func (a *Adapter) logBatchComplete(stats batchStats, event events.SQSEvent) {
	a.handler.Logger().Info("SQS batch processing complete",
		"total_messages", stats.totalCount,
		"success_count", stats.successCount,
		"failure_count", stats.failureCount,
		"partial_batch_enabled", a.config.EnablePartialBatchFailure)
}

func (b *batchProcessor) logMessageStart(record events.SQSMessage, index int) {
	b.handler.Logger().Info("Processing SQS message",
		"message_id", record.MessageId,
		"position", index+1,
		"total", b.stats.totalCount)
}

func (b *batchProcessor) logMessageFailure(record events.SQSMessage, err error, resp handler.Response) {
	b.handler.Logger().Error("Message processing failed",
		"message_id", record.MessageId,
		"error", err,
		"response_success", resp.Success)
}

func (a *Adapter) logDirectRequest(req handler.Request) {
	a.handler.Logger().Info("Processing direct request", "request_id", req.ID)
}

// --- Metrics Helpers ---

// invocationTracker tracks metrics for a single invocation
type invocationTracker struct {
	adapter   *Adapter
	startTime time.Time
	eventType string
}

func (a *Adapter) trackInvocation(event json.RawMessage) *invocationTracker {
	logger := a.handler.Logger()
	metrics := a.handler.Metrics()

	logger.Info("Lambda invoked", "event_size", len(event))
	metrics.IncrementCounter("lambda.invocations", nil)

	return &invocationTracker{
		adapter:   a,
		startTime: time.Now(),
	}
}

func (t *invocationTracker) recordDuration() {
	if t.eventType == "" {
		return
	}

	duration := time.Since(t.startTime)
	t.adapter.handler.Metrics().RecordHistogram("lambda.duration",
		float64(duration.Milliseconds()),
		map[string]string{"event_type": t.eventType})
}

func (a *Adapter) recordBatchMetrics(event events.SQSEvent) {
	metrics := a.handler.Metrics()
	metrics.IncrementCounter("lambda.invocations.sqs", nil)
	metrics.RecordHistogram("lambda.batch_size", float64(len(event.Records)), nil)
}

func (a *Adapter) recordBatchResults(stats batchStats) {
	metrics := a.handler.Metrics()

	metrics.RecordHistogram("lambda.batch.success_count", float64(stats.successCount), nil)
	metrics.RecordHistogram("lambda.batch.failure_count", float64(stats.failureCount), nil)

	switch {
	case stats.failureCount == 0:
		metrics.IncrementCounter("lambda.batch.complete_success", nil)
	case stats.successCount == 0:
		metrics.IncrementCounter("lambda.batch.complete_failure", nil)
	default:
		metrics.IncrementCounter("lambda.batch.partial_failure", nil)
	}
}

func (a *Adapter) recordDirectRequestMetric() {
	a.handler.Metrics().IncrementCounter("lambda.invocations.direct", nil)
}

func (a *Adapter) recordUnsupportedEvent() {
	a.handler.Logger().Error("Unsupported event type")
	a.handler.Metrics().IncrementCounter("lambda.invocations.unsupported", nil)
}
