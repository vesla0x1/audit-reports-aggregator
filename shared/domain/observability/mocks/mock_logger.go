// Package mocks provides mock implementations for testing
package mocks

import (
	"shared/domain/observability"

	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of Logger interface
type MockLogger struct {
	mock.Mock
}

// Info mocks the Info method
func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

// Error mocks the Error method
func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

// WithFields mocks the WithFields method
func (m *MockLogger) WithFields(fields map[string]interface{}) observability.Logger {
	args := m.Called(fields)
	if logger, ok := args.Get(0).(observability.Logger); ok {
		return logger
	}
	return m
}
