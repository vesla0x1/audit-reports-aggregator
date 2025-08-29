package usecase

import (
	"context"
	"fmt"
	"io"

	"downloader/internal/domain"
	"shared/domain/handler"
	"shared/domain/storage"
)

type DownloaderWorker struct {
	downloadService *domain.DownloadService
	objectStorage   storage.ObjectStorage
	//logger          observability.Logger
	//metrics         observability.Metrics
}

func NewDownloaderWorker(
	downloadService *domain.DownloadService,
	objectStorage storage.ObjectStorage,
	//logger observability.Logger,
	//metrics observability.Metrics,
) *DownloaderWorker {
	return &DownloaderWorker{
		downloadService: downloadService,
		objectStorage:   objectStorage,
		//logger:          logger,
		//metrics:         metrics,
	}
}

func (w *DownloaderWorker) Name() string {
	return "downloader"
}

func (w *DownloaderWorker) Run(ctx context.Context, request handler.Request) (handler.Response, error) {
	/*startTime := time.Now()
	w.logger.Info("Processing download request", map[string]interface{}{
		"request_id":   request.ID,
		"request_type": request.Type,
	})*/

	// Parse request
	var downloadReq domain.DownloadRequest
	if err := request.Unmarshal(&downloadReq); err != nil {
		/*w.metrics.IncrementCounter("worker.errors", map[string]string{
			"error_type": "invalid_payload",
		})*/

		fmt.Println("ops", err)
		return handler.Response{
			Success: false,
			Error: &handler.ErrorInfo{
				Code:      "INVALID_PAYLOAD",
				Message:   "Failed to parse download request",
				Retryable: false,
			},
		}, nil
	}

	if downloadReq.ID == "" {
		downloadReq.ID = request.ID
	}

	// Execute download
	result, reader, err := w.downloadService.Execute(ctx, downloadReq)
	if err != nil {
		/*errorType := w.categorizeError(err)
		w.metrics.IncrementCounter("worker.errors", map[string]string{
			"error_type": errorType,
		})

		w.logger.Error("Download failed", map[string]interface{}{
			"request_id": request.ID,
			"error":      err.Error(),
			"error_type": errorType,
		})*/

		if domainErr, ok := err.(*domain.DomainError); ok {
			return handler.Response{
				Success: false,
				Error: &handler.ErrorInfo{
					Code:      domainErr.Code,
					Message:   domainErr.Message,
					Retryable: domainErr.Retryable,
				},
			}, nil
		}

		return handler.Response{
			Success: false,
			Error: &handler.ErrorInfo{
				Code:      "PROCESSING_ERROR",
				Message:   "Failed to process download request",
				Retryable: true,
			},
		}, nil
	}

	// Process content
	if reader != nil {
		defer reader.Close()

		_, err := w.processContent(reader)
		if err != nil {
			/*w.metrics.IncrementCounter("worker.errors", map[string]string{
				"error_type": "content_read",
			})*/

			return handler.Response{
				Success: false,
				Error: &handler.ErrorInfo{
					Code:      "READ_ERROR",
					Message:   "Failed to read downloaded content",
					Retryable: true,
				},
			}, nil
		}

		/*w.metrics.RecordHistogram("worker.bytes_processed", float64(bytesRead), map[string]string{
			"worker": w.Name(),
		})*/
	}

	// Record success metrics
	/*duration := time.Since(startTime).Seconds()
	w.metrics.RecordHistogram("worker.duration", duration, map[string]string{
		"worker": w.Name(),
	})
	w.metrics.IncrementCounter("worker.success", map[string]string{
		"worker": w.Name(),
	})

	w.logger.Info("Request processed successfully", map[string]interface{}{
		"request_id": request.ID,
		"duration":   duration,
	})*/

	return handler.NewSuccessResponse(request.ID, result)
}

func (w *DownloaderWorker) processContent(reader io.Reader) (int64, error) {
	metadata := storage.ObjectMetadata{ContentType: "application/octet-stream"}
	err := w.objectStorage.Put(context.Background(), "audit-reports-local-reports", "cantina/cantina-report.pdf", reader, metadata)
	return 0, err
}
