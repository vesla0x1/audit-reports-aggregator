package main

import (
	"downloader/internal/domain"
	"downloader/internal/infrastructure/adapters/http"
	"downloader/internal/usecase"
	"log"
	"shared/config"
	"shared/domain/handler"
	"shared/domain/observability"
	"shared/domain/storage"
	infrahandler "shared/infrastructure/handlers"
	infraobs "shared/infrastructure/observability"
	infrastorage "shared/infrastructure/storage"
)

func main() {
	// Load centralized configuration
	cfgProvider := config.GetProvider()
	cfgProvider.MustLoad()
	cfg := cfgProvider.MustGet()

	// Initialize observability
	if err := observability.Initialize(cfg, &infraobs.Factory{}); err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}

	// Get observability for main component
	mainLogger, mainMetrics := observability.MustGetObservability("main")
	mainLogger.Info("Starting application",
		"service", cfg.ServiceName,
		"version", cfg.Version,
		"environment", cfg.Environment)
	mainMetrics.IncrementCounter("application.starts", nil)

	// Initialize storage with its own observability
	storageLogger, storageMetrics := observability.MustGetObservability("storage.s3")
	storageFactory := infrastorage.NewFactoryWithObservability(
		storageLogger,
		storageMetrics,
	)
	if err := storage.Initialize(cfg, storageFactory); err != nil {
		storageLogger.Error("Failed to initialize storage", "error", err)
		storageMetrics.IncrementCounter("init.failures", nil)
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	storageLogger.Info("Storage initialized successfully")
	storageMetrics.IncrementCounter("init.success", nil)

	// Create services and use case
	objectStorage := storage.MustGetStorage()

	// Create HTTP client with its observability
	httpLogger, httpMetrics := observability.MustGetObservability("client.http")
	httpClient := http.NewClientWithConfig(cfg.HTTP).
		WithLogger(httpLogger).
		WithMetrics(httpMetrics)

	// service layer should be pure business logic - no need to use logs and metrics
	downloadService := domain.NewDownloadService(httpClient)

	useCaseLogger, useCaseMetrics := observability.MustGetObservability("usecase.download")
	useCase := usecase.NewDownloaderWorker(
		downloadService,
		objectStorage,
		useCaseLogger,
		useCaseMetrics,
	)

	// Initialize handler with factory and observability
	handlerLogger, handlerMetrics := observability.MustGetObservability("handler.download")
	if err := handler.Initialize(useCase, cfg, &infrahandler.Factory{}); err != nil {
		handlerLogger.Error("Failed to initialize handler", "error", err)
		handlerMetrics.IncrementCounter("init.failures", nil)
		log.Fatalf("Failed to initialize handler: %v", err)
	}

	// Start the application
	handlerLogger.Info("Starting handler")
	handlerMetrics.IncrementCounter("handler.starts", nil)
	if err := handler.Start(); err != nil {
		handlerLogger.Error("Failed to start handler", "error", err)
		handlerMetrics.IncrementCounter("start.failures", nil)
		log.Fatalf("Failed to start: %v", err)
	}
}
