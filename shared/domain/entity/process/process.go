package process

import "time"

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

	// not persisted
	MaxAttempts int `db:"-"`
}

func NewProcess(downloadID int64) *Process {
	now := time.Now()
	return &Process{
		DownloadID:   downloadID,
		Status:       StatusPending,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
		MaxAttempts:  3, // Default, can be configured
	}
}
