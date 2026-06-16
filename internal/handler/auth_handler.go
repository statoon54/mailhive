package handler

import (
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
)

// AuthHandler gère les endpoints d'authentification.
type AuthHandler struct {
	authService port.AuthService
}

// NewAuthHandler crée un nouveau handler d'authentification.
func NewAuthHandler(authService port.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// tokenRequest représente une demande de génération de token.
type tokenRequest struct {
	APIKey string `json:"api_key"`
}

// tokenResponse représente la réponse contenant un token JWT.
type tokenResponse struct {
	Token string `json:"token"`
}

// refreshRequest représente une demande de renouvellement de token.
type refreshRequest struct {
	Token string `json:"token"`
}

// GenerateToken génère un JWT à partir d'une clé API.
func (h *AuthHandler) GenerateToken(c *echo.Context) error {
	var req tokenRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, err)
	}

	if req.APIKey == "" {
		return c.JSON(400, ErrorResponse{Error: i18n.T(lang(c), "err.api_key_required")})
	}

	token, err := h.authService.GenerateToken(c.Request().Context(), req.APIKey)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tokenResponse{Token: token})
}

// RefreshToken renouvelle un JWT existant.
func (h *AuthHandler) RefreshToken(c *echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return handleError(c, err)
	}

	if req.Token == "" {
		return c.JSON(400, ErrorResponse{Error: i18n.T(lang(c), "err.token_required")})
	}

	token, err := h.authService.RefreshToken(c.Request().Context(), req.Token)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, tokenResponse{Token: token})
}
