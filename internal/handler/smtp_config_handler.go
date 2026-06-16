package handler

import (
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
)

// SMTPConfigHandler gère les endpoints de configuration SMTP.
type SMTPConfigHandler struct {
	smtpService port.SMTPConfigService
}

// NewSMTPConfigHandler crée un nouveau handler config SMTP.
func NewSMTPConfigHandler(smtpService port.SMTPConfigService) *SMTPConfigHandler {
	return &SMTPConfigHandler{smtpService: smtpService}
}

// Create crée une nouvelle config SMTP.
func (h *SMTPConfigHandler) Create(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.CreateSMTPConfigRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	if errs := validateRequest(&req); len(errs) > 0 {
		return validationFailed(c, errs)
	}

	cfg, err := h.smtpService.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return handleError(c, err)
	}

	return created(c, cfg)
}

// List liste les configs SMTP du tenant.
func (h *SMTPConfigHandler) List(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	configs, err := h.smtpService.List(c.Request().Context(), tenantID)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, configs)
}

// GetByID retourne le détail d'une config SMTP.
func (h *SMTPConfigHandler) GetByID(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	cfg, err := h.smtpService.GetByID(c.Request().Context(), tenantID, id)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, cfg)
}

// Update modifie une config SMTP.
func (h *SMTPConfigHandler) Update(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.UpdateSMTPConfigRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	cfg, err := h.smtpService.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, cfg)
}

// Delete supprime une config SMTP.
func (h *SMTPConfigHandler) Delete(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.smtpService.Delete(c.Request().Context(), tenantID, id); err != nil {
		return handleError(c, err)
	}

	return c.NoContent(204)
}

// Test teste une config SMTP.
func (h *SMTPConfigHandler) Test(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.smtpService.Test(c.Request().Context(), tenantID, id); err != nil {
		return handleError(c, err)
	}

	return ok(c, map[string]string{"message": i18n.T(lang(c), "msg.smtp_test_ok")})
}
