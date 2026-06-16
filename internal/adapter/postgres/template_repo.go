package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// TemplateRepository implémente port.TemplateRepository avec PostgreSQL.
type TemplateRepository struct {
	pool *pgxpool.Pool
}

// NewTemplateRepository crée un nouveau repository template.
func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

// Create insère un nouveau template en base de données.
func (r *TemplateRepository) Create(ctx context.Context, tmpl *domain.Template) error {
	varsJSON, err := json.Marshal(tmpl.Variables)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation des variables : %w", err)
	}

	query := `
		INSERT INTO mail_templates (id, tenant_id, name, slug, subject_tmpl, text_body, html_body,
			variables, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = r.pool.Exec(ctx, query,
		tmpl.ID, tmpl.TenantID, tmpl.Name, tmpl.Slug, tmpl.SubjectTmpl,
		tmpl.TextBody, tmpl.HTMLBody, varsJSON, tmpl.IsActive,
		tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("erreur de création du template : %w", err)
	}
	return nil
}

// GetByID retourne un template par son identifiant.
func (r *TemplateRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	query := `
		SELECT id, tenant_id, name, slug, subject_tmpl, text_body, html_body,
			variables, is_active, created_at, updated_at
		FROM mail_templates WHERE id = $1 AND tenant_id = $2`

	return r.scanTemplate(ctx, query, id, tenantID)
}

// GetBySlug retourne un template par son slug.
func (r *TemplateRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Template, error) {
	query := `
		SELECT id, tenant_id, name, slug, subject_tmpl, text_body, html_body,
			variables, is_active, created_at, updated_at
		FROM mail_templates WHERE tenant_id = $1 AND slug = $2`

	return r.scanTemplate(ctx, query, tenantID, slug)
}

// List retourne tous les templates du tenant.
func (r *TemplateRepository) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Template, error) {
	query := `
		SELECT id, tenant_id, name, slug, subject_tmpl, text_body, html_body,
			variables, is_active, created_at, updated_at
		FROM mail_templates WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des templates : %w", err)
	}
	defer rows.Close()

	var templates []domain.Template
	for rows.Next() {
		var tmpl domain.Template
		var varsJSON []byte
		err := rows.Scan(&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Slug,
			&tmpl.SubjectTmpl, &tmpl.TextBody, &tmpl.HTMLBody,
			&varsJSON, &tmpl.IsActive, &tmpl.CreatedAt, &tmpl.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erreur de lecture du template : %w", err)
		}
		if err := json.Unmarshal(varsJSON, &tmpl.Variables); err != nil {
			return nil, fmt.Errorf("erreur de désérialisation des variables : %w", err)
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}

// Update met à jour un template existant.
func (r *TemplateRepository) Update(ctx context.Context, tmpl *domain.Template) error {
	varsJSON, err := json.Marshal(tmpl.Variables)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation des variables : %w", err)
	}

	query := `
		UPDATE mail_templates SET name = $3, slug = $4, subject_tmpl = $5, text_body = $6,
			html_body = $7, variables = $8, is_active = $9, updated_at = $10
		WHERE id = $1 AND tenant_id = $2`

	_, err = r.pool.Exec(ctx, query,
		tmpl.ID, tmpl.TenantID, tmpl.Name, tmpl.Slug, tmpl.SubjectTmpl,
		tmpl.TextBody, tmpl.HTMLBody, varsJSON, tmpl.IsActive, tmpl.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour du template : %w", err)
	}
	return nil
}

// Delete supprime un template.
func (r *TemplateRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM mail_templates WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("erreur de suppression du template : %w", err)
	}
	return nil
}

// scanTemplate exécute une requête et scanne le résultat en Template.
func (r *TemplateRepository) scanTemplate(ctx context.Context, query string, args ...any) (*domain.Template, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	var tmpl domain.Template
	var varsJSON []byte

	err := row.Scan(&tmpl.ID, &tmpl.TenantID, &tmpl.Name, &tmpl.Slug,
		&tmpl.SubjectTmpl, &tmpl.TextBody, &tmpl.HTMLBody,
		&varsJSON, &tmpl.IsActive, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("erreur de lecture du template : %w", err)
	}

	if err := json.Unmarshal(varsJSON, &tmpl.Variables); err != nil {
		return nil, fmt.Errorf("erreur de désérialisation des variables : %w", err)
	}
	return &tmpl, nil
}
