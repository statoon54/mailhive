package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestBrandingService_Update_Title(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	newTitle := "My App"
	_, err := svc.Update(context.Background(), domain.UpdateBrandingRequest{AppTitle: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "My App", repo.Branding.AppTitle)
}

func TestBrandingService_Update_ValidTimezone(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	tz := "America/New_York"
	_, err := svc.Update(context.Background(), domain.UpdateBrandingRequest{Timezone: &tz})
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", repo.Branding.Timezone)
}

func TestBrandingService_Update_InvalidTimezone(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	tz := "Invalid/Timezone"
	_, err := svc.Update(context.Background(), domain.UpdateBrandingRequest{Timezone: &tz})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestBrandingService_UploadLogo_Valid(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	err := svc.UploadLogo(context.Background(), []byte("png-data"), "image/png")
	require.NoError(t, err)
	assert.NotNil(t, repo.Logo)
}

func TestBrandingService_UploadLogo_InvalidType(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	err := svc.UploadLogo(context.Background(), []byte("data"), "application/pdf")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestBrandingService_UploadLogo_TooLarge(t *testing.T) {
	repo := mocks.NewMockBrandingRepo()
	svc := NewBrandingService(repo)

	data := make([]byte, 600*1024) // 600 Ko > 512 Ko max
	err := svc.UploadLogo(context.Background(), data, "image/png")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}
