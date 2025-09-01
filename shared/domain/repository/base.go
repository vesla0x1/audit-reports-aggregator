package repository

import "context"

// BaseRepository defines common operations for all repositories
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity T) error
	Get(ctx context.Context, id string) (T, error)
	Update(ctx context.Context, id string, entity T) error
	Delete(ctx context.Context, id string) error
	ListAll(ctx context.Context) ([]T, error)
	CountAll(ctx context.Context) (int64, error)
}
