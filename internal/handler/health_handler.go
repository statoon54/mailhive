package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
)

// healthCheckTimeout borne la vérification des dépendances.
//
// Sans borne, /health hérite du contexte de la requête, sans échéance : une
// dépendance injoignable — et non pas simplement en erreur — y bloque aussi
// longtemps que dure sa propre logique de reconnexion. Un répartiteur de charge
// qui sonde cet endpoint attendrait d'autant. Mieux vaut répondre « dégradé »
// vite que juste tard.
const healthCheckTimeout = 2 * time.Second

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
	ctx, cancel := context.WithTimeout(c.Request().Context(), healthCheckTimeout)
	defer cancel()

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
