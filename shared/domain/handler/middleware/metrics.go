package middleware

import (
	"context"
	"time"

	"shared/domain/handler"
	"shared/domain/observability"
)

func MetricsMiddleware(metrics observability.Metrics) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			start := time.Now()

			// Record request
			metrics.IncrementCounter("handler.requests", map[string]string{
				"type":   req.Type,
				"source": req.Source,
			})

			// Process request
			resp, err := next(ctx, req)

			// Record duration
			duration := time.Since(start).Seconds()
			metrics.RecordHistogram("handler.duration", duration, map[string]string{
				"type":   req.Type,
				"source": req.Source,
			})

			// Record outcome
			tags := map[string]string{
				"type":   req.Type,
				"source": req.Source,
			}

			if err != nil || !resp.Success {
				if resp.Error != nil {
					tags["error_code"] = resp.Error.Code
				}
				metrics.IncrementCounter("handler.errors", tags)
			} else {
				metrics.IncrementCounter("handler.success", tags)
			}

			return resp, err
		}
	}
}
