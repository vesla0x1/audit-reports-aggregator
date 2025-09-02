package usecase

import (
	"context"
	"downloader/internal/domain/service"
	"fmt"
	"time"

	"shared/application/ports"
	"shared/domain/dto"
	"shared/infrastructure/config"
)

type DownloaderWorker struct {
	storage      ports.Storage
	queue        ports.Queue
	queueNames   config.QueueNames
	repositories ports.Repositories
	logger       ports.Logger
	metrics      ports.Metrics
	validator    *RequestValidator
	processor    *DownloadProcessor
}

func NewDownloaderWorker(
	downloadService *service.DownloadService,
	storagePathService *service.StoragePathService,
	storage ports.Storage,
	queue ports.Queue,
	queueNames config.QueueNames,
	repositories ports.Repositories,
	obs ports.Observability,
) *DownloaderWorker {
	logger, metrics, err := obs.ComponentsScoped("worker.downloader")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize observability: %v", err))
	}

	return &DownloaderWorker{
		storage:      storage,
		queue:        queue,
		queueNames:   queueNames,
		repositories: repositories,
		logger:       logger,
		metrics:      metrics,
		validator:    NewRequestValidator(),
		processor: NewDownloadProcessor(
			downloadService,
			storagePathService,
			storage,
			queue,
			queueNames,
			repositories,
			logger,
			metrics,
		),
	}
}

func (w *DownloaderWorker) Handle(ctx context.Context, request ports.RuntimeRequest) (ports.RuntimeResponse, error) {
	startTime := time.Now()
	defer w.recordDuration(startTime)

	// Parse request
	downloadReq, err := w.parseRequest(request)
	if err != nil {
		return w.handleParseError(request.ID, err)
	}

	// Create scoped logger
	logger := w.createRequestLogger(downloadReq)
	logger.Info("processing download request",
		"request_timestamp", downloadReq.Timestamp,
		"message_age_seconds", time.Since(downloadReq.Timestamp).Seconds())

	// Validate
	if err := w.validator.Validate(downloadReq); err != nil {
		return w.handleValidationError(downloadReq, logger, err)
	}

	// Process
	if err := w.processor.Process(ctx, downloadReq); err != nil {
		return w.handleProcessingError(downloadReq, logger, err)
	}

	return w.handleSuccess(downloadReq, logger)
}

func (w *DownloaderWorker) parseRequest(request ports.RuntimeRequest) (*dto.DownloadRequest, error) {
	var downloadReq dto.DownloadRequest
	if err := request.Unmarshal(&downloadReq); err != nil {
		w.logger.Error("failed to unmarshal download request",
			"error", err,
			"request_id", request.ID)
		w.metrics.IncrementCounter("download.error",
			map[string]string{"error_type": "unmarshal_failed"})
		return nil, err
	}
	return &downloadReq, nil
}

func (w *DownloaderWorker) createRequestLogger(req *dto.DownloadRequest) ports.Logger {
	return w.logger.WithFields(map[string]interface{}{
		"event_id":    req.EventID,
		"download_id": req.DownloadID,
		"event_type":  req.EventType,
	})
}

func (w *DownloaderWorker) recordDuration(startTime time.Time) {
	w.metrics.RecordHistogram("download.duration",
		time.Since(startTime).Seconds(),
		map[string]string{"operation": "handle"})
}
