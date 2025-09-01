package repository

import (
	"context"
	domain "shared/domain/entity"

	"github.com/Masterminds/squirrel"
)

type auditReportRepository struct {
	*baseRepository[*domain.AuditReport]
}

func (r *auditReportRepository) Create(ctx context.Context, report *domain.AuditReport) error {
	query := r.qb.Insert("reports").
		Columns("id", "title", "details_page_url", "created_at", "updated_at").
		Values(report.ID, report.Title, report.DetailsPageURL, report.CreatedAt, report.UpdatedAt)

	sql, args, _ := query.ToSql()
	r.db.Execute(ctx, sql, args...)
	return nil
}

func (r *auditReportRepository) Update(ctx context.Context, id string, report *domain.AuditReport) error {
	query := r.qb.Update("reports").
		//Set("name", report.Name).
		//Set("status", report.Status).
		Set("updated_at", report.UpdatedAt).
		Where(squirrel.Eq{"id": id})

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}
