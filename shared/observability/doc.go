/*
Package observability provides structured logging and metrics collection
for the audit system's distributed workers.

This package implements a clean, production-ready observability layer that
integrates with modern monitoring stacks (Loki for logs, Prometheus for metrics).
It's designed specifically for the audit reports aggregator system's needs while
maintaining flexibility for future extensions.

# Architecture

The package uses a clean, minimalist design with essential features:

	Provider (manages instances)
	    ├── Logger (JSON formatted for Loki)
	    └── Metrics (Prometheus compatible)

# Design Principles

Provider Pattern: Manages singleton instances per component, ensuring
consistent configuration and preventing duplicate metric registrations.
Each component (downloader, processor, uploader) gets its own logger
and metrics instance with appropriate labels.

Dependency Inversion: All components depend on interfaces, not concrete
implementations, enabling easy testing with mocks and future implementation
changes without affecting consumers.

Structured Logging: All logs are JSON-formatted with consistent field
naming, making them easily searchable in Loki. Context values like
trace_id and request_id are automatically extracted.

Semantic Metrics: Metrics follow Prometheus naming conventions with
meaningful labels, enabling powerful queries and alerting rules.

# Package Structure

	observability/
	├── types/
	│   └── interfaces.go   # Core contracts and types
	├── provider.go         # Provider implementation
	├── doc.go             # Package documentation
	├── logger/
	│   ├── loki_logger.go     # Loki-optimized JSON logger
	│   └── loki_logger_test.go # Logger tests
	├── metrics/
	│   ├── prometheus_metrics.go     # Prometheus metrics
	│   └── prometheus_metrics_test.go # Metrics tests
	└── mocks/
	    ├── mock_logger.go   # Logger mock
	    ├── mock_metrics.go  # Metrics mock
	    └── mock_provider.go # Provider mock

# Usage

Initialize the provider once at application startup:

	config := &observability.Config{
	    ServiceName: "audit-system",
	    Environment: "production",
	    LogLevel:    "info",
	    LogOutput:   os.Stdout,  // Or custom io.Writer
	    AdditionalFields: observability.Fields{
	        "version": "1.0.0",
	        "region":  "us-east-1",
	    },
	}

	provider := observability.NewProvider(config)
	defer provider.Close()

	// Each worker gets its components
	logger := provider.Logger("downloader")
	metrics := provider.Metrics("downloader")

	// Use in business logic with context
	ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
	logger.Info(ctx, "Download started", observability.Fields{
	    "url": "https://example.com/report.pdf",
	    "size_bytes": 1024000,
	})

	// Track operation timing
	start := time.Now()
	metrics.StartOperation("download")
	defer func() {
	    metrics.EndOperation("download")
	    metrics.RecordDuration("download", time.Since(start).Seconds())
	}()

	// Record results
	if err != nil {
	    logger.Error(ctx, "Download failed", err, observability.Fields{
	        "url": url,
	        "retry_count": retries,
	    })
	    metrics.RecordError("download", "network_timeout")
	} else {
	    metrics.RecordSuccess("pdf")
	    metrics.RecordFileSize("pdf", fileSize)
	}

# Context Integration

The logger automatically extracts these context values if present:
  - trace_id: Distributed tracing identifier
  - request_id: Request correlation identifier  
  - report_id: Audit report identifier

Example:

	ctx := context.WithValue(ctx, "trace_id", "xyz-789")
	logger.Info(ctx, "Processing", nil)
	// Output includes: {"trace_id": "xyz-789", ...}

# Testing

Use provided mocks for unit testing:

	mockProvider := new(mocks.MockProvider)
	mockLogger := new(mocks.MockLogger)
	mockMetrics := new(mocks.MockMetrics)

	mockProvider.On("Logger", "downloader").Return(mockLogger)
	mockProvider.On("Metrics", "downloader").Return(mockMetrics)
	
	mockLogger.On("Info", ctx, "Download started", mock.Anything).Return()
	mockMetrics.On("StartOperation", "download").Return()
	mockMetrics.On("EndOperation", "download").Return()

# Configuration

Configure through the Config struct:

  - ServiceName: Base service identifier for all metrics and logs
  - Environment: Deployment environment (development/staging/production)
  - LogLevel: Minimum log level (debug/info/warn/error)
  - LogOutput: Where to write logs (defaults to os.Stdout)
  - AdditionalFields: Fields added to all log entries

# Log Levels

  - Debug: Detailed debugging information, disabled in production
  - Info: General operational information
  - Warn: Warning conditions that don't prevent operation
  - Error: Error conditions that indicate failures

# Metrics Details

Pre-configured metrics with appropriate labels:

  - {service}_processed_total: Counter with labels [status, type]
  - {service}_errors_total: Counter with labels [error_type, operation]
  - {service}_duration_seconds: Histogram with label [operation]
  - {service}_file_size_bytes: Histogram with label [file_type]
  - {service}_in_progress: Gauge with label [operation]

# Integration Points

Loki Integration:
  - Logs are JSON formatted with proper fields for indexing
  - Use Promtail, Fluentd, or Fluent Bit to ship logs to Loki
  - Query examples:
    {service="audit-system"} |= "error"
    {component="downloader"} | json | duration > 5

Prometheus Integration:
  - Metrics are registered with the default registry
  - Expose them via promhttp.Handler() on /metrics endpoint
  - Example setup:
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)

# Performance Considerations

  - Loggers and metrics are cached per component (singleton pattern)
  - JSON marshaling is done efficiently with minimal allocations
  - Metrics use pre-allocated label values where possible
  - Thread-safe operations with minimal lock contention

# Thread Safety

All components are thread-safe and can be used concurrently
from multiple goroutines without external synchronization.

# Best Practices

  1. Initialize the provider once at startup
  2. Use defer for closing the provider
  3. Always use defer for EndOperation after StartOperation
  4. Include relevant context in log fields
  5. Use consistent operation names across metrics
  6. Keep error types consistent for effective alerting
  7. Use WithFields for request-scoped loggers

# Future Extensions

The package is designed to easily support:
  - Tracing integration (OpenTelemetry)
  - Dynamic log level changes
  - Metrics aggregation and roll-ups
  - Custom metric types
  - Log sampling for high-volume scenarios
*/
package observability
