package repository

import (
	"fmt"
	"shared/application/ports"
	domain "shared/domain/entity"
	"shared/domain/repository"
)

type Repositories struct {
	auditReport repository.AuditReportRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db ports.Database, obs ports.Observability) (*Repositories, error) {
	logger, metrics, err := obs.ComponentsScoped("repository")
	if err != nil {
		return nil, fmt.Errorf("failed to get observability: %w", err)
	}

	return &Repositories{
		auditReport: newAuditReportRepository(db, logger, metrics),
	}, nil
}

// Each repository constructor
func newAuditReportRepository(db ports.Database, logger ports.Logger, metrics ports.Metrics) repository.AuditReportRepository {
	repo := &auditReportRepository{}
	repo.baseRepository = newBaseRepository[*domain.AuditReport](db, logger, metrics, "audit_reports")
	return repo
}

func (r *Repositories) AuditReport() repository.AuditReportRepository {
	return r.auditReport
}
