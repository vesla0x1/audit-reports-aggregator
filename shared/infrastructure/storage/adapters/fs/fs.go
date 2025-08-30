package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"shared/domain/observability"
	"shared/domain/storage"
)

// Storage implements ObjectStorage using the local filesystem
type Storage struct {
	basePath string
	logger   observability.Logger
	metrics  observability.Metrics
}

// NewStorage creates a new filesystem-based object storage
func NewStorage(basePath string, logger observability.Logger, metrics observability.Metrics) (storage.ObjectStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		logger.Error("Failed to create base path", "path", basePath, "error", err)
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	logger.Info("Filesystem storage initialized", "base_path", basePath)
	metrics.IncrementCounter("storage.filesystem.initialized", nil)

	return &Storage{
		basePath: basePath,
		logger:   logger.WithFields(map[string]interface{}{"component": "filesystem_storage"}),
		metrics:  metrics.WithTags(map[string]string{"storage": "filesystem"}),
	}, nil
}

// Put stores an object
func (s *Storage) Put(ctx context.Context, bucket, key string, reader io.Reader, metadata storage.ObjectMetadata) error {
	startTime := time.Now()
	s.logger.Info("Storing object", "bucket", bucket, "key", key)
	s.metrics.IncrementCounter("storage.put.attempts", map[string]string{"bucket": bucket})

	objectPath := s.getObjectPath(bucket, key)

	// Create bucket directory if needed
	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		s.logger.Error("Failed to create bucket directory", "bucket", bucket, "error", err)
		s.metrics.IncrementCounter("storage.put.errors", map[string]string{"bucket": bucket, "error": "mkdir"})
		return fmt.Errorf("failed to create bucket directory: %w", err)
	}

	// Write object data
	file, err := os.Create(objectPath)
	if err != nil {
		s.logger.Error("Failed to create file", "path", objectPath, "error", err)
		s.metrics.IncrementCounter("storage.put.errors", map[string]string{"bucket": bucket, "error": "create"})
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	bytesWritten, err := io.Copy(file, reader)
	if err != nil {
		s.logger.Error("Failed to write data", "bucket", bucket, "key", key, "error", err)
		s.metrics.IncrementCounter("storage.put.errors", map[string]string{"bucket": bucket, "error": "write"})
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Save metadata
	if err := s.saveMetadata(bucket, key, metadata); err != nil {
		s.logger.Error("Failed to save metadata", "bucket", bucket, "key", key, "error", err)
		s.metrics.IncrementCounter("storage.put.errors", map[string]string{"bucket": bucket, "error": "metadata"})
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Info("Object stored successfully",
		"bucket", bucket,
		"key", key,
		"bytes", bytesWritten,
		"duration_ms", duration.Milliseconds())

	s.metrics.IncrementCounter("storage.put.success", map[string]string{"bucket": bucket})
	s.metrics.RecordHistogram("storage.put.bytes", float64(bytesWritten), map[string]string{"bucket": bucket})
	s.metrics.RecordHistogram("storage.put.duration_ms", float64(duration.Milliseconds()), map[string]string{"bucket": bucket})

	return nil
}

// Get retrieves an object
func (s *Storage) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	startTime := time.Now()
	s.logger.Info("Retrieving object", "bucket", bucket, "key", key)
	s.metrics.IncrementCounter("storage.get.attempts", map[string]string{"bucket": bucket})

	objectPath := s.getObjectPath(bucket, key)

	file, err := os.Open(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Error("Object not found", "bucket", bucket, "key", key)
			s.metrics.IncrementCounter("storage.get.errors", map[string]string{"bucket": bucket, "error": "not_found"})
			return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
		}
		s.logger.Error("Failed to open file", "path", objectPath, "error", err)
		s.metrics.IncrementCounter("storage.get.errors", map[string]string{"bucket": bucket, "error": "open"})
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Get file size for metrics
	if stat, err := file.Stat(); err == nil {
		s.metrics.RecordHistogram("storage.get.bytes", float64(stat.Size()), map[string]string{"bucket": bucket})
	}

	duration := time.Since(startTime)
	s.logger.Info("Object retrieved successfully", "bucket", bucket, "key", key, "duration_ms", duration.Milliseconds())
	s.metrics.IncrementCounter("storage.get.success", map[string]string{"bucket": bucket})
	s.metrics.RecordHistogram("storage.get.duration_ms", float64(duration.Milliseconds()), map[string]string{"bucket": bucket})

	return file, nil
}

// GetWithMetadata retrieves an object with its metadata
func (s *Storage) GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.ObjectMetadata, error) {
	s.logger.Info("Retrieving object with metadata", "bucket", bucket, "key", key)

	reader, err := s.Get(ctx, bucket, key)
	if err != nil {
		return nil, &storage.ObjectMetadata{}, err
	}

	metadata, err := s.loadMetadata(bucket, key)
	if err != nil {
		reader.Close()
		s.logger.Error("Failed to load metadata", "bucket", bucket, "key", key, "error", err)
		return nil, &storage.ObjectMetadata{}, err
	}

	return reader, &metadata, nil
}

