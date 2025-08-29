package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"shared/config"
	"shared/domain/observability"
	"shared/domain/storage"
)

// Client implements the ObjectStorage interface for AWS S3
type client struct {
	s3Client *s3.Client
	config   *config.S3Config
	logger   observability.Logger
	metrics  observability.Metrics
}

// NewClient creates a new S3 storage client
func New(cfg *config.StorageConfig, logger observability.Logger, metrics observability.Metrics) (storage.ObjectStorage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 configuration: %w", err)
	}

	// Build AWS configuration
	awsCfg, err := buildAWSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS config: %w", err)
	}

	// Create S3 client with custom options
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	c := &client{
		s3Client: s3Client,
		config:   &cfg.S3,
		logger:   logger,
		metrics:  metrics,
	}

	// Test connection by checking if the bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.ensureBucketExists(ctx); err != nil {
		logger.Error("Failed to verify bucket existence", "error", err, "bucket", cfg.S3.Bucket)
		return nil, fmt.Errorf("failed to verify bucket existence: %w", err)
	}

	logger.Info("S3 client initialized successfully", "bucket", cfg.S3.Bucket, "region", cfg.S3.Region)
	return c, nil
}

// Put stores an object in S3
func (c *client) Put(ctx context.Context, bucket, key string, reader io.Reader, metadata storage.ObjectMetadata) error {
	start := time.Now()

	// If bucket is not specified, use the default from config
	if bucket == "" {
		bucket = c.config.Bucket
	}

	// Read the content into a buffer to determine size
	buf := &bytes.Buffer{}
	bytesRead, err := io.Copy(buf, reader)
	if err != nil {
		c.logger.Error("Failed to read content",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.put.errors", map[string]string{
			"error_type": "read_error",
		})
		return fmt.Errorf("failed to read content: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(metadata.ContentType),
	}

	// Add optional metadata
	if metadata.ContentEncoding != "" {
		input.ContentEncoding = aws.String(metadata.ContentEncoding)
	}
	if metadata.CacheControl != "" {
		input.CacheControl = aws.String(metadata.CacheControl)
	}
	if len(metadata.UserMetadata) > 0 {
		input.Metadata = metadata.UserMetadata
	}

	_, err = c.s3Client.PutObject(ctx, input)
	if err != nil {
		c.logger.Error("Failed to put object",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.put.errors", map[string]string{
			"error_type": "s3_error",
		})
		return fmt.Errorf("failed to put object: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Object stored successfully",
		"bucket", bucket,
		"key", key,
		"size_bytes", bytesRead,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.put.success", nil)
	c.metrics.RecordHistogram("s3.put.duration", float64(duration.Milliseconds()), nil)
	c.metrics.RecordHistogram("s3.put.size", float64(bytesRead), nil)

	return nil
}

// Get retrieves an object from S3
func (c *client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	start := time.Now()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			c.logger.Info("Object not found",
				"bucket", bucket,
				"key", key)
			c.metrics.IncrementCounter("s3.get.not_found", nil)
			return nil, storage.ErrObjectNotFound
		}

		c.logger.Error("Failed to get object",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.get.errors", nil)
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Object retrieved successfully",
		"bucket", bucket,
		"key", key,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.get.success", nil)
	c.metrics.RecordHistogram("s3.get.duration", float64(duration.Milliseconds()), nil)

	return result.Body, nil
}

// GetWithMetadata retrieves an object along with its metadata
func (c *client) GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.ObjectMetadata, error) {
	start := time.Now()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			c.metrics.IncrementCounter("s3.get_with_metadata.not_found", nil)
			return nil, nil, storage.ErrObjectNotFound
		}

		c.logger.Error("Failed to get object with metadata",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.get_with_metadata.errors", nil)
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	metadata := &storage.ObjectMetadata{
		ContentType:     aws.ToString(result.ContentType),
		ContentLength:   aws.ToInt64(result.ContentLength),
		ContentEncoding: aws.ToString(result.ContentEncoding),
		CacheControl:    aws.ToString(result.CacheControl),
		LastModified:    aws.ToTime(result.LastModified),
		ETag:            aws.ToString(result.ETag),
		UserMetadata:    result.Metadata,
	}

	duration := time.Since(start)
	c.metrics.IncrementCounter("s3.get_with_metadata.success", nil)
	c.metrics.RecordHistogram("s3.get_with_metadata.duration", float64(duration.Milliseconds()), nil)

	return result.Body, metadata, nil
}

