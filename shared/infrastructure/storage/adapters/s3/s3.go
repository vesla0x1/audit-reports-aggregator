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

// Client implements ObjectStorage interface for AWS S3
type Client struct {
	s3      *s3.Client
	config  *config.S3Config
	logger  observability.Logger
	metrics observability.Metrics
}

// New creates a new S3 storage client
func New(cfg *config.StorageConfig, logger observability.Logger, metrics observability.Metrics) (storage.ObjectStorage, error) {
	builder := newClientBuilder(cfg, logger, metrics)
	return builder.Build()
}

// --- Client Builder ---

type clientBuilder struct {
	config  *config.StorageConfig
	logger  observability.Logger
	metrics observability.Metrics
}

func newClientBuilder(cfg *config.StorageConfig, logger observability.Logger, metrics observability.Metrics) *clientBuilder {
	return &clientBuilder{
		config:  cfg,
		logger:  logger,
		metrics: metrics,
	}
}

func (b *clientBuilder) Build() (*Client, error) {
	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 configuration: %w", err)
	}

	awsCfg, err := b.buildAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS config: %w", err)
	}

	client := b.createClient(awsCfg)

	if err := client.verifyConnection(); err != nil {
		return nil, err
	}

	b.logSuccess()
	return client, nil
}

func (b *clientBuilder) buildAWSConfig() (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(b.config.S3.Region),
		awsconfig.WithRetryMaxAttempts(b.config.MaxRetries),
		awsconfig.WithHTTPClient(&http.Client{
			Timeout: b.config.Timeout,
		}),
	}

	if b.hasStaticCredentials() {
		opts = append(opts, b.staticCredentialsOption())
	}

	return awsconfig.LoadDefaultConfig(context.Background(), opts...)
}

func (b *clientBuilder) hasStaticCredentials() bool {
	return b.config.S3.AccessKeyID != "" && b.config.S3.SecretAccessKey != ""
}

func (b *clientBuilder) staticCredentialsOption() func(*awsconfig.LoadOptions) error {
	return awsconfig.WithCredentialsProvider(
		credentials.NewStaticCredentialsProvider(
			b.config.S3.AccessKeyID,
			b.config.S3.SecretAccessKey,
			"",
		),
	)
}

func (b *clientBuilder) createClient(awsCfg aws.Config) *Client {
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &Client{
		s3:      s3Client,
		config:  &b.config.S3,
		logger:  b.logger,
		metrics: b.metrics,
	}
}

func (b *clientBuilder) logSuccess() {
	b.logger.Info("S3 client initialized successfully",
		"bucket", b.config.S3.Bucket,
		"region", b.config.S3.Region)
}

// --- Connection Verification ---

func (c *Client) verifyConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.ensureBucketExists(ctx); err != nil {
		c.logger.Error("Failed to verify bucket existence",
			"error", err,
			"bucket", c.config.Bucket)
		return fmt.Errorf("failed to verify bucket existence: %w", err)
	}
	return nil
}

func (c *Client) ensureBucketExists(ctx context.Context) error {
	exists, err := c.bucketExists(ctx, c.config.Bucket)
	if err != nil {
		return err
	}

	if !exists {
		c.logger.Info("Bucket does not exist, creating", "bucket", c.config.Bucket)
		return c.CreateBucket(ctx, c.config.Bucket)
	}

	c.logger.Info("Bucket exists", "bucket", c.config.Bucket)
	return nil
}

func (c *Client) bucketExists(ctx context.Context, bucket string) (bool, error) {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})

	if err == nil {
		return true, nil
	}

	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check bucket existence: %w", err)
}

// --- Core Operations ---

// Put stores an object in S3
func (c *Client) Put(ctx context.Context, bucket, key string, reader io.Reader, metadata storage.ObjectMetadata) error {
	op := c.newPutOperation(bucket, key, reader, metadata)
	return op.Execute(ctx)
}

// Get retrieves an object from S3
func (c *Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	op := c.newGetOperation(bucket, key)
	return op.Execute(ctx)
}

// GetWithMetadata retrieves an object with its metadata
func (c *Client) GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.ObjectMetadata, error) {
	op := c.newGetWithMetadataOperation(bucket, key)
	return op.Execute(ctx)
}

