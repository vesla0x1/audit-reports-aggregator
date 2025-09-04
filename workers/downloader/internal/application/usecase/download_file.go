package usecase

import (
	"bytes"
	"context"
	"downloader/internal/application/dto"
	"downloader/internal/application/ports"
	downloadPkg "downloader/internal/domain/entity/download"
	"downloader/internal/domain/entity/process"
	"downloader/internal/domain/service"
	"fmt"
	"shared/infrastructure/config"
	"time"
)

type DownloadFile struct {
	downloadService *service.DownloadService
	storage         ports.Storage
	queue           ports.Queue
	queueNames      config.QueueNames
	repositories    ports.Repositories
	logger          ports.Logger
	metrics         ports.Metrics
}

func NewDownloadFile(
	downloadService *service.DownloadService,
	storage ports.Storage,
	queue ports.Queue,
	queueNames config.QueueNames,
	repositories ports.Repositories,
	obs ports.Observability,
) (*DownloadFile, error) {
	logger, metrics, _ := obs.ComponentsScoped("usecase.download_file")
	return &DownloadFile{
		downloadService: downloadService,
		storage:         storage,
		queue:           queue,
		queueNames:      queueNames,
		repositories:    repositories,
		logger:          logger,
		metrics:         metrics,
	}, nil
}

func (p *DownloadFile) Download(ctx context.Context, req *dto.DownloadRequest) error {
	// 1. Query download record from database
	download, err := p.repositories.Download().Get(ctx, req.DownloadID)
	if err != nil {
		return ErrDownloadFileAuditReportNotFound(err)
	}

	// 2. Check current status (skip if completed)
	if download.IsCompleted() {
		return downloadPkg.ErrAlreadyCompleted
	}

	// 3. Check if can start (business rule in entity)
	if !download.CanStart() {
		if download.HasExceededMaxAttempts() {
			return downloadPkg.ErrMaxAttemptsExceeded
		}
		return downloadPkg.ErrInvalidStateTransition
	}

	// 3. Update status to in_progress ("lock" the register so no other worker will be able to get it)
	download.Start()
	if err := p.repositories.Download().Update(ctx, download); err != nil {
		return ErrDownloadFileUpdateFailed(err)
	}

	// 4. Query audit report for metadata
	report, err := p.repositories.AuditReport().Get(ctx, download.ReportID)
	if err != nil {
		return p.commitDownloadFailWithError(ctx, download, ErrDownloadFileAuditReportNotFound(err))
	}

	// Get provider for slug
	provider, err := p.repositories.AuditProvider().Get(ctx, report.ProviderID)
	if err != nil {
		return p.commitDownloadFailWithError(ctx, download, ErrAuditProviderNotFound(err))
	}

	// 5. Download file from URL
	result, err := p.downloadService.Download(ctx, report.SourceDownloadURL)
	if err != nil {
		return p.commitDownloadFailWithError(ctx, download, ErrDownloadFileDownloadFailed(err))
	}

	// 6. Generate storage path
	storagePath := report.StoragePath(provider, result.Extension())

	// Create metadata for the upload
	metadata := ports.ObjectMetadata{
		ContentType:   result.ContentType(),
		ContentLength: result.Size(),
		UserMetadata: map[string]string{
			"report_id": fmt.Sprintf("%d", report.ID),
			"provider":  provider.Slug,
			"file_hash": result.Hash(),
		},
	}

	// Convert byte slice to reader
	resultContent := bytes.NewReader(result.Content())
	// Upload with bucket and metadata
	if err := p.storage.Put(ctx, "", storagePath, resultContent, metadata); err != nil {
		return p.commitDownloadFailWithError(ctx, download, ErrDownloadFileUploadFailed(err))
	}

	// 8. Update download record with results
	if err := download.Complete(storagePath, result.Hash(), result.Extension()); err != nil {
		// This should never happen, so we don't need a custom error for it
		return err
	}

	if err := p.repositories.Download().Update(ctx, download); err != nil {
		return ErrDownloadFileUpdateFailed(err)
	}

	// 9. Create process record
	if err := p.createProcessAndPublishEvent(ctx, download.ID); err != nil {
		return err
	}

	return nil
}

func (d *DownloadFile) commitDownloadFailWithError(
	ctx context.Context,
	download *downloadPkg.Download,
	err error,
) error {
	if err := download.Fail(err.Error()); err != nil {
		return err
	}

	if err := d.repositories.Download().Update(ctx, download); err != nil {
		return ErrDownloadFileUpdateFailed(err)
	}
	return err
}

func (p *DownloadFile) createProcessAndPublishEvent(ctx context.Context, downloadID int64) error {
	// Create process record
	proc := process.NewProcess(downloadID)
	if err := p.repositories.Process().Create(ctx, proc); err != nil {
		return ErrProcessCreationFailed(err)
	}

	// Publish event
	event := &dto.ProcessRequest{
		EventID:   fmt.Sprintf("process-%d-%d", proc.ID, time.Now().Unix()),
		EventType: "process.requested",
		ProcessID: proc.ID,
		Timestamp: time.Now(),
	}

	message := &ports.QueueMessage{
		Target: p.queueNames.Processor,
		Body:   event,
	}

	if err := p.queue.Publish(ctx, message); err != nil {
		return ErrPublishProcessEvent(err)
	}

	return nil
}
