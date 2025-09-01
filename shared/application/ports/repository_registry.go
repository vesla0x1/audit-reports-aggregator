package ports

import (
	"shared/domain/repository"
)

type Repositories interface {
	AuditReport() repository.AuditReportRepository
}
