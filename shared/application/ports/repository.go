package ports

import (
	"context"
	"shared/domain/entity"
)

// BaseRepository defines common operations for all repositories
type BaseRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Get(ctx context.Context, id int64) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id int64) error
	ListAll(ctx context.Context) ([]*T, error)
	CountAll(ctx context.Context) (int64, error)
}

type AuditProviderRepository interface {
	BaseRepository[entity.AuditProvider]
	GetBySlug(ctx context.Context, slug string) (*entity.AuditProvider, error)
	List(ctx context.Context, onlyActive bool) ([]*entity.AuditProvider, error)
}

type AuditReportRepository interface {
	BaseRepository[entity.AuditReport]
	ExistsByURL(ctx context.Context, sourceID int64, detailsURL string) (bool, error)
}

type DownloadRepository interface {
	BaseRepository[entity.Download]
	GetByReportID(ctx context.Context, reportID int64) (*entity.Download, error)
	GetPendingDownloads(ctx context.Context, limit int) ([]*entity.Download, error)
}

type ProcessRepository interface {
	BaseRepository[entity.Process]
	GetByDownloadID(ctx context.Context, downloadID int64) (*entity.Process, error)
	GetPendingProcesses(ctx context.Context, limit int) ([]*entity.Process, error)
}

type Repositories interface {
	AuditReport() AuditReportRepository
	Download() DownloadRepository
	Process() ProcessRepository
	AuditProvider() AuditProviderRepository
}