// Delete removes an object
func (s *Storage) Delete(ctx context.Context, bucket, key string) error {
	s.logger.Info("Deleting object", "bucket", bucket, "key", key)
	s.metrics.IncrementCounter("storage.delete.attempts", map[string]string{"bucket": bucket})

	objectPath := s.getObjectPath(bucket, key)
	metadataPath := s.getMetadataPath(bucket, key)

	// Remove object
	if err := os.Remove(objectPath); err != nil && !os.IsNotExist(err) {
		s.logger.Error("Failed to delete object", "path", objectPath, "error", err)
		s.metrics.IncrementCounter("storage.delete.errors", map[string]string{"bucket": bucket})
		return fmt.Errorf("failed to delete object: %w", err)
	}

	// Remove metadata
	os.Remove(metadataPath) // Ignore error if metadata doesn't exist

	s.logger.Info("Object deleted successfully", "bucket", bucket, "key", key)
	s.metrics.IncrementCounter("storage.delete.success", map[string]string{"bucket": bucket})

	return nil
}

// Exists checks if an object exists
func (s *Storage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	s.metrics.IncrementCounter("storage.exists.calls", map[string]string{"bucket": bucket})

	objectPath := s.getObjectPath(bucket, key)
	_, err := os.Stat(objectPath)
	if err == nil {
		s.logger.Info("Object exists", "bucket", bucket, "key", key)
		return true, nil
	}
	if os.IsNotExist(err) {
		s.logger.Info("Object does not exist", "bucket", bucket, "key", key)
		return false, nil
	}

	s.logger.Error("Failed to check object existence", "bucket", bucket, "key", key, "error", err)
	return false, err
}

// List returns objects in a bucket with optional prefix
func (s *Storage) List(ctx context.Context, bucket, prefix string) ([]storage.ObjectInfo, error) {
	startTime := time.Now()
	s.logger.Info("Listing objects", "bucket", bucket, "prefix", prefix)
	s.metrics.IncrementCounter("storage.list.attempts", map[string]string{"bucket": bucket})

	bucketPath := filepath.Join(s.basePath, bucket)

	var objects []storage.ObjectInfo

	err := filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories and metadata files
		if info.IsDir() || strings.HasSuffix(path, ".metadata.json") {
			return nil
		}

		// Get relative path from bucket
		relPath, _ := filepath.Rel(bucketPath, path)
		key := filepath.ToSlash(relPath) // Use forward slashes

		// Check prefix
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			return nil
		}

		// Load metadata if exists
		//metadata, _ := s.loadMetadata(bucket, key)

		objects = append(objects, storage.ObjectInfo{
			Key:          key,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			//Metadata:     &metadata,
		})

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to list objects", "bucket", bucket, "error", err)
		s.metrics.IncrementCounter("storage.list.errors", map[string]string{"bucket": bucket})
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Info("Listed objects successfully",
		"bucket", bucket,
		"prefix", prefix,
		"count", len(objects),
		"duration_ms", duration.Milliseconds())

	s.metrics.IncrementCounter("storage.list.success", map[string]string{"bucket": bucket})
	s.metrics.RecordHistogram("storage.list.count", float64(len(objects)), map[string]string{"bucket": bucket})
	s.metrics.RecordHistogram("storage.list.duration_ms", float64(duration.Milliseconds()), map[string]string{"bucket": bucket})

	return objects, nil
}

