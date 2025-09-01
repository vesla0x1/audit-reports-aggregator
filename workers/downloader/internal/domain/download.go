package domain

import (
	"context"
	"shared/application/ports"
)

// DownloadService handles file download operations
type DownloadService struct {
	httpClient ports.HTTPClient
}

// NewDownloadService creates a new download service
func NewDownloadService(
	httpClient ports.HTTPClient,
) *DownloadService {
	return &DownloadService{
		httpClient: httpClient,
	}
}

// Execute downloads a file and returns its content and metadata
func (s *DownloadService) Execute(ctx context.Context) error {
	return nil
}
