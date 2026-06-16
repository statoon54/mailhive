package worker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

// TestGCHandler_CollectsOrphans vérifie que le handler GC supprime une pièce
// jointe sans lien mail_attachments et laisse une pièce jointe référencée.
func TestGCHandler_CollectsOrphans(t *testing.T) {
	repo := mocks.NewMockAttachmentRepo()
	blobs := mocks.NewMockBlobStore()
	svc := service.NewAttachmentService(repo, blobs, domain.AttachmentStoragePostgres)
	ctx := context.Background()
	tenantID := uuid.New()

	orphan, err := svc.Store(ctx, tenantID, []byte("orphelin"), "text/plain")
	require.NoError(t, err)
	referenced, err := svc.Store(ctx, tenantID, []byte("référencé"), "text/plain")
	require.NoError(t, err)
	repo.SetLinks(referenced, 1) // 1 mail référence cette pièce jointe

	h := NewGCHandler(svc)
	require.NoError(t, h.HandleAttachmentGC(ctx, asynq.NewTask(TypeAttachmentGC, nil)))

	_, err = svc.Load(ctx, tenantID, orphan)
	assert.ErrorIs(t, err, domain.ErrAttachmentNotFound, "l'orphelin doit être collecté")

	_, err = svc.Load(ctx, tenantID, referenced)
	assert.NoError(t, err, "la pièce jointe référencée doit subsister")
}
