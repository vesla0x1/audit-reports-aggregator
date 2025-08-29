package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockMetrics is a mock implementation of Metrics interface
type MockMetrics struct {
	mock.Mock
}

// IncrementCounter mocks the IncrementCounter method
func (m *MockMetrics) IncrementCounter(name string, tags map[string]string) {
	m.Called(name, tags)
}

// RecordHistogram mocks the RecordHistogram method
func (m *MockMetrics) RecordHistogram(name string, value float64, tags map[string]string) {
	m.Called(name, value, tags)
}

// RecordGauge mocks the RecordGauge method
func (m *MockMetrics) RecordGauge(name string, value float64, tags map[string]string) {
	m.Called(name, value, tags)
}
