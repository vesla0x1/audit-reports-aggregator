package base

import "shared/config"

// Factory is a standardized factory interface for creating resources
type Factory[T any] interface {
	Create(cfg *config.Config) (T, error)
}
