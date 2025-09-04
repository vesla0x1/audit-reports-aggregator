package download

import (
	"errors"
)

var (
	// State transition errors
	ErrAlreadyCompleted       = errors.New("download already completed")
	ErrAlreadyInProgress      = errors.New("download already in progress")
	ErrInvalidStateTransition = errors.New("invalid download state transition")
	ErrNotInProgress          = errors.New("download is not in progress")

	// Retry/attempt errors
	ErrMaxAttemptsExceeded = errors.New("maximum download attempts exceeded")
	ErrCannotRetry         = errors.New("cannot retry download")

	// Validation errors
	ErrEmptyStoragePath   = errors.New("storage path cannot be empty")
	ErrEmptyFileHash      = errors.New("file hash cannot be empty")
	ErrEmptyFileExtension = errors.New("file extension cannot be empty")
)
