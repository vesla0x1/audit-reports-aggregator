package util

import (
	"crypto/sha256"
	"fmt"
)

// CalculateHash computes SHA-256 hash of content
func CalculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}
