package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

// HealthHandler gère les endpoints de vérification de santé.
type HealthHandler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

// NewHealthHandler crée un nouveau handler de santé.
func NewHealthHandler(pool *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, redis: redis}
}

// Health vérifie la santé de l'application (DB + Redis).
func (h *HealthHandler) Health(c *echo.Context) error {
	ctx := c.Request().Context()

	status := map[string]string{
		"status":   "ok",
		"database": "ok",
		"redis":    "ok",
	}

	// Vérifier PostgreSQL
	if err := h.pool.Ping(ctx); err != nil {
		status["status"] = "dégradé"
		status["database"] = "erreur"
	}

	// Vérifier Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		status["status"] = "dégradé"
		status["redis"] = "erreur"
	}

	code := http.StatusOK
	if status["status"] != "ok" {
		code = http.StatusServiceUnavailable
	}

	return c.JSON(code, status)
}
