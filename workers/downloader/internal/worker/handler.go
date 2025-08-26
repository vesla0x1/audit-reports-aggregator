package worker

import (
	"context"
	"io"
	"time"

	"downloader/internal/domain"

	"shared/handler"
	"shared/observability/types"
)

// DownloadService defines the interface for download operations
type DownloadService interface {
	Execute(ctx context.Context, req domain.DownloadRequest) (*domain.DownloadResult, io.ReadCloser, error)
}

// checksumGetter provides access to checksum and size information
type checksumGetter interface {
	GetChecksum() string
	GetSize() int64
}

// DownloaderWorker implements the handler.Worker interface with comprehensive metrics
type DownloaderWorker struct {
	downloadService DownloadService
	logger          types.Logger
	metrics         types.Metrics
}

// NewDownloaderWorker creates a new downloader worker with observability
func NewDownloaderWorker(
	downloadService DownloadService,
	logger types.Logger,
	metrics types.Metrics,
) *DownloaderWorker {
	return &DownloaderWorker{
		downloadService: downloadService,
		logger:          logger,
		metrics:         metrics,
	}
}

// Name returns the worker name
func (w *DownloaderWorker) Name() string {
	return "downloader"
}

// Process handles download requests with comprehensive metrics tracking
func (w *DownloaderWorker) Process(ctx context.Context, request handler.Request) (handler.Response, error) {
	// Track concurrent requests
	w.metrics.StartOperation("worker_process")
	defer w.metrics.EndOperation("worker_process")

	startTime := time.Now()
	defer func() {
		// Always record request processing duration
		w.metrics.RecordDuration("worker_process", time.Since(startTime).Seconds())
	}()

	w.logger.Info(ctx, "Processing download request", types.Fields{
		"request_id":   request.ID,
		"request_type": request.Type,
	})

	// Parse request payload
	var downloadReq domain.DownloadRequest
	if err := request.Unmarshal(&downloadReq); err != nil {
		w.metrics.RecordError("worker_process", "invalid_payload")
		w.logger.Error(ctx, "Failed to parse request payload", err, types.Fields{
			"request_id": request.ID,
		})

		return handler.NewErrorResponse(
			request.ID,
			"INVALID_PAYLOAD",
			"Failed to parse download request",
			err.Error(),
		), nil
	}

	// Set request ID if not provided
	if downloadReq.ID == "" {
		downloadReq.ID = request.ID
	}

	// Execute download
	result, reader, err := w.downloadService.Execute(ctx, downloadReq)
	if err != nil {
		// Track different types of processing errors
		errorType := w.categorizeWorkerError(err)
		w.metrics.RecordError("worker_process", errorType)

		w.logger.Error(ctx, "Download execution failed", err, types.Fields{
			"request_id": request.ID,
			"url":        downloadReq.URL,
			"error_type": errorType,
		})

		// Handle domain errors
		if domainErr, ok := err.(*domain.DomainError); ok {
			errorResp := handler.NewErrorResponse(
				request.ID,
				domainErr.Code,
				domainErr.Message,
				domainErr.Error(),
			)
			// Set retryable flag
			errorResp.Error.Retryable = domainErr.Retryable

			// Track retryable errors separately
			if domainErr.Retryable {
				w.metrics.RecordError("worker_retryable", domainErr.Code)
			}

			return errorResp, nil
		}

		// Generic error
		return handler.NewErrorResponse(
			request.ID,
			"PROCESSING_ERROR",
			"Failed to process download request",
			err.Error(),
		), nil
	}

	// Process the downloaded content
	if reader != nil {
		// Track content processing
		processingStart := time.Now()

		defer func() {
			reader.Close()
			// Record content processing duration
			w.metrics.RecordDuration("content_processing", time.Since(processingStart).Seconds())
		}()

		// Read content to calculate checksum and size
		// In production, this would stream to MinIO/S3
		bytesRead, err := w.processContent(reader)
		if err != nil {
			w.metrics.RecordError("worker_process", "content_read")
			w.logger.Error(ctx, "Failed to read content", err, types.Fields{
				"request_id": request.ID,
				"bytes_read": bytesRead,
			})

			return handler.NewErrorResponse(
				request.ID,
				"READ_ERROR",
				"Failed to read downloaded content",
				err.Error(),
			), nil
		}

		// Get final checksum and size
		if cr, ok := reader.(checksumGetter); ok {
			result.Checksum = cr.GetChecksum()
			result.Size = cr.GetSize()

			w.logger.Info(ctx, "Content processed successfully", types.Fields{
				"request_id": request.ID,
				"size":       result.Size,
				"checksum":   result.Checksum,
			})
		}
	}

	// Create success response
	response, err := handler.NewSuccessResponse(request.ID, result)
	if err != nil {
		w.metrics.RecordError("worker_process", "response_creation")
		w.logger.Error(ctx, "Failed to create response", err, types.Fields{
			"request_id": request.ID,
		})

		return handler.NewErrorResponse(
			request.ID,
			"RESPONSE_ERROR",
			"Failed to create response",
			err.Error(),
		), nil
	}

	// Record successful processing
	w.metrics.RecordSuccess("worker_process")

	w.logger.Info(ctx, "Request processed successfully", types.Fields{
		"request_id":   request.ID,
		"storage_path": result.StoragePath,
		"content_type": result.ContentType,
	})

	return response, nil
}

// processContent reads and processes the downloaded content
func (w *DownloaderWorker) processContent(reader io.Reader) (int64, error) {
	// In production, this would stream to object storage
	// For now, we just consume the content to calculate checksum
	return io.Copy(io.Discard, reader)
}

// categorizeWorkerError categorizes errors for metrics tracking
func (w *DownloaderWorker) categorizeWorkerError(err error) string {
	if domainErr, ok := err.(*domain.DomainError); ok {
		switch domainErr.Code {
		case "INVALID_URL":
			return "invalid_url"
		case "DOWNLOAD_FAILED":
			return "download_failed"
		case "TIMEOUT":
			return "timeout"
		default:
			return "domain_error"
		}
	}

	// Check error message for common patterns
	errMsg := err.Error()
	switch {
	case errMsg == "":
		return "unknown"
	default:
		return "processing_error"
	}
}

// Health checks worker health with metrics
func (w *DownloaderWorker) Health(ctx context.Context) error {
	// Track health check calls
	w.metrics.RecordSuccess("health_check")

	// Could be extended to check dependencies
	// For example, checking if download service is healthy
	return nil
}

// GetMetrics returns the metrics instance for testing
func (w *DownloaderWorker) GetMetrics() types.Metrics {
	return w.metrics
}
