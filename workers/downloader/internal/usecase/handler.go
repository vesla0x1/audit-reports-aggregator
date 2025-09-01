package usecase

import (
	"context"

	"shared/application/ports"
)

type DownloaderWorker struct {
	//downloadService .DownloadService
	storage      ports.Storage
	repositories ports.Repositories
	logger       ports.Logger
	metrics      ports.Metrics
}

func NewDownloaderWorker(
	//downloadService ports.DownloadService,
	storage ports.Storage,
	repositories ports.Repositories,
	obs ports.Observability,
) *DownloaderWorker {
	logger, metrics, err := obs.ComponentsScoped("worker.downloader")
	if err != nil {
		// TODO: handle
	}
	return &DownloaderWorker{
		//downloadService: downloadService,
		storage:      storage,
		repositories: repositories,
		logger:       logger,
		metrics:      metrics,
	}
}

func (w *DownloaderWorker) Handle(ctx context.Context, request ports.RuntimeRequest) (ports.RuntimeResponse, error) {
	return ports.RuntimeResponse{}, nil
}
