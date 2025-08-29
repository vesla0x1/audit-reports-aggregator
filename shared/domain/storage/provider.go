package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"shared/config"
)

// Provider manages storage lifecycle and ensures singleton behavior
type Provider struct {
	storage     ObjectStorage
	config      *config.Config
	mu          sync.RWMutex
	initialized bool
}

var (
	instance *Provider
	once     sync.Once
)

// GetProvider returns the singleton storage provider instance
// This ensures only one S3 client exists throughout the application lifecycle
func GetProvider() *Provider {
	once.Do(func() {
		instance = &Provider{}
	})
	return instance
}

// Initialize initializes the storage provider with configuration and dependencies
// This should be called once at application startup
func (p *Provider) Initialize(cfg *config.Config, factory StorageFactory) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil // Already initialized
	}

	// Check if storage is configured
	if !cfg.IsStorageEnabled() {
		return fmt.Errorf("storage is not configured")
	}
	// Create storage implementation using internal factory
	storage, err := factory.CreateStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Test connection
	if err := p.testConnection(storage); err != nil {
		return fmt.Errorf("failed to verify storage connection: %w", err)
	}

	//logger.Info("storage initialized successfully", map[string]interface{}{
	//	"provider": cfg.GetStorageProvider(),
	//})

	p.storage = storage
	p.config = cfg
	p.initialized = true

	return nil
}

// MustInitialize initializes the storage provider and panics on error
// Use this for application initialization where errors are fatal
func (p *Provider) MustInitialize(cfg *config.Config, factory StorageFactory) {
	if err := p.Initialize(cfg, factory); err != nil {
		panic(fmt.Sprintf("failed to initialize storage: %v", err))
	}
}

// GetStorage returns the storage instance
// Returns error if storage hasn't been initialized
func (p *Provider) GetStorage() (ObjectStorage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized || p.storage == nil {
		return nil, fmt.Errorf("storage not initialized; call Initialize() first")
	}

	return p.storage, nil
}

// MustGetStorage returns the storage or panics if not initialized
// Use this when you're certain storage has been initialized
func (p *Provider) MustGetStorage() ObjectStorage {
	storage, err := p.GetStorage()
	if err != nil {
		panic(fmt.Sprintf("failed to get storage: %v", err))
	}
	return storage
}

// IsInitialized returns whether storage has been initialized
func (p *Provider) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}

// Close cleanly shuts down the storage provider
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil
	}

	//p.logger.Info("closing storage provider", nil)

	// S3 client doesn't need explicit cleanup, but we reset state
	p.storage = nil
	p.initialized = false

	return nil
}

// Reset resets the provider (useful for testing)
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.storage = nil
	p.config = nil
	p.initialized = false
}

// testConnection tests the storage connection
func (p *Provider) testConnection(storage ObjectStorage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try a simple operation to verify connection
	_, err := storage.Exists(ctx, "", ".health-check")
	if err != nil {
		// Check if it's a "not found" error (which is fine) or a connection error
		if !isNotFoundError(err) {
			return err
		}
	}

	return nil
}

// isNotFoundError checks if error is a not found error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's the specific ErrObjectNotFound
	if errors.Is(err, ErrObjectNotFound) {
		return true
	}

	// Check for common S3 not found error strings
	return err != nil &&
		(contains(err.Error(), "NoSuchKey") ||
			contains(err.Error(), "NotFound") ||
			contains(err.Error(), "404"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && strings.Contains(s, substr)
}
