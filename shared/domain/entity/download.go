package entity

import "time"

type DownloadStatus string

const (
	DownloadStatusPending    DownloadStatus = "pending"
	DownloadStatusInProgress DownloadStatus = "in_progress"
	DownloadStatusCompleted  DownloadStatus = "completed"
	DownloadStatusFailed     DownloadStatus = "failed"
)

type Download struct {
	ID            int64          `db:"id"`
	ReportID      int64          `db:"report_id"`
	StoragePath   *string        `db:"storage_path"`
	FileHash      *string        `db:"file_hash"`
	FileExtension *string        `db:"file_extension"`
	Status        DownloadStatus `db:"status"`
	ErrorMessage  *string        `db:"error_message"`
	AttemptCount  int            `db:"attempt_count"`
	CreatedAt     time.Time      `db:"created_at"`
	StartedAt     *time.Time     `db:"started_at"`
	CompletedAt   *time.Time     `db:"completed_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}
