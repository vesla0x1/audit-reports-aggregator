package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockMetrics is a mock implementation of Metrics interface
type MockMetrics struct {
	mock.Mock
}

// RecordSuccess mocks the RecordSuccess method
func (m *MockMetrics) RecordSuccess(operationType string) {
	m.Called(operationType)
}

// RecordError mocks the RecordError method
func (m *MockMetrics) RecordError(operationType string, errorType string) {
	m.Called(operationType, errorType)
}

// RecordDuration mocks the RecordDuration method
func (m *MockMetrics) RecordDuration(operation string, duration float64) {
	m.Called(operation, duration)
}

// RecordFileSize mocks the RecordFileSize method
func (m *MockMetrics) RecordFileSize(fileType string, bytes int64) {
	m.Called(fileType, bytes)
}

// StartOperation mocks the StartOperation method
func (m *MockMetrics) StartOperation(operation string) {
	m.Called(operation)
}

// EndOperation mocks the EndOperation method
func (m *MockMetrics) EndOperation(operation string) {
	m.Called(operation)
}
