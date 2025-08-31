package repository

// Repositories contains all repository instances
type Repositories struct {
	AuditReport AuditReportRepository
}

// NewRepositories creates a new repository container
func NewRepositories(
	auditReport AuditReportRepository,
) *Repositories {
	return &Repositories{
		AuditReport: auditReport,
	}
}

// HasDatabase checks if repositories are available (database is configured)
func (r *Repositories) HasDatabase() bool {
	return r != nil && r.AuditReport != nil
}