// Delete removes an object from S3
func (c *Client) Delete(ctx context.Context, bucket, key string) error {
	op := c.newDeleteOperation(bucket, key)
	return op.Execute(ctx)
}

// Exists checks if an object exists
func (c *Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	op := c.newExistsOperation(bucket, key)
	return op.Execute(ctx)
}

// List returns objects with the given prefix
func (c *Client) List(ctx context.Context, bucket, prefix string) ([]storage.ObjectInfo, error) {
	op := c.newListOperation(bucket, prefix)
	return op.Execute(ctx)
}

// CreateBucket creates a new S3 bucket
func (c *Client) CreateBucket(ctx context.Context, bucket string) error {
	op := c.newCreateBucketOperation(bucket)
	return op.Execute(ctx)
}

// DeleteBucket removes an S3 bucket
func (c *Client) DeleteBucket(ctx context.Context, bucket string) error {
	op := c.newDeleteBucketOperation(bucket)
	return op.Execute(ctx)
}

// --- Operation Types ---

// putOperation encapsulates a put operation
type putOperation struct {
	client   *Client
	bucket   string
	key      string
	reader   io.Reader
	metadata storage.ObjectMetadata
}

func (c *Client) newPutOperation(bucket, key string, reader io.Reader, metadata storage.ObjectMetadata) *putOperation {
	return &putOperation{
		client:   c,
		bucket:   c.resolveBucket(bucket),
		key:      key,
		reader:   reader,
		metadata: metadata,
	}
}

func (op *putOperation) Execute(ctx context.Context) error {
	timer := op.client.startTimer()

	content, size, err := op.readContent()
	if err != nil {
		op.recordError("read_error")
		return err
	}

	input := op.buildInput(content)

	if err := op.upload(ctx, input); err != nil {
		op.recordError("s3_error")
		return err
	}

	op.recordSuccess(timer.Elapsed(), size)
	return nil
}

func (op *putOperation) readContent() ([]byte, int64, error) {
	buf := &bytes.Buffer{}
	size, err := io.Copy(buf, op.reader)
	if err != nil {
		op.client.logger.Error("Failed to read content",
			"error", err,
			"bucket", op.bucket,
			"key", op.key)
		return nil, 0, fmt.Errorf("failed to read content: %w", err)
	}
	return buf.Bytes(), size, nil
}

func (op *putOperation) buildInput(content []byte) *s3.PutObjectInput {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(op.bucket),
		Key:         aws.String(op.key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(op.metadata.ContentType),
	}

	op.applyOptionalMetadata(input)
	return input
}

func (op *putOperation) applyOptionalMetadata(input *s3.PutObjectInput) {
	if op.metadata.ContentEncoding != "" {
		input.ContentEncoding = aws.String(op.metadata.ContentEncoding)
	}
	if op.metadata.CacheControl != "" {
		input.CacheControl = aws.String(op.metadata.CacheControl)
	}
	if len(op.metadata.UserMetadata) > 0 {
		input.Metadata = op.metadata.UserMetadata
	}
}

