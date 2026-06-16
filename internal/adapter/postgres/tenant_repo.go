package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// TenantRepository implémente port.TenantRepository avec PostgreSQL.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository crée un nouveau repository tenant.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// Create insère un nouveau tenant en base de données.
func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation des paramètres : %w", err)
	}

	query := `
		INSERT INTO tenants (id, name, slug, api_key, is_active, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.pool.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.APIKey,
		tenant.IsActive, settingsJSON, tenant.CreatedAt, tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de création du tenant : %w", err)
	}
	return nil
}

// GetByID retourne un tenant par son identifiant.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, api_key, is_active, settings, created_at, updated_at
		FROM tenants WHERE id = $1`

	return r.scanTenant(ctx, query, id)
}

// GetByAPIKey retourne un tenant par sa clé API.
func (r *TenantRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, api_key, is_active, settings, created_at, updated_at
		FROM tenants WHERE api_key = $1`

	return r.scanTenant(ctx, query, apiKey)
}

// GetBySlug retourne un tenant par son slug.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, api_key, is_active, settings, created_at, updated_at
		FROM tenants WHERE slug = $1`

	return r.scanTenant(ctx, query, slug)
}

// List retourne la liste paginée des tenants.
func (r *TenantRepository) List(ctx context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error) {
	offset := (page - 1) * limit

	var total int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tenants").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("erreur de comptage des tenants : %w", err)
	}

	query := `
		SELECT id, name, slug, api_key, is_active, settings, created_at, updated_at
		FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des tenants : %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		t, err := r.scanTenantRow(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, *t)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &domain.PaginatedList[domain.Tenant]{
		Items:      tenants,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// Update met à jour un tenant existant.
func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation des paramètres : %w", err)
	}

	query := `
		UPDATE tenants SET name = $2, slug = $3, is_active = $4, settings = $5, updated_at = $6
		WHERE id = $1`

	_, err = r.pool.Exec(ctx, query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.IsActive, settingsJSON, tenant.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour du tenant : %w", err)
	}
	return nil
}

// UpdateAPIKey met à jour la clé API d'un tenant.
func (r *TenantRepository) UpdateAPIKey(ctx context.Context, id uuid.UUID, newKey string) error {
	query := `UPDATE tenants SET api_key = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, newKey)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour de la clé API : %w", err)
	}
	return nil
}

// Delete désactive un tenant par son identifiant.
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erreur de désactivation du tenant : %w", err)
	}
	return nil
}

// scanTenant exécute une requête et scanne le résultat en Tenant.
func (r *TenantRepository) scanTenant(ctx context.Context, query string, args ...any) (*domain.Tenant, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	var t domain.Tenant
	var settingsJSON []byte

	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.APIKey, &t.IsActive, &settingsJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("erreur de lecture du tenant : %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("erreur de désérialisation des paramètres : %w", err)
	}
	return &t, nil
}

// scanTenantRow scanne une ligne de résultat en Tenant.
func (r *TenantRepository) scanTenantRow(rows pgx.Rows) (*domain.Tenant, error) {
	var t domain.Tenant
	var settingsJSON []byte

	err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.APIKey, &t.IsActive, &settingsJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("erreur de lecture du tenant : %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &t.Settings); err != nil {
		return nil, fmt.Errorf("erreur de désérialisation des paramètres : %w", err)
	}
	return &t, nil
}
