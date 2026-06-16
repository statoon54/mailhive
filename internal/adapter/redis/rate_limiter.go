package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// tokenBucketScript est un script Lua atomique implémentant un token bucket dans Redis.
// Clé : rate_limit:{tenantID}
// ARGV[1] = rate (tokens/s), ARGV[2] = burst (capacité max), ARGV[3] = now (Unix microsecondes)
//
// Le bucket stocke deux champs dans un hash :
//   - tokens : nombre de tokens disponibles
//   - last   : dernier timestamp de rafraîchissement (microsecondes)
//
// Retourne 1 si le token est accordé, 0 sinon.
var tokenBucketScript = redis.NewScript(`
local key       = KEYS[1]
local rate      = tonumber(ARGV[1])
local burst     = tonumber(ARGV[2])
local now       = tonumber(ARGV[3])

local data = redis.call("HMGET", key, "tokens", "last")
local tokens = tonumber(data[1])
local last   = tonumber(data[2])

if tokens == nil then
    tokens = burst
    last   = now
end

-- Ajouter les tokens accumulés depuis le dernier appel
local elapsed = (now - last) / 1e6 -- microsecondes → secondes
local newTokens = tokens + elapsed * rate
if newTokens > burst then
    newTokens = burst
end

local allowed = 0
if newTokens >= 1 then
    newTokens = newTokens - 1
    allowed   = 1
end

redis.call("HSET", key, "tokens", newTokens, "last", now)
-- TTL de sécurité : 2× le temps pour remplir le bucket (min 60s)
local ttl = math.max(math.ceil(burst / rate) * 2, 60)
redis.call("EXPIRE", key, ttl)

return allowed
`)

// RateLimiter implémente un rate limiter distribué via Redis (token bucket).
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter crée un nouveau rate limiter Redis.
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow vérifie si une requête est autorisée pour le tenant donné.
func (rl *RateLimiter) Allow(
	ctx context.Context,
	tenantID uuid.UUID,
	rateLimit float64,
	burst int,
) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", tenantID)
	now := time.Now().UnixMicro()

	result, err := tokenBucketScript.Run(ctx, rl.client, []string{key}, rateLimit, burst, now).Int()
	if err != nil {
		return false, fmt.Errorf("erreur du rate limiter Redis : %w", err)
	}

	return result == 1, nil
}
