package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
)

// AnalysisHandler gère les endpoints d'analyse de templates.
type AnalysisHandler struct {
	analysisService port.AnalysisService
	redisClient     *redis.Client
}

// NewAnalysisHandler crée un nouveau handler d'analyse.
func NewAnalysisHandler(
	analysisService port.AnalysisService,
	redisClient *redis.Client,
) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: analysisService,
		redisClient:     redisClient,
	}
}

// SpamCheck analyse un template pour les indicateurs de spam.
func (h *AnalysisHandler) SpamCheck(c *echo.Context) error {
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

	result, err := h.analysisService.SpamCheck(c.Request().Context(), tenantID, id, req.Data)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}

// HTMLCheck analyse un template pour la compatibilité clients email.
func (h *AnalysisHandler) HTMLCheck(c *echo.Context) error {
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

	result, err := h.analysisService.HTMLCheck(c.Request().Context(), tenantID, id, req.Data)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}

// LinkCheck analyse un template pour vérifier les liens (rate limité à 1/min par tenant).
func (h *AnalysisHandler) LinkCheck(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	// Rate limit : 1 check par minute par tenant via Redis SET NX
	ctx := c.Request().Context()
	key := fmt.Sprintf("link_check:%s", tenantID)
	result, err := h.redisClient.SetArgs(ctx, key, 1, redis.SetArgs{
		TTL:  60 * time.Second,
		Mode: "nx",
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return handleError(c, err)
	}
	if result != "OK" {
		l := lang(c)
		return c.JSON(http.StatusTooManyRequests, ErrorResponse{
			Error: i18n.T(l, "err.link_check_rate_limited"),
		})
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.PreviewTemplateRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	checkResult, err := h.analysisService.LinkCheck(ctx, tenantID, id, req.Data)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, checkResult)
}
