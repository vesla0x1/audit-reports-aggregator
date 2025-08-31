package repository

import (
	"context"
	"database/sql"
	"fmt"

	"shared/domain/database"
	"shared/domain/model"
	repo "shared/domain/repository"

	sq "github.com/Masterminds/squirrel"
)

// AuditReportRepository handles audit report persistence
type AuditReportRepository struct {
	*database.Repository
	qb sq.StatementBuilderType
}

// Ensure AuditReportRepository implements the interface
var _ repo.AuditReportRepository = (*AuditReportRepository)(nil)

// NewAuditReportRepository creates a new audit report repository
func NewAuditReportRepository(db database.Database) repo.AuditReportRepository {
	return &AuditReportRepository{
		Repository: database.NewRepository(db, "audit_reports"),
		qb:         sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Create creates a new audit report
func (r *AuditReportRepository) Create(ctx context.Context, report *model.AuditReport) error {
	query := r.qb.
		Insert("audit_reports").
		Columns(
			"source_id", "title", "type", "audited_company",
			"period", "details_page_url", "source_download_url",
			"summary", "findings_summary",
		).
		Values(
			report.SourceID, report.Title, report.Type, report.AuditedCompany,
			report.Period, report.DetailsPageURL, report.SourceDownloadURL,
			report.Summary, report.FindingsSummary,
		).
		Suffix("RETURNING id, platform_id, created_at, updated_at")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	row := r.QueryRow(ctx, sql, args...)
	return row.Scan(&report.ID, &report.PlatformID, &report.CreatedAt, &report.UpdatedAt)
}

// FindByID finds an audit report by ID
func (r *AuditReportRepository) FindByID(ctx context.Context, id int64) (*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Eq{"id": id})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	report := &model.AuditReport{}
	row := r.QueryRow(ctx, sqlQuery, args...)

	err = row.Scan(
		&report.ID, &report.SourceID, &report.PlatformID,
		&report.Title, &report.Type, &report.AuditedCompany,
		&report.Period, &report.DetailsPageURL, &report.SourceDownloadURL,
		&report.Summary, &report.FindingsSummary,
		&report.CreatedAt, &report.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit report with id %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan audit report: %w", err)
	}

	return report, nil
}

// ExistsByURL checks if a report with the same details URL already exists
func (r *AuditReportRepository) ExistsByURL(ctx context.Context, sourceID int64, detailsURL string) (bool, error) {
	query := r.qb.
		Select("1").
		From("audit_reports").
		Where(sq.Eq{
			"source_id":        sourceID,
			"details_page_url": detailsURL,
		}).
		Limit(1)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build query: %w", err)
	}

	var exists int
	err = r.QueryRow(ctx, sqlQuery, args...).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Update updates an audit report
func (r *AuditReportRepository) Update(ctx context.Context, report *model.AuditReport) error {
	query := r.qb.
		Update("audit_reports").
		Set("title", report.Title).
		Set("type", report.Type).
		Set("audited_company", report.AuditedCompany).
		Set("period", report.Period).
		Set("details_page_url", report.DetailsPageURL).
		Set("source_download_url", report.SourceDownloadURL).
		Set("summary", report.Summary).
		Set("findings_summary", report.FindingsSummary).
		Where(sq.Eq{"id": report.ID})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := r.Execute(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update audit report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("audit report with id %d not found", report.ID)
	}

	return nil
}

// UpdateSummaries updates only the summary fields
/*
// FindBySourceID finds all audit reports from a specific source
func (r *AuditReportRepository) FindBySourceID(ctx context.Context, sourceID int64) ([]*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Eq{"source_id": sourceID}).
		OrderBy("created_at DESC")

	return r.queryReports(ctx, query)
}

// FindByPlatformID finds all audit reports from a specific platform
func (r *AuditReportRepository) FindByPlatformID(ctx context.Context, platformID int64) ([]*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Eq{"platform_id": platformID}).
		OrderBy("created_at DESC")

	return r.queryReports(ctx, query)
}

// FindByType finds all audit reports of a specific type
func (r *AuditReportRepository) FindByType(ctx context.Context, reportType model.AuditReportType) ([]*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Eq{"type": reportType}).
		OrderBy("created_at DESC")

	return r.queryReports(ctx, query)
}

// FindByAuditedCompany finds all audit reports for a specific company
func (r *AuditReportRepository) FindByAuditedCompany(ctx context.Context, company string) ([]*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Like{"audited_company": "%" + company + "%"}).
		OrderBy("created_at DESC")

	return r.queryReports(ctx, query)
}

func (r *AuditReportRepository) UpdateSummaries(ctx context.Context, id int64, summary, findingsSummary string) error {
	query := r.qb.
		Update("audit_reports").
		Set("summary", summary).
		Set("findings_summary", findingsSummary).
		Where(sq.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	result, err := r.Execute(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update summaries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("audit report with id %d not found", id)
	}

	return nil
}

// Delete deletes an audit report
func (r *AuditReportRepository) Delete(ctx context.Context, id int64) error {
	query := r.qb.
		Delete("audit_reports").
		Where(sq.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	result, err := r.Execute(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to delete audit report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("audit report with id %d not found", id)
	}

	return nil
}

// GetUnprocessedReports gets reports without summaries
func (r *AuditReportRepository) GetUnprocessedReports(ctx context.Context, limit int) ([]*model.AuditReport, error) {
	query := r.qb.
		Select(
			"id", "source_id", "platform_id", "title", "type",
			"audited_company", "period", "details_page_url",
			"source_download_url", "summary", "findings_summary",
			"created_at", "updated_at",
		).
		From("audit_reports").
		Where(sq.Or{
			sq.Eq{"summary": nil},
			sq.Eq{"findings_summary": nil},
		}).
		OrderBy("created_at ASC").
		Limit(uint64(limit))

	return r.queryReports(ctx, query)
}

// Helper method to query multiple reports
func (r *AuditReportRepository) queryReports(ctx context.Context, query sq.SelectBuilder) ([]*model.AuditReport, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit reports: %w", err)
	}
	defer rows.Close()

	var reports []*model.AuditReport
	for rows.Next() {
		report := &model.AuditReport{}
		err := rows.Scan(
			&report.ID, &report.SourceID, &report.PlatformID,
			&report.Title, &report.Type, &report.AuditedCompany,
			&report.Period, &report.DetailsPageURL, &report.SourceDownloadURL,
			&report.Summary, &report.FindingsSummary,
			&report.CreatedAt, &report.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit report: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, rows.Err()
}*/
