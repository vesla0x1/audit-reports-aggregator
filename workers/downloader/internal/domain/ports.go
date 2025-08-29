package domain

import (
	"context"
	"io"
)

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	// Download retrieves content from a URL
	// Returns: reader, response headers, error
	Download(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, map[string]string, error)
}
