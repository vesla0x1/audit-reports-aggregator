package cloudwatch

import (
	"context"
	"fmt"
	"shared/config"
	"shared/domain/observability"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Metrics implements observability.Metrics using AWS CloudWatch Metrics
type Metrics struct {
	client    *cloudwatch.Client
	namespace string
	buffer    []types.MetricDatum
	bufferCh  chan types.MetricDatum
}

// NewMetrics creates a new CloudWatch metrics client
func NewMetrics(cfg config.Config) (observability.Metrics, error) {
	namespace := cfg.Observability.CloudWatchNamespace
	if namespace == "" {
		// Fallback to service-based namespace
		namespace = fmt.Sprintf("%s/%s", cfg.ServiceName, cfg.Environment)
	}

	// Determine region
	region := cfg.Observability.CloudWatchRegion
	if region == "" {
		region = cfg.Storage.S3.Region // Fallback to S3 region
	}

	if region == "" {
		return nil, fmt.Errorf("no AWS region specified for metrics")
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for metrics: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsCfg)

	m := &Metrics{
		client:    client,
		namespace: namespace,
		buffer:    make([]types.MetricDatum, 0, 20),
		bufferCh:  make(chan types.MetricDatum, 100),
	}

	// Start background flusher
	go m.backgroundFlusher()

	return m, nil
}

// IncrementCounter increments a counter metric
func (m *Metrics) IncrementCounter(name string, tags map[string]string) {
	datum := types.MetricDatum{
		MetricName: aws.String(name),
		Value:      aws.Float64(1),
		Unit:       types.StandardUnitCount,
		Timestamp:  aws.Time(time.Now()),
		Dimensions: m.tagsToDimensions(tags),
	}

	select {
	case m.bufferCh <- datum:
	default:
		// Buffer full, drop metric (or handle differently)
	}
}

// RecordHistogram records a value in a histogram
func (m *Metrics) RecordHistogram(name string, value float64, tags map[string]string) {
	datum := types.MetricDatum{
		MetricName: aws.String(name),
		Value:      aws.Float64(value),
		Unit:       types.StandardUnitNone,
		Timestamp:  aws.Time(time.Now()),
		Dimensions: m.tagsToDimensions(tags),
	}

	select {
	case m.bufferCh <- datum:
	default:
		// Buffer full, drop metric
	}
}

// RecordGauge records a gauge value
func (m *Metrics) RecordGauge(name string, value float64, tags map[string]string) {
	datum := types.MetricDatum{
		MetricName: aws.String(name),
		Value:      aws.Float64(value),
		Unit:       types.StandardUnitNone,
		Timestamp:  aws.Time(time.Now()),
		Dimensions: m.tagsToDimensions(tags),
	}

	select {
	case m.bufferCh <- datum:
	default:
		// Buffer full, drop metric
	}
}

// tagsToDimensions converts tags to CloudWatch dimensions
func (m *Metrics) tagsToDimensions(tags map[string]string) []types.Dimension {
	dimensions := make([]types.Dimension, 0, len(tags))
	for name, value := range tags {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(name),
			Value: aws.String(value),
		})
	}
	return dimensions
}

// backgroundFlusher periodically flushes metrics to CloudWatch
func (m *Metrics) backgroundFlusher() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case metric := <-m.bufferCh:
			m.buffer = append(m.buffer, metric)
			// Flush if buffer is full
			if len(m.buffer) >= 20 {
				m.flush()
			}

		case <-ticker.C:
			// Periodic flush
			if len(m.buffer) > 0 {
				m.flush()
			}
		}
	}
}

// flush sends buffered metrics to CloudWatch
func (m *Metrics) flush() {
	if len(m.buffer) == 0 {
		return
	}

	// Send asynchronously
	go func(data []types.MetricDatum) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, _ = m.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(m.namespace),
			MetricData: data,
		})
	}(m.buffer)

	// Clear buffer
	m.buffer = make([]types.MetricDatum, 0, 20)
}
