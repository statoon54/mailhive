package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func TestBrandingRepo_Get(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewBrandingRepository(pool)
	ctx := context.Background()

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "MailHive", got.AppTitle)
	assert.Equal(t, "Europe/Paris", got.Timezone)
}

func TestBrandingRepo_Update(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewBrandingRepository(pool)
	ctx := context.Background()

	branding := &domain.AppBranding{
		AppTitle:    "New Title",
		AppSubtitle: "New Subtitle",
		Timezone:    "America/New_York",
	}
	require.NoError(t, repo.Update(ctx, branding))

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "New Title", got.AppTitle)
	assert.Equal(t, "New Subtitle", got.AppSubtitle)
	assert.Equal(t, "America/New_York", got.Timezone)
}

func TestBrandingRepo_UploadAndGetLogo(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewBrandingRepository(pool)
	ctx := context.Background()

	logoData := []byte("fake-png-data")
	require.NoError(t, repo.UpdateLogo(ctx, logoData, "image/png"))

	logo, err := repo.GetLogo(ctx)
	require.NoError(t, err)
	assert.Equal(t, logoData, logo.Data)
	assert.Equal(t, "image/png", logo.ContentType)
}

func TestBrandingRepo_GetLogo_NotFound(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewBrandingRepository(pool)
	ctx := context.Background()

	_, err := repo.GetLogo(ctx)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
