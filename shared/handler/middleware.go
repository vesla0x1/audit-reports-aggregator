package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"runtime/debug"
	"time"

	"shared/config"
	"shared/observability"
	"shared/observability/types"

	"github.com/google/uuid"
)

// LoggingMiddleware adds structured logging to request processing
func LoggingMiddleware(provider observability.Provider) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			logger := provider.Logger("handler")

			// Extract context values
			workerName, _ := ctx.Value("worker").(string)
			platform, _ := ctx.Value("platform").(string)

			// Create logger with request context
			requestLogger := logger.WithFields(types.Fields{
				"request_id": req.ID,
				"type":       req.Type,
				"source":     req.Source,
				"worker":     workerName,
				"platform":   platform,
			})

			// Log request start
			requestLogger.Info(ctx, "Processing request", types.Fields{
				"payload_size": len(req.Payload),
			})

			// Track timing
			start := time.Now()

			// Process request
			resp, err := next(ctx, req)

			// Calculate duration
			duration := time.Since(start)

			// Log completion
			if err != nil {
				requestLogger.Error(ctx, "Request failed with error", err, types.Fields{
					"duration_ms": duration.Milliseconds(),
				})
			} else if !resp.Success {
				requestLogger.Warn(ctx, "Request completed with failure", types.Fields{
					"error_code":  resp.Error.Code,
					"error_msg":   resp.Error.Message,
					"duration_ms": duration.Milliseconds(),
				})
			} else {
				requestLogger.Info(ctx, "Request completed successfully", types.Fields{
					"duration_ms": duration.Milliseconds(),
				})
			}

			// Add duration to response
			resp.Duration = duration

			return resp, err
		}
	}
}

// MetricsMiddleware records metrics for request processing
func MetricsMiddleware(provider observability.Provider) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			metrics := provider.Metrics("handler")

			// Get worker name from context
			workerName, _ := ctx.Value("worker").(string)
			if workerName == "" {
				workerName = "unknown"
			}

			// Start operation tracking
			metrics.StartOperation(workerName)
			defer metrics.EndOperation(workerName)

			// Track timing
			start := time.Now()

			// Process request
			resp, err := next(ctx, req)

			// Record duration
			duration := time.Since(start).Seconds()
			metrics.RecordDuration(workerName, duration)

			// Record outcome
			if err != nil {
				metrics.RecordError(workerName, "processing_error")
			} else if !resp.Success {
				errorType := "unknown_error"
				if resp.Error != nil {
					errorType = resp.Error.Code
				}
				metrics.RecordError(workerName, errorType)
			} else {
				metrics.RecordSuccess(workerName)
			}

			return resp, err
		}
	}
}

// Recovery recovers from panics and returns an error response.
// This middleware should be the outermost layer to catch all panics.
func RecoveryMiddleware(provider observability.Provider) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (resp Response, err error) {
			logger := provider.Logger("handler")
			metrics := provider.Metrics("handler")

			defer func() {
				if r := recover(); r != nil {
					// Log the panic with stack trace
					logger.Error(ctx, "Panic recovered", fmt.Errorf("%v", r), types.Fields{
						"request_id": req.ID,
						"worker":     ctx.Value("worker"),
						"stack":      string(debug.Stack()),
					})

					// Record panic metric
					metrics.RecordError("panic", "panic_recovered")

					// Return error response
					resp = NewErrorResponse(
						req.ID,
						"INTERNAL_ERROR",
						"An internal error occurred",
						"", // Don't expose panic details to client
					)

					// Set error for middleware chain
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()

			return next(ctx, req)
		}
	}
}

// Tracing adds distributed tracing context to requests.
// It ensures each request has a trace ID for correlation across services.
func TracingMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			// Extract or generate trace ID
			traceID := extractTraceID(req)
			if traceID == "" {
				traceID = uuid.New().String()
			}

			// Extract or generate span ID
			spanID := uuid.New().String()

			// Get parent span ID if exists
			parentSpanID, _ := req.Metadata["parent_span_id"]

			// Add to context
			ctx = context.WithValue(ctx, "trace_id", traceID)
			ctx = context.WithValue(ctx, "span_id", spanID)
			if parentSpanID != "" {
				ctx = context.WithValue(ctx, "parent_span_id", parentSpanID)
			}

			// Add to request metadata for downstream services
			req.Metadata["trace_id"] = traceID
			req.Metadata["span_id"] = spanID

			// Process request
			resp, err := next(ctx, req)

			// Add trace info to response
			if resp.Metadata == nil {
				resp.Metadata = make(map[string]string)
			}
			resp.Metadata["trace_id"] = traceID
			resp.Metadata["span_id"] = spanID

			return resp, err
		}
	}
}

