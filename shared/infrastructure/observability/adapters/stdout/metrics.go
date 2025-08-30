package stdout

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"shared/domain/observability"
)

// Metrics implements observability.Metrics using stdout
type Metrics struct {
	tags   map[string]string
	logger *log.Logger
	mu     sync.RWMutex

	// Optional: store metrics in memory for debugging/testing
	counters   map[string]int64
	histograms map[string][]float64
	gauges     map[string]float64
}

// NewMetrics creates a new stdout metrics instance
func NewMetrics() observability.Metrics {
	return &Metrics{
		tags:       make(map[string]string),
		logger:     log.New(os.Stdout, "", 0),
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

// IncrementCounter increments a counter metric
func (m *Metrics) IncrementCounter(name string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build metric key
	key := m.buildKey(name, tags)

	// Update in-memory counter
	m.counters[key]++

	// Log the metric
	m.logMetric("COUNTER", name, float64(m.counters[key]), tags)
}

// RecordHistogram records a histogram value
func (m *Metrics) RecordHistogram(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build metric key
	key := m.buildKey(name, tags)

	// Store value in memory
	m.histograms[key] = append(m.histograms[key], value)

	// Calculate basic stats
	stats := m.calculateStats(m.histograms[key])

	// Log the metric with stats
	m.logHistogram(name, value, tags, stats)
}

// RecordGauge records a gauge value
func (m *Metrics) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build metric key
	key := m.buildKey(name, tags)

	// Update in-memory gauge
	m.gauges[key] = value

	// Log the metric
	m.logMetric("GAUGE", name, value, tags)
}

// WithTags returns a new Metrics instance with additional tags
func (m *Metrics) WithTags(tags map[string]string) observability.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create new tags map with combined tags
	newTags := make(map[string]string)

	// Copy existing tags
	for k, v := range m.tags {
		newTags[k] = v
	}

	// Add new tags
	for k, v := range tags {
		newTags[k] = v
	}

	return &Metrics{
		tags:       newTags,
		logger:     m.logger,
		counters:   m.counters,   // Share the same storage
		histograms: m.histograms, // Share the same storage
		gauges:     m.gauges,     // Share the same storage
	}
}

// GetCounter returns the current value of a counter (useful for testing)
func (m *Metrics) GetCounter(name string, tags map[string]string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.counters[key]
}

// GetHistogram returns all values recorded for a histogram (useful for testing)
func (m *Metrics) GetHistogram(name string, tags map[string]string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	values := m.histograms[key]

	// Return a copy to avoid race conditions
	result := make([]float64, len(values))
	copy(result, values)
	return result
}

// GetGauge returns the current value of a gauge (useful for testing)
func (m *Metrics) GetGauge(name string, tags map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.buildKey(name, tags)
	return m.gauges[key]
}

// Reset clears all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]int64)
	m.histograms = make(map[string][]float64)
	m.gauges = make(map[string]float64)
}

// buildKey creates a unique key for a metric with tags
func (m *Metrics) buildKey(name string, tags map[string]string) string {
	// Combine default tags with provided tags
	allTags := make(map[string]string)
	for k, v := range m.tags {
		allTags[k] = v
	}
	for k, v := range tags {
		allTags[k] = v
	}

	// Build sorted tag string for consistent keys
	var tagPairs []string
	for k, v := range allTags {
		tagPairs = append(tagPairs, fmt.Sprintf("%s:%s", k, v))
	}

	if len(tagPairs) > 0 {
		return fmt.Sprintf("%s{%s}", name, strings.Join(tagPairs, ","))
	}
	return name
}

// logMetric logs a metric to stdout
func (m *Metrics) logMetric(metricType string, name string, value float64, tags map[string]string) {
	// Combine tags
	allTags := m.combineTags(tags)

	// Format output
	if jsonMetrics {
		m.logMetricJSON(metricType, name, value, allTags)
	} else {
		m.logMetricText(metricType, name, value, allTags)
	}
}

// logHistogram logs a histogram metric with statistics
func (m *Metrics) logHistogram(name string, value float64, tags map[string]string, stats histogramStats) {
	allTags := m.combineTags(tags)

	if jsonMetrics {
		m.logHistogramJSON(name, value, allTags, stats)
	} else {
		m.logHistogramText(name, value, allTags, stats)
	}
}

