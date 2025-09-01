package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// handles Lambda runtime integration
type lambdaRuntime struct {
	handler ports.Handler
	logger  ports.Logger
	metrics ports.Metrics
	config  *config.LambdaConfig
}

// NewLambdaRuntime creates a new Lambda runtime
func NewLambdaRuntime(cfg *config.LambdaConfig, handler ports.Handler, obs ports.Observability) ports.Runtime {
	logger, metrics, err := obs.ComponentsScoped("runtime.lambda")
	if err != nil {
		panic(fmt.Errorf("failed to create runtime: Observability was not initialized %w", err))
	}

	if handler == nil {
		panic(fmt.Errorf("failed to create runtime: handler is required"))
	}

	return &lambdaRuntime{
		handler: handler,
		logger:  logger,
		metrics: metrics,
		config:  cfg,
	}
}

// Start begins the Lambda runtime
func (runtime *lambdaRuntime) Start() error {
	runtime.logStartup()
	lambda.Start(runtime.handleEvent)
	return nil
}

// handleEvent is the main Lambda entry point
func (runtime *lambdaRuntime) handleEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	invocation := runtime.trackInvocation(event)
	defer invocation.recordDuration()

	return runtime.routeEvent(ctx, event)
}

// routeEvent determines event type and routes to appropriate handler
func (runtime *lambdaRuntime) routeEvent(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try SQS event
	if sqsEvent, ok := runtime.tryParseSQSEvent(event); ok {
		return runtime.processSQSEvent(ctx, sqsEvent)
	}

	// Try direct request
	if request, ok := runtime.tryParseDirectRequest(event); ok {
		return runtime.processDirectRequest(ctx, request)
	}

	// Unsupported event type
	runtime.recordUnsupportedEvent()
	return nil, fmt.Errorf("unsupported event type")
}

// --- SQS Event Processing ---

// processSQSEvent handles SQS batch events
func (runtime *lambdaRuntime) processSQSEvent(ctx context.Context, event events.SQSEvent) (interface{}, error) {
	batch := newBatchProcessor(runtime.handler, runtime.logger, runtime.metrics, runtime.config)

	runtime.logBatchStart(event)
	runtime.recordBatchMetrics(event)

	response := batch.process(ctx, event)

	runtime.logBatchComplete(batch.getStats(), event)
	runtime.recordBatchResults(batch.getStats())

	return response, batch.getError()
}

// batchProcessor encapsulates batch processing logic
type batchProcessor struct {
	handler  ports.Handler
	logger   ports.Logger
	metrics  ports.Metrics
	config   *config.LambdaConfig
	stats    batchStats
	response events.SQSEventResponse
}

type batchStats struct {
	successCount int
	failureCount int
	totalCount   int
}

