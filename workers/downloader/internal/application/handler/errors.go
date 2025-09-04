package handler

import (
	"fmt"
)

func ErrHandlerInvalidPayload(err error) error {
	return fmt.Errorf("invalid payload: %w", err)
}

func ErrHandlerParseRequest(err error) error {
	return fmt.Errorf("failed to parse request: %w", err)
}

func ErrHandlerUnmarshal(err error) error {
	return fmt.Errorf("failed to unmarshal payload: %w", err)
}
