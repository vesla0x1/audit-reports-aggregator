package domain

import "fmt"

// DomainError represents a domain-specific error
type DomainError struct {
	Code      string
	Message   string
	Err       error
	Retryable bool
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error, retryable bool) *DomainError {
	return &DomainError{
		Code:      code,
		Message:   message,
		Err:       err,
		Retryable: retryable,
	}
}

// Common domain errors
var (
	ErrInvalidURL = &DomainError{
		Code:      "INVALID_URL",
		Message:   "The provided URL is invalid",
		Retryable: false,
	}

	ErrDownloadFailed = &DomainError{
		Code:      "DOWNLOAD_FAILED",
		Message:   "Failed to download file",
		Retryable: true,
	}

	ErrStorageFailed = &DomainError{
		Code:      "STORAGE_FAILED",
		Message:   "Failed to store file",
		Retryable: true,
	}
)
