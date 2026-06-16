package service

import (
	"context"
	"fmt"
	"time"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

const maxLogoSize = 512 * 1024 // 512 Ko

var allowedLogoTypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/svg+xml": true,
}

// BrandingService implémente port.BrandingService.
type BrandingService struct {
	repo port.BrandingRepository
}

// NewBrandingService crée un nouveau service de branding.
func NewBrandingService(repo port.BrandingRepository) *BrandingService {
	return &BrandingService{repo: repo}
}

// Get retourne le branding courant.
func (s *BrandingService) Get(ctx context.Context) (*domain.AppBranding, error) {
	return s.repo.Get(ctx)
}

// Update met à jour le branding (titre, sous-titre).
func (s *BrandingService) Update(
	ctx context.Context,
	req domain.UpdateBrandingRequest,
) (*domain.AppBranding, error) {
	branding, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if req.AppTitle != nil {
		branding.AppTitle = *req.AppTitle
	}
	if req.AppSubtitle != nil {
		branding.AppSubtitle = *req.AppSubtitle
	}
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			return nil, fmt.Errorf(
				"timezone invalide (%s) : %w",
				*req.Timezone,
				domain.ErrValidation,
			)
		}
		branding.Timezone = *req.Timezone
	}

	if err := s.repo.Update(ctx, branding); err != nil {
		return nil, err
	}

	// Mettre à jour la timezone globale pour le parsing des dates.
	ApplyTimezone(branding.Timezone)

	return s.repo.Get(ctx)
}

// ApplyTimezone met à jour domain.AppTimezone à partir d'un nom de timezone IANA.
func ApplyTimezone(tz string) {
	if loc, err := time.LoadLocation(tz); err == nil {
		domain.AppTimezone = loc
	}
}

// UploadLogo valide et enregistre le logo.
func (s *BrandingService) UploadLogo(ctx context.Context, data []byte, contentType string) error {
	if !allowedLogoTypes[contentType] {
		return fmt.Errorf(
			"type de fichier non autorisé (%s) : %w",
			contentType,
			domain.ErrValidation,
		)
	}
	if len(data) > maxLogoSize {
		return fmt.Errorf("le logo dépasse la taille maximale de 512 Ko : %w", domain.ErrValidation)
	}
	return s.repo.UpdateLogo(ctx, data, contentType)
}

// GetLogo retourne les données brutes du logo.
func (s *BrandingService) GetLogo(ctx context.Context) (*domain.LogoData, error) {
	return s.repo.GetLogo(ctx)
}
