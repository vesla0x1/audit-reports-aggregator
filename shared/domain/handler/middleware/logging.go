package middleware

import (
	"context"
	"time"

	"shared/domain/handler"
	"shared/domain/observability"
)

func LoggingMiddleware(logger observability.Logger) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			start := time.Now()

			// Log request start
			logger.Info("Processing request", map[string]interface{}{
				"request_id":   req.ID,
				"type":         req.Type,
				"source":       req.Source,
				"payload_size": len(req.Payload),
			})

			// Process request
			resp, err := next(ctx, req)

			// Calculate duration
			duration := time.Since(start)

			// Log completion
			fields := map[string]interface{}{
				"request_id":  req.ID,
				"duration_ms": duration.Milliseconds(),
			}

			if err != nil {
				fields["error"] = err.Error()
				logger.Error("Request failed", fields)
			} else if !resp.Success {
				if resp.Error != nil {
					fields["error_code"] = resp.Error.Code
					fields["error_msg"] = resp.Error.Message
				}
				logger.Info("Request completed with failure", fields)
			} else {
				logger.Info("Request completed successfully", fields)
			}

			return resp, err
		}
	}
}
