package repository

import (
	"context"
	"shared/domain/model"
)

// AuditReportRepository defines the interface for audit report persistence
type AuditReportRepository interface {
	Create(ctx context.Context, report *model.AuditReport) error
	FindByID(ctx context.Context, id int64) (*model.AuditReport, error)
	Update(ctx context.Context, report *model.AuditReport) error
	ExistsByURL(ctx context.Context, sourceID int64, detailsURL string) (bool, error)
}
