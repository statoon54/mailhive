package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestTenantHandler_Create_Success(t *testing.T) {
	svc := &mocks.MockTenantService{
		Tenant: &domain.Tenant{ID: uuid.New(), Name: "Test", Slug: "test"},
	}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodPost, "/api/v1/admin/tenants", map[string]string{"name": "Test"})
	setTenantCtx(c, uuid.New().String(), "admin")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestTenantHandler_Create_ValidationError(t *testing.T) {
	svc := &mocks.MockTenantService{}
	h := NewTenantHandler(svc)

	// Missing required 'name' field
	c, rec := newTestContext(http.MethodPost, "/api/v1/admin/tenants", map[string]string{})
	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTenantHandler_List(t *testing.T) {
	svc := &mocks.MockTenantService{
		Tenants: &domain.PaginatedList[domain.Tenant]{
			Items: []domain.Tenant{{ID: uuid.New(), Name: "T1"}},
			Total: 1, Page: 1, Limit: 20, TotalPages: 1,
		},
	}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/tenants", nil)
	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantHandler_GetByID(t *testing.T) {
	id := uuid.New()
	svc := &mocks.MockTenantService{
		Tenant: &domain.Tenant{ID: id, Name: "Test"},
	}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/tenants/"+id.String(), nil)
	setPathParams(c, map[string]string{"id": id.String()})

	err := h.GetByID(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantHandler_GetByID_InvalidUUID(t *testing.T) {
	svc := &mocks.MockTenantService{}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/tenants/invalid", nil)
	setPathParams(c, map[string]string{"id": "invalid"})

	err := h.GetByID(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTenantHandler_Update(t *testing.T) {
	id := uuid.New()
	svc := &mocks.MockTenantService{
		Tenant: &domain.Tenant{ID: id, Name: "Updated"},
	}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodPut, "/api/v1/admin/tenants/"+id.String(), map[string]string{"name": "Updated"})
	setPathParams(c, map[string]string{"id": id.String()})

	err := h.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTenantHandler_Delete(t *testing.T) {
	id := uuid.New()
	svc := &mocks.MockTenantService{}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/admin/tenants/"+id.String(), nil)
	setPathParams(c, map[string]string{"id": id.String()})

	err := h.Delete(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestTenantHandler_Me(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockTenantService{
		Tenant: &domain.Tenant{ID: tenantID, Name: "My Tenant", CreatedAt: time.Now()},
	}
	h := NewTenantHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/tenant/me", nil)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Me(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
