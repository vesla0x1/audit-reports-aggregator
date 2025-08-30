package storage

import (
	"context"
	"io"
	"shared/domain/base"
)

// ObjectStorage defines the interface for object storage operations
// This interface abstracts the underlying storage implementation,
// allowing for easy swapping between different providers (S3, MinIO, etc.)
type ObjectStorage interface {
	// Put stores an object in the specified bucket with the given key
	Put(ctx context.Context, bucket, key string, reader io.Reader, metadata ObjectMetadata) error

	// Get retrieves an object from the specified bucket by key
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)

	// GetWithMetadata retrieves an object along with its metadata
	GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectMetadata, error)

	// Delete removes an object from the specified bucket
	Delete(ctx context.Context, bucket, key string) error

	// Exists checks if an object exists in the specified bucket
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// List returns a list of objects in the specified bucket with optional prefix
	List(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)

	// CreateBucket creates a new bucket if it doesn't exist
	CreateBucket(ctx context.Context, bucket string) error

	// DeleteBucket removes a bucket (must be empty)
	DeleteBucket(ctx context.Context, bucket string) error
}

type StorageFactory interface {
	base.Factory[ObjectStorage]
}
