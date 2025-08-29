package mocks

import (
	"context"
	"io"
	"shared/domain/storage"

	"github.com/stretchr/testify/mock"
)

// MockObjectStorage is a mock implementation of ObjectStorage interface
type MockObjectStorage struct {
	mock.Mock
}

// Put mocks the Put method
func (m *MockObjectStorage) Put(ctx context.Context, bucket, key string, reader io.Reader, metadata storage.ObjectMetadata) error {
	args := m.Called(ctx, bucket, key, reader, metadata)
	return args.Error(0)
}

// Get mocks the Get method
func (m *MockObjectStorage) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, bucket, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// GetWithMetadata mocks the GetWithMetadata method
func (m *MockObjectStorage) GetWithMetadata(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.ObjectMetadata, error) {
	args := m.Called(ctx, bucket, key)

	var reader io.ReadCloser
	var metadata *storage.ObjectMetadata

	if args.Get(0) != nil {
		reader = args.Get(0).(io.ReadCloser)
	}
	if args.Get(1) != nil {
		metadata = args.Get(1).(*storage.ObjectMetadata)
	}

	return reader, metadata, args.Error(2)
}

// Delete mocks the Delete method
func (m *MockObjectStorage) Delete(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

// Exists mocks the Exists method
func (m *MockObjectStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	args := m.Called(ctx, bucket, key)
	return args.Bool(0), args.Error(1)
}

// List mocks the List method
func (m *MockObjectStorage) List(ctx context.Context, bucket, prefix string) ([]storage.ObjectInfo, error) {
	args := m.Called(ctx, bucket, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.ObjectInfo), args.Error(1)
}

// CreateBucket mocks the CreateBucket method
func (m *MockObjectStorage) CreateBucket(ctx context.Context, bucket string) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

// DeleteBucket mocks the DeleteBucket method
func (m *MockObjectStorage) DeleteBucket(ctx context.Context, bucket string) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

// MockStorageProvider is a mock implementation of StorageProvider interface
type MockStorageProvider struct {
	mock.Mock
}

// GetStorage mocks the GetStorage method
func (m *MockStorageProvider) GetStorage() (storage.ObjectStorage, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(storage.ObjectStorage), args.Error(1)
}

// Close mocks the Close method
func (m *MockStorageProvider) Close() error {
	args := m.Called()
	return args.Error(0)
}
