package download

import "shared/domain/entity/download"

type (
	Download = download.Download
)

const (
	StatusPending    download.Status = "pending"
	StatusInProgress download.Status = "in_progress"
	StatusCompleted  download.Status = "completed"
	StatusFailed     download.Status = "failed"
)

var (
	// State transition errors
	ErrAlreadyCompleted       = download.ErrAlreadyCompleted
	ErrAlreadyInProgress      = download.ErrAlreadyInProgress
	ErrInvalidStateTransition = download.ErrInvalidStateTransition
	ErrNotInProgress          = download.ErrNotInProgress

	// Retry/attempt errors
	ErrMaxAttemptsExceeded = download.ErrMaxAttemptsExceeded
	ErrCannotRetry         = download.ErrCannotRetry

	// Validation errors
	ErrEmptyStoragePath   = download.ErrEmptyStoragePath
	ErrEmptyFileHash      = download.ErrEmptyFileHash
	ErrEmptyFileExtension = download.ErrEmptyFileExtension
)
