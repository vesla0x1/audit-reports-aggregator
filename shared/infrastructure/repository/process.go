package repository

import (
	"context"
	"shared/domain/entity"

	"github.com/Masterminds/squirrel"
)

type processRepository struct {
	*baseRepository[entity.Process]
}

func (r *processRepository) Create(ctx context.Context, process *entity.Process) error {
	query := r.qb.Insert("processes").
		Columns("download_id", "status", "attempt_count", "created_at", "updated_at").
		Values(process.DownloadID, process.Status, process.AttemptCount, process.CreatedAt, process.UpdatedAt)

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *processRepository) Update(ctx context.Context, process *entity.Process) error {
	query := r.qb.Update("processes").
		Set("status", process.Status).
		Set("attempt_count", process.AttemptCount).
		Set("updated_at", process.UpdatedAt)

	if process.ErrorMessage != nil {
		query = query.Set("error_message", *process.ErrorMessage)
	}
	if process.ProcessorVersion != nil {
		query = query.Set("processor_version", *process.ProcessorVersion)
	}
	if process.StartedAt != nil {
		query = query.Set("started_at", *process.StartedAt)
	}
	if process.CompletedAt != nil {
		query = query.Set("completed_at", *process.CompletedAt)
	}

	query = query.Where(squirrel.Eq{"id": process.ID})

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *processRepository) GetByDownloadID(ctx context.Context, downloadID int64) (*entity.Process, error) {
	query := r.qb.Select("*").
		From("processes").
		Where(squirrel.Eq{"download_id": downloadID})

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)

	var p entity.Process
	err := row.Scan(
		&p.ID, &p.DownloadID, &p.Status, &p.ErrorMessage,
		&p.AttemptCount, &p.ProcessorVersion, &p.CreatedAt,
		&p.StartedAt, &p.CompletedAt, &p.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *processRepository) GetPendingProcesses(ctx context.Context, limit int) ([]*entity.Process, error) {
	query := r.qb.Select("*").
		From("processes").
		Where(squirrel.Eq{"status": entity.ProcessStatusPending}).
		OrderBy("created_at ASC").
		Limit(uint64(limit))

	sql, args, _ := query.ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var processes []*entity.Process
	for rows.Next() {
		var p entity.Process
		err := rows.Scan(
			&p.ID, &p.DownloadID, &p.Status, &p.ErrorMessage,
			&p.AttemptCount, &p.ProcessorVersion, &p.CreatedAt,
			&p.StartedAt, &p.CompletedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		processes = append(processes, &p)
	}

	return processes, rows.Err()
}
