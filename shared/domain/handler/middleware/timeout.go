package middleware

import (
	"context"
	"fmt"
	"time"

	"shared/domain/handler"
)

func TimeoutMiddleware(timeout time.Duration) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (handler.Response, error) {
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			type result struct {
				resp handler.Response
				err  error
			}
			resultChan := make(chan result, 1)

			go func() {
				resp, err := next(timeoutCtx, req)
				resultChan <- result{resp, err}
			}()

			select {
			case res := <-resultChan:
				return res.resp, res.err
			case <-timeoutCtx.Done():
				return handler.Response{
					Success: false,
					Error: &handler.ErrorInfo{
						Code:      "TIMEOUT",
						Message:   fmt.Sprintf("Request timed out after %v", timeout),
						Retryable: true,
					},
				}, timeoutCtx.Err()
			}
		}
	}
}
