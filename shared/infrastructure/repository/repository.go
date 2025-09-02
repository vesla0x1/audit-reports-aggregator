package repository

import (
	"fmt"
	"shared/application/ports"
	"shared/domain/entity"
	"shared/domain/repository"
)

type Repositories struct {
	download      repository.DownloadRepository
	auditReport   repository.AuditReportRepository
	auditProvider repository.AuditProviderRepository
	process       repository.ProcessRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db ports.Database, obs ports.Observability) (*Repositories, error) {
	logger, metrics, err := obs.ComponentsScoped("repository")
	if err != nil {
		return nil, fmt.Errorf("failed to get observability: %w", err)
	}

	return &Repositories{
		download:      newDownloadRepository(db, logger, metrics),
		auditReport:   newAuditReportRepository(db, logger, metrics),
		auditProvider: newAuditProviderRepository(db, logger, metrics),
		process:       newProcessRepository(db, logger, metrics),
	}, nil
}

// Each repository constructor
func newDownloadRepository(db ports.Database, logger ports.Logger, metrics ports.Metrics) repository.DownloadRepository {
	repo := &downloadRepository{}
	repo.baseRepository = newBaseRepository[entity.Download](db, logger, metrics, "downloads")
	return repo
}

func newAuditReportRepository(db ports.Database, logger ports.Logger, metrics ports.Metrics) repository.AuditReportRepository {
	repo := &auditReportRepository{}
	repo.baseRepository = newBaseRepository[entity.AuditReport](db, logger, metrics, "audit_reports")
	return repo
}

func newAuditProviderRepository(db ports.Database, logger ports.Logger, metrics ports.Metrics) repository.AuditProviderRepository {
	repo := &auditProviderRepository{}
	repo.baseRepository = newBaseRepository[entity.AuditProvider](db, logger, metrics, "audit_providers")
	return repo
}

func newProcessRepository(db ports.Database, logger ports.Logger, metrics ports.Metrics) repository.ProcessRepository {
	repo := &processRepository{}
	repo.baseRepository = newBaseRepository[entity.Process](db, logger, metrics, "processes")
	return repo
}

func (r *Repositories) Download() repository.DownloadRepository {
	return r.download
}

func (r *Repositories) AuditReport() repository.AuditReportRepository {
	return r.auditReport
}

func (r *Repositories) AuditProvider() repository.AuditProviderRepository {
	return r.auditProvider
}

func (r *Repositories) Process() repository.ProcessRepository {
	return r.process
}
