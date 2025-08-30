package storage_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"shared/config"
	"shared/domain/storage"
	mockStorage "shared/domain/storage/mocks"
)

// Create a test factory that returns our mock
type testStorageFactory struct {
	mockStorage storage.ObjectStorage
	shouldError bool
}

func (f *testStorageFactory) Create(cfg *config.Config) (storage.ObjectStorage, error) {
	if f.shouldError {
		return nil, errors.New("failed to create storage")
	}
	return f.mockStorage, nil
}

func TestProvider_Singleton(t *testing.T) {
	provider1 := storage.GetProvider()
	provider2 := storage.GetProvider()

	assert.Same(t, provider1, provider2, "should return same instance")
}

func TestProvider_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		setupMocks    func(*mockStorage.MockObjectStorage)
		factoryError  bool
		expectedError string
	}{
		{
			name: "successful initialization",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider:   "s3",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
					S3: config.S3Config{
						Region: "us-east-1",
						Bucket: "test-bucket",
					},
				},
			},
			setupMocks: func(mockObj *mockStorage.MockObjectStorage) {
				// Test connection will call Exists
				mockObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound)
			},
			factoryError:  false,
			expectedError: "",
		},
		{
			name: "storage not configured",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider: "", // No provider
				},
			},
			setupMocks:    func(mockObj *mockStorage.MockObjectStorage) {},
			expectedError: "storage is not configured",
		},
		{
			name: "factory error",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider: "s3",
					S3: config.S3Config{
						Region: "us-east-1",
						Bucket: "test-bucket",
					},
				},
			},
			setupMocks:    func(mockObj *mockStorage.MockObjectStorage) {},
			factoryError:  true,
			expectedError: "failed to create storage",
		},
		{
			name: "connection test fails",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider: "s3",
					S3: config.S3Config{
						Region: "us-east-1",
						Bucket: "test-bucket",
					},
				},
			},
			setupMocks: func(mockObj *mockStorage.MockObjectStorage) {
				// Test connection will fail
				mockObj.On("Exists", mock.Anything, "", ".health-check").Return(false, errors.New("connection failed"))
			},
			expectedError: "failed to verify storage connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset provider for each test
			provider := storage.GetProvider()
			provider.Reset()

			mockStorageObj := new(mockStorage.MockObjectStorage)
			tt.setupMocks(mockStorageObj)

			factory := &testStorageFactory{
				mockStorage: mockStorageObj,
				shouldError: tt.factoryError,
			}

			err := provider.Initialize(tt.config, factory)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.False(t, provider.IsInitialized())
			} else {
				assert.NoError(t, err)
				assert.True(t, provider.IsInitialized())
			}

			mockStorageObj.AssertExpectations(t)
		})
	}
}

func TestProvider_InitializeIdempotent(t *testing.T) {
	provider := storage.GetProvider()
	provider.Reset()

	mockStorageObj := new(mockStorage.MockObjectStorage)

	// First initialization
	mockStorageObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound).Once()

	factory := &testStorageFactory{
		mockStorage: mockStorageObj,
	}

	config := &config.Config{
		Storage: config.StorageConfig{
			Provider: "s3",
			S3: config.S3Config{
				Region: "us-east-1",
				Bucket: "test-bucket",
			},
		},
	}

	// First call should initialize
	err := provider.Initialize(config, factory)
	assert.NoError(t, err)
	assert.True(t, provider.IsInitialized())

	// Second call should return immediately without doing anything
	err = provider.Initialize(config, factory)
	assert.NoError(t, err)
	assert.True(t, provider.IsInitialized())

	// Mock should have been called only once
	mockStorageObj.AssertExpectations(t)
}

func TestProvider_GetStorage(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		provider := storage.GetProvider()
		provider.Reset()

		storageObj, err := provider.Get()
		assert.Error(t, err)
		assert.Nil(t, storageObj)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("initialized", func(t *testing.T) {
		provider := storage.GetProvider()
		provider.Reset()

		mockStorageObj := new(mockStorage.MockObjectStorage)
		mockStorageObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound)

		factory := &testStorageFactory{
			mockStorage: mockStorageObj,
		}

		config := &config.Config{
			Storage: config.StorageConfig{
				Provider: "s3",
				S3: config.S3Config{
					Region: "us-east-1",
					Bucket: "test-bucket",
				},
			},
		}

		err := provider.Initialize(config, factory)
		assert.NoError(t, err)

		storageObj, err := provider.Get()
		assert.NoError(t, err)
		assert.NotNil(t, storageObj)
		assert.Same(t, mockStorageObj, storageObj)

		mockStorageObj.AssertExpectations(t)
	})
}

func TestProvider_MustGetStorage(t *testing.T) {
	t.Run("panics when not initialized", func(t *testing.T) {
		provider := storage.GetProvider()
		provider.Reset()

		assert.Panics(t, func() {
			provider.MustGet()
		})
	})

	t.Run("returns storage when initialized", func(t *testing.T) {
		provider := storage.GetProvider()
		provider.Reset()

		mockStorageObj := new(mockStorage.MockObjectStorage)
		mockStorageObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound)

		factory := &testStorageFactory{
			mockStorage: mockStorageObj,
		}

		config := &config.Config{
			Storage: config.StorageConfig{
				Provider: "s3",
				S3: config.S3Config{
					Region: "us-east-1",
					Bucket: "test-bucket",
				},
			},
		}

		err := provider.Initialize(config, factory)
		assert.NoError(t, err)

		assert.NotPanics(t, func() {
			storageObj := provider.MustGet()
			assert.Same(t, mockStorageObj, storageObj)
		})

		mockStorageObj.AssertExpectations(t)
	})
}

func TestProvider_Reset(t *testing.T) {
	provider := storage.GetProvider()
	provider.Reset()

	mockStorageObj := new(mockStorage.MockObjectStorage)
	mockStorageObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound)

	factory := &testStorageFactory{
		mockStorage: mockStorageObj,
	}

	config := &config.Config{
		Storage: config.StorageConfig{
			Provider: "s3",
			S3: config.S3Config{
				Region: "us-east-1",
				Bucket: "test-bucket",
			},
		},
	}

	err := provider.Initialize(config, factory)
	assert.NoError(t, err)
	assert.True(t, provider.IsInitialized())

	provider.Reset()

	assert.False(t, provider.IsInitialized())

	mockStorageObj.AssertExpectations(t)
}

func TestProvider_ConcurrentAccess(t *testing.T) {
	provider := storage.GetProvider()
	provider.Reset()

	mockStorageObj := new(mockStorage.MockObjectStorage)
	mockStorageObj.On("Exists", mock.Anything, "", ".health-check").Return(false, storage.ErrObjectNotFound)

	factory := &testStorageFactory{
		mockStorage: mockStorageObj,
	}

	config := &config.Config{
		Storage: config.StorageConfig{
			Provider: "s3",
			S3: config.S3Config{
				Region: "us-east-1",
				Bucket: "test-bucket",
			},
		},
	}

	err := provider.Initialize(config, factory)
	assert.NoError(t, err)

	// Test concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.Get()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Should have no errors
	assert.Len(t, errors, 0)

	mockStorageObj.AssertExpectations(t)
}
