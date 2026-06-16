package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestSMTPConfigHandler_Create(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockSMTPConfigService{
		Config: &domain.SMTPConfig{ID: uuid.New(), Name: "Test SMTP"},
	}
	h := NewSMTPConfigHandler(svc)

	body := map[string]any{
		"name":        "Test SMTP",
		"host":        "smtp.example.com",
		"port":        587,
		"auth_method": "PLAIN",
		"tls_policy":  "mandatory",
		"from_email":  "noreply@example.com",
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/smtp-configs", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestSMTPConfigHandler_Create_ValidationError(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockSMTPConfigService{}
	h := NewSMTPConfigHandler(svc)

	body := map[string]any{"name": "Test"} // missing required fields
	c, rec := newTestContext(http.MethodPost, "/api/v1/smtp-configs", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSMTPConfigHandler_List(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockSMTPConfigService{
		Configs: []domain.SMTPConfig{{ID: uuid.New(), Name: "SMTP1"}},
	}
	h := NewSMTPConfigHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/smtp-configs", nil)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSMTPConfigHandler_GetByID(t *testing.T) {
	tenantID := uuid.New()
	cfgID := uuid.New()
	svc := &mocks.MockSMTPConfigService{
		Config: &domain.SMTPConfig{ID: cfgID, Name: "Test"},
	}
	h := NewSMTPConfigHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/smtp-configs/"+cfgID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": cfgID.String()})

	err := h.GetByID(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSMTPConfigHandler_Update(t *testing.T) {
	tenantID := uuid.New()
	cfgID := uuid.New()
	svc := &mocks.MockSMTPConfigService{
		Config: &domain.SMTPConfig{ID: cfgID, Name: "Updated"},
	}
	h := NewSMTPConfigHandler(svc)

	body := map[string]any{"name": "Updated"}
	c, rec := newTestContext(http.MethodPut, "/api/v1/smtp-configs/"+cfgID.String(), body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": cfgID.String()})

	err := h.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSMTPConfigHandler_Delete(t *testing.T) {
	tenantID := uuid.New()
	cfgID := uuid.New()
	svc := &mocks.MockSMTPConfigService{}
	h := NewSMTPConfigHandler(svc)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/smtp-configs/"+cfgID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": cfgID.String()})

	err := h.Delete(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestSMTPConfigHandler_Test(t *testing.T) {
	tenantID := uuid.New()
	cfgID := uuid.New()
	svc := &mocks.MockSMTPConfigService{}
	h := NewSMTPConfigHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/smtp-configs/"+cfgID.String()+"/test", nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": cfgID.String()})

	err := h.Test(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
