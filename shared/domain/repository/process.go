package repository

import (
	"context"
	"shared/domain/entity"
)

type ProcessRepository interface {
	BaseRepository[entity.Process]
	GetByDownloadID(ctx context.Context, downloadID int64) (*entity.Process, error)
	GetPendingProcesses(ctx context.Context, limit int) ([]*entity.Process, error)
}
