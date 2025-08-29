package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/url"
	"path"
	"strings"
	"time"
)

// DownloadService handles file download operations
type DownloadService struct {
	httpClient HTTPClient
}

// NewDownloadService creates a new download service
func NewDownloadService(
	httpClient HTTPClient,
) *DownloadService {
	return &DownloadService{
		httpClient: httpClient,
	}
}

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

// Execute downloads a file and returns its content and metadata
func (s *DownloadService) Execute(ctx context.Context, req DownloadRequest) (*DownloadResult, io.ReadCloser, error) {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return nil, nil, err
	}

	// Download file
	reader, headers, err := s.httpClient.Download(ctx, req.URL, req.Headers)
	if err != nil {
		return nil, nil, NewDomainError(
			ErrDownloadFailed.Code,
			ErrDownloadFailed.Message,
			err,
			true,
		)
	}

	// Determine file type from content type or URL
	fileType := s.extractFileType(headers["Content-Type"], req.URL)

	// Calculate checksum while reading
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	// Create a custom reader that will calculate checksum on read
	checksumReader := &ChecksumReader{
		reader:   teeReader,
		hasher:   hasher,
		original: reader,
		fileType: fileType,
		onClose: func(size int64) {
			// Record file size when reader is closed
		},
	}

	// Determine storage path
	storagePath := req.StoragePath
	if storagePath == "" {
		storagePath = s.generateStoragePath(req.ID, req.URL)
	}

	result := &DownloadResult{
		ID:           req.ID,
		URL:          req.URL,
		StoragePath:  storagePath,
		ContentType:  headers["Content-Type"],
		DownloadedAt: time.Now().UTC(),
	}

	return result, checksumReader, nil
}

// validateRequest validates the download request
func (s *DownloadService) validateRequest(req DownloadRequest) error {
	if req.URL == "" {
		return ErrInvalidURL
	}

	// Parse and validate URL
	u, err := url.Parse(req.URL)
	if err != nil {
		return NewDomainError(
			ErrInvalidURL.Code,
			"Failed to parse URL",
			err,
			false,
		)
	}

	// Only allow HTTP and HTTPS
	if u.Scheme != "http" && u.Scheme != "https" {
		return NewDomainError(
			ErrInvalidURL.Code,
			"Only HTTP and HTTPS URLs are supported",
			nil,
			false,
		)
	}

	return nil
}

// generateStoragePath generates a storage path from URL
func (s *DownloadService) generateStoragePath(id, downloadURL string) string {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Sprintf("%s/download", id)
	}

	// Extract filename from URL path
	filename := path.Base(u.Path)
	if filename == "" || filename == "/" || filename == "." {
		filename = "download"
	}

	return fmt.Sprintf("%s/%s", id, filename)
}

// extractFileType determines file type from content type or URL extension
func (s *DownloadService) extractFileType(contentType, urlPath string) string {
	// Try to extract from content type first
	if contentType != "" {
		switch {
		case strings.Contains(contentType, "pdf"):
			return "pdf"
		case strings.Contains(contentType, "html"):
			return "html"
		case strings.Contains(contentType, "markdown"):
			return "markdown"
		case strings.Contains(contentType, "json"):
			return "json"
		case strings.Contains(contentType, "text"):
			return "text"
		}
	}

	// Fallback to URL extension
	if u, err := url.Parse(urlPath); err == nil {
		ext := strings.ToLower(path.Ext(u.Path))
		switch ext {
		case ".pdf":
			return "pdf"
		case ".html", ".htm":
			return "html"
		case ".md", ".markdown":
			return "markdown"
		case ".json":
			return "json"
		case ".txt":
			return "text"
		}
	}

	return "unknown"
}

// categorizeError categorizes errors for metrics
func (s *DownloadService) categorizeError(err error) string {
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection"):
		return "connection"
	case strings.Contains(errStr, "404") || strings.Contains(errStr, "not found"):
		return "not_found"
	case strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden"):
		return "forbidden"
	case strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized"):
		return "unauthorized"
	case strings.Contains(errStr, "500") || strings.Contains(errStr, "server"):
		return "server_error"
	default:
		return "unknown"
	}
}

// checksumReader wraps a reader to calculate checksum on read
type ChecksumReader struct {
	reader   io.Reader
	hasher   hash.Hash
	original io.ReadCloser
	size     int64
	fileType string
	onClose  func(int64)
}

// Read implements io.Reader
func (r *ChecksumReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.size += int64(n)
	return
}

// Close implements io.Closer
func (r *ChecksumReader) Close() error {
	// Call the onClose callback with final size
	if r.onClose != nil {
		r.onClose(r.size)
	}
	return r.original.Close()
}

// GetChecksum returns the calculated checksum
func (r *ChecksumReader) GetChecksum() string {
	return hex.EncodeToString(r.hasher.Sum(nil))
}

// GetSize returns the total bytes read
func (r *ChecksumReader) GetSize() int64 {
	return r.size
}
