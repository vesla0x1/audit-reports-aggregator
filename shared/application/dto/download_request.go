package dto

import (
	"errors"
	"time"
)

// DownloadRequest represents the message payload for requesting a file download
type DownloadRequest struct {
	// EventID is the unique identifier for this message (for idempotency)
	EventID string `json:"event_id"`

	// EventType identifies this as a download request
	// can be: downlaod.retry or download.requested -> also useful for tracking
	EventType string `json:"event_type"`

	// DownloadID is the database ID of the download record to process
	DownloadID int64 `json:"download_id"`

	// Timestamp when the message was created
	Timestamp time.Time `json:"timestamp"`
}

func (r *DownloadRequest) Validate() error {
	if r.DownloadID <= 0 {
		return errors.New("download ID must be positive")
	}
	if r.EventID == "" {
		return errors.New("event ID is required")
	}
	if r.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	return nil
}
