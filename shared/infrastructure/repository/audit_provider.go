package repository

import (
	"context"
	"fmt"
	"shared/domain/entity"

	"github.com/Masterminds/squirrel"
)

type auditProviderRepository struct {
	*baseRepository[entity.AuditProvider]
}

func (r *auditProviderRepository) Create(ctx context.Context, provider *entity.AuditProvider) error {
	query := r.qb.Insert("audit_providers").
		Columns(
			"name", "slug", "website_url", "description",
			"provider_type", "is_active", "created_at", "updated_at",
		).
		Values(
			provider.Name, provider.Slug, provider.WebsiteURL, provider.Description,
			provider.ProviderType, provider.IsActive, provider.CreatedAt, provider.UpdatedAt,
		).
		Suffix("RETURNING id")

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)
	err := row.Scan(&provider.ID)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

func (r *auditProviderRepository) Update(ctx context.Context, provider *entity.AuditProvider) error {
	query := r.qb.Update("audit_providers").
		Set("name", provider.Name).
		Set("slug", provider.Slug).
		Set("provider_type", provider.ProviderType).
		Set("is_active", provider.IsActive).
		Set("updated_at", provider.UpdatedAt)

	if provider.WebsiteURL != nil {
		query = query.Set("website_url", *provider.WebsiteURL)
	}
	if provider.Description != nil {
		query = query.Set("description", *provider.Description)
	}

	query = query.Where(squirrel.Eq{"id": provider.ID})

	sql, args, _ := query.ToSql()
	_, err := r.db.Execute(ctx, sql, args...)
	return err
}

func (r *auditProviderRepository) GetBySlug(ctx context.Context, slug string) (*entity.AuditProvider, error) {
	query := r.qb.Select("*").
		From("audit_providers").
		Where(squirrel.Eq{"slug": slug})

	sql, args, _ := query.ToSql()
	row := r.db.QueryRow(ctx, sql, args...)

	var p entity.AuditProvider
	err := row.Scan(
		&p.ID, &p.Name, &p.Slug, &p.WebsiteURL, &p.Description,
		&p.ProviderType, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *auditProviderRepository) List(ctx context.Context, onlyActive bool) ([]*entity.AuditProvider, error) {
	query := r.qb.Select("*").From("audit_providers")

	if onlyActive {
		query = query.Where(squirrel.Eq{"is_active": true})
	}

	query = query.OrderBy("name ASC")

	sql, args, _ := query.ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*entity.AuditProvider
	for rows.Next() {
		var p entity.AuditProvider
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.WebsiteURL, &p.Description,
			&p.ProviderType, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		providers = append(providers, &p)
	}

	return providers, rows.Err()
}
