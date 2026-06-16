package worker

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// État du circuit breaker.
type cbState int

const (
	cbClosed   cbState = iota // Fonctionnement normal
	cbOpen                    // SMTP considéré down, on bloque les envois
	cbHalfOpen                // On laisse passer 1 essai pour tester
)

// circuitBreaker implémente le pattern circuit breaker pour un endpoint SMTP.
type circuitBreaker struct {
	cooldownPeriod time.Duration
	lastFailure    time.Time // Date du dernier échec
	mu             sync.Mutex
	state          cbState
	failures       int // Échecs consécutifs
	threshold      int // Seuil d'échecs avant ouverture
}

// CircuitBreakerRegistry gère un circuit breaker par config SMTP.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[uuid.UUID]*circuitBreaker
	config   CircuitBreakerConfig
}

// CircuitBreakerConfig contient les paramètres du circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Nombre d'échecs consécutifs avant ouverture (défaut: 5)
	CooldownPeriod   time.Duration // Durée avant de passer en half-open (défaut: 30s)
}

// DefaultCircuitBreakerConfig retourne la configuration par défaut.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		CooldownPeriod:   30 * time.Second,
	}
}

// NewCircuitBreakerRegistry crée un nouveau registre de circuit breakers.
func NewCircuitBreakerRegistry(cfg CircuitBreakerConfig) *CircuitBreakerRegistry {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.CooldownPeriod <= 0 {
		cfg.CooldownPeriod = 30 * time.Second
	}
	return &CircuitBreakerRegistry{
		breakers: make(map[uuid.UUID]*circuitBreaker),
		config:   cfg,
	}
}

// get retourne le circuit breaker associé à une config SMTP, ou en crée un nouveau.
func (r *CircuitBreakerRegistry) get(smtpConfigID uuid.UUID) *circuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[smtpConfigID]
	r.mu.RUnlock()
	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if cb, ok = r.breakers[smtpConfigID]; ok {
		return cb
	}
	cb = &circuitBreaker{
		threshold:      r.config.FailureThreshold,
		cooldownPeriod: r.config.CooldownPeriod,
	}
	r.breakers[smtpConfigID] = cb
	return cb
}

// Allow vérifie si un envoi est autorisé pour cette config SMTP.
// Retourne true si autorisé, false si le circuit est ouvert.
func (r *CircuitBreakerRegistry) Allow(smtpConfigID uuid.UUID) bool {
	cb := r.get(smtpConfigID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		// Vérifier si le cooldown est écoulé
		if time.Since(cb.lastFailure) >= cb.cooldownPeriod {
			cb.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		// Un seul essai à la fois en half-open (celui en cours)
		return false
	}
	return true
}

// RecordSuccess signale un envoi réussi.
func (r *CircuitBreakerRegistry) RecordSuccess(smtpConfigID uuid.UUID) {
	cb := r.get(smtpConfigID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = cbClosed
}

// RecordFailure signale un échec d'envoi.
func (r *CircuitBreakerRegistry) RecordFailure(smtpConfigID uuid.UUID) {
	cb := r.get(smtpConfigID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = cbOpen
	}
}

// IsOpen retourne true si le circuit est ouvert pour cette config SMTP.
func (r *CircuitBreakerRegistry) IsOpen(smtpConfigID uuid.UUID) bool {
	cb := r.get(smtpConfigID)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state == cbOpen
}
