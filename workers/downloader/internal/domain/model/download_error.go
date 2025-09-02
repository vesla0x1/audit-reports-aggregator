package model

type ErrorType string

const (
	NetworkError      ErrorType = "network_error"
	FileTooLargeError ErrorType = "file_too_large"
	ReadError         ErrorType = "read_error"
	InvalidURLError   ErrorType = "invalid_url"
)

type DownloadError struct {
	Type    ErrorType
	Message string
	URL     string
}

func NewDownloadError(errType ErrorType, message, url string) *DownloadError {
	return &DownloadError{
		Type:    errType,
		Message: message,
		URL:     url,
	}
}

func (e *DownloadError) Error() string {
	return e.Message
}

func (e *DownloadError) IsRetryable() bool {
	switch e.Type {
	case NetworkError, ReadError:
		return true
	default:
		return false
	}
}
