package repository

import (
	"context"
	"shared/domain/entity"

	"github.com/Masterminds/squirrel"
)

type downloadRepository struct {
	*baseRepository[entity.Download]
}

func (r *downloadRepository) Create(ctx context.Context, download *entity.Download) error {
	query := r.qb.Insert("downloads").
		Columns("report_id", "status", "attempt_count", "created_at", "updated_at").
		Values(download.ReportID, download.Status, download.AttemptCount, download.CreatedAt, download.UpdatedAt)

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *downloadRepository) Update(ctx context.Context, download *entity.Download) error {
	query := r.qb.Update("downloads").
		Set("status", download.Status).
		Set("attempt_count", download.AttemptCount).
		Set("updated_at", download.UpdatedAt)

	// Update nullable fields only if they have values
	if download.StoragePath != nil {
		query = query.Set("storage_path", *download.StoragePath)
	}
	if download.FileHash != nil {
		query = query.Set("file_hash", *download.FileHash)
	}
	if download.FileExtension != nil {
		query = query.Set("file_extension", *download.FileExtension)
	}
	if download.ErrorMessage != nil {
		query = query.Set("error_message", *download.ErrorMessage)
	}
	if download.StartedAt != nil {
		query = query.Set("started_at", *download.StartedAt)
	}
	if download.CompletedAt != nil {
		query = query.Set("completed_at", *download.CompletedAt)
	}

	query = query.Where(squirrel.Eq{"id": download.ID})

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *downloadRepository) GetByReportID(ctx context.Context, reportID int64) (*entity.Download, error) {
	query := r.qb.Select("*").
		From("downloads").
		Where(squirrel.Eq{"report_id": reportID})

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)

	var d entity.Download
	err := row.Scan(
		&d.ID, &d.ReportID, &d.StoragePath, &d.FileHash,
		&d.FileExtension, &d.Status, &d.ErrorMessage,
		&d.AttemptCount, &d.CreatedAt, &d.StartedAt,
		&d.CompletedAt, &d.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *downloadRepository) GetPendingDownloads(ctx context.Context, limit int) ([]*entity.Download, error) {
	query := r.qb.Select("*").
		From("downloads").
		Where(squirrel.Eq{"status": entity.DownloadStatusPending}).
		OrderBy("created_at ASC").
		Limit(uint64(limit))

	sql, args, _ := query.ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*entity.Download
	for rows.Next() {
		var d entity.Download
		err := rows.Scan(
			&d.ID, &d.ReportID, &d.StoragePath, &d.FileHash,
			&d.FileExtension, &d.Status, &d.ErrorMessage,
			&d.AttemptCount, &d.CreatedAt, &d.StartedAt,
			&d.CompletedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, &d)
	}

	return downloads, rows.Err()
}