func (op *putOperation) upload(ctx context.Context, input *s3.PutObjectInput) error {
	_, err := op.client.s3.PutObject(ctx, input)
	if err != nil {
		op.client.logger.Error("Failed to put object",
			"error", err,
			"bucket", op.bucket,
			"key", op.key)
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (op *putOperation) recordError(errorType string) {
	op.client.metrics.IncrementCounter("s3.put.errors", map[string]string{
		"error_type": errorType,
	})
}

func (op *putOperation) recordSuccess(duration time.Duration, size int64) {
	op.client.logger.Info("Object stored successfully",
		"bucket", op.bucket,
		"key", op.key,
		"size_bytes", size,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.put.success", nil)
	op.client.metrics.RecordHistogram("s3.put.duration", float64(duration.Milliseconds()), nil)
	op.client.metrics.RecordHistogram("s3.put.size", float64(size), nil)
}

// getOperation encapsulates a get operation
type getOperation struct {
	client *Client
	bucket string
	key    string
}

func (c *Client) newGetOperation(bucket, key string) *getOperation {
	return &getOperation{
		client: c,
		bucket: c.resolveBucket(bucket),
		key:    key,
	}
}

func (op *getOperation) Execute(ctx context.Context) (io.ReadCloser, error) {
	timer := op.client.startTimer()

	result, err := op.client.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(op.bucket),
		Key:    aws.String(op.key),
	})

	if err != nil {
		return nil, op.handleError(err)
	}

	op.recordSuccess(timer.Elapsed())
	return result.Body, nil
}

func (op *getOperation) handleError(err error) error {
	if isNotFoundError(err) {
		op.client.logger.Info("Object not found",
			"bucket", op.bucket,
			"key", op.key)
		op.client.metrics.IncrementCounter("s3.get.not_found", nil)
		return storage.ErrObjectNotFound
	}

	op.client.logger.Error("Failed to get object",
		"error", err,
		"bucket", op.bucket,
		"key", op.key)
	op.client.metrics.IncrementCounter("s3.get.errors", nil)
	return fmt.Errorf("failed to get object: %w", err)
}

func (op *getOperation) recordSuccess(duration time.Duration) {
	op.client.logger.Info("Object retrieved successfully",
		"bucket", op.bucket,
		"key", op.key,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.get.success", nil)
	op.client.metrics.RecordHistogram("s3.get.duration", float64(duration.Milliseconds()), nil)
}

// getWithMetadataOperation encapsulates get with metadata
type getWithMetadataOperation struct {
	client *Client
	bucket string
	key    string
}

func (c *Client) newGetWithMetadataOperation(bucket, key string) *getWithMetadataOperation {
	return &getWithMetadataOperation{
		client: c,
		bucket: c.resolveBucket(bucket),
		key:    key,
	}
}

func (op *getWithMetadataOperation) Execute(ctx context.Context) (io.ReadCloser, *storage.ObjectMetadata, error) {
	timer := op.client.startTimer()

	result, err := op.client.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(op.bucket),
		Key:    aws.String(op.key),
	})

	if err != nil {
		if isNotFoundError(err) {
			op.client.metrics.IncrementCounter("s3.get_with_metadata.not_found", nil)
			return nil, nil, storage.ErrObjectNotFound
		}
		op.recordError(err)
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	metadata := op.extractMetadata(result)
	op.recordSuccess(timer.Elapsed())

	return result.Body, metadata, nil
}

func (op *getWithMetadataOperation) extractMetadata(result *s3.GetObjectOutput) *storage.ObjectMetadata {
	return &storage.ObjectMetadata{
		ContentType:     aws.ToString(result.ContentType),
		ContentLength:   aws.ToInt64(result.ContentLength),
		ContentEncoding: aws.ToString(result.ContentEncoding),
		CacheControl:    aws.ToString(result.CacheControl),
		LastModified:    aws.ToTime(result.LastModified),
		ETag:            aws.ToString(result.ETag),
		UserMetadata:    result.Metadata,
	}
}

func (op *getWithMetadataOperation) recordError(err error) {
	op.client.logger.Error("Failed to get object with metadata",
		"error", err,
		"bucket", op.bucket,
		"key", op.key)
	op.client.metrics.IncrementCounter("s3.get_with_metadata.errors", nil)
}

func (op *getWithMetadataOperation) recordSuccess(duration time.Duration) {
	op.client.metrics.IncrementCounter("s3.get_with_metadata.success", nil)
	op.client.metrics.RecordHistogram("s3.get_with_metadata.duration", float64(duration.Milliseconds()), nil)
}

// deleteOperation encapsulates delete operation
type deleteOperation struct {
	client *Client
	bucket string
	key    string
}

func (c *Client) newDeleteOperation(bucket, key string) *deleteOperation {
	return &deleteOperation{
		client: c,
		bucket: c.resolveBucket(bucket),
		key:    key,
	}
}

func (op *deleteOperation) Execute(ctx context.Context) error {
	timer := op.client.startTimer()

	_, err := op.client.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(op.bucket),
		Key:    aws.String(op.key),
	})

	if err != nil {
		op.recordError(err)
		return fmt.Errorf("failed to delete object: %w", err)
	}

	op.recordSuccess(timer.Elapsed())
	return nil
}

