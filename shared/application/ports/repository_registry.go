package ports

import (
	"shared/domain/repository"
)

type Repositories interface {
	AuditReport() repository.AuditReportRepository
	Download() repository.DownloadRepository
	Process() repository.ProcessRepository
	AuditProvider() repository.AuditProviderRepository
}
