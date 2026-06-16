package handler

import (
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// TenantHandler gère les endpoints admin des tenants.
type TenantHandler struct {
	tenantService port.TenantService
}

// NewTenantHandler crée un nouveau handler tenant.
func NewTenantHandler(tenantService port.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// Me retourne les informations du tenant connecté.
func (h *TenantHandler) Me(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	tenant, err := h.tenantService.GetByID(c.Request().Context(), tenantID)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tenant)
}

// Create crée un nouveau tenant.
func (h *TenantHandler) Create(c *echo.Context) error {
	var req domain.CreateTenantRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	if errs := validateRequest(&req); len(errs) > 0 {
		return validationFailed(c, errs)
	}

	tenant, err := h.tenantService.Create(c.Request().Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return created(c, tenant)
}

// List liste tous les tenants avec pagination.
func (h *TenantHandler) List(c *echo.Context) error {
	page, limit := paginationParams(c)

	result, err := h.tenantService.List(c.Request().Context(), page, limit)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}

// GetByID retourne le détail d'un tenant.
func (h *TenantHandler) GetByID(c *echo.Context) error {
	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	tenant, err := h.tenantService.GetByID(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tenant)
}

// Update modifie un tenant.
func (h *TenantHandler) Update(c *echo.Context) error {
	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.UpdateTenantRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	tenant, err := h.tenantService.Update(c.Request().Context(), id, req)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tenant)
}

// RegenerateAPIKey génère une nouvelle clé API pour un tenant.
func (h *TenantHandler) RegenerateAPIKey(c *echo.Context) error {
	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	tenant, err := h.tenantService.RegenerateAPIKey(c.Request().Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tenant)
}

// Delete désactive un tenant.
func (h *TenantHandler) Delete(c *echo.Context) error {
	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.tenantService.Delete(c.Request().Context(), id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}
