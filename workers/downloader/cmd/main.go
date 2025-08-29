package main

import (
	"downloader/internal/domain"
	"downloader/internal/infrastructure/adapters/http"
	"downloader/internal/usecase"
	"log"
	"shared/config"
	"shared/domain/handler"
	"shared/domain/storage"
	infrahandler "shared/infrastructure/handlers"
	infrastorage "shared/infrastructure/storage"
)

func main() {
	// Load centralized configuration
	cfgProvider := config.GetProvider()
	cfgProvider.MustLoad()
	cfg := cfgProvider.MustGet()

	// Initialize observability
	//obsProvider := observability.GetProvider()
	//if err := obsProvider.Initialize(cfg, &infraobs.Factory{}); err != nil {
	//	log.Fatalf("Failed to initialize observability: %v", err)
	//}

	storageProvider := storage.GetProvider()
	if err := storageProvider.Initialize(cfg, &infrastorage.Factory{}); err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create services and use case
	objectStorage := storageProvider.MustGetStorage()
	httpClient := http.NewClient().WithConfig(cfg.HTTP)
	downloadService := domain.NewDownloadService(httpClient)

	useCase := usecase.NewDownloaderWorker(
		downloadService,
		objectStorage,
		//obsProvider.MustGetLogger(""),
		//obsProvider.MustGetMetrics(),
	)

	// Initialize handler with factory
	handlerProvider := handler.GetProvider()
	handlerFactory := &infrahandler.Factory{}
	if err := handlerProvider.Initialize(useCase, cfg, handlerFactory); err != nil {
		log.Fatalf("Failed to initialize handler: %v", err)
	}

	// Start the application
	if err := handlerProvider.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}
