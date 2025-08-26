package worker

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"downloader/internal/domain"
	"downloader/mocks"

	"shared/handler"
	obmocks "shared/observability/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDownloaderWorker_Name(t *testing.T) {
	mockLogger := &obmocks.MockLogger{}
	mockMetrics := &obmocks.MockMetrics{}
	mockService := &mocks.MockDownloadService{}

	worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

	assert.Equal(t, "downloader", worker.Name())
}

func TestDownloaderWorker_Process(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockService := &mocks.MockDownloadService{}

		// Setup metrics expectations
		mockMetrics.On("StartOperation", "worker_process").Return()
		mockMetrics.On("EndOperation", "worker_process").Return()
		mockMetrics.On("RecordDuration", "worker_process", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordDuration", "content_processing", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordSuccess", "worker_process").Return()

		// Setup logger expectations
		mockLogger.On("Info", mock.Anything, "Processing download request", mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, "Content processed successfully", mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, "Request processed successfully", mock.Anything).Return()

		expectedResult := &domain.DownloadResult{
			ID:          "test-123",
			URL:         "https://example.com/file.pdf",
			StoragePath: "test-123/file.pdf",
			ContentType: "application/pdf",
		}

		// Mock reader that simulates downloaded content
		content := "PDF content here"
		reader := &mockChecksumReader{
			reader:   io.NopCloser(strings.NewReader(content)),
			checksum: "abc123",
			size:     int64(len(content)),
		}

		mockService.On("Execute", mock.Anything, mock.MatchedBy(func(req domain.DownloadRequest) bool {
			return req.URL == "https://example.com/file.pdf"
		})).Return(expectedResult, reader, nil)

		worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

		request := handler.Request{
			ID:      "test-123",
			Type:    "download",
			Payload: []byte(`{"url": "https://example.com/file.pdf"}`),
		}

		resp, err := worker.Process(context.Background(), request)

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "test-123", resp.ID)
		assert.Nil(t, resp.Error)

		mockService.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("invalid payload", func(t *testing.T) {
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockService := &mocks.MockDownloadService{}

		// Setup metrics expectations
		mockMetrics.On("StartOperation", "worker_process").Return()
		mockMetrics.On("EndOperation", "worker_process").Return()
		mockMetrics.On("RecordDuration", "worker_process", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "worker_process", "invalid_payload").Return()

		// Setup logger expectations
		mockLogger.On("Info", mock.Anything, "Processing download request", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Failed to parse request payload", mock.Anything, mock.Anything).Return()

		worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

		request := handler.Request{
			ID:      "test-456",
			Type:    "download",
			Payload: []byte(`invalid json`),
		}

		resp, err := worker.Process(context.Background(), request)

		assert.NoError(t, err) // Worker returns error in response, not as error
		assert.False(t, resp.Success)
		assert.Equal(t, "INVALID_PAYLOAD", resp.Error.Code)
		assert.Contains(t, resp.Error.Message, "parse")

		mockMetrics.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("domain error - retryable", func(t *testing.T) {
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockService := &mocks.MockDownloadService{}

		// Setup metrics expectations
		mockMetrics.On("StartOperation", "worker_process").Return()
		mockMetrics.On("EndOperation", "worker_process").Return()
		mockMetrics.On("RecordDuration", "worker_process", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "worker_process", "download_failed").Return()
		mockMetrics.On("RecordError", "worker_retryable", "DOWNLOAD_FAILED").Return()

		// Setup logger expectations
		mockLogger.On("Info", mock.Anything, "Processing download request", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Download execution failed", mock.Anything, mock.Anything).Return()

		domainErr := &domain.DomainError{
			Code:      "DOWNLOAD_FAILED",
			Message:   "Network error",
			Err:       errors.New("connection timeout"),
			Retryable: true,
		}

		mockService.On("Execute", mock.Anything, mock.Anything).Return(nil, nil, domainErr)

		worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

		request := handler.Request{
			ID:      "test-789",
			Type:    "download",
			Payload: []byte(`{"url": "https://example.com/file.pdf"}`),
		}

		resp, err := worker.Process(context.Background(), request)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "DOWNLOAD_FAILED", resp.Error.Code)
		assert.True(t, resp.Error.Retryable)
		assert.Contains(t, resp.Error.Details, "connection timeout")

		mockService.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("sets request ID when not provided", func(t *testing.T) {
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockService := &mocks.MockDownloadService{}

		// Setup expectations
		mockMetrics.On("StartOperation", "worker_process").Return()
		mockMetrics.On("EndOperation", "worker_process").Return()
		mockMetrics.On("RecordDuration", mock.Anything, mock.Anything).Return()
		mockMetrics.On("RecordSuccess", "worker_process").Return()

		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

		result := &domain.DownloadResult{
			ID:          "req-999",
			URL:         "https://example.com/file.txt",
			StoragePath: "req-999/file.txt",
		}

		reader := &mockChecksumReader{
			reader: io.NopCloser(strings.NewReader("content")),
		}

		mockService.On("Execute", mock.Anything, mock.MatchedBy(func(req domain.DownloadRequest) bool {
			return req.ID == "req-999" // Should use request ID
		})).Return(result, reader, nil)

		worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

		request := handler.Request{
			ID:      "req-999",
			Type:    "download",
			Payload: []byte(`{"url": "https://example.com/file.txt"}`), // No ID in payload
		}

		resp, err := worker.Process(context.Background(), request)

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		mockService.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("content read error", func(t *testing.T) {
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}
		mockService := &mocks.MockDownloadService{}

		// Setup metrics expectations
		mockMetrics.On("StartOperation", "worker_process").Return()
		mockMetrics.On("EndOperation", "worker_process").Return()
		mockMetrics.On("RecordDuration", "worker_process", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordDuration", "content_processing", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "worker_process", "content_read").Return()

		// Setup logger expectations
		mockLogger.On("Info", mock.Anything, "Processing download request", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Failed to read content", mock.Anything, mock.Anything).Return()

		result := &domain.DownloadResult{
			ID:  "test-read-error",
			URL: "https://example.com/file.txt",
		}

		// Create a reader that fails
		reader := &failingReader{
			err: errors.New("read error"),
		}

		mockService.On("Execute", mock.Anything, mock.Anything).Return(result, reader, nil)

		worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

		request := handler.Request{
			ID:      "test-read-error",
			Type:    "download",
			Payload: []byte(`{"url": "https://example.com/file.txt"}`),
		}

		resp, err := worker.Process(context.Background(), request)

		assert.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "READ_ERROR", resp.Error.Code)

		mockService.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}

func TestDownloaderWorker_Health(t *testing.T) {
	mockLogger := &obmocks.MockLogger{}
	mockMetrics := &obmocks.MockMetrics{}
	mockService := &mocks.MockDownloadService{}

	// Expect health check metric
	mockMetrics.On("RecordSuccess", "health_check").Return()

	worker := NewDownloaderWorker(mockService, mockLogger, mockMetrics)

	err := worker.Health(context.Background())
	assert.NoError(t, err)

	mockMetrics.AssertExpectations(t)
}

// Test helpers

type mockChecksumReader struct {
	reader   io.ReadCloser
	checksum string
	size     int64
}

func (m *mockChecksumReader) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *mockChecksumReader) Close() error {
	return m.reader.Close()
}

func (m *mockChecksumReader) GetChecksum() string {
	return m.checksum
}

func (m *mockChecksumReader) GetSize() int64 {
	return m.size
}

type failingReader struct {
	err error
}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, f.err
}

func (f *failingReader) Close() error {
	return nil
}
