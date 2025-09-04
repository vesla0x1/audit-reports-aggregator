package downloadresult

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyContent = errors.New("download content cannot be empty")
	ErrEmptyUrl     = errors.New("download URL cannot be empty")
)

func ErrSizeExceeded(maxLen int64) error {
	return fmt.Errorf("content size exceeds maximum %d", maxLen)
}

func ErrReadContent(err error) error {
	return fmt.Errorf("failed to read content: %w", err)
}
