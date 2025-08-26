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

	"shared/config"
	"shared/handler"
	"shared/handler/platforms"
	"shared/observability"
)

func main() {
	// Load centralized configuration
	cfgProvider := config.GetProvider()
	cfgProvider.MustLoad()
	cfg := cfgProvider.MustGet()

	// Initialize observability
	obsConfig := &observability.Config{
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
		LogLevel:    cfg.LogLevel,
		LogOutput:   os.Stdout,
	}

	obsProvider := observability.NewProvider(obsConfig)
	logger := obsProvider.Logger("main")
	metrics := obsProvider.Metrics("main")

	// Record service startup
	startTime := time.Now()
	metrics.RecordSuccess("service_start")

	logger.Info(context.Background(), "Starting downloader worker", observability.Fields{
		"service":     cfg.ServiceName,
		"environment": cfg.Environment,
	})

	// Initialize HTTP client with defaults, then override with config
	httpClient := httpAdapter.NewClient()
	if cfg != nil {
		httpClient.WithConfig(cfg.HTTP)
	}

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

	// Create handler factory with defaults
	factory := handler.NewFactory(downloaderWorker, obsProvider)

	// Override with centralized config if available
	if cfg != nil {
		factory.WithHandlerConfig(cfg.Handler).WithRetryConfig(cfg.Retry)
	}

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
	case "lambda":
		// Lambda mode
		logger.Info(context.Background(), "Starting Lambda runtime", observability.Fields{
			"mode":          "lambda",
			"function_name": os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
			"memory_size":   os.Getenv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE"),
		})

		// Configure Lambda adapter
		lambdaConfig := &platforms.LambdaConfig{
			MaxConcurrency:            cfg.Lambda.MaxConcurrency,
			ProcessingTimeout:         cfg.Lambda.ProcessingTimeout,
			EnablePartialBatchFailure: cfg.Lambda.EnablePartialBatchFailure,
			AutoBase64Decode:          cfg.Lambda.AutoBase64Decode,
		}

		// Create and start Lambda adapter
		adapter := platforms.NewLambdaAdapter(h, lambdaConfig)
		adapter.Start() // This blocks until Lambda runtime stops
	default:
		// HTTP mode
		logger.Info(context.Background(), "Starting HTTP server", observability.Fields{
			"address": cfg.HTTP.Addr,
			"mode":    "http",
		})

		adapter := platforms.NewHTTPAdapter(h)
		// Start server in a goroutine
		serverErrChan := make(chan error, 1)
		go func() {
			if err := adapter.Serve(cfg.HTTP.Addr); err != nil {
				serverErrChan <- err
			}
		}()

		// Wait for shutdown signal or server error
		select {
		case <-shutdownChan:
			handler.GracefulShutdown(logger, metrics, startTime)
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

		if err := adapter.Serve(cfg.HTTP.Addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
