package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"shared/domain/handler"
	"shared/domain/observability"
)

func RecoveryMiddleware(logger observability.Logger) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(ctx context.Context, req handler.Request) (resp handler.Response, err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Panic recovered", map[string]interface{}{
						"request_id": req.ID,
						"panic":      fmt.Sprintf("%v", r),
						"stack":      string(debug.Stack()),
					})

					err = fmt.Errorf("panic recovered: %v", r)
					resp = handler.Response{
						Success: false,
						Error: &handler.ErrorInfo{
							Code:      "INTERNAL_ERROR",
							Message:   "An internal error occurred",
							Retryable: false,
						},
					}
				}
			}()

			return next(ctx, req)
		}
	}
}
