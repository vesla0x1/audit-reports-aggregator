package types

import "errors"

// Common storage errors
var (
	// ErrObjectNotFound is returned when an object is not found in storage
	ErrObjectNotFound = errors.New("object not found")
)