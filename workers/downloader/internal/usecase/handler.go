package usecase

import (
	"context"
	"io"
	"time"

	"downloader/internal/domain"
	"shared/domain/handler"
	"shared/domain/model"
	"shared/domain/observability"
	"shared/domain/repository"
	"shared/domain/storage"
)

type DownloaderWorker struct {
	downloadService *domain.DownloadService
	objectStorage   storage.ObjectStorage
	repositories    *repository.Repositories
	logger          observability.Logger
	metrics         observability.Metrics
}

func NewDownloaderWorker(
	downloadService *domain.DownloadService,
	objectStorage storage.ObjectStorage,
	repositories *repository.Repositories,
	logger observability.Logger,
	metrics observability.Metrics,
) *DownloaderWorker {
	return &DownloaderWorker{
		downloadService: downloadService,
		objectStorage:   objectStorage,
		repositories:    repositories,
		logger:          logger,
		metrics:         metrics,
	}
}

func (w *DownloaderWorker) Name() string {
	return "downloader"
}

func (w *DownloaderWorker) Run(ctx context.Context, request handler.Request) (handler.Response, error) {
	startTime := time.Now()

	w.logger.Info("Processing download request",
		"request_id", request.ID,
		"request_type", request.Type,
		"source", request.Source)

	w.metrics.IncrementCounter("worker.requests", map[string]string{
		"worker": w.Name(),
		"type":   request.Type,
	})

	auditReportModel, err := model.NewAuditReport(1, "test", model.AuditReportTypeCompetition, "foo")
	if err != nil {
		w.logger.Error("Download execution failed",
			"error", err,
			"request_id", request.ID,
		)

		return handler.Response{
			Success: false,
			Error: &handler.ErrorInfo{
				Code:      "",
				Message:   "",
				Retryable: false,
			},
		}, nil
	}

	if err := w.repositories.AuditReport.Create(ctx, auditReportModel); err != nil {
		w.logger.Error("Download execution failed",
			"error", err,
			"request_id", request.ID,
		)

		return handler.Response{
			Success: false,
			Error: &handler.ErrorInfo{
				Code:      "",
				Message:   "",
				Retryable: false,
			},
		}, nil
	}

	w.logger.Info("audit report inserted!")

	// Parse request
	var downloadReq domain.DownloadRequest
	if err := request.Unmarshal(&downloadReq); err != nil {
		w.logger.Error("Failed to parse download request",
			"error", err,
			"request_id", request.ID)

		w.metrics.IncrementCounter("worker.errors", map[string]string{
			"worker":     w.Name(),
			"error_type": "invalid_payload",
		})

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

	w.logger.Info("Download request parsed",
		"request_id", downloadReq.ID,
		"url", downloadReq.URL,
		"has_storage_path", downloadReq.StoragePath != "")

	// Execute download
	result, reader, err := w.downloadService.Execute(ctx, downloadReq)
	if err != nil {
		errorType := w.categorizeError(err)

		w.logger.Error("Download execution failed",
			"error", err,
			"request_id", request.ID,
			"url", downloadReq.URL,
			"error_type", errorType)

		w.metrics.IncrementCounter("worker.errors", map[string]string{
			"worker":     w.Name(),
			"error_type": errorType,
		})

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

		storageStart := time.Now()
		bytesProcessed, err := w.processContent(reader)
		storageDuration := time.Since(storageStart)
		if err != nil {
			w.logger.Error("Failed to store downloaded content",
				"error", err,
				"request_id", request.ID,
				"storage_path", result.StoragePath)

			w.metrics.IncrementCounter("worker.errors", map[string]string{
				"worker":     w.Name(),
				"error_type": "storage_error",
			})

			return handler.Response{
				Success: false,
				Error: &handler.ErrorInfo{
					Code:      "READ_ERROR",
					Message:   "Failed to read downloaded content",
					Retryable: true,
				},
			}, nil
		}

		w.logger.Info("Content stored successfully",
			"request_id", request.ID,
			"bytes", bytesProcessed,
			"storage_duration_ms", storageDuration.Milliseconds())

		w.metrics.RecordHistogram("worker.bytes_processed", float64(bytesProcessed), map[string]string{
			"worker": w.Name(),
		})
		w.metrics.RecordHistogram("worker.storage_duration", float64(storageDuration.Milliseconds()), nil)

	}

	// Record success metrics
	duration := time.Since(startTime)

	w.logger.Info("Request processed successfully",
		"request_id", request.ID,
		"url", downloadReq.URL,
		"storage_path", result.StoragePath,
		"duration_ms", duration.Milliseconds(),
		"file_size", result.Size,
		"content_type", result.ContentType)

	w.metrics.RecordHistogram("worker.duration", float64(duration.Milliseconds()), map[string]string{
		"worker": w.Name(),
		"status": "success",
	})
	w.metrics.IncrementCounter("worker.success", map[string]string{
		"worker": w.Name(),
	})

	return handler.NewSuccessResponse(request.ID, result)
}

func (w *DownloaderWorker) processContent(reader io.Reader) (int64, error) {
	metadata := storage.ObjectMetadata{ContentType: "application/octet-stream"}
	err := w.objectStorage.Put(context.Background(), "audit-reports-local", "cantina/cantina-report.pdf", reader, metadata)
	return 0, err
}

func (w *DownloaderWorker) categorizeError(err error) string {
	if err == nil {
		return "none"
	}

	if domainErr, ok := err.(*domain.DomainError); ok {
		return domainErr.Code
	}

	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection"):
		return "connection_error"
	case contains(errStr, "invalid"):
		return "validation_error"
	case contains(errStr, "not found"):
		return "not_found"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && s[:len(substr)] == substr ||
			len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
			len(substr) > 0 && len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
