package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// Client implements the HTTPClient port
type Client struct {
	client  *http.Client
	config  config.HTTPConfig
	logger  ports.Logger
	metrics ports.Metrics
}

// NewClientWithConfig creates a new HTTP client with custom configuration
func CreateHTTPClient(cfg config.HTTPConfig, obs ports.Observability) (ports.HTTPClient, error) {
	logger, metrics, _ := obs.ComponentsScoped("http.client")
	return &Client{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		config:  cfg,
		logger:  logger,
		metrics: metrics,
	}, nil
}

// Download implements the HTTPClient interface
func (c *Client) Download(ctx context.Context, url string, headers map[string]string) (io.ReadCloser, map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", c.config.UserAgent)

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request with retries
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}

		resp, lastErr = c.client.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break // Success or client error (no retry needed)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		return nil, nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Extract response headers
	responseHeaders := make(map[string]string)
	for key := range resp.Header {
		responseHeaders[key] = resp.Header.Get(key)
	}

	return resp.Body, responseHeaders, nil
}
