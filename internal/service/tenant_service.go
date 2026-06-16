package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// TenantService implémente port.TenantService.
type TenantService struct {
	repo port.TenantRepository
}

// NewTenantService crée un nouveau service tenant.
func NewTenantService(repo port.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

// Create crée un nouveau tenant avec une clé API générée et des paramètres par défaut.
func (s *TenantService) Create(ctx context.Context, req domain.CreateTenantRequest) (*domain.Tenant, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	settings := domain.TenantSettings{
		RateLimit:        100,
		RateBurst:        200,
		MaxDestinataires: 500,
		DefaultPriority:  domain.MailPriorityDefault,
	}
	if req.Settings != nil {
		settings = *req.Settings
	}
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	slug := req.Slug
	if slug == "" {
		slug = slugify(req.Name)
	}

	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      slug,
		APIKey:    apiKey,
		IsActive:  true,
		Settings:  settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// GetByID retourne un tenant par son identifiant.
func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return s.repo.GetByID(ctx, id)
}

// List retourne la liste paginée des tenants.
func (s *TenantService) List(ctx context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, page, limit)
}

// Update met à jour les champs modifiables d'un tenant existant.
func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req domain.UpdateTenantRequest) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Slug != nil {
		tenant.Slug = *req.Slug
	}
	if req.IsActive != nil {
		tenant.IsActive = *req.IsActive
	}
	if req.Settings != nil {
		tenant.Settings = *req.Settings
	}
	if err := tenant.Settings.Validate(); err != nil {
		return nil, err
	}
	tenant.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// Delete supprime un tenant par son identifiant.
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// RegenerateAPIKey génère une nouvelle clé API pour un tenant existant.
func (s *TenantService) RegenerateAPIKey(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateAPIKey(ctx, id, newKey); err != nil {
		return nil, err
	}

	tenant.APIKey = newKey
	tenant.UpdatedAt = time.Now()
	return tenant, nil
}

// generateAPIKey génère une clé API aléatoire de 64 caractères hexadécimaux.
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