func (op *deleteOperation) recordError(err error) {
	op.client.logger.Error("Failed to delete object",
		"error", err,
		"bucket", op.bucket,
		"key", op.key)
	op.client.metrics.IncrementCounter("s3.delete.errors", nil)
}

func (op *deleteOperation) recordSuccess(duration time.Duration) {
	op.client.logger.Info("Object deleted successfully",
		"bucket", op.bucket,
		"key", op.key,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.delete.success", nil)
	op.client.metrics.RecordHistogram("s3.delete.duration", float64(duration.Milliseconds()), nil)
}

// existsOperation encapsulates existence check
type existsOperation struct {
	client *Client
	bucket string
	key    string
}

func (c *Client) newExistsOperation(bucket, key string) *existsOperation {
	return &existsOperation{
		client: c,
		bucket: c.resolveBucket(bucket),
		key:    key,
	}
}

func (op *existsOperation) Execute(ctx context.Context) (bool, error) {
	timer := op.client.startTimer()

	_, err := op.client.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(op.bucket),
		Key:    aws.String(op.key),
	})

	duration := timer.Elapsed()
	op.recordMetrics(duration)

	if err != nil {
		if isNotFoundError(err) {
			op.client.metrics.IncrementCounter("s3.exists.not_found", nil)
			return false, nil
		}
		op.recordError(err)
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	op.client.metrics.IncrementCounter("s3.exists.found", nil)
	return true, nil
}

func (op *existsOperation) recordError(err error) {
	op.client.logger.Error("Failed to check object existence",
		"error", err,
		"bucket", op.bucket,
		"key", op.key)
	op.client.metrics.IncrementCounter("s3.exists.errors", nil)
}

func (op *existsOperation) recordMetrics(duration time.Duration) {
	op.client.metrics.RecordHistogram("s3.exists.duration", float64(duration.Milliseconds()), nil)
}

// listOperation encapsulates list operation
type listOperation struct {
	client *Client
	bucket string
	prefix string
}

func (c *Client) newListOperation(bucket, prefix string) *listOperation {
	return &listOperation{
		client: c,
		bucket: c.resolveBucket(bucket),
		prefix: prefix,
	}
}

func (op *listOperation) Execute(ctx context.Context) ([]storage.ObjectInfo, error) {
	timer := op.client.startTimer()

	objects, pageCount, err := op.fetchAllObjects(ctx)
	if err != nil {
		return nil, err
	}

	op.recordSuccess(timer.Elapsed(), len(objects), pageCount)
	return objects, nil
}

func (op *listOperation) fetchAllObjects(ctx context.Context) ([]storage.ObjectInfo, int, error) {
	input := op.buildInput()
	paginator := s3.NewListObjectsV2Paginator(op.client.s3, input)

	var objects []storage.ObjectInfo
	pageCount := 0

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			op.recordError(err, pageCount)
			return nil, 0, fmt.Errorf("failed to list objects: %w", err)
		}

		objects = append(objects, op.extractObjects(page)...)
		pageCount++
	}

	return objects, pageCount, nil
}

func (op *listOperation) buildInput() *s3.ListObjectsV2Input {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(op.bucket),
	}
	if op.prefix != "" {
		input.Prefix = aws.String(op.prefix)
	}
	return input
}

func (op *listOperation) extractObjects(page *s3.ListObjectsV2Output) []storage.ObjectInfo {
	objects := make([]storage.ObjectInfo, 0, len(page.Contents))
	for _, obj := range page.Contents {
		objects = append(objects, storage.ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
		})
	}
	return objects
}

func (op *listOperation) recordError(err error, pagesProcessed int) {
	op.client.logger.Error("Failed to list objects",
		"error", err,
		"bucket", op.bucket,
		"prefix", op.prefix,
		"pages_processed", pagesProcessed)
	op.client.metrics.IncrementCounter("s3.list.errors", nil)
}

