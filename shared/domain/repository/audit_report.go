package repository

import (
	"context"
	domain "shared/domain/entity"
)

// AuditReportRepository defines the interface for audit report persistence
type AuditReportRepository interface {
	BaseRepository[domain.AuditReport]
	ExistsByURL(ctx context.Context, sourceID int64, detailsURL string) (bool, error)
}
