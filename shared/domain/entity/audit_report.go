package domain

import (
	"fmt"
	"strings"
	"time"
)

// AuditReportType represents the type of audit report
type AuditReportType string

const (
	AuditReportTypeCompetition AuditReportType = "competition"
	AuditReportTypePrivate     AuditReportType = "private"
	AuditReportTypeBugBounty   AuditReportType = "bug_bounty"
)

// AuditReport represents an audit report entity
type AuditReport struct {
	ID                int64           `json:"id"`
	SourceID          int64           `json:"source_id"`
	PlatformID        int64           `json:"platform_id"`
	Title             string          `json:"title"`
	Type              AuditReportType `json:"type"`
	AuditedCompany    string          `json:"audited_company"`
	Period            *string         `json:"period,omitempty"`
	DetailsPageURL    *string         `json:"details_page_url,omitempty"`
	SourceDownloadURL *string         `json:"source_download_url,omitempty"`
	Summary           *string         `json:"summary,omitempty"`
	FindingsSummary   *string         `json:"findings_summary,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// NewAuditReport creates a new audit report with validation
func NewAuditReport(sourceID int64, title string, reportType AuditReportType, company string) (*AuditReport, error) {
	report := &AuditReport{
		SourceID:       sourceID,
		Title:          strings.TrimSpace(title),
		Type:           reportType,
		AuditedCompany: strings.TrimSpace(company),
	}

	if err := report.Validate(); err != nil {
		return nil, err
	}

	return report, nil
}

// Validate checks if the audit report has valid data
func (a *AuditReport) Validate() error {
	if a.SourceID <= 0 {
		return fmt.Errorf("invalid source ID")
	}

	if strings.TrimSpace(a.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}

	if !a.Type.IsValid() {
		return fmt.Errorf("invalid report type: %s", a.Type)
	}

	if strings.TrimSpace(a.AuditedCompany) == "" {
		return fmt.Errorf("audited company cannot be empty")
	}

	return nil
}

// IsValid checks if the report type is valid
func (t AuditReportType) IsValid() bool {
	switch t {
	case AuditReportTypeCompetition, AuditReportTypePrivate, AuditReportTypeBugBounty:
		return true
	default:
		return false
	}
}

// IsProcessed checks if the report has been processed
func (a *AuditReport) IsProcessed() bool {
	return a.Summary != nil && a.FindingsSummary != nil
}

// SetPeriod sets the period with validation
func (a *AuditReport) SetPeriod(period string) error {
	period = strings.TrimSpace(period)
	if period == "" {
		a.Period = nil
		return nil
	}

	if len(period) > 100 {
		return fmt.Errorf("period cannot exceed 100 characters")
	}

	a.Period = &period
	return nil
}
