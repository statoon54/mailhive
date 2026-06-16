package worker

import (
	"sync"
	"time"
)

// TTL par défaut des caches du worker.
//
// L'API et le worker étant des processus distincts, l'invalidation se fait
// uniquement par expiration (pas de signal in-process possible). Le TTL borne
// donc le délai de prise en compte d'une modification côté API.
const (
	// defaultTemplateCacheTTL : les templates sont du contenu, ils changent rarement.
	defaultTemplateCacheTTL = 5 * time.Minute
	// defaultConfigCacheTTL : config tenant / SMTP, opérationnelle — propagation rapide.
	defaultConfigCacheTTL = 30 * time.Second
)

// cacheEntry est une entrée horodatée d'un ttlCache.
type cacheEntry[V any] struct {
	value    V
	storedAt time.Time
}

// ttlCache est un cache générique en mémoire avec expiration par TTL,
// sûr pour un usage concurrent.
type ttlCache[K comparable, V any] struct {
	ttl     time.Duration
	now     func() time.Time
	mu      sync.Mutex
	entries map[K]cacheEntry[V]
}

// newTTLCache crée un cache avec le TTL donné.
func newTTLCache[K comparable, V any](ttl time.Duration) *ttlCache[K, V] {
	return &ttlCache[K, V]{
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[K]cacheEntry[V]),
	}
}

// get retourne la valeur si elle est présente et non expirée.
// Une entrée expirée est supprimée et traitée comme un miss.
func (c *ttlCache[K, V]) get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	if c.now().Sub(entry.storedAt) >= c.ttl {
		delete(c.entries, key)
		var zero V
		return zero, false
	}
	return entry.value, true
}

// set stocke une valeur et l'horodate.
func (c *ttlCache[K, V]) set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry[V]{value: value, storedAt: c.now()}
}
