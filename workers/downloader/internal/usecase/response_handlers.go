package usecase

import (
	"fmt"
	"shared/application/ports"
	"shared/domain/dto"
)

func (w *DownloaderWorker) handleParseError(requestID string, err error) (ports.RuntimeResponse, error) {
	return ports.RuntimeResponse{
		Success: false,
		Error: &ports.ErrorInfo{
			Code:      "INVALID_PAYLOAD",
			Message:   fmt.Sprintf("failed to parse download request: %v", err),
			Retryable: false,
		},
	}, nil
}

func (w *DownloaderWorker) handleValidationError(req *dto.DownloadRequest, logger ports.Logger, err error) (ports.RuntimeResponse, error) {
	logger.Error("invalid download request", "error", err)
	w.metrics.IncrementCounter("download.validation_error",
		map[string]string{"event_type": req.EventType})

	return ports.RuntimeResponse{
		Success: false,
		Error: &ports.ErrorInfo{
			Code:      "VALIDATION_ERROR",
			Message:   err.Error(),
			Retryable: false,
		},
	}, nil
}

func (w *DownloaderWorker) handleDuplicateEvent(req *dto.DownloadRequest, logger ports.Logger) (ports.RuntimeResponse, error) {
	logger.Info("event already processed, skipping")
	w.metrics.IncrementCounter("download.skipped",
		map[string]string{"reason": "already_processed"})

	return ports.RuntimeResponse{Success: true}, nil
}

func (w *DownloaderWorker) handleProcessingError(req *dto.DownloadRequest, logger ports.Logger, err error) (ports.RuntimeResponse, error) {
	logger.Error("failed to process download", "error", err)
	w.metrics.IncrementCounter("download.failed",
		map[string]string{"event_type": req.EventType})

	// Processing errors should go to DLQ for retry by orchestrator
	return ports.RuntimeResponse{
		Success: false,
		Error: &ports.ErrorInfo{
			Code:      "DOWNLOAD_FAILED",
			Message:   err.Error(),
			Retryable: true, // Let orchestrator handle retry logic
		},
	}, nil
}

func (w *DownloaderWorker) handleSuccess(req *dto.DownloadRequest, logger ports.Logger) (ports.RuntimeResponse, error) {
	w.metrics.IncrementCounter("download.success",
		map[string]string{"event_type": req.EventType})
	logger.Info("download completed successfully")

	return ports.RuntimeResponse{Success: true}, nil
}
