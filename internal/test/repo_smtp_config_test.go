package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func insertTenant(t *testing.T, pool *pgxpool.Pool) *domain.Tenant {
	t.Helper()
	repo := postgres.NewTenantRepository(pool)
	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      "Test Tenant",
		Slug:      "t-" + uuid.New().String()[:8],
		APIKey:    "k-" + uuid.New().String()[:16],
		IsActive:  true,
		Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(context.Background(), tenant))
	return tenant
}

func newSMTPConfig(tenantID uuid.UUID, isDefault bool) *domain.SMTPConfig {
	return &domain.SMTPConfig{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "SMTP " + uuid.New().String()[:6],
		Host:       "smtp.example.com",
		Port:       587,
		AuthMethod: domain.AuthPlain,
		TLSPolicy:  domain.TLSMandatory,
		FromEmail:  "noreply@example.com",
		FromName:   "Test",
		Charset:    domain.CharsetUTF8,
		Encoding:   domain.EncodingQP,
		IsDefault:  isDefault,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestSMTPConfigRepo_CreateAndGetByID(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	cfg := newSMTPConfig(tenant.ID, false)

	require.NoError(t, repo.Create(ctx, cfg))

	got, err := repo.GetByID(ctx, tenant.ID, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, cfg.Name, got.Name)
	assert.Equal(t, cfg.Host, got.Host)
	assert.Equal(t, cfg.Port, got.Port)
}

func TestSMTPConfigRepo_GetDefault(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	cfg := newSMTPConfig(tenant.ID, true)
	require.NoError(t, repo.Create(ctx, cfg))

	got, err := repo.GetDefault(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, got.ID)
	assert.True(t, got.IsDefault)
}

func TestSMTPConfigRepo_GetDefault_NotFound(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	_, err := repo.GetDefault(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrSMTPConfigNotFound)
}

func TestSMTPConfigRepo_List(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	require.NoError(t, repo.Create(ctx, newSMTPConfig(tenant.ID, true)))
	require.NoError(t, repo.Create(ctx, newSMTPConfig(tenant.ID, false)))

	list, err := repo.List(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSMTPConfigRepo_Update(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	cfg := newSMTPConfig(tenant.ID, false)
	require.NoError(t, repo.Create(ctx, cfg))

	cfg.Name = "Updated SMTP"
	cfg.Port = 465
	cfg.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, cfg))

	got, err := repo.GetByID(ctx, tenant.ID, cfg.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated SMTP", got.Name)
	assert.Equal(t, 465, got.Port)
}

func TestSMTPConfigRepo_Delete(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	cfg := newSMTPConfig(tenant.ID, false)
	require.NoError(t, repo.Create(ctx, cfg))

	require.NoError(t, repo.Delete(ctx, tenant.ID, cfg.ID))

	_, err := repo.GetByID(ctx, tenant.ID, cfg.ID)
	assert.ErrorIs(t, err, domain.ErrSMTPConfigNotFound)
}

func TestSMTPConfigRepo_ClearDefault(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewSMTPConfigRepository(pool)
	ctx := context.Background()

	tenant := insertTenant(t, pool)
	cfg := newSMTPConfig(tenant.ID, true)
	require.NoError(t, repo.Create(ctx, cfg))

	require.NoError(t, repo.ClearDefault(ctx, tenant.ID))

	got, err := repo.GetByID(ctx, tenant.ID, cfg.ID)
	require.NoError(t, err)
	assert.False(t, got.IsDefault)
}
