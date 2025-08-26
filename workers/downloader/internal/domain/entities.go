package domain

import (
	"time"
)

// DownloadRequest represents a file download request
type DownloadRequest struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	StoragePath string            `json:"storage_path,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DownloadResult represents the result of a download operation
type DownloadResult struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	StoragePath  string    `json:"storage_path"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	Checksum     string    `json:"checksum"`
	DownloadedAt time.Time `json:"downloaded_at"`
}