// Delete removes an object from S3
func (c *client) Delete(ctx context.Context, bucket, key string) error {
	start := time.Now()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.DeleteObject(ctx, input)
	if err != nil {
		c.logger.Error("Failed to delete object",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.delete.errors", nil)
		return fmt.Errorf("failed to delete object: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Object deleted successfully",
		"bucket", bucket,
		"key", key,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.delete.success", nil)
	c.metrics.RecordHistogram("s3.delete.duration", float64(duration.Milliseconds()), nil)

	return nil
}

// Exists checks if an object exists in S3
func (c *client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	start := time.Now()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.HeadObject(ctx, input)
	if err != nil {
		if isNotFoundError(err) {
			c.metrics.IncrementCounter("s3.exists.not_found", nil)
			return false, nil
		}
		c.logger.Error("Failed to check object existence",
			"error", err,
			"bucket", bucket,
			"key", key)
		c.metrics.IncrementCounter("s3.exists.errors", nil)
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	c.metrics.IncrementCounter("s3.exists.found", nil)
	c.metrics.RecordHistogram("s3.exists.duration", float64(time.Since(start).Milliseconds()), nil)
	return true, nil
}

// List returns a list of objects in S3
func (c *client) List(ctx context.Context, bucket, prefix string) ([]storage.ObjectInfo, error) {
	start := time.Now()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	var objects []storage.ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, input)

	pageCount := 0
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			c.logger.Error("Failed to list objects",
				"error", err,
				"bucket", bucket,
				"prefix", prefix,
				"pages_processed", pageCount)
			c.metrics.IncrementCounter("s3.list.errors", nil)
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, storage.ObjectInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
				ETag:         aws.ToString(obj.ETag),
			})
		}
		pageCount++
	}

	duration := time.Since(start)
	c.logger.Info("Objects listed successfully",
		"bucket", bucket,
		"prefix", prefix,
		"count", len(objects),
		"pages", pageCount,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.list.success", nil)
	c.metrics.RecordHistogram("s3.list.duration", float64(duration.Milliseconds()), nil)
	c.metrics.RecordHistogram("s3.list.count", float64(len(objects)), nil)

	return objects, nil
}

// CreateBucket creates a new S3 bucket
func (c *client) CreateBucket(ctx context.Context, bucket string) error {
	start := time.Now()

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Add location constraint for non us-east-1 regions
	if c.config.Region != "" && c.config.Region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(c.config.Region),
		}
	}

	_, err := c.s3Client.CreateBucket(ctx, input)
	if err != nil {
		// Check if bucket already exists
		var bae *s3types.BucketAlreadyExists
		var baoyb *s3types.BucketAlreadyOwnedByYou
		if errors.As(err, &bae) || errors.As(err, &baoyb) {
			c.logger.Info("Bucket already exists", "bucket", bucket)
			c.metrics.IncrementCounter("s3.create_bucket.already_exists", nil)
			return nil
		}

		c.logger.Error("Failed to create bucket",
			"error", err,
			"bucket", bucket)
		c.metrics.IncrementCounter("s3.create_bucket.errors", nil)
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Bucket created successfully",
		"bucket", bucket,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.create_bucket.success", nil)
	c.metrics.RecordHistogram("s3.create_bucket.duration", float64(duration.Milliseconds()), nil)

	return nil
}

// DeleteBucket removes an S3 bucket
func (c *client) DeleteBucket(ctx context.Context, bucket string) error {
	start := time.Now()

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := c.s3Client.DeleteBucket(ctx, input)
	if err != nil {
		c.logger.Error("Failed to delete bucket",
			"error", err,
			"bucket", bucket)
		c.metrics.IncrementCounter("s3.delete_bucket.errors", nil)
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	duration := time.Since(start)
	c.logger.Info("Bucket deleted successfully",
		"bucket", bucket,
		"duration_ms", duration.Milliseconds())

	c.metrics.IncrementCounter("s3.delete_bucket.success", nil)
	c.metrics.RecordHistogram("s3.delete_bucket.duration", float64(duration.Milliseconds()), nil)

	return nil
}

// ensureBucketExists checks if the configured bucket exists
func (c *client) ensureBucketExists(ctx context.Context) error {
	_, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.config.Bucket),
	})

	if err != nil {
		var nse *s3types.NotFound
		if errors.As(err, &nse) {
			c.logger.Info("Bucket does not exist, attempting to create",
				"bucket", c.config.Bucket)
			return c.CreateBucket(ctx, c.config.Bucket)
		}
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	c.logger.Info("Bucket exists", "bucket", c.config.Bucket)
	return nil
}

// buildAWSConfig builds the AWS configuration from the S3 config
func buildAWSConfig(storageConfig *config.StorageConfig) (aws.Config, error) {
	var optFns []func(*awsconfig.LoadOptions) error
	s3Config := storageConfig.S3

	if s3Config.Region != "" {
		optFns = append(optFns, awsconfig.WithRegion(s3Config.Region))
	}

	// Use static credentials if provided
	if s3Config.AccessKeyID != "" && s3Config.SecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s3Config.AccessKeyID,
				s3Config.SecretAccessKey,
				"",
			),
		))
	}

	// Set custom retry configuration
	optFns = append(optFns, awsconfig.WithRetryMaxAttempts(storageConfig.MaxRetries))

	// Set timeout
	optFns = append(optFns, awsconfig.WithHTTPClient(&http.Client{
		Timeout: storageConfig.Timeout,
	}))

	return awsconfig.LoadDefaultConfig(context.Background(), optFns...)
}

// isNotFoundError checks if an error is a not found error
func isNotFoundError(err error) bool {
	var nsk *s3types.NoSuchKey
	var nse *s3types.NotFound
	return errors.As(err, &nsk) || errors.As(err, &nse)
}
