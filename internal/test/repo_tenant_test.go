package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func TestTenantRepo_CreateAndGetByID(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	tenant := &domain.Tenant{
		ID:       uuid.New(),
		Name:     "Test Tenant",
		Slug:     "test-tenant-" + uuid.New().String()[:8],
		APIKey:   "key-" + uuid.New().String()[:16],
		IsActive: true,
		Settings: domain.TenantSettings{
			RateLimit:        10,
			RateBurst:        10,
			MaxDestinataires: 100,
			DefaultPriority:  domain.MailPriorityDefault,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, tenant)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, tenant.Name, got.Name)
	assert.Equal(t, tenant.Slug, got.Slug)
	assert.Equal(t, tenant.Settings.RateLimit, got.Settings.RateLimit)
	assert.Equal(t, tenant.Settings.RateBurst, got.Settings.RateBurst)
}

func TestTenantRepo_GetByAPIKey(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	apiKey := "apikey-" + uuid.New().String()[:16]
	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      "API Key Tenant",
		Slug:      "api-key-" + uuid.New().String()[:8],
		APIKey:    apiKey,
		IsActive:  true,
		Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tenant))

	got, err := repo.GetByAPIKey(ctx, apiKey)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, got.ID)
}

func TestTenantRepo_GetBySlug(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	slug := "slug-" + uuid.New().String()[:8]
	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      "Slug Tenant",
		Slug:      slug,
		APIKey:    "key-" + uuid.New().String()[:16],
		IsActive:  true,
		Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tenant))

	got, err := repo.GetBySlug(ctx, slug)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, got.ID)
}

func TestTenantRepo_GetByID_NotFound(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestTenantRepo_List(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	for i := range 3 {
		tenant := &domain.Tenant{
			ID:        uuid.New(),
			Name:      "Tenant " + uuid.New().String()[:4],
			Slug:      "list-" + uuid.New().String()[:8],
			APIKey:    "key-" + uuid.New().String()[:16],
			IsActive:  true,
			Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, tenant))
	}

	list, err := repo.List(ctx, 1, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 3)
	assert.Equal(t, 1, list.Page)
}

func TestTenantRepo_Update(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      "Before Update",
		Slug:      "update-" + uuid.New().String()[:8],
		APIKey:    "key-" + uuid.New().String()[:16],
		IsActive:  true,
		Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tenant))

	tenant.Name = "After Update"
	tenant.Settings.RateLimit = 20
	tenant.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, tenant))

	got, err := repo.GetByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, float64(20), got.Settings.RateLimit)
}

func TestTenantRepo_Delete(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Name:      "To Delete",
		Slug:      "delete-" + uuid.New().String()[:8],
		APIKey:    "key-" + uuid.New().String()[:16],
		IsActive:  true,
		Settings:  domain.TenantSettings{RateLimit: 5, RateBurst: 5, MaxDestinataires: 50, DefaultPriority: domain.MailPriorityDefault},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tenant))

	// Delete = soft delete (is_active = false)
	require.NoError(t, repo.Delete(ctx, tenant.ID))

	got, err := repo.GetByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
}

func TestTenantRepo_SettingsJSONB(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTenantRepository(pool)
	ctx := context.Background()

	threshold := float32(5.0)
	action := domain.SpamScoreActionBlock
	tenant := &domain.Tenant{
		ID:       uuid.New(),
		Name:     "Settings Test",
		Slug:     "settings-" + uuid.New().String()[:8],
		APIKey:   "key-" + uuid.New().String()[:16],
		IsActive: true,
		Settings: domain.TenantSettings{
			RateLimit:          10,
			RateBurst:          10,
			MaxDestinataires:   200,
			DefaultPriority:    domain.MailPriorityCritical,
			SpamScoreThreshold: &threshold,
			SpamScoreAction:    &action,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tenant))

	got, err := repo.GetByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailPriorityCritical, got.Settings.DefaultPriority)
	require.NotNil(t, got.Settings.SpamScoreThreshold)
	assert.Equal(t, float32(5.0), *got.Settings.SpamScoreThreshold)
	require.NotNil(t, got.Settings.SpamScoreAction)
	assert.Equal(t, domain.SpamScoreActionBlock, *got.Settings.SpamScoreAction)
}
