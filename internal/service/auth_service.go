package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// JWTClaims contient les claims personnalisés du JWT.
type JWTClaims struct {
	jwt.RegisteredClaims
	Role       string `json:"role"`
	TenantID   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
}

// AuthService implémente port.AuthService.
type AuthService struct {
	tenantRepo port.TenantRepository
	cfg        config.JWTConfig
	adminKey   string
}

// NewAuthService crée un nouveau service d'authentification.
func NewAuthService(
	tenantRepo port.TenantRepository,
	cfg config.JWTConfig,
	adminKey string,
) *AuthService {
	return &AuthService{
		tenantRepo: tenantRepo,
		cfg:        cfg,
		adminKey:   adminKey,
	}
}

// GenerateToken génère un JWT à partir d'une clé API (admin ou tenant).
func (s *AuthService) GenerateToken(ctx context.Context, apiKey string) (string, error) {
	// Vérification de la clé admin
	if apiKey == s.adminKey {
		tenant, err := s.getOrCreateAdminTenant(ctx)
		if err != nil {
			return "", fmt.Errorf("erreur tenant admin : %w", err)
		}
		return s.createToken(tenant.ID.String(), tenant.Slug, "admin")
	}

	// Recherche du tenant par clé API
	tenant, err := s.tenantRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return "", domain.ErrInvalidAPIKey
	}

	if !tenant.IsActive {
		return "", domain.ErrTenantInactive
	}

	return s.createToken(tenant.ID.String(), tenant.Slug, "tenant")
}

// getOrCreateAdminTenant récupère ou crée le tenant système pour l'admin.
func (s *AuthService) getOrCreateAdminTenant(ctx context.Context) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetBySlug(ctx, "admin")
	if err == nil {
		return tenant, nil
	}

	now := time.Now()
	tenant = &domain.Tenant{
		ID:       uuid.New(),
		Name:     "Administration",
		Slug:     "admin",
		APIKey:   s.adminKey,
		IsActive: true,
		Settings: domain.TenantSettings{
			RateLimit:        100,
			RateBurst:        200,
			MaxDestinataires: 100,
			DefaultPriority:  domain.MailPriorityCritical,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		// En cas de conflit (création concurrente), on retente la lecture
		t, err2 := s.tenantRepo.GetBySlug(ctx, "admin")
		if err2 != nil {
			return nil, err
		}
		return t, nil
	}
	return tenant, nil
}

// RefreshToken renouvelle un JWT valide en prolongeant son expiration.
func (s *AuthService) RefreshToken(ctx context.Context, tokenString string) (string, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue : %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})

	if err != nil || !token.Valid {
		return "", domain.ErrUnauthorized
	}

	return s.createToken(claims.TenantID, claims.TenantSlug, claims.Role)
}

// createToken crée et signe un JWT avec les claims tenant et rôle.
func (s *AuthService) createToken(tenantID, tenantSlug, role string) (string, error) {
	claims := &JWTClaims{
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.Expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mailhive",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("erreur de signature du JWT : %w", err)
	}

	return signedToken, nil
}
