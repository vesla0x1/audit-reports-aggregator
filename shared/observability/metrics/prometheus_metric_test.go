package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Create a new registry for testing
	reg := prometheus.NewRegistry()

	// Clear default registry for test isolation
	prometheus.DefaultRegisterer = reg

	metrics := New("test-service")

	assert.NotNil(t, metrics)
	assert.Equal(t, "test-service", metrics.serviceName)
}

func TestPrometheusMetrics_RecordSuccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg

	metrics := New("test")

	metrics.RecordSuccess("pdf")
	metrics.RecordSuccess("pdf")
	metrics.RecordSuccess("html")

	// Verify counters
	pdfCount := testutil.ToFloat64(metrics.processedTotal.WithLabelValues("success", "pdf"))
	htmlCount := testutil.ToFloat64(metrics.processedTotal.WithLabelValues("success", "html"))

	assert.Equal(t, 2.0, pdfCount)
	assert.Equal(t, 1.0, htmlCount)
}

func TestPrometheusMetrics_RecordError(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg

	metrics := New("test")

	metrics.RecordError("pdf", "network_error")
	metrics.RecordError("pdf", "network_error")
	metrics.RecordError("html", "parsing_error")

	// Verify processed counter
	pdfErrorCount := testutil.ToFloat64(metrics.processedTotal.WithLabelValues("error", "pdf"))
	htmlErrorCount := testutil.ToFloat64(metrics.processedTotal.WithLabelValues("error", "html"))

	assert.Equal(t, 2.0, pdfErrorCount)
	assert.Equal(t, 1.0, htmlErrorCount)

	// Verify error counter
	networkErrors := testutil.ToFloat64(metrics.errorsTotal.WithLabelValues("network_error", "pdf"))
	parsingErrors := testutil.ToFloat64(metrics.errorsTotal.WithLabelValues("parsing_error", "html"))

	assert.Equal(t, 2.0, networkErrors)
	assert.Equal(t, 1.0, parsingErrors)
}

func TestPrometheusMetrics_Operations(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg

	metrics := New("test")

	// Start operations
	metrics.StartOperation("download")
	metrics.StartOperation("download")
	metrics.StartOperation("process")

	// Verify gauges
	downloadGauge := testutil.ToFloat64(metrics.inProgress.WithLabelValues("download"))
	processGauge := testutil.ToFloat64(metrics.inProgress.WithLabelValues("process"))

	assert.Equal(t, 2.0, downloadGauge)
	assert.Equal(t, 1.0, processGauge)

	// End operations
	metrics.EndOperation("download")
	metrics.EndOperation("process")

	// Verify updated gauges
	downloadGauge = testutil.ToFloat64(metrics.inProgress.WithLabelValues("download"))
	processGauge = testutil.ToFloat64(metrics.inProgress.WithLabelValues("process"))

	assert.Equal(t, 1.0, downloadGauge)
	assert.Equal(t, 0.0, processGauge)
}
