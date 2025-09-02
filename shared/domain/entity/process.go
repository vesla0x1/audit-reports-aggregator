package entity

import "time"

type ProcessStatus string

const (
	ProcessStatusPending    ProcessStatus = "pending"
	ProcessStatusInProgress ProcessStatus = "in_progress"
	ProcessStatusCompleted  ProcessStatus = "completed"
	ProcessStatusFailed     ProcessStatus = "failed"
)

type Process struct {
	ID               int64         `db:"id"`
	DownloadID       int64         `db:"download_id"`
	Status           ProcessStatus `db:"status"`
	ErrorMessage     *string       `db:"error_message"`
	AttemptCount     int           `db:"attempt_count"`
	ProcessorVersion *string       `db:"processor_version"`
	CreatedAt        time.Time     `db:"created_at"`
	StartedAt        *time.Time    `db:"started_at"`
	CompletedAt      *time.Time    `db:"completed_at"`
	UpdatedAt        time.Time     `db:"updated_at"`
}
