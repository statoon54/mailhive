package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTLCache_MissThenHit(t *testing.T) {
	c := newTTLCache[string, int](time.Minute)

	_, ok := c.get("k")
	assert.False(t, ok, "cache vide doit rater")

	c.set("k", 42)
	v, ok := c.get("k")
	assert.True(t, ok, "doit toucher après set dans le TTL")
	assert.Equal(t, 42, v)
}

func TestTTLCache_ExpiresAfterTTL(t *testing.T) {
	c := newTTLCache[string, int](5 * time.Minute)
	now := time.Now()
	c.now = func() time.Time { return now }

	c.set("k", 7)

	// Juste avant l'expiration : toujours présent.
	now = now.Add(5*time.Minute - time.Second)
	_, ok := c.get("k")
	assert.True(t, ok, "doit toucher juste avant l'expiration")

	// Après le TTL : expiré, miss (force le rechargement par l'appelant).
	now = now.Add(2 * time.Second)
	_, ok = c.get("k")
	assert.False(t, ok, "doit rater après expiration du TTL")
}
