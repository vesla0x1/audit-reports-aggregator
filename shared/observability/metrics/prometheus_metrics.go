// Package metrics provides Prometheus-compatible metrics collection
// for monitoring and alerting in the audit reports aggregator system.
// It follows Prometheus naming conventions and best practices.
package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics implements the Metrics interface using Prometheus client library.
// It provides pre-configured metrics for tracking operations, errors, durations,
// file sizes, and concurrent operations. All metrics follow Prometheus naming
// conventions with the service name as a prefix.
type PrometheusMetrics struct {
	// mu provides thread-safe access (currently unused but kept for future extensions)
	mu          sync.RWMutex
	// serviceName is used as a prefix for all metric names
	serviceName string

	// Pre-defined metrics following Prometheus conventions:
	
	// processedTotal tracks the total number of processed items by status and type
	processedTotal  *prometheus.CounterVec
	// errorsTotal tracks the total number of errors by error type and operation
	errorsTotal     *prometheus.CounterVec
	// durationSeconds tracks operation duration using a histogram with default buckets
	durationSeconds *prometheus.HistogramVec
	// fileSizeBytes tracks file sizes using a histogram with exponential buckets
	fileSizeBytes   *prometheus.HistogramVec
	// inProgress tracks the number of operations currently in progress
	inProgress      *prometheus.GaugeVec
}

// New creates a new PrometheusMetrics instance with pre-configured metrics.
// All metrics are automatically registered with the default Prometheus registry.
// The service name is used as a prefix for all metric names to ensure uniqueness.
//
// Pre-configured metrics:
//   - {serviceName}_processed_total: Counter for successful and failed operations
//   - {serviceName}_errors_total: Counter for errors by type and operation
//   - {serviceName}_duration_seconds: Histogram for operation durations
//   - {serviceName}_file_size_bytes: Histogram for file sizes with exponential buckets
//   - {serviceName}_in_progress: Gauge for concurrent operations
//
// Parameters:
//   - serviceName: Name to prefix all metrics (e.g., "audit_processor")
//
// Returns:
//   - A configured PrometheusMetrics instance with registered metrics
//
// Panics:
//   - If metrics registration fails (e.g., duplicate metric names)
func New(serviceName string) *PrometheusMetrics {
	m := &PrometheusMetrics{
		serviceName: serviceName,
	}

	// Initialize metrics with proper naming and labels
	
	// processedTotal counts items by status (success/error) and operation type
	m.processedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_processed_total", serviceName),
			Help: fmt.Sprintf("Total processed items by %s", serviceName),
		},
		[]string{"status", "type"},
	)

	// errorsTotal provides detailed error tracking by type and operation
	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_errors_total", serviceName),
			Help: fmt.Sprintf("Total errors in %s", serviceName),
		},
		[]string{"error_type", "operation"},
	)

	// durationSeconds tracks operation latency with default exponential buckets
	// Default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 seconds
	m.durationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_duration_seconds", serviceName),
			Help:    fmt.Sprintf("Operation duration in %s", serviceName),
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// fileSizeBytes tracks file sizes with exponential buckets suitable for file operations
	// Buckets: 1KB, 10KB, 100KB, 1MB, 10MB, 100MB, 1GB
	m.fileSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: fmt.Sprintf("%s_file_size_bytes", serviceName),
			Help: fmt.Sprintf("File sizes processed by %s", serviceName),
			Buckets: []float64{
				1024,        // 1KB
				10240,       // 10KB
				102400,      // 100KB
				1048576,     // 1MB
				10485760,    // 10MB
				104857600,   // 100MB
				1073741824,  // 1GB
			},
		},
		[]string{"file_type"},
	)

	// inProgress tracks concurrent operations for capacity planning and bottleneck detection
	m.inProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_in_progress", serviceName),
			Help: fmt.Sprintf("Operations in progress in %s", serviceName),
		},
		[]string{"operation"},
	)

	// Register all metrics with the default Prometheus registry
	// MustRegister panics if registration fails (e.g., duplicate names)
	prometheus.MustRegister(
		m.processedTotal,
		m.errorsTotal,
		m.durationSeconds,
		m.fileSizeBytes,
		m.inProgress,
	)

	return m
}

// RecordSuccess increments the success counter for a specific operation type.
// This updates the {serviceName}_processed_total metric with status="success".
//
// Parameters:
//   - operationType: The type of operation that succeeded (e.g., "parse", "upload", "validate")
//
// Example:
//
//	metrics.RecordSuccess("parse_json")
func (m *PrometheusMetrics) RecordSuccess(operationType string) {
	m.processedTotal.WithLabelValues("success", operationType).Inc()
}

// RecordError increments both the processed counter (with status="error") and
// the detailed error counter. This provides both high-level failure rates
// and detailed error breakdowns for debugging.
//
// Parameters:
//   - operationType: The operation that failed (e.g., "parse", "upload")
//   - errorType: The category of error (e.g., "timeout", "validation", "network")
//
// Example:
//
//	metrics.RecordError("upload_s3", "network_timeout")
func (m *PrometheusMetrics) RecordError(operationType string, errorType string) {
	m.processedTotal.WithLabelValues("error", operationType).Inc()
	m.errorsTotal.WithLabelValues(errorType, operationType).Inc()
}

// RecordDuration records the duration of an operation in seconds.
// The duration is added to a histogram, allowing calculation of percentiles,
// averages, and other statistics. Use time.Since(start).Seconds() for accuracy.
//
// Parameters:
//   - operation: Name of the operation being measured (e.g., "db_query", "file_processing")
//   - duration: Duration in seconds (use time.Since(start).Seconds())
//
// Example:
//
//	start := time.Now()
//	// ... perform operation ...
//	metrics.RecordDuration("process_report", time.Since(start).Seconds())
func (m *PrometheusMetrics) RecordDuration(operation string, duration float64) {
	m.durationSeconds.WithLabelValues(operation).Observe(duration)
}

// RecordFileSize records the size of a processed file in bytes.
// The size is added to a histogram with exponential buckets suitable
// for file size distribution analysis.
//
// Parameters:
//   - fileType: Type of file processed (e.g., "json", "pdf", "markdown")
//   - bytes: Size of the file in bytes
//
// Example:
//
//	fileInfo, _ := os.Stat(filename)
//	metrics.RecordFileSize("json", fileInfo.Size())
func (m *PrometheusMetrics) RecordFileSize(fileType string, bytes int64) {
	m.fileSizeBytes.WithLabelValues(fileType).Observe(float64(bytes))
}

// StartOperation increments the in-progress gauge for an operation.
// Use this to track concurrent operations and identify bottlenecks.
// Must be paired with EndOperation to maintain accurate counts.
//
// Parameters:
//   - operation: Name of the operation starting (e.g., "report_processing")
//
// Example:
//
//	metrics.StartOperation("report_processing")
//	defer metrics.EndOperation("report_processing")
//	// ... perform operation ...
func (m *PrometheusMetrics) StartOperation(operation string) {
	m.inProgress.WithLabelValues(operation).Inc()
}

// EndOperation decrements the in-progress gauge for an operation.
// This should be called when an operation completes (successfully or not).
// Typically called in a defer statement to ensure it runs even on errors.
//
// Parameters:
//   - operation: Name of the operation ending (must match StartOperation)
//
// Example:
//
//	metrics.StartOperation("report_processing")
//	defer metrics.EndOperation("report_processing")
//	// ... perform operation ...
func (m *PrometheusMetrics) EndOperation(operation string) {
	m.inProgress.WithLabelValues(operation).Dec()
}
