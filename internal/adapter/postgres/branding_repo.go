package postgres

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// BrandingRepository implémente port.BrandingRepository.
type BrandingRepository struct {
	pool *pgxpool.Pool
}

// NewBrandingRepository crée un nouveau repository branding.
func NewBrandingRepository(pool *pgxpool.Pool) *BrandingRepository {
	return &BrandingRepository{pool: pool}
}

// Get retourne le branding courant.
func (r *BrandingRepository) Get(ctx context.Context) (*domain.AppBranding, error) {
	var b domain.AppBranding
	var hasLogo bool

	err := r.pool.QueryRow(ctx,
		`SELECT app_title, app_subtitle, timezone, logo_content_type != '' AS has_logo, updated_at
		 FROM app_branding WHERE id = 1`,
	).Scan(&b.AppTitle, &b.AppSubtitle, &b.Timezone, &hasLogo, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if hasLogo {
		b.LogoURL = "/api/v1/branding/logo"
	}
	return &b, nil
}

// Update met à jour le titre et le sous-titre.
func (r *BrandingRepository) Update(ctx context.Context, branding *domain.AppBranding) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE app_branding SET app_title = $1, app_subtitle = $2, timezone = $3, updated_at = $4 WHERE id = 1`,
		branding.AppTitle,
		branding.AppSubtitle,
		branding.Timezone,
		time.Now(),
	)
	return err
}

// UpdateLogo met à jour le logo (stocké en base64).
func (r *BrandingRepository) UpdateLogo(
	ctx context.Context,
	data []byte,
	contentType string,
) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	_, err := r.pool.Exec(
		ctx,
		`UPDATE app_branding SET logo_data = $1, logo_content_type = $2, updated_at = $3 WHERE id = 1`,
		encoded,
		contentType,
		time.Now(),
	)
	return err
}

// GetLogo retourne les données brutes du logo.
func (r *BrandingRepository) GetLogo(ctx context.Context) (*domain.LogoData, error) {
	var encoded, contentType string
	err := r.pool.QueryRow(ctx,
		`SELECT logo_data, logo_content_type FROM app_branding WHERE id = 1`,
	).Scan(&encoded, &contentType)
	if err != nil {
		return nil, err
	}

	if encoded == "" || contentType == "" {
		return nil, domain.ErrNotFound
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return &domain.LogoData{Data: data, ContentType: contentType}, nil
}
