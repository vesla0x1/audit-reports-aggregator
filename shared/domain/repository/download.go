package repository

import (
	"context"
	"shared/domain/entity"
)

type DownloadRepository interface {
	BaseRepository[entity.Download]
	GetByReportID(ctx context.Context, reportID int64) (*entity.Download, error)
	GetPendingDownloads(ctx context.Context, limit int) ([]*entity.Download, error)
}
