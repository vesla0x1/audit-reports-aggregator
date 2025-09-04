package download

import (
	"fmt"
	"time"
)

type Download struct {
	ID            int64      `db:"id"`
	ReportID      int64      `db:"report_id"`
	StoragePath   *string    `db:"storage_path"`
	FileHash      *string    `db:"file_hash"`
	FileExtension *string    `db:"file_extension"`
	Status        Status     `db:"status"`
	ErrorMessage  *string    `db:"error_message"`
	AttemptCount  int        `db:"attempt_count"`
	CreatedAt     time.Time  `db:"created_at"`
	StartedAt     *time.Time `db:"started_at"`
	CompletedAt   *time.Time `db:"completed_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

func NewDownload(reportID int64, maxAttempts int) *Download {
	now := time.Now()
	return &Download{
		ReportID:     reportID,
		Status:       StatusPending,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func NewDownloadWithDefaults(reportID int64) *Download {
	return NewDownload(reportID, 3)
}

// ============================================================================
// BUSINESS METHODS (State transitions with rules)
// ============================================================================

// CanStart checks if the download can be started
func (d *Download) CanStart() bool {
	return d.Status == StatusPending ||
		(d.Status == StatusFailed && d.AttemptCount < d.MaxAttempts())
}

// Start begins or retries a download
func (d *Download) Start() error {
	// Business rule: Cannot start if already completed
	if d.Status == StatusCompleted {
		return ErrAlreadyCompleted
	}

	// Business rule: Cannot start if already in progress
	if d.Status == StatusInProgress {
		return ErrAlreadyInProgress
	}

	// Business rule: Check max attempts
	if d.AttemptCount >= d.MaxAttempts() {
		return ErrMaxAttemptsExceeded
	}

	// State transition
	now := time.Now()
	d.Status = StatusInProgress
	d.StartedAt = &now
	d.AttemptCount++
	d.UpdatedAt = now
	d.ErrorMessage = nil

	return nil
}

// Complete marks the download as successfully completed
func (d *Download) Complete(storagePath, fileHash, fileExtension string) error {
	// Business rule: Can only complete if in progress
	if d.Status != StatusInProgress {
		return fmt.Errorf("%w: cannot complete download in status %s",
			ErrInvalidStateTransition, d.Status)
	}

	// Validate inputs
	if storagePath == "" {
		return ErrEmptyStoragePath
	}
	if fileHash == "" {
		return ErrEmptyFileHash
	}
	if fileExtension == "" {
		return ErrEmptyFileExtension
	}

	// State transition
	now := time.Now()
	d.Status = StatusCompleted
	d.StoragePath = &storagePath
	d.FileHash = &fileHash
	d.FileExtension = &fileExtension
	d.CompletedAt = &now
	d.UpdatedAt = now
	d.ErrorMessage = nil // Clear any error

	return nil
}

// Fail marks the download as failed - simple state transition only
func (d *Download) Fail(errorMessage string) error {
	if d.Status == StatusCompleted {
		return ErrAlreadyCompleted
	}
	if d.Status != StatusInProgress {
		return ErrNotInProgress // Can only fail from in_progress
	}

	d.Status = StatusFailed
	d.ErrorMessage = &errorMessage
	d.UpdatedAt = time.Now()

	return nil
}

func (d *Download) MaxAttempts() int {
	return 3
}

// ============================================================================
// QUERY METHODS (Business logic queries)
// ============================================================================

// IsCompleted checks if the download is completed
func (d *Download) IsCompleted() bool {
	return d.Status == StatusCompleted
}

// IsInProgress checks if the download is currently in progress
func (d *Download) IsInProgress() bool {
	return d.Status == StatusInProgress
}

// IsFailed checks if the download has failed
func (d *Download) IsFailed() bool {
	return d.Status == StatusFailed
}

// CanRetry checks if the download can be retried
func (d *Download) CanRetry() bool {
	return d.Status == StatusFailed && d.AttemptCount < d.MaxAttempts()
}

// ShouldRetry determines if download should be retried
func (d *Download) ShouldRetry() bool {
	return d.Status == StatusFailed && d.AttemptCount < d.MaxAttempts()
}

// HasExceededMaxAttempts checks if max attempts have been exceeded
func (d *Download) HasExceededMaxAttempts() bool {
	return d.AttemptCount >= d.MaxAttempts()
}

// AttemptsRemaining returns the number of attempts remaining
func (d *Download) AttemptsRemaining() int {
	remaining := d.MaxAttempts() - d.AttemptCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Duration returns the download duration (if completed)
func (d *Download) Duration() *time.Duration {
	if d.StartedAt == nil || d.CompletedAt == nil {
		return nil
	}
	duration := d.CompletedAt.Sub(*d.StartedAt)
	return &duration
}
