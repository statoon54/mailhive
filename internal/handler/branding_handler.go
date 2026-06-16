package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
)

// BrandingHandler gère les endpoints de branding.
type BrandingHandler struct {
	brandingService port.BrandingService
}

// NewBrandingHandler crée un nouveau handler de branding.
func NewBrandingHandler(service port.BrandingService) *BrandingHandler {
	return &BrandingHandler{brandingService: service}
}

// Get retourne le branding courant.
func (h *BrandingHandler) Get(c *echo.Context) error {
	branding, err := h.brandingService.Get(c.Request().Context())
	if err != nil {
		return handleError(c, err)
	}
	return ok(c, branding)
}

// Update met à jour le titre et le sous-titre.
func (h *BrandingHandler) Update(c *echo.Context) error {
	var req domain.UpdateBrandingRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, domain.ErrValidation)
	}

	branding, err := h.brandingService.Update(c.Request().Context(), req)
	if err != nil {
		return handleError(c, err)
	}
	return ok(c, branding)
}

// UploadLogo gère l'upload du logo via multipart.
func (h *BrandingHandler) UploadLogo(c *echo.Context) error {
	file, err := c.FormFile("logo")
	if err != nil {
		return handleError(c, domain.ErrValidation)
	}

	src, err := file.Open()
	if err != nil {
		return handleError(c, domain.ErrValidation)
	}
	defer func() { _ = src.Close() }()

	data, err := io.ReadAll(src)
	if err != nil {
		return handleError(c, domain.ErrValidation)
	}

	contentType := file.Header.Get("Content-Type")
	if err := h.brandingService.UploadLogo(c.Request().Context(), data, contentType); err != nil {
		return handleError(c, err)
	}

	return ok(c, map[string]string{"message": i18n.T(lang(c), "msg.logo_updated")})
}

// GetLogo sert l'image du logo avec le bon Content-Type.
func (h *BrandingHandler) GetLogo(c *echo.Context) error {
	logo, err := h.brandingService.GetLogo(c.Request().Context())
	if err != nil {
		return handleError(c, err)
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return c.Blob(http.StatusOK, logo.ContentType, logo.Data)
}
