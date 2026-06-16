package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestAnalysisHandler_SpamCheck(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockAnalysisService{
		SpamResult: &domain.SpamCheckResult{Score: 1.5, MaxScore: 10},
	}
	h := NewAnalysisHandler(svc, nil)

	body := map[string]any{"data": map[string]string{"name": "Alice"}}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/spam-check", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.SpamCheck(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAnalysisHandler_HTMLCheck(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockAnalysisService{
		HTMLResult: &domain.HTMLCheckResult{TotalCount: 0},
	}
	h := NewAnalysisHandler(svc, nil)

	body := map[string]any{"data": map[string]string{"name": "Alice"}}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/html-check", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.HTMLCheck(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAnalysisHandler_SpamCheck_NotFound(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockAnalysisService{
		Err: domain.ErrNotFound,
	}
	h := NewAnalysisHandler(svc, nil)

	body := map[string]any{"data": map[string]string{}}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/spam-check", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.SpamCheck(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAnalysisHandler_HTMLCheck_ValidationError(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockAnalysisService{
		Err: domain.ErrValidation,
	}
	h := NewAnalysisHandler(svc, nil)

	body := map[string]any{"data": map[string]string{}}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/html-check", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.HTMLCheck(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
