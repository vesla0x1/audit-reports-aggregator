package storage

import (
	"errors"
	"time"
)

// Common storage errors
var (
	// ErrObjectNotFound is returned when an object is not found in storage
	ErrObjectNotFound = errors.New("object not found")
)

// ObjectMetadata represents metadata associated with stored objects
type ObjectMetadata struct {
	ContentType     string
	ContentLength   int64
	ContentEncoding string
	CacheControl    string
	LastModified    time.Time
	ETag            string
	UserMetadata    map[string]string
}

// ObjectInfo represents information about a stored object
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}
