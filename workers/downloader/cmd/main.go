package main

import (
	"context"
	"downloader/internal/domain"
	"downloader/internal/infrastructure/adapters/http"
	"downloader/internal/usecase"
	"log"
	"shared/config"
	"shared/domain/database"
	"shared/domain/handler"
	"shared/domain/observability"
	"shared/domain/storage"
	infradatabase "shared/infrastructure/database"
	infrahandler "shared/infrastructure/handlers"
	infraobs "shared/infrastructure/observability"
	infrastorage "shared/infrastructure/storage"
	"time"
)

func main() {
	cfg := loadConfiguration()

	deps := initializeDependencies(cfg)

	app := buildApplication(cfg, deps)

	startApplication(app)
}

// Dependencies holds all initialized infrastructure components
type Dependencies struct {
	storage    storage.ObjectStorage
	database   database.Database
	httpClient *http.Client
	logger     observability.Logger
	metrics    observability.Metrics
}

// Application holds the complete application stack
type Application struct {
	handler handler.Handler
	logger  observability.Logger
	metrics observability.Metrics
}

// loadConfiguration loads and validates the application configuration
func loadConfiguration() *config.Config {
	config.MustLoad()
	return config.MustGet()
}

// initializeDependencies sets up all infrastructure dependencies
func initializeDependencies(cfg *config.Config) *Dependencies {
	initializeObservability(cfg)

	logStartup(cfg)

	db := initializeDatabase(cfg)
	storageClient := initializeStorage(cfg)
	httpClient := createHTTPClient(cfg)

	logger, metrics := observability.MustGetObservability("app")

	return &Dependencies{
		storage:    storageClient,
		database:   db,
		httpClient: httpClient,
		logger:     logger,
		metrics:    metrics,
	}
}

// initializeObservability sets up logging and metrics infrastructure
func initializeObservability(cfg *config.Config) {
	if err := observability.Initialize(cfg, &infraobs.Factory{}); err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
}

// logStartup logs application startup information
func logStartup(cfg *config.Config) {
	logger, metrics := observability.MustGetObservability("main")

	logger.Info("Starting application",
		"service", cfg.ServiceName,
		"version", cfg.Version,
		"environment", cfg.Environment)

	metrics.IncrementCounter("application.starts", nil)
}

// initializeDatabase sets up the database connection
func initializeDatabase(cfg *config.Config) database.Database {
	logger, metrics := observability.MustGetObservability("database")
	logger.Info("Initializing database", "adapter", cfg.Adapters.Database)

	factory := infradatabase.NewFactory(logger, metrics)
	if err := database.Initialize(cfg, factory); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		metrics.IncrementCounter("init.failures", map[string]string{"component": "database"})
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Test connection
	db := database.MustGetDatabase()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		logger.Error("Database ping failed", "error", err)
		log.Fatalf("Failed to ping database: %v", err)
	}

	logger.Info("Database initialized successfully")
	metrics.IncrementCounter("init.success", map[string]string{"component": "database"})

	return db
}

// initializeStorage sets up the storage provider with observability
func initializeStorage(cfg *config.Config) storage.ObjectStorage {
	logger, metrics := observability.MustGetObservability("storage.s3")
	logger.Info("calling storage creation")

	factory := infrastorage.NewFactory(logger, metrics)
	if err := storage.Initialize(cfg, factory); err != nil {
		logger.Error("Failed to initialize storage", "error", err)
		metrics.IncrementCounter("init.failures", nil)
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	logger.Info("Storage initialized successfully")
	metrics.IncrementCounter("init.success", nil)

	return storage.MustGetStorage()
}

// createHTTPClient creates an HTTP client with observability
func createHTTPClient(cfg *config.Config) *http.Client {
	logger, metrics := observability.MustGetObservability("client.http")

	return http.NewClientWithConfig(cfg.HTTP).
		WithLogger(logger).
		WithMetrics(metrics)
}

// buildApplication assembles the application layers
func buildApplication(cfg *config.Config, deps *Dependencies) *Application {
	useCase := createUseCase(deps)

	initializeHandler(useCase, cfg)

	logger, metrics := observability.MustGetObservability("handler.download")

	return &Application{
		handler: handler.MustGetHandler(),
		logger:  logger,
		metrics: metrics,
	}
}

// createUseCase builds the business logic layer
func createUseCase(deps *Dependencies) handler.UseCase {
	downloadService := domain.NewDownloadService(deps.httpClient)

	logger, metrics := observability.MustGetObservability("usecase.download")

	return usecase.NewDownloaderWorker(
		downloadService,
		deps.storage,
		logger,
		metrics,
	)
}

// initializeHandler sets up the request handler
func initializeHandler(useCase handler.UseCase, cfg *config.Config) {
	logger, metrics := observability.MustGetObservability("handler.download")

	factory := infrahandler.NewFactory(logger, metrics)
	if err := handler.Initialize(useCase, cfg, factory); err != nil {
		logger.Error("Failed to initialize handler", "error", err)
		metrics.IncrementCounter("init.failures", nil)
		log.Fatalf("Failed to initialize handler: %v", err)
	}
}

// startApplication starts the application and begins processing
func startApplication(app *Application) {
	app.logger.Info("Starting handler")
	app.metrics.IncrementCounter("handler.starts", nil)

	if err := handler.Start(); err != nil {
		app.logger.Error("Failed to start handler", "error", err)
		app.metrics.IncrementCounter("start.failures", nil)
		log.Fatalf("Failed to start: %v", err)
	}
}
