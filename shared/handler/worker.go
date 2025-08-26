package handler

import (
	"context"
)

// Worker defines the interface that each worker must implement.
// This is the core business logic interface that remains platform-agnostic.
// Workers process requests and return responses without knowing about
// the underlying platform or transport mechanism.
type Worker interface {
	// Name returns the worker name for identification.
	// This is used for logging, metrics, and routing.
	Name() string

	// Process handles the actual work with platform-agnostic request/response.
	// This is where the business logic lives. The worker should unmarshal
	// the request payload, process it, and return an appropriate response.
	Process(ctx context.Context, request Request) (Response, error)

	// Health checks if the worker is healthy and ready to process requests.
	// This should verify that all dependencies (database, storage, etc.) are accessible.
	Health(ctx context.Context) error
}