// logMetricText outputs metric in text format
func (m *Metrics) logMetricText(metricType string, name string, value float64, tags map[string]string) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	tagStr := ""
	if len(tags) > 0 {
		var tagPairs []string
		for k, v := range tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		tagStr = " " + strings.Join(tagPairs, " ")
	}

	m.logger.Printf("%s [METRIC] %s %s=%.2f%s", timestamp, metricType, name, value, tagStr)
}

// logMetricJSON outputs metric in JSON format
func (m *Metrics) logMetricJSON(metricType string, name string, value float64, tags map[string]string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"type":      "metric",
		"metric":    metricType,
		"name":      name,
		"value":     value,
		"tags":      tags,
	}

	// Use the logger's JSON method if available, or format manually
	jsonStr := formatJSON(entry)
	m.logger.Println(jsonStr)
}

// logHistogramText outputs histogram in text format
func (m *Metrics) logHistogramText(name string, value float64, tags map[string]string, stats histogramStats) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	tagStr := ""
	if len(tags) > 0 {
		var tagPairs []string
		for k, v := range tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		tagStr = " " + strings.Join(tagPairs, " ")
	}

	m.logger.Printf("%s [METRIC] HISTOGRAM %s=%.2f count=%d min=%.2f max=%.2f avg=%.2f%s",
		timestamp, name, value, stats.count, stats.min, stats.max, stats.avg, tagStr)
}

// logHistogramJSON outputs histogram in JSON format
func (m *Metrics) logHistogramJSON(name string, value float64, tags map[string]string, stats histogramStats) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"type":      "metric",
		"metric":    "HISTOGRAM",
		"name":      name,
		"value":     value,
		"stats": map[string]interface{}{
			"count": stats.count,
			"min":   stats.min,
			"max":   stats.max,
			"avg":   stats.avg,
		},
		"tags": tags,
	}

	jsonStr := formatJSON(entry)
	m.logger.Println(jsonStr)
}

// combineTags merges default tags with provided tags
func (m *Metrics) combineTags(tags map[string]string) map[string]string {
	allTags := make(map[string]string)
	for k, v := range m.tags {
		allTags[k] = v
	}
	for k, v := range tags {
		allTags[k] = v
	}
	return allTags
}

// histogramStats holds basic statistics for histogram values
type histogramStats struct {
	count int
	min   float64
	max   float64
	avg   float64
}

// calculateStats computes basic statistics for histogram values
func (m *Metrics) calculateStats(values []float64) histogramStats {
	if len(values) == 0 {
		return histogramStats{}
	}

	stats := histogramStats{
		count: len(values),
		min:   values[0],
		max:   values[0],
	}

	sum := 0.0
	for _, v := range values {
		sum += v
		if v < stats.min {
			stats.min = v
		}
		if v > stats.max {
			stats.max = v
		}
	}

	stats.avg = sum / float64(len(values))
	return stats
}

// formatJSON is a simple JSON formatter
func formatJSON(data map[string]interface{}) string {
	// Simple implementation - in production you'd use json.Marshal
	pairs := []string{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			pairs = append(pairs, fmt.Sprintf(`"%s":"%s"`, k, val))
		case float64:
			pairs = append(pairs, fmt.Sprintf(`"%s":%.2f`, k, val))
		case int:
			pairs = append(pairs, fmt.Sprintf(`"%s":%d`, k, val))
		case map[string]string:
			tagPairs := []string{}
			for tk, tv := range val {
				tagPairs = append(tagPairs, fmt.Sprintf(`"%s":"%s"`, tk, tv))
			}
			pairs = append(pairs, fmt.Sprintf(`"%s":{%s}`, k, strings.Join(tagPairs, ",")))
		case map[string]interface{}:
			pairs = append(pairs, fmt.Sprintf(`"%s":%s`, k, formatJSON(val)))
		default:
			pairs = append(pairs, fmt.Sprintf(`"%s":"%v"`, k, val))
		}
	}
	return "{" + strings.Join(pairs, ",") + "}"
}

// Configuration

var jsonMetrics = false

// UseJSONMetrics enables JSON output format for metrics
func UseJSONMetrics(enabled bool) {
	jsonMetrics = enabled
}
