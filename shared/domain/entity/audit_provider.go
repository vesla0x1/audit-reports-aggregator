package entity

import "time"

type ProviderType string

const (
	ProviderTypeMarketplace ProviderType = "marketplace"
	ProviderTypeFirm        ProviderType = "firm"
	ProviderTypeIndividual  ProviderType = "individual"
)

type AuditProvider struct {
	ID           int64        `db:"id"`
	Name         string       `db:"name"`
	Slug         string       `db:"slug"`
	WebsiteURL   *string      `db:"website_url"`
	Description  *string      `db:"description"`
	ProviderType ProviderType `db:"provider_type"`
	IsActive     bool         `db:"is_active"`
	CreatedAt    time.Time    `db:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"`
}
