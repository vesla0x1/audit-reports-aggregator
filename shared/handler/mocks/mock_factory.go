package mocks

import (
	"shared/config"
	"shared/handler"

	"github.com/stretchr/testify/mock"
)

// MockFactory is a mock implementation of the Factory.
// Use this to test factory creation patterns.
type MockFactory struct {
	mock.Mock
}

// Create mocks handler creation
func (m *MockFactory) Create() *handler.Handler {
	args := m.Called()
	if h, ok := args.Get(0).(*handler.Handler); ok {
		return h
	}
	return nil
}

// CreateHTTP mocks HTTP handler creation
func (m *MockFactory) CreateHTTP() *handler.Handler {
	args := m.Called()
	if h, ok := args.Get(0).(*handler.Handler); ok {
		return h
	}
	return nil
}

// WithHandlerConfig mocks handler configuration setting
func (m *MockFactory) WithHandlerConfig(cfg config.HandlerConfig) *MockFactory {
	m.Called(cfg)
	return m
}

// WithRetryConfig mocks retry configuration setting
func (m *MockFactory) WithRetryConfig(cfg config.RetryConfig) *MockFactory {
	m.Called(cfg)
	return m
}
