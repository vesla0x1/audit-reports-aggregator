package cloudwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// logger implements domain.Logger using AWS CloudWatch Logs
type logger struct {
	client        *cloudwatchlogs.Client
	logGroup      string
	logStream     string
	sequenceToken *string
	baseFields    map[string]interface{}
	level         LogLevel
}

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// NewLogger creates a new CloudWatch logger that implements domain.Logger
func NewCloudwatchLogger(cfg config.Config) (ports.Logger, error) {
	logGroup := cfg.Observability.CloudWatchLogGroup
	if logGroup == "" {
		// Fallback to service-based log group
		logGroup = fmt.Sprintf("/aws/lambda/%s", cfg.ServiceName)
	}

	region := cfg.Observability.CloudWatchRegion
	if region == "" {
		region = cfg.Storage.S3.Region // Fallback to S3 region
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(awsCfg)

	// Generate log stream name with timestamp and hostname
	logStream := fmt.Sprintf("%s-%s-%d",
		cfg.ServiceName,
		cfg.Environment,
		time.Now().Unix(),
	)

	l := &logger{
		client:     client,
		logGroup:   logGroup,
		logStream:  logStream,
		baseFields: make(map[string]interface{}),
		level:      parseLogLevel(cfg.LogLevel),
	}

	// Add base fields
	l.baseFields["service"] = cfg.ServiceName
	l.baseFields["environment"] = cfg.Environment
	if cfg.Version != "" {
		l.baseFields["version"] = cfg.Version
	}

	// Initialize CloudWatch resources
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := l.ensureLogGroup(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure log group: %w", err)
	}

	if err := l.ensureLogStream(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure log stream: %w", err)
	}

	return l, nil
}

// Info logs informational messages
func (l *logger) Info(msg string, fields ...interface{}) {
	if l.level > InfoLevel {
		return
	}
	l.log(context.Background(), "INFO", msg, nil, fieldsToMap(fields...))
}

// Error logs error messages
func (l *logger) Error(msg string, fields ...interface{}) {
	if l.level > ErrorLevel {
		return
	}

	// Extract error from fields if present
	fieldMap := fieldsToMap(fields...)
	var err error
	if errVal, ok := fieldMap["error"]; ok {
		if e, ok := errVal.(error); ok {
			err = e
		}
	}

	l.log(context.Background(), "ERROR", msg, err, fieldMap)
}

// WithFields returns a new logger with additional default fields
func (l *logger) WithFields(fields map[string]interface{}) ports.Logger {
	newFields := make(map[string]interface{})
	// Copy existing base fields
	for k, v := range l.baseFields {
		newFields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newFields[k] = v
	}

	return &logger{
		client:        l.client,
		logGroup:      l.logGroup,
		logStream:     l.logStream,
		sequenceToken: l.sequenceToken,
		baseFields:    newFields,
		level:         l.level,
	}
}

// log sends the log entry to CloudWatch
func (l *logger) log(ctx context.Context, level, msg string, err error, fields map[string]interface{}) {
	logEntry := l.buildLogEntry(ctx, level, msg, err, fields)

	// Marshal to JSON
	data, marshalErr := json.Marshal(logEntry)
	if marshalErr != nil {
		data = []byte(fmt.Sprintf(`{"level":"%s","message":"%s","error":"failed to marshal log"}`, level, msg))
	}

	l.sendToCloudWatch(string(data))
}

// sendToCloudWatch sends the log message to CloudWatch
func (l *logger) sendToCloudWatch(message string) {
	token := l.sequenceToken

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(l.logGroup),
		LogStreamName: aws.String(l.logStream),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(message),
				Timestamp: aws.Int64(time.Now().UnixMilli()),
			},
		},
	}

	if token != nil {
		input.SequenceToken = token
	}

	// Send asynchronously to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		output, err := l.client.PutLogEvents(ctx, input)
		if err == nil && output.NextSequenceToken != nil {
			l.sequenceToken = output.NextSequenceToken
		}
		// Consider implementing retry logic here for failed sends
	}()
}

// buildLogEntry constructs the log entry with all fields
func (l *logger) buildLogEntry(ctx context.Context, level, msg string, err error, fields map[string]interface{}) map[string]interface{} {
	entry := make(map[string]interface{})

	// Add base fields
	for k, v := range l.baseFields {
		entry[k] = v
	}

	// Add provided fields
	for k, v := range fields {
		entry[k] = v
	}

	// Add standard fields
	entry["level"] = level
	entry["message"] = msg
	entry["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)

	// Add error if present
	if err != nil {
		entry["error"] = err.Error()
		// Add error type for better filtering
		entry["error_type"] = fmt.Sprintf("%T", err)
	}

	return entry
}

// ensureLogGroup creates the log group if it doesn't exist
func (l *logger) ensureLogGroup(ctx context.Context) error {
	_, err := l.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(l.logGroup),
	})

	if err != nil {
		var alreadyExists *types.ResourceAlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return nil // Log group already exists, that's fine
		}
		return fmt.Errorf("failed to create log group: %w", err)
	}

	// Set retention policy (optional)
	_, err = l.client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(l.logGroup),
		RetentionInDays: aws.Int32(30), // Configure based on your needs
	})
	if err != nil {
		// Non-fatal, just log the error
		fmt.Printf("Warning: failed to set retention policy: %v\n", err)
	}

	return nil
}

// ensureLogStream creates the log stream if it doesn't exist
func (l *logger) ensureLogStream(ctx context.Context) error {
	_, err := l.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(l.logGroup),
		LogStreamName: aws.String(l.logStream),
	})

	if err != nil {
		var alreadyExists *types.ResourceAlreadyExistsException
		if errors.As(err, &alreadyExists) {
			return nil // Log stream already exists, that's fine
		}
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	return nil
}

func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// fieldsToMap converts variadic key-value pairs to a map
func fieldsToMap(fields ...interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Handle odd number of fields
	if len(fields)%2 != 0 {
		fields = append(fields, "")
	}

	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", fields[i])
		}
		result[key] = fields[i+1]
	}

	return result
}
