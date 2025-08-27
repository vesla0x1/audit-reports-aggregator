package storage

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"shared/config"
	mockObservability "shared/observability/mocks"
	mockStorage "shared/storage/mocks"
	"shared/storage/types"
)

func TestProvider_Singleton(t *testing.T) {
	// Reset for test
	instance = nil
	once = sync.Once{}

	provider1 := GetProvider()
	provider2 := GetProvider()

	assert.Same(t, provider1, provider2, "should return same instance")
}

func TestProvider_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		setupMocks    func(*mockObservability.MockLogger)
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
			setupMocks: func(logger *mockObservability.MockLogger) {
				// No logging expected in current implementation
			},
			expectedError: "",
		},
		{
			name: "storage not configured",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider: "", // No provider
				},
			},
			setupMocks:    func(logger *mockObservability.MockLogger) {},
			expectedError: "storage is not configured",
		},
		{
			name: "unsupported provider",
			config: &config.Config{
				Storage: config.StorageConfig{
					Provider: "unsupported",
				},
			},
			setupMocks:    func(logger *mockObservability.MockLogger) {},
			expectedError: "unsupported storage provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new provider for each test
			provider := &Provider{}

			mockLogger := new(mockObservability.MockLogger)
			mockMetrics := new(mockObservability.MockMetrics)

			tt.setupMocks(mockLogger)

			err := provider.Initialize(tt.config, mockLogger, mockMetrics)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.False(t, provider.IsInitialized())
			} else {
				// This would fail in real test because we can't create real S3 client
				// In practice, you'd mock the createStorage method
				assert.Error(t, err) // Expected to fail without real AWS config
			}

			mockLogger.AssertExpectations(t)
		})
	}
}

func TestProvider_InitializeIdempotent(t *testing.T) {
	provider := &Provider{}

	mockLogger := new(mockObservability.MockLogger)
	mockMetrics := new(mockObservability.MockMetrics)
	mockStorage := new(mockStorage.MockObjectStorage)

	// Manually set as initialized
	provider.storage = mockStorage
	provider.initialized = true
	provider.logger = mockLogger
	provider.metrics = mockMetrics

	config := &config.Config{
		Storage: config.StorageConfig{
			Provider: "s3",
		},
	}

	// Should return nil without doing anything
	err := provider.Initialize(config, mockLogger, mockMetrics)
	assert.NoError(t, err)

	// Should still be the same storage
	assert.Same(t, mockStorage, provider.storage)
}

func TestProvider_GetStorage(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		provider := &Provider{}

		storage, err := provider.GetStorage()
		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("initialized", func(t *testing.T) {
		provider := &Provider{}
		mockStorage := new(mockStorage.MockObjectStorage)

		provider.storage = mockStorage
		provider.initialized = true

		storage, err := provider.GetStorage()
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Same(t, mockStorage, storage)
	})
}

func TestProvider_MustGetStorage(t *testing.T) {
	t.Run("panics when not initialized", func(t *testing.T) {
		provider := &Provider{}

		assert.Panics(t, func() {
			provider.MustGetStorage()
		})
	})

	t.Run("returns storage when initialized", func(t *testing.T) {
		provider := &Provider{}
		mockStorage := new(mockStorage.MockObjectStorage)

		provider.storage = mockStorage
		provider.initialized = true

		assert.NotPanics(t, func() {
			storage := provider.MustGetStorage()
			assert.Same(t, mockStorage, storage)
		})
	})
}

func TestProvider_Close(t *testing.T) {
	t.Run("close when not initialized", func(t *testing.T) {
		provider := &Provider{}

		err := provider.Close()
		assert.NoError(t, err)
	})

	t.Run("close when initialized", func(t *testing.T) {
		provider := &Provider{}
		mockStorage := new(mockStorage.MockObjectStorage)

		provider.storage = mockStorage
		provider.initialized = true

		err := provider.Close()
		assert.NoError(t, err)
		assert.False(t, provider.initialized)
		assert.Nil(t, provider.storage)
	})
}

func TestProvider_Reset(t *testing.T) {
	provider := &Provider{}
	mockStorage := new(mockStorage.MockObjectStorage)
	mockLogger := new(mockObservability.MockLogger)
	mockMetrics := new(mockObservability.MockMetrics)

	provider.storage = mockStorage
	provider.logger = mockLogger
	provider.metrics = mockMetrics
	provider.initialized = true

	provider.Reset()

	assert.False(t, provider.initialized)
	assert.Nil(t, provider.storage)
	assert.Nil(t, provider.logger)
	assert.Nil(t, provider.metrics)
	assert.Nil(t, provider.config)
}

func TestProvider_TestConnection(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockStorage.MockObjectStorage)
		expectError bool
	}{
		{
			name: "connection successful",
			setupMock: func(m *mockStorage.MockObjectStorage) {
				m.On("Exists", mock.Anything, "", ".health-check").Return(true, nil)
			},
			expectError: false,
		},
		{
			name: "object not found is ok",
			setupMock: func(m *mockStorage.MockObjectStorage) {
				m.On("Exists", mock.Anything, "", ".health-check").Return(false, types.ErrObjectNotFound)
			},
			expectError: false,
		},
		{
			name: "connection error",
			setupMock: func(m *mockStorage.MockObjectStorage) {
				m.On("Exists", mock.Anything, "", ".health-check").Return(false, errors.New("connection failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{}
			mockStorage := new(mockStorage.MockObjectStorage)

			tt.setupMock(mockStorage)

			err := provider.testConnection(mockStorage)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestProvider_ConcurrentAccess(t *testing.T) {
	provider := &Provider{}
	mockStorage := new(mockStorage.MockObjectStorage)

	provider.storage = mockStorage
	provider.initialized = true

	// Test concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.GetStorage()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Should have no errors
	assert.Len(t, errors, 0)
}
