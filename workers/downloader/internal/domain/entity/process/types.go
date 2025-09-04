package process

import (
	"shared/domain/entity/process"
)

type (
	Process = process.Process
)

func NewProcess(downloadId int64) *Process { return process.NewProcess(downloadId) }
