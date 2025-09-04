package process

type ProcessStatus string

const (
	StatusPending    ProcessStatus = "pending"
	StatusInProgress ProcessStatus = "in_progress"
	StatusCompleted  ProcessStatus = "completed"
	StatusFailed     ProcessStatus = "failed"
)
