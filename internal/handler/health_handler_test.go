package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Healthy(t *testing.T) {
	// Use real Redis/PG connections if available, otherwise skip
	// For unit testing, we test the handler structure and response format

	// Create a minimal test with mock-like setup
	// Since HealthHandler requires real *pgxpool.Pool and *redis.Client,
	// we test with nil-safe approach via integration tests.
	// Here we verify the handler can be constructed.
	h := NewHealthHandler(nil, nil)
	assert.NotNil(t, h)
}

func TestHealthHandler_DegradedDB(t *testing.T) {
	// Adresses volontairement injoignables mais qui échouent immédiatement :
	// 127.0.0.1:1 refuse la connexion sans passer par le DNS. L'ancienne
	// adresse Redis « localhost:0 » désignait un port non assignable, que
	// go-redis réessayait avec un backoff — à lui seul, ce test occupait deux
	// minutes de la suite unitaire.
	const unreachable = "127.0.0.1:1"

	pool, err := pgxpool.New(context.Background(), "postgres://"+unreachable+"/nonexistent")
	require.NoError(t, err)
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:        unreachable,
		MaxRetries:  -1, // aucune reprise : l'échec est le comportement testé
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = redisClient.Close() }()

	h := NewHealthHandler(pool, redisClient)

	c, rec := newTestContext(http.MethodGet, "/health", nil)
	err = h.Health(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "dégradé")
}

// TestHealthHandler_BoundedByTimeout vérifie que /health répond même quand une
// dépendance ne répond pas du tout, plutôt que de rester bloqué sur elle.
func TestHealthHandler_BoundedByTimeout(t *testing.T) {
	// 203.0.113.0/24 est réservé à la documentation (RFC 5737) : les paquets
	// n'aboutissent nulle part, la connexion n'est ni acceptée ni refusée.
	const blackhole = "203.0.113.1:5432"

	pool, err := pgxpool.New(context.Background(), "postgres://"+blackhole+"/nonexistent")
	require.NoError(t, err)
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:        blackhole,
		MaxRetries:  -1,
		DialTimeout: 100 * time.Millisecond,
	})
	defer func() { _ = redisClient.Close() }()

	h := NewHealthHandler(pool, redisClient)

	c, rec := newTestContext(http.MethodGet, "/health", nil)
	start := time.Now()
	err = h.Health(c)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Less(t, elapsed, 2*healthCheckTimeout,
		"la vérification doit être bornée par healthCheckTimeout, pas par la pile réseau")
}
