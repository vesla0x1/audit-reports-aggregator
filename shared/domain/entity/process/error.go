package process

import "errors"

var (
	ErrAlreadyCompleted       = errors.New("process already completed")
	ErrAlreadyInProgress      = errors.New("process already in progress")
	ErrNotInProgress          = errors.New("process not in progress")
	ErrInvalidStateTransition = errors.New("invalid process state transition")
	ErrMaxAttemptsExceeded    = errors.New("maximum process attempts exceeded")
)
