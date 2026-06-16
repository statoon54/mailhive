package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/config"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{Secret: "test-secret-key-32bytes-long!!", Expiration: time.Hour}
}

func TestAuthService_GenerateToken_AdminKey(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	svc := NewAuthService(repo, testJWTConfig(), "admin-key-123")

	token, err := svc.GenerateToken(context.Background(), "admin-key-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Le tenant admin doit être créé
	assert.True(t, repo.Called("GetBySlug"))
	assert.True(t, repo.Called("Create"))
}

func TestAuthService_GenerateToken_TenantKey(t *testing.T) {
	tenantID := uuid.New()
	repo := mocks.NewMockTenantRepo()
	repo.Tenants[tenantID] = &domain.Tenant{
		ID:       tenantID,
		Name:     "Test",
		Slug:     "test",
		APIKey:   "tenant-key-abc",
		IsActive: true,
	}
	svc := NewAuthService(repo, testJWTConfig(), "admin-key-123")

	token, err := svc.GenerateToken(context.Background(), "tenant-key-abc")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_GenerateToken_InvalidKey(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	svc := NewAuthService(repo, testJWTConfig(), "admin-key-123")

	_, err := svc.GenerateToken(context.Background(), "invalid-key")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAPIKey)
}

func TestAuthService_GenerateToken_InactiveTenant(t *testing.T) {
	tenantID := uuid.New()
	repo := mocks.NewMockTenantRepo()
	repo.Tenants[tenantID] = &domain.Tenant{
		ID:       tenantID,
		APIKey:   "tenant-key-abc",
		IsActive: false,
	}
	svc := NewAuthService(repo, testJWTConfig(), "admin-key-123")

	_, err := svc.GenerateToken(context.Background(), "tenant-key-abc")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTenantInactive)
}

func TestAuthService_GenerateToken_AdminConcurrentCreate(t *testing.T) {
	// Simulate admin tenant already exists on second call
	repo := mocks.NewMockTenantRepo()
	adminID := uuid.New()
	repo.Tenants[adminID] = &domain.Tenant{
		ID:       adminID,
		Name:     "Administration",
		Slug:     "admin",
		APIKey:   "admin-key",
		IsActive: true,
	}
	svc := NewAuthService(repo, testJWTConfig(), "admin-key")

	token, err := svc.GenerateToken(context.Background(), "admin-key")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_RefreshToken_Valid(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	cfg := testJWTConfig()
	svc := NewAuthService(repo, cfg, "admin-key")

	// Create a valid token first
	claims := &JWTClaims{
		TenantID:   uuid.New().String(),
		TenantSlug: "test",
		Role:       "tenant",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	newToken, err := svc.RefreshToken(context.Background(), signed)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, signed, newToken)
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	svc := NewAuthService(repo, testJWTConfig(), "admin-key")

	_, err := svc.RefreshToken(context.Background(), "invalid-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestAuthService_RefreshToken_WrongSigningMethod(t *testing.T) {
	repo := mocks.NewMockTenantRepo()
	svc := NewAuthService(repo, testJWTConfig(), "admin-key")

	// Create token with wrong signing method (none)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"tenant_id": uuid.New().String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := svc.RefreshToken(context.Background(), signed)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}
