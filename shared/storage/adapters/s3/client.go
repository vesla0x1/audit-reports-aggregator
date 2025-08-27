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
	"shared/observability"
	"shared/storage/types"
)

// Client implements the ObjectStorage interface for AWS S3
type Client struct {
	s3Client *s3.Client
	config   *config.S3Config
	logger   observability.Logger
	metrics  observability.Metrics
}

// NewClient creates a new S3 storage client
func NewClient(cfg *config.StorageConfig, logger observability.Logger, metrics observability.Metrics) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 configuration: %w", err)
	}

	// Build AWS configuration
	awsCfg, err := buildAWSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS config: %w", err)
	}

	// Create S3 client with custom options
	s3Client := s3.NewFromConfig(awsCfg)

	client := &Client{
		s3Client: s3Client,
		config:   &cfg.S3,
		logger:   logger,
		metrics:  metrics,
	}

	// Test connection by checking if the bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.ensureBucketExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify bucket existence: %w", err)
	}

	return client, nil
}

// Put stores an object in S3
func (c *Client) Put(ctx context.Context, bucket, key string, reader io.Reader, metadata types.ObjectMetadata) error {
	start := time.Now()
	defer func() {
		c.metrics.RecordDuration("storage.s3.put", float64(time.Since(start).Milliseconds()))
	}()

	// If bucket is not specified, use the default from config
	if bucket == "" {
		bucket = c.config.Bucket
	}

	// Read the content into a buffer to determine size
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, reader)
	if err != nil {
		//c.metrics.IncrementCounter("storage.s3.put.error", 1)
		c.logger.Error(ctx, "failed to read content", err, observability.Fields{
			"bucket": bucket,
			"key":    key,
			"error":  err.Error(),
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
		//c.metrics.IncrementCounter(ctx, "storage.s3.put.error", 1)
		c.logger.Error(ctx, "failed to put object", err, observability.Fields{
			"bucket": bucket,
			"key":    key,
			"error":  err.Error(),
		})
		return fmt.Errorf("failed to put object: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.put.success", 1)
	//c.metrics.RecordValue("storage.s3.put.bytes", float64(buf.Len()))
	c.logger.Debug(ctx, "object stored successfully", observability.Fields{
		"bucket": bucket,
		"key":    key,
		"size":   buf.Len(),
	})

	return nil
}

// Get retrieves an object from S3
func (c *Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.get", float64(time.Since(start).Milliseconds()))
	//}()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		//c.metrics.IncrementCounter(ctx, "storage.s3.get.error", 1)
		if isNotFoundError(err) {
			c.logger.Debug(ctx, "object not found", observability.Fields{
				"bucket": bucket,
				"key":    key,
			})
			return nil, types.ErrObjectNotFound
		}
		c.logger.Error(ctx, "failed to get object", err, observability.Fields{
			"bucket": bucket,
			"key":    key,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.get.success", 1)
	c.logger.Debug(ctx, "object retrieved successfully", observability.Fields{
		"bucket": bucket,
		"key":    key,
	})

	return result.Body, nil
}

// GetWithMetadata retrieves an object along with its metadata
func (c *Client) GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *types.ObjectMetadata, error) {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.get_with_metadata", float64(time.Since(start).Milliseconds()))
	//}()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		//c.metrics.IncrementCounter(ctx, "storage.s3.get_with_metadata.error", 1)
		if isNotFoundError(err) {
			return nil, nil, types.ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	metadata := &types.ObjectMetadata{
		ContentType:     aws.ToString(result.ContentType),
		ContentLength:   aws.ToInt64(result.ContentLength),
		ContentEncoding: aws.ToString(result.ContentEncoding),
		CacheControl:    aws.ToString(result.CacheControl),
		LastModified:    aws.ToTime(result.LastModified),
		ETag:            aws.ToString(result.ETag),
		UserMetadata:    result.Metadata,
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.get_with_metadata.success", 1)
	return result.Body, metadata, nil
}

// Delete removes an object from S3
func (c *Client) Delete(ctx context.Context, bucket, key string) error {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.delete", float64(time.Since(start).Milliseconds()))
	//}()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.DeleteObject(ctx, input)
	if err != nil {
		//c.metrics.IncrementCounter(ctx, "storage.s3.delete.error", 1)
		c.logger.Error(ctx, "failed to delete object", err, observability.Fields{
			"bucket": bucket,
			"key":    key,
			"error":  err.Error(),
		})
		return fmt.Errorf("failed to delete object: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.delete.success", 1)
	c.logger.Debug(ctx, "object deleted successfully", observability.Fields{
		"bucket": bucket,
		"key":    key,
	})

	return nil
}

// Exists checks if an object exists in S3
func (c *Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.exists", float64(time.Since(start).Milliseconds()))
	//}()

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
			return false, nil
		}
		//c.metrics.IncrementCounter(ctx, "storage.s3.exists.error", 1)
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.exists.success", 1)
	return true, nil
}

// List returns a list of objects in S3
func (c *Client) List(ctx context.Context, bucket, prefix string) ([]types.ObjectInfo, error) {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.list", float64(time.Since(start).Milliseconds()))
	//}()

	if bucket == "" {
		bucket = c.config.Bucket
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	var objects []types.ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			//c.metrics.IncrementCounter(ctx, "storage.s3.list.error", 1)
			c.logger.Error(ctx, "failed to list objects", err, observability.Fields{
				"bucket": bucket,
				"prefix": prefix,
				"error":  err.Error(),
			})
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
				ETag:         aws.ToString(obj.ETag),
			})
		}
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.list.success", 1)
	//c.metrics.RecordValue(ctx, "storage.s3.list.count", float64(len(objects)))
	c.logger.Debug(ctx, "objects listed successfully", observability.Fields{
		"bucket": bucket,
		"prefix": prefix,
		"count":  len(objects),
	})

	return objects, nil
}

// CreateBucket creates a new S3 bucket
func (c *Client) CreateBucket(ctx context.Context, bucket string) error {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.create_bucket", float64(time.Since(start).Milliseconds()))
	//}()

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
			c.logger.Debug(ctx, "bucket already exists", observability.Fields{
				"bucket": bucket,
			})
			return nil
		}

		//c.metrics.IncrementCounter(ctx, "storage.s3.create_bucket.error", 1)
		c.logger.Error(ctx, "failed to create bucket", err, observability.Fields{
			"bucket": bucket,
			"error":  err.Error(),
		})
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.create_bucket.success", 1)
	c.logger.Info(ctx, "bucket created successfully", observability.Fields{
		"bucket": bucket,
	})

	return nil
}

// DeleteBucket removes an S3 bucket
func (c *Client) DeleteBucket(ctx context.Context, bucket string) error {
	//start := time.Now()
	//defer func() {
	//	c.metrics.RecordDuration(ctx, "storage.s3.delete_bucket", float64(time.Since(start).Milliseconds()))
	//}()

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := c.s3Client.DeleteBucket(ctx, input)
	if err != nil {
		//c.metrics.IncrementCounter(ctx, "storage.s3.delete_bucket.error", 1)
		c.logger.Error(ctx, "failed to delete bucket", err, observability.Fields{
			"bucket": bucket,
			"error":  err.Error(),
		})
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	//c.metrics.IncrementCounter(ctx, "storage.s3.delete_bucket.success", 1)
	c.logger.Info(ctx, "bucket deleted successfully", observability.Fields{
		"bucket": bucket,
	})

	return nil
}

// ensureBucketExists checks if the configured bucket exists
func (c *Client) ensureBucketExists(ctx context.Context) error {
	_, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.config.Bucket),
	})

	if err != nil {
		var nse *s3types.NotFound
		if errors.As(err, &nse) {
			// Bucket doesn't exist, try to create it
			c.logger.Info(ctx, "bucket does not exist, attempting to create", observability.Fields{
				"bucket": c.config.Bucket,
			})
			return c.CreateBucket(ctx, c.config.Bucket)
		}
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

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
