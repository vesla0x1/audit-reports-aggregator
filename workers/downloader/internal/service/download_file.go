package service

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

	"shared/observability/types"

	"downloader/internal/domain"
	"downloader/internal/domain/ports"
)

// DownloadService handles file download operations
type DownloadService struct {
	httpClient ports.HTTPClient
	logger     types.Logger
	metrics    types.Metrics
}

// NewDownloadService creates a new download service
func NewDownloadService(
	httpClient ports.HTTPClient,
	logger types.Logger,
	metrics types.Metrics,
) *DownloadService {
	return &DownloadService{
		httpClient: httpClient,
		logger:     logger,
		metrics:    metrics,
	}
}

// Execute downloads a file and returns its content and metadata
func (s *DownloadService) Execute(ctx context.Context, req domain.DownloadRequest) (*domain.DownloadResult, io.ReadCloser, error) {
	s.metrics.StartOperation("download")
	defer s.metrics.EndOperation("download")
	startTime := time.Now()
	defer func() {
		// Always record duration regardless of success/failure
		s.metrics.RecordDuration("download", time.Since(startTime).Seconds())
	}()

	s.logger.Info(ctx, "Starting file download", types.Fields{
		"url": req.URL,
		"id":  req.ID,
	})

	// Validate request
	if err := s.validateRequest(req); err != nil {
		s.metrics.RecordError("download", "validation_error")
		s.logger.Error(ctx, "Request validation failed", err, types.Fields{
			"url": req.URL,
			"id":  req.ID,
		})
		return nil, nil, err
	}

	// Download file
	reader, headers, err := s.httpClient.Download(ctx, req.URL, req.Headers)
	if err != nil {
		errorType := s.categorizeError(err)
		s.metrics.RecordError("download", errorType)

		s.logger.Error(ctx, "Failed to download file", err, types.Fields{
			"url":        req.URL,
			"error_type": errorType,
		})

		return nil, nil, domain.NewDomainError(
			domain.ErrDownloadFailed.Code,
			domain.ErrDownloadFailed.Message,
			err,
			true, // Retryable
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
		metrics:  s.metrics,
		fileType: fileType,
		onClose: func(size int64) {
			// Record file size when reader is closed
			s.metrics.RecordFileSize(fileType, size)
		},
	}

	// Determine storage path
	storagePath := req.StoragePath
	if storagePath == "" {
		storagePath = s.generateStoragePath(req.ID, req.URL)
	}

	result := &domain.DownloadResult{
		ID:           req.ID,
		URL:          req.URL,
		StoragePath:  storagePath,
		ContentType:  headers["Content-Type"],
		DownloadedAt: time.Now().UTC(),
	}

	s.metrics.RecordSuccess("download")

	s.logger.Info(ctx, "File download prepared", types.Fields{
		"id":           req.ID,
		"path":         storagePath,
		"content_type": result.ContentType,
		"file_type":    fileType,
	})

	return result, checksumReader, nil
}

// validateRequest validates the download request
func (s *DownloadService) validateRequest(req domain.DownloadRequest) error {
	if req.URL == "" {
		return domain.ErrInvalidURL
	}

	// Parse and validate URL
	u, err := url.Parse(req.URL)
	if err != nil {
		return domain.NewDomainError(
			domain.ErrInvalidURL.Code,
			"Failed to parse URL",
			err,
			false,
		)
	}

	// Only allow HTTP and HTTPS
	if u.Scheme != "http" && u.Scheme != "https" {
		return domain.NewDomainError(
			domain.ErrInvalidURL.Code,
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
	metrics  types.Metrics
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