// Timeout enforces a timeout on request processing.
// If the timeout is exceeded, it returns a timeout error response.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			// Create timeout context
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Channel for result
			type result struct {
				resp Response
				err  error
			}
			resultChan := make(chan result, 1)

			// Process in goroutine
			go func() {
				resp, err := next(timeoutCtx, req)
				resultChan <- result{resp, err}
			}()

			// Wait for result or timeout
			select {
			case res := <-resultChan:
				return res.resp, res.err

			case <-timeoutCtx.Done():
				return NewErrorResponse(
					req.ID,
					"TIMEOUT",
					"Request processing timed out",
					fmt.Sprintf("Exceeded timeout of %v", timeout),
				), timeoutCtx.Err()
			}
		}
	}
}

// Retry adds retry logic for transient failures.
// It uses exponential backoff between retry attempts.
func RetryMiddleware(cfg *config.RetryConfig) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
			var lastResp Response
			var lastErr error

			for attempt := 0; attempt <= cfg.MaxAttempts; attempt++ {
				// Add attempt number to context
				attemptCtx := context.WithValue(ctx, "retry_attempt", attempt)

				// Process request
				resp, err := next(attemptCtx, req)

				// Success - return immediately
				if err == nil && resp.Success {
					return resp, nil
				}

				// Check if error is retryable
				if !isRetryable(resp, err) {
					return resp, err
				}

				lastResp = resp
				lastErr = err

				// Don't sleep after last attempt
				if attempt < cfg.MaxAttempts {
					backoff := calculateBackoff(attempt, cfg)

					// Check context cancellation during backoff
					select {
					case <-ctx.Done():
						return NewErrorResponse(
							req.ID,
							"CANCELLED",
							"Request cancelled during retry",
							"",
						), ctx.Err()
					case <-time.After(backoff):
						// Continue to next retry
					}
				}
			}

			// All retries exhausted
			if lastErr != nil {
				return lastResp, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
			}

			// Update error message to indicate retries were exhausted
			if lastResp.Error != nil {
				lastResp.Error.Details = fmt.Sprintf("Failed after %d retries", cfg.MaxAttempts)
			}

			return lastResp, nil
		}
	}
}

// Validation validates and enriches incoming requests.
// It ensures requests have required fields and adds defaults where needed.
func ValidationMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request) (Response, error) {
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
				return NewErrorResponse(
					req.ID,
					"VALIDATION_ERROR",
					"Request type is required",
					"Missing 'type' field in request",
				), nil
			}

			// Validate payload
			if len(req.Payload) == 0 {
				return NewErrorResponse(
					req.ID,
					"VALIDATION_ERROR",
					"Request payload is required",
					"Empty payload",
				), nil
			}

			// Validate JSON payload
			if !json.Valid(req.Payload) {
				return NewErrorResponse(
					req.ID,
					"VALIDATION_ERROR",
					"Invalid JSON payload",
					"Payload must be valid JSON",
				), nil
			}

			// Initialize metadata if nil
			if req.Metadata == nil {
				req.Metadata = make(map[string]string)
			}

			// Add validation timestamp
			req.Metadata["validated_at"] = time.Now().UTC().Format(time.RFC3339)

			return next(ctx, req)
		}
	}
}

// isRetryable determines if a response/error is retryable
func isRetryable(resp Response, err error) bool {
	// Don't retry if context is cancelled
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check response error code
	if resp.Error != nil {
		// Explicit retryable flag
		if resp.Error.Retryable {
			return true
		}

		// Check known retryable error codes
		retryableCodes := map[string]bool{
			"TIMEOUT":             true,
			"NETWORK_ERROR":       true,
			"RATE_LIMITED":        true,
			"TEMPORARY_ERROR":     true,
			"SERVICE_UNAVAILABLE": true,
			"GATEWAY_TIMEOUT":     true,
		}

		return retryableCodes[resp.Error.Code]
	}

	// Retry on generic errors
	return err != nil
}

// calculateBackoff calculates the backoff duration for a retry attempt
func calculateBackoff(attempt int, cfg *config.RetryConfig) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt))

	// Cap at max backoff
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	return time.Duration(backoff)
}

// extractTraceID attempts to extract trace ID from various sources
func extractTraceID(req Request) string {
	// Check metadata for common trace ID keys
	traceKeys := []string{
		"trace_id",
		"x-trace-id",
		"x-b3-traceid",
		"x-request-id",
		"correlation-id",
	}

	for _, key := range traceKeys {
		if val, ok := req.Metadata[key]; ok && val != "" {
			return val
		}
	}

	return ""
}
