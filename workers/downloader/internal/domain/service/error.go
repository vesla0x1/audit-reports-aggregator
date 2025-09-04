package service

import (
	"errors"
	"fmt"
)

var (
	ErrFileTooLarge = errors.New("file exceeds maximum size")
)

// Error wrapping functions with context
func ErrRequestCreation(err error) error {
	return fmt.Errorf("failed to create HTTP request: %w", err)
}

func ErrHTTPRequest(err error) error {
	return fmt.Errorf("HTTP request failed: %w", err)
}

func ErrUnexpectedStatus(statusCode int) error {
	return fmt.Errorf("unexpected HTTP status code: %d", statusCode)
}

func ErrReadResponse(err error) error {
	return fmt.Errorf("failed to read response: %w", err)
}
