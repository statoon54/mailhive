package mocks

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// MockAttachmentRepo est un faux port.AttachmentRepository en mémoire.
// L'orphelinage est modélisé par links : une pièce jointe sans lien (compteur
// nul ou absent) est orpheline, comme en base où l'on teste NOT EXISTS sur
// mail_attachments.
type MockAttachmentRepo struct {
	mu    sync.RWMutex
	byID  map[uuid.UUID]*domain.AttachmentMeta
	byKey map[string]uuid.UUID // clé = tenant|sha256hex
	links map[uuid.UUID]int    // nombre de liens mail_attachments par attachment
}

// NewMockAttachmentRepo crée un faux repository de pièces jointes.
func NewMockAttachmentRepo() *MockAttachmentRepo {
	return &MockAttachmentRepo{
		byID:  make(map[uuid.UUID]*domain.AttachmentMeta),
		byKey: make(map[string]uuid.UUID),
		links: make(map[uuid.UUID]int),
	}
}

// SetLinks fixe le nombre de liens mail_attachments d'une pièce jointe (helper de
// test). 0 = orpheline.
func (m *MockAttachmentRepo) SetLinks(attachmentID uuid.UUID, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links[attachmentID] = n
}

func attachmentKey(tenantID uuid.UUID, sha []byte) string {
	return tenantID.String() + "|" + hex.EncodeToString(sha)
}

// UpsertMeta insère la métadonnée si absente (dédup par tenant+sha256).
func (m *MockAttachmentRepo) UpsertMeta(_ context.Context, meta domain.AttachmentMeta) (uuid.UUID, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := attachmentKey(meta.TenantID, meta.SHA256)
	if id, ok := m.byKey[k]; ok {
		return id, false, nil
	}
	id := uuid.New()
	meta.ID = id
	m.byID[id] = &meta
	m.byKey[k] = id
	return id, true, nil
}

// GetMeta retourne les métadonnées d'une pièce jointe d'un tenant.
func (m *MockAttachmentRepo) GetMeta(_ context.Context, tenantID, attachmentID uuid.UUID) (*domain.AttachmentMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta, ok := m.byID[attachmentID]
	if !ok || meta.TenantID != tenantID {
		return nil, domain.ErrAttachmentNotFound
	}
	cp := *meta
	return &cp, nil
}

// ListOrphans retourne les métadonnées sans aucun lien mail_attachments.
func (m *MockAttachmentRepo) ListOrphans(_ context.Context, _ time.Time, limit int) ([]domain.AttachmentMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.AttachmentMeta
	for id, meta := range m.byID {
		if m.links[id] <= 0 {
			out = append(out, *meta)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// DeleteOrphanMeta supprime la métadonnée seulement si elle n'a aucun lien
// mail_attachments. Retourne true si une ligne a été supprimée.
func (m *MockAttachmentRepo) DeleteOrphanMeta(_ context.Context, attachmentID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	meta, ok := m.byID[attachmentID]
	if !ok || m.links[attachmentID] > 0 {
		return false, nil
	}
	delete(m.byKey, attachmentKey(meta.TenantID, meta.SHA256))
	delete(m.byID, attachmentID)
	delete(m.links, attachmentID)
	return true, nil
}

// MockBlobStore est un faux port.BlobStore en mémoire.
type MockBlobStore struct {
	mu   sync.RWMutex
	data map[string][]byte // clé = tenant|sha256hex
}

// NewMockBlobStore crée un faux BlobStore en mémoire.
func NewMockBlobStore() *MockBlobStore {
	return &MockBlobStore{data: make(map[string][]byte)}
}

// Put stocke le contenu sous (tenantID, sha256).
func (b *MockBlobStore) Put(_ context.Context, tenantID uuid.UUID, sha, content []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[attachmentKey(tenantID, sha)] = content
	return nil
}

// Get retourne le contenu stocké sous (tenantID, sha256).
func (b *MockBlobStore) Get(_ context.Context, tenantID uuid.UUID, sha []byte) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	c, ok := b.data[attachmentKey(tenantID, sha)]
	if !ok {
		return nil, domain.ErrAttachmentNotFound
	}
	return c, nil
}

// Delete supprime le contenu sous (tenantID, sha256).
func (b *MockBlobStore) Delete(_ context.Context, tenantID uuid.UUID, sha []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.data, attachmentKey(tenantID, sha))
	return nil
}

// Exists indique si un contenu est présent sous (tenantID, sha256).
func (b *MockBlobStore) Exists(_ context.Context, tenantID uuid.UUID, sha []byte) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.data[attachmentKey(tenantID, sha)]
	return ok, nil
}
