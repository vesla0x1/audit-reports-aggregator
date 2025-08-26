package mocks

import (
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

// CreateOpenFaaS mocks OpenFaaS handler creation
func (m *MockFactory) CreateOpenFaaS() *handler.Handler {
	args := m.Called()
	if h, ok := args.Get(0).(*handler.Handler); ok {
		return h
	}
	return nil
}

// WithConfig mocks configuration setting
func (m *MockFactory) WithConfig(config *handler.Config) *MockFactory {
	m.Called(config)
	return m
}
