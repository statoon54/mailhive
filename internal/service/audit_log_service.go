package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

const (
	// auditQueueSize borne le nombre d'entrées d'audit en attente d'écriture.
	auditQueueSize = 1024
	// auditWorkers borne le nombre de goroutines écrivant en base (et donc les
	// connexions du pool consommées par l'audit).
	auditWorkers = 4
	// auditWriteTimeout borne la durée d'une écriture d'audit en base.
	auditWriteTimeout = 5 * time.Second
)

// AuditLogService implémente port.AuditLogService.
//
// Les écritures d'audit sont asynchrones : Log() dépose l'entrée dans une file
// bornée consommée par un petit pool de workers. Cela évite la création non
// bornée de goroutines (et la contention du pool DB) sous forte charge, tout en
// permettant un arrêt propre qui draine les écritures en attente.
type AuditLogService struct {
	repo    port.AuditLogRepository
	queue   chan *domain.AuditLog
	wg      sync.WaitGroup
	closed  atomic.Bool
	dropped atomic.Int64
}

// NewAuditLogService crée le service et démarre le pool de workers d'écriture.
// Close() doit être appelé à l'arrêt pour drainer les entrées en attente.
func NewAuditLogService(repo port.AuditLogRepository) *AuditLogService {
	s := &AuditLogService{
		repo:  repo,
		queue: make(chan *domain.AuditLog, auditQueueSize),
	}
	s.wg.Add(auditWorkers)
	for range auditWorkers {
		go s.worker()
	}
	return s
}

// worker consomme la file et écrit les entrées en base jusqu'à fermeture de la file.
func (s *AuditLogService) worker() {
	defer s.wg.Done()
	for entry := range s.queue {
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		if err := s.repo.Create(ctx, entry); err != nil {
			slog.Error("échec d'écriture du log d'audit", "err", err)
		}
		cancel()
	}
}

// Log met en file une entrée d'audit (non bloquant). Si la file est pleine,
// l'entrée est ignorée et un avertissement est émis (préférable à une création
// non bornée de goroutines ou au blocage de la requête HTTP).
func (s *AuditLogService) Log(tenantID uuid.UUID, action, resourceType, resourceID, status string, statusCode int, errorMessage, details string, method, path string) {
	if s.closed.Load() {
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		slog.Error("échec de génération d'UUID v7 pour l'audit", "err", err)
		return
	}
	entry := &domain.AuditLog{
		ID:           id,
		TenantID:     tenantID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       status,
		StatusCode:   statusCode,
		ErrorMessage: errorMessage,
		Details:      details,
		Method:       method,
		Path:         path,
		CreatedAt:    time.Now(),
	}

	select {
	case s.queue <- entry:
	default:
		n := s.dropped.Add(1)
		slog.Warn("file d'audit pleine, entrée ignorée", "dropped_total", n, "tenant_id", tenantID)
	}
}

// Dropped retourne le nombre cumulé d'entrées d'audit ignorées (file pleine).
func (s *AuditLogService) Dropped() int64 {
	return s.dropped.Load()
}

// Close ferme la file et attend que les workers aient écrit les entrées en
// attente. Idempotent. À appeler après l'arrêt du serveur HTTP (plus aucun Log).
func (s *AuditLogService) Close() {
	if s.closed.Swap(true) {
		return
	}
	close(s.queue)
	s.wg.Wait()
}

// List retourne les logs d'audit paginés avec filtres.
func (s *AuditLogService) List(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	return s.repo.List(ctx, filter)
}
