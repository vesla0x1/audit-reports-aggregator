package mocks

import (
	"context"

	"shared/handler"

	"github.com/stretchr/testify/mock"
)

// MockWorker is a mock implementation of the Worker interface.
// Use this to test handlers and middleware without real business logic.
type MockWorker struct {
	mock.Mock
}

// Ensure MockWorker implements handler.Worker
var _ handler.Worker = (*MockWorker)(nil)

// Name returns the mock worker name
func (m *MockWorker) Name() string {
	args := m.Called()
	return args.String(0)
}

// Process mocks the request processing
func (m *MockWorker) Process(ctx context.Context, request handler.Request) (handler.Response, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(handler.Response), args.Error(1)
}

// Health mocks the health check
func (m *MockWorker) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ExpectProcess sets up an expectation for Process with a specific request type
func (m *MockWorker) ExpectProcess(requestType string, response handler.Response, err error) *mock.Call {
	return m.On("Process",
		mock.Anything, // ctx
		mock.MatchedBy(func(req handler.Request) bool {
			return req.Type == requestType
		}),
	).Return(response, err)
}

// ExpectProcessAny sets up an expectation for any Process call
func (m *MockWorker) ExpectProcessAny(response handler.Response, err error) *mock.Call {
	return m.On("Process", mock.Anything, mock.Anything).Return(response, err)
}
