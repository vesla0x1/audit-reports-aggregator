package main

import (
	"context"
	"fmt"
	"log"
	"shared/application/ports"
	"time"

	// Domain layer
	"downloader/internal/domain/service"
	"downloader/internal/usecase"

	// Infrastructure layer
	"shared/infrastructure/config"
	"shared/infrastructure/database"
	"shared/infrastructure/http"
	"shared/infrastructure/observability"
	"shared/infrastructure/queue"
	"shared/infrastructure/repository"
	"shared/infrastructure/runtime"
	"shared/infrastructure/storage"
)

func main() {
	cfg := loadConfiguration()
	obs := initializeObservability(cfg)
	deps := initializeDependencies(cfg, obs)
	initializeApplication(cfg, deps, obs)
}

// Dependencies holds all initialized infrastructure components
type Dependencies struct {
	storage      ports.Storage
	database     ports.Database
	httpClient   ports.HTTPClient
	repositories ports.Repositories
	queue        ports.Queue
}

// loadConfiguration loads and validates the application configuration
func loadConfiguration() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	return cfg
}

// initializeDependencies sets up all infrastructure dependencies
func initializeDependencies(cfg *config.Config, obs ports.Observability) *Dependencies {
	// Database initialization - component handles its own logging
	db, err := database.CreateDB(cfg, obs)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Storage initialization - component handles its own logging
	storageClient, err := storage.CreateStorage(cfg, obs)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// HTTP client - component handles its own logging
	httpClient, err := http.CreateHTTPClient(cfg.HTTP, obs)
	if err != nil {
		log.Fatalf("Failed to create http client: %v", err)
	}

	// Repositories - component handles its own logging
	repositories, err := repository.NewRepositories(db, obs)
	if err != nil {
		log.Fatalf("Failed to create http client: %v", err)
	}

	// Queue initialization - optional component
	var publisher ports.Queue
	if cfg.Adapters.Queue != "" {
		publisher, err = queue.CreateQueue(cfg, obs)
		if err != nil {
			log.Fatalf("Failed to create queue: %v", err)
		}
	}

	return &Dependencies{
		storage:      storageClient,
		database:     db,
		httpClient:   httpClient,
		repositories: repositories,
		queue:        publisher,
	}
}

// initializeObservability sets up logging and metrics infrastructure
func initializeObservability(cfg *config.Config) ports.Observability {
	obs, err := observability.CreateObservability(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	return obs
}

// buildApplication assembles the application layers
func buildApplication(cfg *config.Config, deps *Dependencies, obs ports.Observability) (ports.Runtime, error) {
	// Create use case
	handler := usecase.NewDownloaderWorker(
		service.NewDownloadService(deps.httpClient),
		service.NewStoragePathService(),
		deps.storage,
		deps.queue,
		cfg.Queue.Queues,
		deps.repositories,
		obs,
	)

	// Create runtime
	runtime, err := runtime.Create(cfg, handler, obs)
	if err != nil {
		return nil, fmt.Errorf("runtime creation: %w", err)
	}

	return runtime, nil
}

func initializeApplication(cfg *config.Config, deps *Dependencies, obs ports.Observability) {
	app, err := buildApplication(cfg, deps, obs)
	if err != nil {
		log.Fatalf("error building the application: %w", err)
	}
	app.Start()
}
