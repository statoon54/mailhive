package service

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestAuditLogService_Log_CreatesEntry(t *testing.T) {
	repo := mocks.NewMockAuditLogRepo()
	svc := NewAuditLogService(repo)
	tenantID := uuid.New()

	svc.Log(tenantID, "create", "mail", "123", "success", 201, "", "", "POST", "/api/v1/mails")

	// Close draine la file et arrête les workers : lecture déterministe et sans data race.
	svc.Close()

	assert.True(t, repo.Called("Create"))
	assert.GreaterOrEqual(t, len(repo.Logs), 1)
}

func TestAuditLogService_List_Pagination(t *testing.T) {
	repo := mocks.NewMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	list, err := svc.List(context.Background(), domain.AuditLogFilter{Page: 0, Limit: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, list.Page)
	assert.Equal(t, 20, list.Limit)
}

func TestAuditLogService_List_WithFilter(t *testing.T) {
	repo := mocks.NewMockAuditLogRepo()
	svc := NewAuditLogService(repo)

	status := "error"
	list, err := svc.List(context.Background(), domain.AuditLogFilter{
		Status: &status,
		Page:   1,
		Limit:  50,
	})
	require.NoError(t, err)
	assert.NotNil(t, list)
}

// countingAuditRepo compte les écritures via un compteur atomique.
type countingAuditRepo struct {
	created atomic.Int64
}

func (r *countingAuditRepo) Create(_ context.Context, _ *domain.AuditLog) error {
	r.created.Add(1)
	return nil
}

func (r *countingAuditRepo) List(_ context.Context, _ domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	return nil, nil
}

// blockingAuditRepo bloque dans Create jusqu'à libération, pour saturer la file.
type blockingAuditRepo struct {
	release chan struct{}
}

func (r *blockingAuditRepo) Create(_ context.Context, _ *domain.AuditLog) error {
	<-r.release
	return nil
}

func (r *blockingAuditRepo) List(_ context.Context, _ domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	return nil, nil
}

func TestAuditLogService_Close_DrainsPending(t *testing.T) {
	repo := &countingAuditRepo{}
	svc := NewAuditLogService(repo)

	const n = 50
	for range n {
		svc.Log(uuid.New(), "create", "mail", "res", "success", 201, "", "", "POST", "/x")
	}

	// Close doit drainer toutes les entrées en attente avant de rendre la main.
	svc.Close()
	assert.Equal(t, int64(n), repo.created.Load())
}

func TestAuditLogService_DropsWhenQueueFull(t *testing.T) {
	repo := &blockingAuditRepo{release: make(chan struct{})}
	svc := NewAuditLogService(repo)

	// Les workers bloquent dans Create : la capacité totale est file + workers.
	// Au-delà, Log doit ignorer les entrées sans paniquer ni bloquer.
	total := auditQueueSize + auditWorkers + 100
	for range total {
		svc.Log(uuid.New(), "create", "mail", "res", "success", 201, "", "", "POST", "/x")
	}

	assert.Positive(t, svc.Dropped(), "des entrées doivent être ignorées quand la file est pleine")

	close(repo.release) // débloquer les workers
	svc.Close()
}
