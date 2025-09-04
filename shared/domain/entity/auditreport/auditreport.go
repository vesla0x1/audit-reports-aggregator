package auditreport

import (
	"fmt"
	"regexp"
	"shared/domain/entity/auditprovider"
	"strings"
	"time"
)

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

func (r *AuditReport) StoragePath(provider *auditprovider.AuditProvider, extension string) string {
	return r.generatePath(provider.Slug, extension)
}

func (r *AuditReport) generatePath(providerSlug string, extension string) string {
	sanitizedTitle := r.sanitizeTitle()
	return fmt.Sprintf(
		"%s/%d_%s%s",
		providerSlug,
		r.ID,
		sanitizedTitle,
		extension,
	)
}

func (r *AuditReport) sanitizeTitle() string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	sanitized := reg.ReplaceAllString(r.Title, "_")

	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return strings.ToLower(strings.Trim(sanitized, "_"))
}
