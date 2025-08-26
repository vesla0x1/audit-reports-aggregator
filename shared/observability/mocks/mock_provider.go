package mocks

import (
	"shared/observability"

	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of Provider interface
type MockProvider struct {
	mock.Mock
}

// Logger mocks the Logger method
func (m *MockProvider) Logger(component string) observability.Logger {
	args := m.Called(component)
	if logger, ok := args.Get(0).(observability.Logger); ok {
		return logger
	}
	return nil
}

// Metrics mocks the Metrics method
func (m *MockProvider) Metrics(component string) observability.Metrics {
	args := m.Called(component)
	if metrics, ok := args.Get(0).(observability.Metrics); ok {
		return metrics
	}
	return nil
}

// Close mocks the Close method
func (m *MockProvider) Close() error {
	args := m.Called()
	return args.Error(0)
}