func newBatchProcessor(h ports.Handler, logger ports.Logger, metrics ports.Metrics, cfg *config.LambdaConfig) *batchProcessor {
	return &batchProcessor{
		handler: h,
		logger:  logger,
		metrics: metrics,
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

func (b *batchProcessor) isFailure(resp ports.RuntimeResponse, err error) bool {
	return err != nil || !resp.Success
}

func (b *batchProcessor) handleFailure(record events.SQSMessage, err error, resp ports.RuntimeResponse) {
	b.stats.failureCount++
	b.logMessageFailure(record, err, resp)

	if b.config.EnablePartialBatchFailure {
		b.response.BatchItemFailures = append(b.response.BatchItemFailures,
			events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
	}
}

func (b *batchProcessor) convertToRequest(record events.SQSMessage) ports.RuntimeRequest {
	return ports.RuntimeRequest{
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
func (runtime *lambdaRuntime) processDirectRequest(ctx context.Context, req ports.RuntimeRequest) (interface{}, error) {
	runtime.logDirectRequest(req)
	runtime.recordDirectRequestMetric()

	return runtime.handler.Handle(ctx, req)
}

// --- Parsing Helpers ---

func (runtime *lambdaRuntime) tryParseSQSEvent(event json.RawMessage) (events.SQSEvent, bool) {
	var sqsEvent events.SQSEvent
	err := json.Unmarshal(event, &sqsEvent)
	return sqsEvent, err == nil && len(sqsEvent.Records) > 0
}

func (runtime *lambdaRuntime) tryParseDirectRequest(event json.RawMessage) (ports.RuntimeRequest, bool) {
	var req ports.RuntimeRequest
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

func (runtime *lambdaRuntime) logStartup() {
	runtime.logger.Info("Starting Lambda runtime")
	runtime.metrics.IncrementCounter("lambda.starts", nil)
}

func (runtime *lambdaRuntime) logBatchStart(event events.SQSEvent) {
	runtime.logger.Info("Processing SQS batch",
		"batch_size", len(event.Records),
		"source", event.Records[0].EventSource)
}

func (runtime *lambdaRuntime) logBatchComplete(stats batchStats, event events.SQSEvent) {
	runtime.logger.Info("SQS batch processing complete",
		"total_messages", stats.totalCount,
		"success_count", stats.successCount,
		"failure_count", stats.failureCount,
		"partial_batch_enabled", runtime.config.EnablePartialBatchFailure)
}

func (b *batchProcessor) logMessageStart(record events.SQSMessage, index int) {
	b.logger.Info("Processing SQS message",
		"message_id", record.MessageId,
		"position", index+1,
		"total", b.stats.totalCount)
}

func (b *batchProcessor) logMessageFailure(record events.SQSMessage, err error, resp ports.RuntimeResponse) {
	b.logger.Error("Message processing failed",
		"message_id", record.MessageId,
		"error", err,
		"response_success", resp.Success)
}

func (runtime *lambdaRuntime) logDirectRequest(req ports.RuntimeRequest) {
	runtime.logger.Info("Processing direct request", "request_id", req.ID)
}

// --- Metrics Helpers ---

// invocationTracker tracks metrics for a single invocation
type invocationTracker struct {
	runtime   *lambdaRuntime
	startTime time.Time
	eventType string
}

func (runtime *lambdaRuntime) trackInvocation(event json.RawMessage) *invocationTracker {
	runtime.logger.Info("Lambda invoked", "event_size", len(event))
	runtime.metrics.IncrementCounter("lambda.invocations", nil)

	return &invocationTracker{
		runtime:   runtime,
		startTime: time.Now(),
	}
}

func (t *invocationTracker) recordDuration() {
	if t.eventType == "" {
		return
	}

	duration := time.Since(t.startTime)
	t.runtime.metrics.RecordHistogram("lambda.duration",
		float64(duration.Milliseconds()),
		map[string]string{"event_type": t.eventType})
}

func (runtime *lambdaRuntime) recordBatchMetrics(event events.SQSEvent) {
	runtime.metrics.IncrementCounter("lambda.invocations.sqs", nil)
	runtime.metrics.RecordHistogram("lambda.batch_size", float64(len(event.Records)), nil)
}

func (runtime *lambdaRuntime) recordBatchResults(stats batchStats) {
	runtime.metrics.RecordHistogram("lambda.batch.success_count", float64(stats.successCount), nil)
	runtime.metrics.RecordHistogram("lambda.batch.failure_count", float64(stats.failureCount), nil)

	switch {
	case stats.failureCount == 0:
		runtime.metrics.IncrementCounter("lambda.batch.complete_success", nil)
	case stats.successCount == 0:
		runtime.metrics.IncrementCounter("lambda.batch.complete_failure", nil)
	default:
		runtime.metrics.IncrementCounter("lambda.batch.partial_failure", nil)
	}
}

func (runtime *lambdaRuntime) recordDirectRequestMetric() {
	runtime.metrics.IncrementCounter("lambda.invocations.direct", nil)
}

func (runtime *lambdaRuntime) recordUnsupportedEvent() {
	runtime.logger.Error("Unsupported event type")
	runtime.metrics.IncrementCounter("lambda.invocations.unsupported", nil)
}
