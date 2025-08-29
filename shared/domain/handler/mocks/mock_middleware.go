package mocks

import (
	"context"

	"shared/domain/handler"

	"github.com/stretchr/testify/mock"
)

// MockMiddleware is a mock implementation of a Middleware function.
// Use this to test middleware chains and ordering.
type MockMiddleware struct {
	mock.Mock
	Name string
}

// Execute returns a middleware function that records calls
func (m *MockMiddleware) Execute() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			// Record that this middleware was called
			m.Called(ctx, req)

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// ExecuteWithModification returns a middleware that modifies the request
func (m *MockMiddleware) ExecuteWithModification(modifier func(handler.Request) handler.Request) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			// Record call
			m.Called(ctx, req)

			// Modify request
			modifiedReq := modifier(req)

			// Call next with modified request
			return next(ctx, modifiedReq)
		}
	}
}

// ExecuteWithResponse returns a middleware that returns a fixed response
func (m *MockMiddleware) ExecuteWithResponse(response handler.Response, err error) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			// Record call
			m.Called(ctx, req)

			// Return fixed response without calling next
			return response, err
		}
	}
}
