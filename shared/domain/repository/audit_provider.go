package repository

import (
	"context"
	"shared/domain/entity"
)

type AuditProviderRepository interface {
	BaseRepository[entity.AuditProvider]
	GetBySlug(ctx context.Context, slug string) (*entity.AuditProvider, error)
	List(ctx context.Context, onlyActive bool) ([]*entity.AuditProvider, error)
}
