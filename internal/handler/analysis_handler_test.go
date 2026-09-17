package handler

import (
	json "encoding/json/v2"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/templates"
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

func TestAnalysisHandler_HTMLCheck_TemplateCasse(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockAnalysisService{
		Err: &templates.Error{Field: templates.FieldHTML, Line: 3, Detail: "bad character U+007D '}'"},
	}
	h := NewAnalysisHandler(svc, nil)

	body := map[string]any{"data": map[string]string{}}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/html-check", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	require.NoError(t, h.HTMLCheck(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"un template cassé ne doit pas produire une erreur serveur")

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Fields, 1, "la réponse doit désigner le champ fautif")
	assert.Equal(t, templates.FieldHTML, resp.Fields[0].Field)
	assert.Equal(t, "ligne 3 : bad character U+007D '}'", resp.Fields[0].Message,
		"le message doit situer l'erreur sans répéter le nom du champ")
}
