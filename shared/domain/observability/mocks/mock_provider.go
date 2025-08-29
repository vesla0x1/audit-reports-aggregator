package mocks

import (
	"shared/domain/observability"

	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of ObservabilityProvider interface
type MockProvider struct {
	mock.Mock
}

// GetLogger mocks the GetLogger method
func (m *MockProvider) GetLogger() (observability.Logger, error) {
	args := m.Called()
	if logger, ok := args.Get(0).(observability.Logger); ok {
		return logger, args.Error(1)
	}
	return nil, args.Error(1)
}

// GetMetrics mocks the GetMetrics method
func (m *MockProvider) GetMetrics() (observability.Metrics, error) {
	args := m.Called()
	if metrics, ok := args.Get(0).(observability.Metrics); ok {
		return metrics, args.Error(1)
	}
	return nil, args.Error(1)
}

// GetObservability mocks the GetObservability method
func (m *MockProvider) GetObservability() (observability.Logger, observability.Metrics, error) {
	args := m.Called()

	var logger observability.Logger
	var metrics observability.Metrics

	if l, ok := args.Get(0).(observability.Logger); ok {
		logger = l
	}
	if m, ok := args.Get(1).(observability.Metrics); ok {
		metrics = m
	}

	return logger, metrics, args.Error(2)
}

// MustGetLogger mocks the MustGetLogger method
func (m *MockProvider) MustGetLogger(component string) observability.Logger {
	args := m.Called(component)
	if logger, ok := args.Get(0).(observability.Logger); ok {
		return logger
	}
	return nil
}

// MustGetMetrics mocks the MustGetMetrics method
func (m *MockProvider) MustGetMetrics() observability.Metrics {
	args := m.Called()
	if metrics, ok := args.Get(0).(observability.Metrics); ok {
		return metrics
	}
	return nil
}

// IsInitialized mocks the IsInitialized method
func (m *MockProvider) IsInitialized() bool {
	args := m.Called()
	return args.Bool(0)
}
