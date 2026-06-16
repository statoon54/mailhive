package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
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
	// Test with an invalid PG pool that will fail Ping
	pool, err := pgxpool.New(context.Background(), "postgres://invalid:5432/nonexistent")
	if err != nil {
		t.Skip("cannot create pgx pool for test")
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:0"})
	defer func() { _ = redisClient.Close() }()

	h := NewHealthHandler(pool, redisClient)

	c, rec := newTestContext(http.MethodGet, "/health", nil)
	err = h.Health(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "dégradé")
}
