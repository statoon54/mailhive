package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestBrandingHandler_Get(t *testing.T) {
	svc := &mocks.MockBrandingService{
		Branding: &domain.AppBranding{AppTitle: "MailHive"},
	}
	h := NewBrandingHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/branding", nil)

	err := h.Get(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "MailHive")
}

func TestBrandingHandler_Update(t *testing.T) {
	svc := &mocks.MockBrandingService{
		Branding: &domain.AppBranding{AppTitle: "New Title"},
	}
	h := NewBrandingHandler(svc)

	body := map[string]any{"app_title": "New Title"}
	c, rec := newTestContext(http.MethodPut, "/api/v1/admin/branding", body)

	err := h.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBrandingHandler_GetLogo(t *testing.T) {
	svc := &mocks.MockBrandingService{
		Logo: &domain.LogoData{Data: []byte("PNG"), ContentType: "image/png"},
	}
	h := NewBrandingHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/branding/logo", nil)

	err := h.GetLogo(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
}

func TestBrandingHandler_GetLogo_NotFound(t *testing.T) {
	svc := &mocks.MockBrandingService{
		Err: domain.ErrNotFound,
	}
	h := NewBrandingHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/branding/logo", nil)

	err := h.GetLogo(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
