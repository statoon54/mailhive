package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/service"
)

// LLMHandler gère les endpoints de génération de contenu par IA.
type LLMHandler struct {
	llmService *service.LLMService
}

// NewLLMHandler crée un nouveau handler LLM.
func NewLLMHandler(llmService *service.LLMService) *LLMHandler {
	return &LLMHandler{llmService: llmService}
}

// Generate génère du contenu HTML pour un email à partir d'un prompt.
func (h *LLMHandler) Generate(c *echo.Context) error {
	var req service.GenerateRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	l := lang(c)
	if req.Prompt == "" {
		return validationFailed(c, []FieldValidationError{
			{Field: "prompt", Message: i18n.T(l, "err.prompt_required")},
		})
	}

	result, err := h.llmService.Generate(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadGateway, ErrorResponse{
			Error: fmt.Sprintf(i18n.T(l, "err.generation"), err.Error()),
		})
	}

	return ok(c, result)
}

// Status retourne le statut du service LLM.
func (h *LLMHandler) Status(c *echo.Context) error {
	return ok(c, map[string]bool{
		"enabled": h.llmService.Enabled(),
	})
}
