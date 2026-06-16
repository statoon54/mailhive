package handler

import (
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// AuditLogHandler gère les endpoints du journal d'audit.
type AuditLogHandler struct {
	auditLogService port.AuditLogService
}

// NewAuditLogHandler crée un nouveau handler audit log.
func NewAuditLogHandler(auditLogService port.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{auditLogService: auditLogService}
}

// List liste les logs d'audit avec pagination et filtrage (admin : tous les tenants).
func (h *AuditLogHandler) List(c *echo.Context) error {
	page, limit := paginationParams(c)
	filter := domain.AuditLogFilter{
		Page:  page,
		Limit: limit,
	}

	if s := c.QueryParam("status"); s != "" {
		filter.Status = &s
	}
	if rt := c.QueryParam("resource_type"); rt != "" {
		filter.ResourceType = &rt
	}

	result, err := h.auditLogService.List(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}

// ListByTenant liste les logs d'audit du tenant connecté.
func (h *AuditLogHandler) ListByTenant(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	page, limit := paginationParams(c)
	filter := domain.AuditLogFilter{
		TenantID: &tenantID,
		Page:     page,
		Limit:    limit,
	}

	if s := c.QueryParam("status"); s != "" {
		filter.Status = &s
	}
	if rt := c.QueryParam("resource_type"); rt != "" {
		filter.ResourceType = &rt
	}

	result, err := h.auditLogService.List(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}
