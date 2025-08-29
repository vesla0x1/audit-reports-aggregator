package middleware

import (
	"context"
	"fmt"
	"math"
	"time"

	"shared/config"
	"shared/domain/handler"
)

func RetryMiddleware(cfg *config.RetryConfig) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			var lastResp handler.Response
			var lastErr error

			for attempt := 0; attempt <= cfg.MaxAttempts; attempt++ {
				// Process request
				resp, err := next(ctx, req)

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
						return handler.Response{
							Success: false,
							Error: &handler.ErrorInfo{
								Code:      "CANCELLED",
								Message:   "Request cancelled during retry",
								Retryable: false,
							},
						}, ctx.Err()
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
				lastResp.Error.Message = fmt.Sprintf("Failed after %d retries", cfg.MaxAttempts)
			}

			return lastResp, nil
		}
	}
}

func isRetryable(resp handler.Response, err error) bool {
	// Don't retry if context is cancelled
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check response error
	if resp.Error != nil {
		// Use explicit retryable flag if set
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

func calculateBackoff(attempt int, cfg *config.RetryConfig) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt))

	// Cap at max backoff
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	return time.Duration(backoff)
}
