package downloadresult

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
	"strings"
)

type DownloadResult struct {
	content     []byte
	contentType string
	url         string
}

// NewDownloadResult creates a valid DownloadResult with validation
func NewDownloadResult(content []byte, url string, contentType string) (*DownloadResult, error) {
	if len(content) == 0 {
		return nil, ErrEmptyContent
	}

	if url == "" {
		return nil, ErrEmptyUrl
	}

	return &DownloadResult{
		content:     content,
		url:         url,
		contentType: normalizeContentType(contentType),
	}, nil
}

// Add this factory method to create DownloadResult from a reader
func NewDownloadResultFromReader(
	reader io.Reader,
	url string,
	contentType string,
	maxSize int64,
) (*DownloadResult, error) {
	if url == "" {
		return nil, ErrEmptyUrl
	}

	// Read with size limit
	limitedReader := io.LimitReader(reader, maxSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, ErrReadContent(err)
	}

	// Check if we exceeded the limit
	currLen := int64(len(content))
	if currLen > maxSize {
		return nil, ErrSizeExceeded(maxSize)
	}

	// Delegate to the existing constructor
	return NewDownloadResult(content, url, contentType)
}

// Extension determines the file extension from URL and content type
func (r *DownloadResult) Extension() string {
	// First try to get from URL
	if ext := r.extensionFromURL(); ext != "" {
		return ext
	}

	// Fallback to content type
	return r.extensionFromContentType()
}

// Content returns the downloaded content (immutable - returns copy)
func (r *DownloadResult) Content() []byte {
	// Return a copy to maintain immutability
	result := make([]byte, len(r.content))
	copy(result, r.content)
	return result
}

// Hash calculates and returns the SHA256 hash (computed, not stored)
func (r *DownloadResult) Hash() string {
	hash := sha256.Sum256(r.content)
	return hex.EncodeToString(hash[:])
}

// Size returns the content size (computed, not stored)
func (r *DownloadResult) Size() int64 {
	return int64(len(r.content))
}

// ContentType returns the normalized content type
func (r *DownloadResult) ContentType() string {
	return r.contentType
}

// URL returns the source URL
func (r *DownloadResult) URL() string {
	return r.url
}

// IsTextContent checks if the content is text-based
func (r *DownloadResult) IsTextContent() bool {
	return strings.HasPrefix(r.contentType, "text/") ||
		r.contentType == "application/json" ||
		r.contentType == "application/xml"
}

// IsPDF checks if the content is a PDF
func (r *DownloadResult) IsPDF() bool {
	return r.contentType == "application/pdf" ||
		r.Extension() == ".pdf"
}

// IsImage checks if the content is an image
func (r *DownloadResult) IsImage() bool {
	return strings.HasPrefix(r.contentType, "image/")
}

// Private helper methods
func (r *DownloadResult) extensionFromURL() string {
	// Remove query parameters
	cleanURL := r.url
	if idx := strings.Index(cleanURL, "?"); idx != -1 {
		cleanURL = cleanURL[:idx]
	}

	ext := filepath.Ext(cleanURL)
	if ext != "" {
		return ext
	}

	return ""
}

func (r *DownloadResult) extensionFromContentType() string {
	// Map content types to extensions
	contentTypeToExt := map[string]string{
		"application/pdf":   ".pdf",
		"application/json":  ".json",
		"application/xml":   ".xml",
		"application/zip":   ".zip",
		"application/gzip":  ".gz",
		"application/x-tar": ".tar",
		"text/html":         ".html",
		"text/plain":        ".txt",
		"image/jpeg":        ".jpg",
		"image/png":         ".png",
		"image/gif":         ".gif",
	}

	if ext, ok := contentTypeToExt[r.contentType]; ok {
		return ext
	}

	// Try to extract from content type (e.g., "image/jpeg" -> ".jpeg")
	if strings.Contains(r.contentType, "/") {
		parts := strings.Split(r.contentType, "/")
		if len(parts) == 2 && parts[1] != "*" {
			return "." + strings.ToLower(parts[1])
		}
	}

	return ".bin" // Default binary extension
}

func normalizeContentType(contentType string) string {
	// Remove charset and other parameters
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	return strings.TrimSpace(strings.ToLower(contentType))
}
