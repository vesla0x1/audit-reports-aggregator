// Package mocks provides mock implementations for testing
package mocks

import (
	"context"

	"shared/observability"
	"shared/observability/types"

	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of Logger interface
type MockLogger struct {
	mock.Mock
}

// Info mocks the Info method
func (m *MockLogger) Info(ctx context.Context, msg string, fields types.Fields) {
	m.Called(ctx, msg, fields)
}

// Error mocks the Error method
func (m *MockLogger) Error(ctx context.Context, msg string, err error, fields types.Fields) {
	m.Called(ctx, msg, err, fields)
}

// Warn mocks the Warn method
func (m *MockLogger) Warn(ctx context.Context, msg string, fields types.Fields) {
	m.Called(ctx, msg, fields)
}

// Debug mocks the Debug method
func (m *MockLogger) Debug(ctx context.Context, msg string, fields types.Fields) {
	m.Called(ctx, msg, fields)
}

// WithFields mocks the WithFields method
func (m *MockLogger) WithFields(fields types.Fields) observability.Logger {
	args := m.Called(fields)
	if logger, ok := args.Get(0).(observability.Logger); ok {
		return logger
	}
	return m
}