func (op *listOperation) recordSuccess(duration time.Duration, count, pages int) {
	op.client.logger.Info("Objects listed successfully",
		"bucket", op.bucket,
		"prefix", op.prefix,
		"count", count,
		"pages", pages,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.list.success", nil)
	op.client.metrics.RecordHistogram("s3.list.duration", float64(duration.Milliseconds()), nil)
	op.client.metrics.RecordHistogram("s3.list.count", float64(count), nil)
}

// createBucketOperation encapsulates bucket creation
type createBucketOperation struct {
	client *Client
	bucket string
}

func (c *Client) newCreateBucketOperation(bucket string) *createBucketOperation {
	return &createBucketOperation{
		client: c,
		bucket: bucket,
	}
}

func (op *createBucketOperation) Execute(ctx context.Context) error {
	timer := op.client.startTimer()

	input := op.buildInput()
	err := op.create(ctx, input)

	if err != nil {
		if op.isAlreadyExists(err) {
			op.recordAlreadyExists()
			return nil
		}
		op.recordError(err)
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	op.recordSuccess(timer.Elapsed())
	return nil
}

func (op *createBucketOperation) buildInput() *s3.CreateBucketInput {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(op.bucket),
	}

	if op.needsLocationConstraint() {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(op.client.config.Region),
		}
	}

	return input
}

func (op *createBucketOperation) needsLocationConstraint() bool {
	return op.client.config.Region != "" && op.client.config.Region != "us-east-1"
}

func (op *createBucketOperation) create(ctx context.Context, input *s3.CreateBucketInput) error {
	_, err := op.client.s3.CreateBucket(ctx, input)
	return err
}

func (op *createBucketOperation) isAlreadyExists(err error) bool {
	var bae *s3types.BucketAlreadyExists
	var baoyb *s3types.BucketAlreadyOwnedByYou
	return errors.As(err, &bae) || errors.As(err, &baoyb)
}

func (op *createBucketOperation) recordAlreadyExists() {
	op.client.logger.Info("Bucket already exists", "bucket", op.bucket)
	op.client.metrics.IncrementCounter("s3.create_bucket.already_exists", nil)
}

func (op *createBucketOperation) recordError(err error) {
	op.client.logger.Error("Failed to create bucket",
		"error", err,
		"bucket", op.bucket)
	op.client.metrics.IncrementCounter("s3.create_bucket.errors", nil)
}

func (op *createBucketOperation) recordSuccess(duration time.Duration) {
	op.client.logger.Info("Bucket created successfully",
		"bucket", op.bucket,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.create_bucket.success", nil)
	op.client.metrics.RecordHistogram("s3.create_bucket.duration", float64(duration.Milliseconds()), nil)
}

// deleteBucketOperation encapsulates bucket deletion
type deleteBucketOperation struct {
	client *Client
	bucket string
}

func (c *Client) newDeleteBucketOperation(bucket string) *deleteBucketOperation {
	return &deleteBucketOperation{
		client: c,
		bucket: bucket,
	}
}

func (op *deleteBucketOperation) Execute(ctx context.Context) error {
	timer := op.client.startTimer()

	_, err := op.client.s3.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(op.bucket),
	})

	if err != nil {
		op.recordError(err)
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	op.recordSuccess(timer.Elapsed())
	return nil
}

func (op *deleteBucketOperation) recordError(err error) {
	op.client.logger.Error("Failed to delete bucket",
		"error", err,
		"bucket", op.bucket)
	op.client.metrics.IncrementCounter("s3.delete_bucket.errors", nil)
}

func (op *deleteBucketOperation) recordSuccess(duration time.Duration) {
	op.client.logger.Info("Bucket deleted successfully",
		"bucket", op.bucket,
		"duration_ms", duration.Milliseconds())

	op.client.metrics.IncrementCounter("s3.delete_bucket.success", nil)
	op.client.metrics.RecordHistogram("s3.delete_bucket.duration", float64(duration.Milliseconds()), nil)
}

// --- Helper Methods ---

func (c *Client) resolveBucket(bucket string) string {
	if bucket == "" {
		return c.config.Bucket
	}
	return bucket
}

// timer tracks operation duration
type timer struct {
	start time.Time
}

func (c *Client) startTimer() *timer {
	return &timer{start: time.Now()}
}

func (t *timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// isNotFoundError checks if error indicates object not found
func isNotFoundError(err error) bool {
	var noSuchKey *s3types.NoSuchKey
	var notFound *s3types.NotFound
	return errors.As(err, &noSuchKey) || errors.As(err, &notFound)
}
