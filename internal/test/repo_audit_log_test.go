package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func TestAuditLogRepo_CreateAndList(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewAuditLogRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	logID, _ := uuid.NewV7()
	entry := &domain.AuditLog{
		ID:           logID,
		TenantID:     tenant.ID,
		Action:       "create",
		ResourceType: "mail",
		ResourceID:   uuid.New().String(),
		Status:       "success",
		StatusCode:   201,
		Method:       "POST",
		Path:         "/api/v1/mails",
		CreatedAt:    time.Now(),
	}

	require.NoError(t, repo.Create(ctx, entry))

	list, err := repo.List(ctx, domain.AuditLogFilter{
		TenantID: &tenant.ID,
		Page:     1,
		Limit:    20,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Items), 1)
	assert.Equal(t, "create", list.Items[0].Action)
	assert.Equal(t, tenant.Name, list.Items[0].TenantName)
}

func TestAuditLogRepo_List_FilterByStatus(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewAuditLogRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	for _, status := range []string{"success", "error"} {
		logID, _ := uuid.NewV7()
		entry := &domain.AuditLog{
			ID: logID, TenantID: tenant.ID, Action: "create",
			ResourceType: "mail", Status: status, StatusCode: 200,
			Method: "POST", Path: "/api/v1/mails", CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, entry))
	}

	status := "error"
	list, err := repo.List(ctx, domain.AuditLogFilter{
		Status: &status,
		Page:   1,
		Limit:  20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
}

func TestAuditLogRepo_List_FilterByResourceType(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewAuditLogRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	for _, rt := range []string{"mail", "template"} {
		logID, _ := uuid.NewV7()
		entry := &domain.AuditLog{
			ID: logID, TenantID: tenant.ID, Action: "create",
			ResourceType: rt, Status: "success", StatusCode: 201,
			Method: "POST", Path: "/api/v1/" + rt + "s", CreatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, entry))
	}

	rt := "template"
	list, err := repo.List(ctx, domain.AuditLogFilter{
		ResourceType: &rt,
		Page:         1,
		Limit:        20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
}
