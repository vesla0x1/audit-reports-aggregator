package entity

import "time"

type EngagementType string

const (
	EngagementTypeCompetition EngagementType = "competition"
	EngagementTypePrivate     EngagementType = "private"
	EngagementTypeBugBounty   EngagementType = "bug_bounty"
	EngagementTypeSolo        EngagementType = "solo"
)

type AuditReport struct {
	ID                int64          `db:"id"`
	SourceID          int64          `db:"source_id"`
	ProviderID        int64          `db:"provider_id"`
	Title             string         `db:"title"`
	EngagementType    EngagementType `db:"engagement_type"`
	ClientCompany     *string        `db:"client_company"`
	AuditStartDate    *time.Time     `db:"audit_start_date"`
	AuditEndDate      *time.Time     `db:"audit_end_date"`
	DetailsPageURL    string         `db:"details_page_url"`
	SourceDownloadURL string         `db:"source_download_url"`
	RepositoryURL     *string        `db:"repository_url"`
	Summary           *string        `db:"summary"`
	FindingsSummary   *string        `db:"findings_summary"` // JSONB as string
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
}
