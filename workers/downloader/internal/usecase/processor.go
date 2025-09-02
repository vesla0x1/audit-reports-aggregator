package usecase

import (
	"bytes"
	"context"
	"downloader/internal/domain/model"
	"downloader/internal/domain/service"
	"fmt"
	"shared/application/ports"
	"shared/domain/dto"
	"shared/domain/entity"
	"time"
)

type DownloadProcessor struct {
	downloadService    *service.DownloadService
	storagePathService *service.StoragePathService
	storage            ports.Storage
	repositories       ports.Repositories
	logger             ports.Logger
	metrics            ports.Metrics
}

func NewDownloadProcessor(
	downloadService *service.DownloadService,
	storagePathService *service.StoragePathService,
	storage ports.Storage,
	repositories ports.Repositories,
	logger ports.Logger,
	metrics ports.Metrics,
) *DownloadProcessor {
	return &DownloadProcessor{
		downloadService:    downloadService,
		storagePathService: storagePathService,
		storage:            storage,
		repositories:       repositories,
		logger:             logger,
		metrics:            metrics,
	}
}

func (p *DownloadProcessor) Process(ctx context.Context, req *dto.DownloadRequest) error {
	logger := p.logger.WithFields(map[string]interface{}{
		"download_id": req.DownloadID,
		"event_id":    req.EventID,
	})

	// 1. Query download record from database
	download, err := p.repositories.Download().Get(ctx, req.DownloadID)
	if err != nil {
		logger.Error("failed to get download record", "error", err)
		return fmt.Errorf("failed to get download record: %w", err)
	}

	// 2. Check current status (skip if completed)
	if download.Status == entity.DownloadStatusCompleted {
		logger.Info("download already completed, skipping")
		p.metrics.IncrementCounter("download.skipped",
			map[string]string{"reason": "already_completed"})
		return nil
	}

	// Check if it's been failing too many times
	// TODO: DOWNLOAD ATTEMPT COUNT SHOULD BE AN ENV VAR
	if download.AttemptCount >= 3 {
		logger.Error("download exceeded max attempts", "attempts", download.AttemptCount)
		return fmt.Errorf("download exceeded maximum attempts: %d", download.AttemptCount)
	}

	// 3. Update status to in_progress
	download.Status = entity.DownloadStatusInProgress
	now := time.Now()
	download.StartedAt = &now
	download.AttemptCount++

	if err := p.repositories.Download().Update(ctx, download); err != nil {
		logger.Error("failed to update download status to in_progress", "error", err)
		return fmt.Errorf("failed to update download status: %w", err)
	}

	// 4. Query audit report for metadata
	report, err := p.repositories.AuditReport().Get(ctx, download.ReportID)
	if err != nil {
		logger.Error("failed to get audit report", "error", err, "report_id", download.ReportID)
		return p.markDownloadFailed(ctx, download, "failed to get audit report", err)
	}

	// Get provider for slug
	provider, err := p.repositories.AuditProvider().Get(ctx, report.ProviderID)
	if err != nil {
		logger.Error("failed to get provider", "error", err, "provider_id", report.ProviderID)
		return p.markDownloadFailed(ctx, download, "failed to get provider", err)
	}

	// 5. Download file from URL
	logger.Info("downloading file", "url", report.SourceDownloadURL)
	//downloadStart := time.Now()

	result, err := p.downloadService.Download(ctx, report.SourceDownloadURL)
	if err != nil {
		// Check if it's a domain error with retry information
		if downloadErr, ok := err.(*model.DownloadError); ok {
			logger.Error("download failed",
				"error_type", downloadErr.Type,
				"error", downloadErr.Message,
				"url", downloadErr.URL,
				"retryable", downloadErr.IsRetryable())

			p.metrics.IncrementCounter("download.file.failed",
				map[string]string{"error_type": string(downloadErr.Type)})

			errorMsg := fmt.Sprintf("download failed: %s", downloadErr.Message)
			return p.markDownloadFailed(ctx, download, errorMsg, err)
		}

		return p.markDownloadFailed(ctx, download, "download failed", err)
	}

	// 6. Generate storage path
	storagePath := p.storagePathService.GeneratePath(
		provider.Slug,
		report.ID,
		report.Title,
		result.Extension,
	)

	logger.Info("uploading to storage", "path", storagePath, "size", result.Size)

	// 7. Upload to object storage
	uploadStart := time.Now()

	// Create metadata for the upload
	metadata := ports.ObjectMetadata{
		ContentType:   result.ContentType,
		ContentLength: result.Size,
		UserMetadata: map[string]string{
			"report_id": fmt.Sprintf("%d", report.ID),
			"provider":  provider.Slug,
			"file_hash": result.Hash,
		},
	}

	// Convert byte slice to reader
	reader := bytes.NewReader(result.Content)

	// Upload with bucket and metadata
	if err := p.storage.Put(ctx, "", storagePath, reader, metadata); err != nil {
		logger.Error("failed to upload to storage", "error", err, "path", storagePath)
		return p.markDownloadFailed(ctx, download, "failed to upload to storage", err)
	}

	p.metrics.RecordHistogram("storage.upload.duration",
		time.Since(uploadStart).Seconds(),
		map[string]string{"provider": provider.Slug})

	// 8. Update download record with results
	download.Status = entity.DownloadStatusCompleted
	download.StoragePath = &storagePath
	download.FileHash = &result.Hash
	download.FileExtension = &result.Extension
	now = time.Now()
	download.CompletedAt = &now
	download.ErrorMessage = nil // Clear any previous error

	if err := p.repositories.Download().Update(ctx, download); err != nil {
		logger.Error("failed to update download record", "error", err)
		// This is critical - we uploaded but couldn't record it
		return fmt.Errorf("critical: file uploaded but failed to update record: %w", err)
	}

	// 9. Create process record
	process := &entity.Process{
		DownloadID: download.ID,
		Status:     entity.ProcessStatusPending,
		CreatedAt:  time.Now(),
	}

	if err := p.repositories.Process().Create(ctx, process); err != nil {
		logger.Error("failed to create process record", "error", err)
		// Non-critical - download succeeded but process creation failed
		// Could be handled by a cleanup job
	}

	// 10. TODO: Publish process.requested event
	// This would be done through a message queue client
	// For now, just log it
	logger.Info("download completed successfully, process record created",
		"storage_path", storagePath,
		"file_hash", result.Hash,
		"process_id", process.ID)

	p.metrics.IncrementCounter("download.completed",
		map[string]string{
			"provider":  provider.Slug,
			"extension": result.Extension,
		})

	return nil
}

func (p *DownloadProcessor) markDownloadFailed(ctx context.Context, download *entity.Download, message string, err error) error {
	download.Status = entity.DownloadStatusFailed
	download.ErrorMessage = &message

	// Check if we should retry based on attempt count
	if download.AttemptCount < 3 {
		// Keep it as failed but retryable
		p.logger.Info("download failed but retryable",
			"attempts", download.AttemptCount,
			"error", message)
	} else {
		// Mark as permanently failed
		p.logger.Error("download permanently failed",
			"attempts", download.AttemptCount,
			"error", message)
	}

	if updateErr := p.repositories.Download().Update(ctx, download); updateErr != nil {
		p.logger.Error("failed to update download status to failed", "error", updateErr)
	}

	return fmt.Errorf("%s: %w", message, err)
}
