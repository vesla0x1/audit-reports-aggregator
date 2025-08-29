package middleware

import (
	"context"
	"encoding/json"
	"time"

	"shared/domain/handler"

	"github.com/google/uuid"
)

func ValidationMiddleware() handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			// Ensure request has an ID
			if req.ID == "" {
				req.ID = uuid.New().String()
			}

			// Ensure request has a timestamp
			if req.Timestamp.IsZero() {
				req.Timestamp = time.Now().UTC()
			}

			// Validate request type
			if req.Type == "" {
				return handler.Response{
					Success: false,
					Error: &handler.ErrorInfo{
						Code:      "VALIDATION_ERROR",
						Message:   "Request type is required",
						Retryable: false,
					},
				}, nil
			}

			// Validate payload
			if len(req.Payload) == 0 {
				return handler.Response{
					Success: false,
					Error: &handler.ErrorInfo{
						Code:      "VALIDATION_ERROR",
						Message:   "Request payload is required",
						Retryable: false,
					},
				}, nil
			}

			// Validate JSON payload
			if !json.Valid(req.Payload) {
				return handler.Response{
					Success: false,
					Error: &handler.ErrorInfo{
						Code:      "VALIDATION_ERROR",
						Message:   "Invalid JSON payload",
						Retryable: false,
					},
				}, nil
			}

			// Initialize metadata if nil
			if req.Metadata == nil {
				req.Metadata = make(map[string]string)
			}

			return next(ctx, req)
		}
	}
}
