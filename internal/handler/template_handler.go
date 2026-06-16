package handler

import (
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
)

// TemplateHandler gère les endpoints de templates.
type TemplateHandler struct {
	templateService port.TemplateService
}

// NewTemplateHandler crée un nouveau handler template.
func NewTemplateHandler(templateService port.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// Create crée un nouveau template.
func (h *TemplateHandler) Create(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.CreateTemplateRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	if errs := validateRequest(&req); len(errs) > 0 {
		return validationFailed(c, errs)
	}

	if req.TextBody == "" && req.HTMLBody == "" {
		l := lang(c)
		return validationFailed(c, []FieldValidationError{
			{Field: "text_body", Message: i18n.T(l, "err.body_required")},
		})
	}

	tmpl, err := h.templateService.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return handleError(c, err)
	}

	return created(c, tmpl)
}

// List liste les templates du tenant.
func (h *TemplateHandler) List(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	templates, err := h.templateService.List(c.Request().Context(), tenantID)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, templates)
}

// GetByID retourne le détail d'un template.
func (h *TemplateHandler) GetByID(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	tmpl, err := h.templateService.GetByID(c.Request().Context(), tenantID, id)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tmpl)
}

// Update modifie un template.
func (h *TemplateHandler) Update(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.UpdateTemplateRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	tmpl, err := h.templateService.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tmpl)
}

// Delete supprime un template.
func (h *TemplateHandler) Delete(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.templateService.Delete(c.Request().Context(), tenantID, id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// Preview prévisualise un template avec des données.
func (h *TemplateHandler) Preview(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.PreviewTemplateRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	result, err := h.templateService.Preview(c.Request().Context(), tenantID, id, req.Data)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}
