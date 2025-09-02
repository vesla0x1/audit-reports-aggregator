package repository

import (
	"context"
	"fmt"
	"shared/domain/entity"

	"github.com/Masterminds/squirrel"
)

type auditReportRepository struct {
	*baseRepository[entity.AuditReport]
}

func (r *auditReportRepository) Create(ctx context.Context, report *entity.AuditReport) error {
	query := r.qb.Insert("audit_reports").
		Columns(
			"source_id", "provider_id", "title", "engagement_type",
			"client_company", "audit_start_date", "audit_end_date",
			"details_page_url", "source_download_url", "repository_url",
			"summary", "findings_summary", "created_at", "updated_at",
		).
		Values(
			report.SourceID, report.ProviderID, report.Title, report.EngagementType,
			report.ClientCompany, report.AuditStartDate, report.AuditEndDate,
			report.DetailsPageURL, report.SourceDownloadURL, report.RepositoryURL,
			report.Summary, report.FindingsSummary, report.CreatedAt, report.UpdatedAt,
		).
		Suffix("RETURNING id")

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)
	err := row.Scan(&report.ID)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}
	return nil
}

func (r *auditReportRepository) Update(ctx context.Context, report *entity.AuditReport) error {
	query := r.qb.Update("audit_reports").
		Set("title", report.Title).
		Set("engagement_type", report.EngagementType).
		Set("updated_at", report.UpdatedAt)

	// Update nullable fields
	if report.ClientCompany != nil {
		query = query.Set("client_company", *report.ClientCompany)
	}
	if report.AuditStartDate != nil {
		query = query.Set("audit_start_date", *report.AuditStartDate)
	}
	if report.AuditEndDate != nil {
		query = query.Set("audit_end_date", *report.AuditEndDate)
	}
	if report.RepositoryURL != nil {
		query = query.Set("repository_url", *report.RepositoryURL)
	}
	if report.Summary != nil {
		query = query.Set("summary", *report.Summary)
	}
	if report.FindingsSummary != nil {
		query = query.Set("findings_summary", *report.FindingsSummary)
	}

	query = query.Where(squirrel.Eq{"id": report.ID})

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *auditReportRepository) ExistsByURL(ctx context.Context, sourceID int64, detailsURL string) (bool, error) {
	query := r.qb.Select("COUNT(*)").
		From("audit_reports").
		Where(squirrel.Eq{
			"source_id":        sourceID,
			"details_page_url": detailsURL,
		})

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
