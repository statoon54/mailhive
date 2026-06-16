package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestAuditLogHandler_List(t *testing.T) {
	svc := &mocks.MockAuditLogService{
		List_: &domain.PaginatedList[domain.AuditLog]{
			Items: []domain.AuditLog{}, Total: 0, Page: 1, Limit: 20,
		},
	}
	h := NewAuditLogHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/audit-logs", nil)

	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuditLogHandler_ListByTenant(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockAuditLogService{
		List_: &domain.PaginatedList[domain.AuditLog]{
			Items: []domain.AuditLog{}, Total: 0, Page: 1, Limit: 20,
		},
	}
	h := NewAuditLogHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/tenant/audit-logs", nil)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.ListByTenant(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuditLogHandler_List_WithFilters(t *testing.T) {
	svc := &mocks.MockAuditLogService{
		List_: &domain.PaginatedList[domain.AuditLog]{
			Items: []domain.AuditLog{}, Total: 0, Page: 1, Limit: 20,
		},
	}
	h := NewAuditLogHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/admin/audit-logs?status=success&resource_type=mail", nil)

	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