// CreateBucket creates a new bucket
func (s *Storage) CreateBucket(ctx context.Context, bucket string) error {
	s.logger.Info("Creating bucket", "bucket", bucket)
	s.metrics.IncrementCounter("storage.bucket.create.attempts", nil)

	bucketPath := filepath.Join(s.basePath, bucket)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		s.logger.Error("Failed to create bucket", "bucket", bucket, "error", err)
		s.metrics.IncrementCounter("storage.bucket.create.errors", nil)
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	s.logger.Info("Bucket created successfully", "bucket", bucket)
	s.metrics.IncrementCounter("storage.bucket.create.success", nil)
	return nil
}

// DeleteBucket removes a bucket (must be empty)
func (s *Storage) DeleteBucket(ctx context.Context, bucket string) error {
	s.logger.Info("Deleting bucket", "bucket", bucket)
	s.metrics.IncrementCounter("storage.bucket.delete.attempts", nil)

	bucketPath := filepath.Join(s.basePath, bucket)

	// Check if bucket is empty
	entries, err := os.ReadDir(bucketPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Error("Bucket not found", "bucket", bucket)
			s.metrics.IncrementCounter("storage.bucket.delete.errors", map[string]string{"error": "not_found"})
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		s.logger.Error("Failed to read bucket", "bucket", bucket, "error", err)
		s.metrics.IncrementCounter("storage.bucket.delete.errors", map[string]string{"error": "read"})
		return fmt.Errorf("failed to read bucket: %w", err)
	}

	if len(entries) > 0 {
		s.logger.Error("Bucket not empty", "bucket", bucket, "entries", len(entries))
		s.metrics.IncrementCounter("storage.bucket.delete.errors", map[string]string{"error": "not_empty"})
		return fmt.Errorf("bucket not empty: %s", bucket)
	}

	if err := os.Remove(bucketPath); err != nil {
		s.logger.Error("Failed to delete bucket", "bucket", bucket, "error", err)
		s.metrics.IncrementCounter("storage.bucket.delete.errors", map[string]string{"error": "remove"})
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	s.logger.Info("Bucket deleted successfully", "bucket", bucket)
	s.metrics.IncrementCounter("storage.bucket.delete.success", nil)
	return nil
}

// Helper methods

func (s *Storage) getObjectPath(bucket, key string) string {
	// Sanitize key to prevent directory traversal
	key = strings.TrimPrefix(key, "/")
	key = filepath.FromSlash(key)
	return filepath.Join(s.basePath, bucket, key)
}

func (s *Storage) getMetadataPath(bucket, key string) string {
	return s.getObjectPath(bucket, key) + ".metadata.json"
}

func (s *Storage) saveMetadata(bucket, key string, metadata storage.ObjectMetadata) error {
	//if metadata.ContentType == "" && metadata.Tags == nil {
	// return nil // No metadata to save
	//A}

	metadataPath := s.getMetadataPath(bucket, key)

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

func (s *Storage) loadMetadata(bucket, key string) (storage.ObjectMetadata, error) {
	metadataPath := s.getMetadataPath(bucket, key)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ObjectMetadata{}, nil // Return empty metadata
		}
		return storage.ObjectMetadata{}, err
	}

	var metadata storage.ObjectMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return storage.ObjectMetadata{}, err
	}

	return storage.ObjectMetadata{}, nil
}

// ObjectInfo represents information about an object
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	Metadata     *storage.ObjectMetadata
}
