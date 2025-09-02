package usecase

import (
	"fmt"
	"shared/domain/dto"
	"time"
)

const maxMessageAge = 24 * time.Hour

type RequestValidator struct {
	maxAge time.Duration
}

func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		maxAge: maxMessageAge,
	}
}

func (v *RequestValidator) Validate(req *dto.DownloadRequest) error {
	if req.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	// add validation for download.retry or download.requested

	if req.DownloadID <= 0 {
		return fmt.Errorf("invalid download_id: %d", req.DownloadID)
	}

	if req.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	if time.Since(req.Timestamp) > v.maxAge {
		return fmt.Errorf("request timestamp is too old: %v", req.Timestamp)
	}

	return nil
}
