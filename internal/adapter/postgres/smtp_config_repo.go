package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// SMTPConfigRepository implémente port.SMTPConfigRepository avec PostgreSQL.
type SMTPConfigRepository struct {
	pool *pgxpool.Pool
}

// NewSMTPConfigRepository crée un nouveau repository config SMTP.
func NewSMTPConfigRepository(pool *pgxpool.Pool) *SMTPConfigRepository {
	return &SMTPConfigRepository{pool: pool}
}

// Create insère une nouvelle configuration SMTP.
func (r *SMTPConfigRepository) Create(ctx context.Context, cfg *domain.SMTPConfig) error {
	query := `
		INSERT INTO smtp_configs (id, tenant_id, name, host, port, username, password, auth_method,
			tls_policy, from_email, from_name, charset, encoding, is_default, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := r.pool.Exec(ctx, query,
		cfg.ID, cfg.TenantID, cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password,
		cfg.AuthMethod, cfg.TLSPolicy, cfg.FromEmail, cfg.FromName, cfg.Charset, cfg.Encoding,
		cfg.IsDefault, cfg.IsActive, cfg.CreatedAt, cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de création de la config SMTP : %w", err)
	}
	return nil
}

// GetByID retourne une configuration SMTP par son identifiant.
func (r *SMTPConfigRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error) {
	query := `
		SELECT id, tenant_id, name, host, port, username, password, auth_method, tls_policy,
			from_email, from_name, charset, encoding, is_default, is_active, created_at, updated_at
		FROM smtp_configs WHERE id = $1 AND tenant_id = $2`

	return r.scanConfig(ctx, query, id, tenantID)
}

// GetDefault retourne la configuration SMTP par défaut du tenant.
func (r *SMTPConfigRepository) GetDefault(ctx context.Context, tenantID uuid.UUID) (*domain.SMTPConfig, error) {
	query := `
		SELECT id, tenant_id, name, host, port, username, password, auth_method, tls_policy,
			from_email, from_name, charset, encoding, is_default, is_active, created_at, updated_at
		FROM smtp_configs WHERE tenant_id = $1 AND is_default = true AND is_active = true`

	return r.scanConfig(ctx, query, tenantID)
}

// List retourne toutes les configurations SMTP du tenant.
func (r *SMTPConfigRepository) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SMTPConfig, error) {
	query := `
		SELECT id, tenant_id, name, host, port, username, password, auth_method, tls_policy,
			from_email, from_name, charset, encoding, is_default, is_active, created_at, updated_at
		FROM smtp_configs WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des configs SMTP : %w", err)
	}
	defer rows.Close()

	var configs []domain.SMTPConfig
	for rows.Next() {
		var cfg domain.SMTPConfig
		err := rows.Scan(&cfg.ID, &cfg.TenantID, &cfg.Name, &cfg.Host, &cfg.Port,
			&cfg.Username, &cfg.Password, &cfg.AuthMethod, &cfg.TLSPolicy,
			&cfg.FromEmail, &cfg.FromName, &cfg.Charset, &cfg.Encoding,
			&cfg.IsDefault, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("erreur de lecture de la config SMTP : %w", err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// Update met à jour une configuration SMTP existante.
func (r *SMTPConfigRepository) Update(ctx context.Context, cfg *domain.SMTPConfig) error {
	query := `
		UPDATE smtp_configs SET name = $3, host = $4, port = $5, username = $6, password = $7,
			auth_method = $8, tls_policy = $9, from_email = $10, from_name = $11,
			charset = $12, encoding = $13, is_default = $14, is_active = $15, updated_at = $16
		WHERE id = $1 AND tenant_id = $2`

	_, err := r.pool.Exec(ctx, query,
		cfg.ID, cfg.TenantID, cfg.Name, cfg.Host, cfg.Port, cfg.Username, cfg.Password,
		cfg.AuthMethod, cfg.TLSPolicy, cfg.FromEmail, cfg.FromName, cfg.Charset, cfg.Encoding,
		cfg.IsDefault, cfg.IsActive, cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour de la config SMTP : %w", err)
	}
	return nil
}

// Delete supprime une configuration SMTP.
func (r *SMTPConfigRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM smtp_configs WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("erreur de suppression de la config SMTP : %w", err)
	}
	return nil
}

// ClearDefault retire le flag par défaut de toutes les configs SMTP du tenant.
func (r *SMTPConfigRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID) error {
	query := `UPDATE smtp_configs SET is_default = false, updated_at = NOW() WHERE tenant_id = $1 AND is_default = true`
	_, err := r.pool.Exec(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("erreur de réinitialisation du défaut SMTP : %w", err)
	}
	return nil
}

// scanConfig exécute une requête et scanne le résultat en SMTPConfig.
func (r *SMTPConfigRepository) scanConfig(ctx context.Context, query string, args ...any) (*domain.SMTPConfig, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	var cfg domain.SMTPConfig

	err := row.Scan(&cfg.ID, &cfg.TenantID, &cfg.Name, &cfg.Host, &cfg.Port,
		&cfg.Username, &cfg.Password, &cfg.AuthMethod, &cfg.TLSPolicy,
		&cfg.FromEmail, &cfg.FromName, &cfg.Charset, &cfg.Encoding,
		&cfg.IsDefault, &cfg.IsActive, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSMTPConfigNotFound
		}
		return nil, fmt.Errorf("erreur de lecture de la config SMTP : %w", err)
	}
	return &cfg, nil
}
