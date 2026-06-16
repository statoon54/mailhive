package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
)

// --- Fakes en mémoire ---

type fakeAttachmentRepo struct {
	byID  map[uuid.UUID]*domain.AttachmentMeta
	byKey map[string]uuid.UUID // clé = tenant|sha256hex
	links map[uuid.UUID]int    // nombre de liens mail_attachments par attachment
}

func newFakeAttachmentRepo() *fakeAttachmentRepo {
	return &fakeAttachmentRepo{
		byID:  make(map[uuid.UUID]*domain.AttachmentMeta),
		byKey: make(map[string]uuid.UUID),
		links: make(map[uuid.UUID]int),
	}
}

func metaKey(tenantID uuid.UUID, sha []byte) string {
	return tenantID.String() + "|" + hex.EncodeToString(sha)
}

func (f *fakeAttachmentRepo) UpsertMeta(_ context.Context, m domain.AttachmentMeta) (uuid.UUID, bool, error) {
	k := metaKey(m.TenantID, m.SHA256)
	if id, ok := f.byKey[k]; ok {
		return id, false, nil
	}
	id := uuid.New()
	m.ID = id
	f.byID[id] = &m
	f.byKey[k] = id
	return id, true, nil
}

func (f *fakeAttachmentRepo) GetMeta(_ context.Context, tenantID, attachmentID uuid.UUID) (*domain.AttachmentMeta, error) {
	m, ok := f.byID[attachmentID]
	if !ok || m.TenantID != tenantID {
		return nil, domain.ErrAttachmentNotFound
	}
	cp := *m
	return &cp, nil
}

// setLinks fixe le nombre de liens mail_attachments d'une pièce jointe (0 = orpheline).
func (f *fakeAttachmentRepo) setLinks(attachmentID uuid.UUID, n int) {
	f.links[attachmentID] = n
}

func (f *fakeAttachmentRepo) ListOrphans(_ context.Context, _ time.Time, limit int) ([]domain.AttachmentMeta, error) {
	var out []domain.AttachmentMeta
	for id, m := range f.byID {
		if f.links[id] <= 0 {
			out = append(out, *m)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (f *fakeAttachmentRepo) DeleteOrphanMeta(_ context.Context, attachmentID uuid.UUID) (bool, error) {
	m, ok := f.byID[attachmentID]
	if !ok || f.links[attachmentID] > 0 {
		return false, nil
	}
	delete(f.byKey, metaKey(m.TenantID, m.SHA256))
	delete(f.byID, attachmentID)
	delete(f.links, attachmentID)
	return true, nil
}

type fakeBlobStore struct {
	data     map[string][]byte // clé = tenant|sha256hex
	putCalls int
}

func newFakeBlobStore() *fakeBlobStore {
	return &fakeBlobStore{data: make(map[string][]byte)}
}

func (f *fakeBlobStore) Put(_ context.Context, tenantID uuid.UUID, sha, content []byte) error {
	f.putCalls++
	f.data[metaKey(tenantID, sha)] = content
	return nil
}

func (f *fakeBlobStore) Get(_ context.Context, tenantID uuid.UUID, sha []byte) ([]byte, error) {
	c, ok := f.data[metaKey(tenantID, sha)]
	if !ok {
		return nil, domain.ErrAttachmentNotFound
	}
	return c, nil
}

func (f *fakeBlobStore) Delete(_ context.Context, tenantID uuid.UUID, sha []byte) error {
	delete(f.data, metaKey(tenantID, sha))
	return nil
}

func (f *fakeBlobStore) Exists(_ context.Context, tenantID uuid.UUID, sha []byte) (bool, error) {
	_, ok := f.data[metaKey(tenantID, sha)]
	return ok, nil
}

// --- Tests ---

func TestAttachmentService_Store_Deduplicates(t *testing.T) {
	repo := newFakeAttachmentRepo()
	blobs := newFakeBlobStore()
	svc := service.NewAttachmentService(repo, blobs, domain.AttachmentStoragePostgres)
	ctx := context.Background()
	tenantID := uuid.New()
	content := []byte("contenu de la pièce jointe")

	id1, err := svc.Store(ctx, tenantID, content, "text/plain")
	require.NoError(t, err)
	id2, err := svc.Store(ctx, tenantID, content, "text/plain")
	require.NoError(t, err)

	assert.Equal(t, id1, id2, "le même contenu doit donner le même id (dédup)")
	assert.Equal(t, 1, blobs.putCalls, "le blob ne doit être écrit qu'une seule fois")
	assert.Len(t, repo.byID, 1, "une seule métadonnée pour un contenu identique")
}

func TestAttachmentService_Store_DistinctContentDistinctID(t *testing.T) {
	repo := newFakeAttachmentRepo()
	svc := service.NewAttachmentService(repo, newFakeBlobStore(), domain.AttachmentStoragePostgres)
	ctx := context.Background()
	tenantID := uuid.New()

	id1, err := svc.Store(ctx, tenantID, []byte("a"), "text/plain")
	require.NoError(t, err)
	id2, err := svc.Store(ctx, tenantID, []byte("b"), "text/plain")
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2)
	assert.Len(t, repo.byID, 2)
}

func TestAttachmentService_Store_IsolatedPerTenant(t *testing.T) {
	repo := newFakeAttachmentRepo()
	blobs := newFakeBlobStore()
	svc := service.NewAttachmentService(repo, blobs, domain.AttachmentStoragePostgres)
	ctx := context.Background()
	content := []byte("même contenu")

	id1, err := svc.Store(ctx, uuid.New(), content, "text/plain")
	require.NoError(t, err)
	id2, err := svc.Store(ctx, uuid.New(), content, "text/plain")
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2, "deux tenants ne partagent pas la même pièce jointe")
	assert.Equal(t, 2, blobs.putCalls, "un blob par tenant (dédup scopée au tenant)")
}

func TestAttachmentService_Load_RoundTrip(t *testing.T) {
	repo := newFakeAttachmentRepo()
	svc := service.NewAttachmentService(repo, newFakeBlobStore(), domain.AttachmentStoragePostgres)
	ctx := context.Background()
	tenantID := uuid.New()
	content := []byte("payload")

	id, err := svc.Store(ctx, tenantID, content, "application/octet-stream")
	require.NoError(t, err)

	got, err := svc.Load(ctx, tenantID, id)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	// La métadonnée doit porter le bon hash et la bonne taille.
	sum := sha256.Sum256(content)
	assert.Equal(t, sum[:], repo.byID[id].SHA256)
	assert.Equal(t, int64(len(content)), repo.byID[id].Size)
}

func TestAttachmentService_CollectOrphans(t *testing.T) {
	repo := newFakeAttachmentRepo()
	blobs := newFakeBlobStore()
	svc := service.NewAttachmentService(repo, blobs, domain.AttachmentStoragePostgres)
	ctx := context.Background()
	tenantID := uuid.New()

	orphan, err := svc.Store(ctx, tenantID, []byte("orphelin"), "text/plain")
	require.NoError(t, err)
	referenced, err := svc.Store(ctx, tenantID, []byte("référencé"), "text/plain")
	require.NoError(t, err)
	repo.setLinks(referenced, 1) // 1 lien mail_attachments

	n, err := svc.CollectOrphans(ctx, time.Now(), 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "seule la pièce jointe sans lien doit être collectée")

	_, err = svc.Load(ctx, tenantID, orphan)
	assert.ErrorIs(t, err, domain.ErrAttachmentNotFound, "l'orphelin a été supprimé")

	_, err = svc.Load(ctx, tenantID, referenced)
	assert.NoError(t, err, "la pièce jointe référencée subsiste")
}
