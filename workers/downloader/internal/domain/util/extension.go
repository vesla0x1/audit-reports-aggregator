package util

import (
	"path"
	"strings"
)

// DetermineExtension extracts file extension from URL or content type
func DetermineExtension(url string, contentType string) string {
	// Try URL first
	if ext := extractExtensionFromURL(url); ext != "" {
		return ext
	}
	// Fall back to content type
	return extensionFromContentType(contentType)
}

func extractExtensionFromURL(url string) string {
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	ext := strings.ToLower(path.Ext(url))
	if ext != "" && len(ext) > 1 {
		ext = ext[1:] // Remove leading dot
		if isValidExtension(ext) {
			return ext
		}
	}
	return ""
}

func extensionFromContentType(contentType string) string {
	// Remove parameters
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	contentTypeMap := map[string]string{
		"application/pdf":  "pdf",
		"text/html":        "html",
		"text/markdown":    "md",
		"application/json": "json",
		"text/plain":       "txt",
		"application/zip":  "zip",
	}

	if ext, ok := contentTypeMap[contentType]; ok {
		return ext
	}
	return "pdf" // Default for audit reports
}

func isValidExtension(ext string) bool {
	validExtensions := map[string]bool{
		"pdf": true, "html": true, "htm": true,
		"md": true, "txt": true, "json": true, "zip": true,
	}
	return validExtensions[ext]
}

func ExtractContentType(headers map[string]string) string {
	if ct := headers["content-type"]; ct != "" {
		return ct
	}
	if ct := headers["Content-Type"]; ct != "" {
		return ct
	}
	return "application/octet-stream"
}
