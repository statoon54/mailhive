package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestTemplateHandler_Create(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockTemplateService{
		Template: &domain.Template{ID: uuid.New(), Name: "Welcome"},
	}
	h := NewTemplateHandler(svc)

	body := map[string]any{
		"name":         "Welcome",
		"subject_tmpl": "Hello {{.name}}",
		"text_body":    "Welcome {{.name}}!",
		"variables":    map[string]string{"name": "string"},
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestTemplateHandler_Create_NoBody(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockTemplateService{}
	h := NewTemplateHandler(svc)

	body := map[string]any{
		"name":         "Test",
		"subject_tmpl": "Hello",
		// Missing both text_body and html_body
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates", body)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTemplateHandler_List(t *testing.T) {
	tenantID := uuid.New()
	svc := &mocks.MockTemplateService{
		Templates: []domain.Template{{ID: uuid.New(), Name: "T1"}},
	}
	h := NewTemplateHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/templates", nil)
	setTenantCtx(c, tenantID.String(), "tenant")

	err := h.List(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTemplateHandler_GetByID(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockTemplateService{
		Template: &domain.Template{ID: tmplID, Name: "Test"},
	}
	h := NewTemplateHandler(svc)

	c, rec := newTestContext(http.MethodGet, "/api/v1/templates/"+tmplID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.GetByID(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTemplateHandler_Update(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockTemplateService{
		Template: &domain.Template{ID: tmplID, Name: "Updated"},
	}
	h := NewTemplateHandler(svc)

	body := map[string]any{"name": "Updated"}
	c, rec := newTestContext(http.MethodPut, "/api/v1/templates/"+tmplID.String(), body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTemplateHandler_Delete(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockTemplateService{}
	h := NewTemplateHandler(svc)

	c, rec := newTestContext(http.MethodDelete, "/api/v1/templates/"+tmplID.String(), nil)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.Delete(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestTemplateHandler_Preview(t *testing.T) {
	tenantID := uuid.New()
	tmplID := uuid.New()
	svc := &mocks.MockTemplateService{
		Preview_: &domain.PreviewTemplateResponse{
			Subject:  "Hello Alice",
			TextBody: "Welcome Alice!",
		},
	}
	h := NewTemplateHandler(svc)

	body := map[string]any{
		"data": map[string]string{"name": "Alice"},
	}
	c, rec := newTestContext(http.MethodPost, "/api/v1/templates/"+tmplID.String()+"/preview", body)
	setTenantCtx(c, tenantID.String(), "tenant")
	setPathParams(c, map[string]string{"id": tmplID.String()})

	err := h.Preview(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
