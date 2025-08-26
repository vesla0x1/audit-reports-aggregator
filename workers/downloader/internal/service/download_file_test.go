package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"downloader/internal/domain"
	"downloader/mocks"

	obmocks "shared/observability/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDownloadService_Execute(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		// Setup logger expectations
		mockLogger.On("Info", mock.Anything, "Starting file download", mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, "File download prepared", mock.Anything).Return()

		// Setup metrics expectations
		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordSuccess", "download").Return()
		mockMetrics.On("RecordFileSize", "pdf", mock.AnythingOfType("int64")).Return().Maybe()

		// Setup HTTP client expectations
		content := "file content for testing"
		reader := io.NopCloser(strings.NewReader(content))
		headers := map[string]string{
			"Content-Type": "application/pdf",
		}
		mockHTTP.On("Download", mock.Anything, "https://example.com/file.pdf", mock.Anything).
			Return(reader, headers, nil)

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:          "test-123",
			URL:         "https://example.com/file.pdf",
			StoragePath: "docs/file.pdf",
		}

		result, downloadReader, err := service.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, downloadReader)
		defer downloadReader.Close()

		assert.Equal(t, "test-123", result.ID)
		assert.Equal(t, "https://example.com/file.pdf", result.URL)
		assert.Equal(t, "docs/file.pdf", result.StoragePath)
		assert.Equal(t, "application/pdf", result.ContentType)

		// Read content to trigger checksum calculation
		content2, err := io.ReadAll(downloadReader)
		assert.NoError(t, err)
		assert.Equal(t, "file content for testing", string(content2))

		mockHTTP.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("invalid URL - empty", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockLogger.On("Info", mock.Anything, "Starting file download", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Request validation failed", mock.Anything, mock.Anything).Return()

		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "download", "validation_error").Return()

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:  "test-456",
			URL: "",
		}

		result, reader, err := service.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Nil(t, reader)
		assert.Equal(t, domain.ErrInvalidURL, err)

		mockLogger.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("invalid URL - non-HTTP scheme", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockLogger.On("Info", mock.Anything, "Starting file download", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Request validation failed", mock.Anything, mock.Anything).Return()

		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "download", "validation_error").Return()

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:  "test-789",
			URL: "ftp://example.com/file.txt", // FTP not supported
		}

		result, reader, err := service.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Nil(t, reader)

		domainErr, ok := err.(*domain.DomainError)
		assert.True(t, ok)
		assert.Equal(t, "INVALID_URL", domainErr.Code)
		assert.Contains(t, domainErr.Message, "HTTP and HTTPS")

		mockLogger.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("download failure", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockLogger.On("Info", mock.Anything, "Starting file download", mock.Anything).Return()
		mockLogger.On("Error", mock.Anything, "Failed to download file", mock.Anything, mock.Anything).Return()

		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordError", "download", "timeout").Return()

		downloadErr := errors.New("network timeout")
		mockHTTP.On("Download", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, nil, downloadErr)

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:  "test-999",
			URL: "https://example.com/file.txt",
		}

		result, reader, err := service.Execute(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Nil(t, reader)

		domainErr, ok := err.(*domain.DomainError)
		assert.True(t, ok)
		assert.Equal(t, "DOWNLOAD_FAILED", domainErr.Code)
		assert.True(t, domainErr.Retryable)

		mockHTTP.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("generates storage path when not provided", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		mockLogger.On("Info", mock.Anything, "Starting file download", mock.Anything).Return()
		mockLogger.On("Info", mock.Anything, "File download prepared", mock.Anything).Return()

		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordSuccess", "download").Return()
		mockMetrics.On("RecordFileSize", "pdf", mock.AnythingOfType("int64")).Return().Maybe()

		reader := io.NopCloser(strings.NewReader("content"))
		headers := map[string]string{"Content-Type": "application/pdf"}
		mockHTTP.On("Download", mock.Anything, "https://example.com/path/document.pdf", mock.Anything).
			Return(reader, headers, nil)

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:  "test-auto-path",
			URL: "https://example.com/path/document.pdf",
			// StoragePath not provided
		}

		result, downloadReader, err := service.Execute(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, downloadReader)
		defer downloadReader.Close()

		// Should generate path from URL
		assert.Equal(t, "test-auto-path/document.pdf", result.StoragePath)

		mockHTTP.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		mockMetrics.AssertExpectations(t)
	})

	t.Run("tracks file size on close", func(t *testing.T) {
		mockHTTP := &mocks.MockHTTPClient{}
		mockLogger := &obmocks.MockLogger{}
		mockMetrics := &obmocks.MockMetrics{}

		// Setup expectations
		mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()

		mockMetrics.On("StartOperation", "download").Return()
		mockMetrics.On("EndOperation", "download").Return()
		mockMetrics.On("RecordDuration", "download", mock.AnythingOfType("float64")).Return()
		mockMetrics.On("RecordSuccess", "download").Return()

		// Expect file size to be recorded when reader is closed
		content := "test content with known size"
		mockMetrics.On("RecordFileSize", "text", int64(len(content))).Return()

		reader := io.NopCloser(strings.NewReader(content))
		headers := map[string]string{"Content-Type": "text/plain"}
		mockHTTP.On("Download", mock.Anything, mock.Anything, mock.Anything).
			Return(reader, headers, nil)

		service := NewDownloadService(mockHTTP, mockLogger, mockMetrics)

		req := domain.DownloadRequest{
			ID:  "test-size",
			URL: "https://example.com/file.txt",
		}

		_, downloadReader, err := service.Execute(context.Background(), req)
		assert.NoError(t, err)

		// Read all content
		data, err := io.ReadAll(downloadReader)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))

		// Close to trigger size recording
		downloadReader.Close()

		mockMetrics.AssertExpectations(t)
	})
}
