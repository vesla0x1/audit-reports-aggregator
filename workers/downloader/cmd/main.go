package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "downloader/internal/adapters/http"
	"downloader/internal/service"
	"downloader/internal/worker"

	"shared/handler"
	"shared/handler/platforms"
	"shared/observability"
	"shared/utils"
)

func main() {
	// Initialize observability
	obsConfig := &observability.Config{
		ServiceName: "downloader-worker",
		Environment: utils.GetEnv("ENVIRONMENT", "development"),
		LogLevel:    utils.GetEnv("LOG_LEVEL", "info"),
		LogOutput:   os.Stdout,
	}

	obsProvider := observability.NewProvider(obsConfig)
	logger := obsProvider.Logger("main")
	metrics := obsProvider.Metrics("main")

	// Record service startup
	startTime := time.Now()
	metrics.RecordSuccess("service_start")

	logger.Info(context.Background(), "Starting downloader worker", observability.Fields{
		"service":     obsConfig.ServiceName,
		"environment": obsConfig.Environment,
	})

	// Initialize HTTP client with timeout tracking
	httpTimeout := utils.GetEnvDuration("HTTP_TIMEOUT", "120s")
	httpClient := httpAdapter.NewClient(httpAdapter.ClientConfig{
		Timeout:    httpTimeout,
		MaxRetries: utils.GetEnvInt("HTTP_MAX_RETRIES", 3),
		UserAgent:  utils.GetEnv("HTTP_USER_AGENT", "audit-reports-downloader/1.0"),
	})

	// Initialize download service
	downloadService := service.NewDownloadService(
		httpClient,
		obsProvider.Logger("download-service"),
		obsProvider.Metrics("dowload-service"),
	)

	// Create worker
	downloaderWorker := worker.NewDownloaderWorker(
		downloadService,
		obsProvider.Logger("worker"),
		obsProvider.Metrics("worker"),
	)

	// Create handler factory
	factory := handler.NewFactory(downloaderWorker, obsProvider)

	// Detect platform
	platform := handler.DetectPlatform()
	logger.Info(context.Background(), "Detected platform", observability.Fields{
		"platform":         platform,
		"startup_duration": time.Since(startTime).Seconds(),
	})

	// Record platform type
	metrics.RecordSuccess("platform_" + platform)

	// Create handler with middleware
	h := factory.Create()

	// Setup graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	// Start platform adapter
	switch platform {
	case "openfaas":
		// OpenFaaS mode
		adapter := platforms.NewOpenFaaSAdapter(h)

		// Monitor for shutdown in a goroutine
		go func() {
			<-shutdownChan
			gracefulShutdown(logger, metrics, startTime)
		}()

		// Run adapter (blocking)
		adapter.Run()
	default:
		// HTTP mode
		addr := utils.GetEnv("HTTP_ADDR", ":8080")
		logger.Info(context.Background(), "Starting HTTP server", observability.Fields{
			"address": addr,
			"mode":    "http",
		})

		adapter := platforms.NewHTTPAdapter(h)
		// Start server in a goroutine
		serverErrChan := make(chan error, 1)
		go func() {
			if err := adapter.Serve(addr); err != nil {
				serverErrChan <- err
			}
		}()

		// Wait for shutdown signal or server error
		select {
		case <-shutdownChan:
			gracefulShutdown(logger, metrics, startTime)
		case err := <-serverErrChan:
			metrics.RecordError("service", "server_error")
			logger.Error(context.Background(), "Server error", err, nil)
			log.Fatalf("Server error: %v", err)
		}

		// Graceful shutdown
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			logger.Info(context.Background(), "Shutting down gracefully", nil)
			os.Exit(0)
		}()

		if err := adapter.Serve(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// gracefulShutdown handles the shutdown process with proper metrics
func gracefulShutdown(logger observability.Logger, metrics observability.Metrics, startTime time.Time) {
	ctx := context.Background()

	// Record shutdown initiation
	metrics.RecordSuccess("shutdown_initiated")

	logger.Info(ctx, "Shutting down gracefully", observability.Fields{
		"uptime_seconds": time.Since(startTime).Seconds(),
	})

	// Create a timeout context for shutdown operations
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Perform cleanup operations here
	// For example: close database connections, flush logs, etc.

	// Record final metrics
	metrics.RecordDuration("service_uptime", time.Since(startTime).Seconds())
	metrics.RecordSuccess("shutdown_complete")

	logger.Info(shutdownCtx, "Shutdown complete", nil)

	// Give time for final metrics to be sent
	time.Sleep(2 * time.Second)

	os.Exit(0)
}
