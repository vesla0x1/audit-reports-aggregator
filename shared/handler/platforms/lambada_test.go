package platforms

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"shared/config"
	"shared/handler"
	"shared/handler/mocks"
	obmocks "shared/observability/mocks"
)

func TestLambdaAdapter_HandleSQSEvent(t *testing.T) {
	t.Run("successful SQS message processing", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			return req.Source == "sqs" && req.ID == "msg-123"
		})).Return(handler.Response{
			ID:      "msg-123",
			Success: true,
			Data:    json.RawMessage(`{"processed": true}`),
		}, nil)

		// Create mock observability
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockProvider.On("Logger", mock.Anything).Return(mockLogger)
		mockProvider.On("Metrics", mock.Anything).Return(mockMetrics)

		// Create handler and adapter
		cfg := config.DefaultHandlerConfig()
		h := handler.NewHandler(mockWorker, mockProvider, &cfg)
		adapter := NewLambdaAdapter(h, nil)

		// Create SQS event
		sqsEvent := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId:     "msg-123",
					ReceiptHandle: "receipt-123",
					Body:          `{"url": "https://example.com/file.pdf"}`,
					MessageAttributes: map[string]events.SQSMessageAttribute{
						"type": {
							StringValue: stringPtr("download"),
						},
					},
				},
			},
		}

		// Process event
		response, err := adapter.handleSQSEvent(context.Background(), sqsEvent)

		assert.NoError(t, err)
		assert.Empty(t, response.BatchItemFailures)
		mockWorker.AssertExpectations(t)
	})

	t.Run("partial batch failure", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")

		// First message succeeds
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			return req.ID == "msg-1"
		})).Return(handler.Response{
			ID:      "msg-1",
			Success: true,
		}, nil)

		// Second message fails with retryable error
		mockWorker.On("Process", mock.Anything, mock.MatchedBy(func(req handler.Request) bool {
			return req.ID == "msg-2"
		})).Return(handler.Response{
			ID:      "msg-2",
			Success: false,
			Error: &handler.ErrorResponse{
				Code:      "DOWNLOAD_FAILED",
				Message:   "Network error",
				Retryable: true,
			},
		}, nil)

		// Create mock observability
		mockProvider := &obmocks.MockProvider{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockProvider.On("Logger", mock.Anything).Return(mockLogger)
		mockProvider.On("Metrics", mock.Anything).Return(mockMetrics)

		// Create handler and adapter with partial batch failure enabled
		defaultCfg := config.DefaultHandlerConfig()
		h := handler.NewHandler(mockWorker, mockProvider, &defaultCfg)
		config := &LambdaConfig{
			EnablePartialBatchFailure: true,
		}
		adapter := NewLambdaAdapter(h, config)

		// Create SQS event with multiple messages
		sqsEvent := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId: "msg-1",
					Body:      `{"id": "1"}`,
				},
				{
					MessageId: "msg-2",
					Body:      `{"id": "2"}`,
				},
			},
		}

		// Process event
		response, err := adapter.handleSQSEvent(context.Background(), sqsEvent)

		assert.NoError(t, err)
		assert.Len(t, response.BatchItemFailures, 1)
		assert.Equal(t, "msg-2", response.BatchItemFailures[0].ItemIdentifier)
		mockWorker.AssertExpectations(t)
	})
}

func TestLambdaAdapter_HandleEvent(t *testing.T) {
	t.Run("routes SQS event correctly", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")
		mockWorker.On("Process", mock.Anything, mock.Anything).Return(
			handler.Response{Success: true}, nil)

		// Create mock observability
		mockProvider := &obmocks.MockProvider{}
		mockProvider.On("Logger", mock.Anything).Return(&obmocks.MockLogger{})
		mockProvider.On("Metrics", mock.Anything).Return(&obmocks.MockMetrics{})

		// Create handler and adapter
		cfg := config.DefaultHandlerConfig()
		h := handler.NewHandler(mockWorker, mockProvider, &cfg)
		adapter := NewLambdaAdapter(h, nil)

		// Create SQS event
		sqsEvent := events.SQSEvent{
			Records: []events.SQSMessage{
				{
					MessageId: "test-msg",
					Body:      `{"test": "data"}`,
				},
			},
		}

		eventBytes, _ := json.Marshal(sqsEvent)

		// Process event
		result, err := adapter.HandleEvent(context.Background(), eventBytes)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Should return SQSEventResponse
		response, ok := result.(events.SQSEventResponse)
		assert.True(t, ok)
		assert.Empty(t, response.BatchItemFailures)

		mockWorker.AssertExpectations(t)
	})

	t.Run("returns error for unsupported event", func(t *testing.T) {
		// Create mock worker
		mockWorker := &mocks.MockWorker{}
		mockWorker.On("Name").Return("test-worker")

		// Create mock observability
		mockProvider := &obmocks.MockProvider{}

		// Create handler and adapter
		cfg := config.DefaultHandlerConfig()
		h := handler.NewHandler(mockWorker, mockProvider, &cfg)
		adapter := NewLambdaAdapter(h, nil)

		// Create unsupported event
		eventBytes := json.RawMessage(`{"unsupported": "event"}`)

		// Process event
		_, err := adapter.HandleEvent(context.Background(), eventBytes)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported event type")
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
