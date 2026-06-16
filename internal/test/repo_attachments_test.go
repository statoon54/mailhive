package test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/blobstore"
	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
)

func TestBlobStore_Postgres_RoundTrip(t *testing.T) {
	pool := PGPool(t)
	store := blobstore.NewPostgresStore(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	content := []byte("contenu binaire de la pièce jointe")
	sum := sha256.Sum256(content)
	hash := sum[:]

	// Absent au départ.
	exists, err := store.Exists(ctx, tenant.ID, hash)
	require.NoError(t, err)
	assert.False(t, exists)

	// Put puis Get.
	require.NoError(t, store.Put(ctx, tenant.ID, hash, content))
	got, err := store.Get(ctx, tenant.ID, hash)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	exists, err = store.Exists(ctx, tenant.ID, hash)
	require.NoError(t, err)
	assert.True(t, exists)

	// Put idempotent (ON CONFLICT DO NOTHING).
	require.NoError(t, store.Put(ctx, tenant.ID, hash, content))

	// Delete puis absence.
	require.NoError(t, store.Delete(ctx, tenant.ID, hash))
	_, err = store.Get(ctx, tenant.ID, hash)
	assert.ErrorIs(t, err, blobstore.ErrBlobNotFound)
}

func TestAttachmentRepo_UpsertMeta_Deduplicates(t *testing.T) {
	pool := PGPool(t)
	repo := postgres.NewAttachmentRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	sum := sha256.Sum256([]byte("doc.pdf content"))
	meta := domain.AttachmentMeta{
		TenantID:    tenant.ID,
		SHA256:      sum[:],
		Size:        15,
		ContentType: "application/pdf",
		Storage:     domain.AttachmentStoragePostgres,
	}

	id1, created1, err := repo.UpsertMeta(ctx, meta)
	require.NoError(t, err)
	assert.True(t, created1, "le premier upsert crée la ligne")

	id2, created2, err := repo.UpsertMeta(ctx, meta)
	require.NoError(t, err)
	assert.False(t, created2, "le second upsert (même tenant+sha256) ne crée pas")
	assert.Equal(t, id1, id2, "dédup : même id retourné")

	// Tenant différent → ligne distincte (dédup scopée au tenant).
	other := insertTenant(t, pool)
	meta.TenantID = other.ID
	id3, created3, err := repo.UpsertMeta(ctx, meta)
	require.NoError(t, err)
	assert.True(t, created3)
	assert.NotEqual(t, id1, id3)
}

// TestAttachmentRepo_Orphans_NoLink vérifie qu'une pièce jointe sans lien
// mail_attachments est orpheline, et qu'elle cesse de l'être dès qu'un mail la
// référence.
func TestAttachmentRepo_Orphans_NoLink(t *testing.T) {
	pool := PGPool(t)
	mailRepo := postgres.NewMailRepository(pool)
	repo := postgres.NewAttachmentRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	sum := sha256.Sum256([]byte("payload ref"))
	id, _, err := repo.UpsertMeta(ctx, domain.AttachmentMeta{
		TenantID: tenant.ID, SHA256: sum[:], Size: 11,
		ContentType: "text/plain", Storage: domain.AttachmentStoragePostgres,
	})
	require.NoError(t, err)

	// Aucun lien → orpheline.
	orphans, err := repo.ListOrphans(ctx, time.Now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.True(t, containsAttachment(orphans, id), "sans lien → orpheline")

	// Lier à un mail → plus orpheline.
	mailID, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID: mailID, TenantID: tenant.ID, FromEmail: "a@example.com", Subject: "M",
		Status: domain.MailStatusPending, Priority: domain.MailPriorityDefault,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	links := []domain.AttachmentLink{{AttachmentID: id, Filename: "doc.pdf", Position: 0}}
	require.NoError(t, mailRepo.CreateWithRecipients(ctx, mail, nil, links))

	orphans, err = repo.ListOrphans(ctx, time.Now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.False(t, containsAttachment(orphans, id), "lien présent → non orpheline")
}

// TestAttachmentRepo_OrphanAfterMailDeleted est le test de non-régression du bug
// d'archivage : quand un mail est supprimé (DELETE, comme le fait l'archivage par
// CASCADE sur mail_attachments), sa pièce jointe doit redevenir collectable par le
// GC. Avec un ref_count dénormalisé ce n'était pas le cas — le compteur restait figé.
func TestAttachmentRepo_OrphanAfterMailDeleted(t *testing.T) {
	pool := PGPool(t)
	mailRepo := postgres.NewMailRepository(pool)
	repo := postgres.NewAttachmentRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	sum := sha256.Sum256([]byte("pièce jointe d'une campagne"))
	attID, _, err := repo.UpsertMeta(ctx, domain.AttachmentMeta{
		TenantID: tenant.ID, SHA256: sum[:], Size: 27,
		ContentType: "application/pdf", Storage: domain.AttachmentStoragePostgres,
	})
	require.NoError(t, err)

	mailID, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID: mailID, TenantID: tenant.ID, FromEmail: "a@example.com", Subject: "Campagne",
		Status: domain.MailStatusSent, Priority: domain.MailPriorityDefault,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	links := []domain.AttachmentLink{{AttachmentID: attID, Filename: "doc.pdf", Position: 0}}
	require.NoError(t, mailRepo.CreateWithRecipients(ctx, mail, nil, links))

	// Référencée → pas orpheline.
	orphans, err := repo.ListOrphans(ctx, time.Now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.False(t, containsAttachment(orphans, attID))

	// Simuler l'archivage : DELETE du mail → CASCADE supprime mail_attachments.
	_, err = pool.Exec(ctx, `DELETE FROM mails WHERE id = $1`, mailID)
	require.NoError(t, err)

	// Le lien a disparu → la pièce jointe est de nouveau collectable.
	orphans, err = repo.ListOrphans(ctx, time.Now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.True(t, containsAttachment(orphans, attID),
		"après suppression du mail, la pièce jointe doit redevenir orpheline")
}

// TestAttachmentRepo_DeleteOrphanMeta_GuardsReReference vérifie le garde anti-race
// du GC : si une pièce jointe que le balayage avait vue orpheline est ré-référencée
// par un mail avant la suppression, DeleteOrphanMeta ne la supprime pas (retourne
// false) — sans quoi le GC effacerait un blob qu'un mail vient de référencer.
func TestAttachmentRepo_DeleteOrphanMeta_GuardsReReference(t *testing.T) {
	pool := PGPool(t)
	mailRepo := postgres.NewMailRepository(pool)
	repo := postgres.NewAttachmentRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	sum := sha256.Sum256([]byte("contenu ré-référencé pendant le GC"))
	attID, _, err := repo.UpsertMeta(ctx, domain.AttachmentMeta{
		TenantID: tenant.ID, SHA256: sum[:], Size: 34,
		ContentType: "application/pdf", Storage: domain.AttachmentStoragePostgres,
	})
	require.NoError(t, err)

	// Orpheline au balayage → puis un mail la référence avant la suppression.
	mailID, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID: mailID, TenantID: tenant.ID, FromEmail: "a@example.com", Subject: "M",
		Status: domain.MailStatusPending, Priority: domain.MailPriorityDefault,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	links := []domain.AttachmentLink{{AttachmentID: attID, Filename: "doc.pdf", Position: 0}}
	require.NoError(t, mailRepo.CreateWithRecipients(ctx, mail, nil, links))

	// Le lien existe désormais → la suppression conditionnelle ne retire rien.
	deleted, err := repo.DeleteOrphanMeta(ctx, attID)
	require.NoError(t, err)
	assert.False(t, deleted, "ne doit pas supprimer une pièce jointe ré-référencée")

	// La métadonnée subsiste (le mail peut toujours résoudre sa pièce jointe).
	_, err = repo.GetMeta(ctx, tenant.ID, attID)
	require.NoError(t, err)

	// Vraiment orpheline → cette fois elle est supprimée.
	_, err = pool.Exec(ctx, `DELETE FROM mails WHERE id = $1`, mailID)
	require.NoError(t, err)
	deleted, err = repo.DeleteOrphanMeta(ctx, attID)
	require.NoError(t, err)
	assert.True(t, deleted, "sans lien, la pièce jointe doit être supprimée")
}

// TestCreateBatchWithRecipients_SharesAttachment vérifie la déduplication de bout
// en bout : une pièce jointe partagée par N mails (mode campagne individuelle)
// donne 1 seule métadonnée et N liens mail_attachments.
func TestCreateBatchWithRecipients_SharesAttachment(t *testing.T) {
	pool := PGPool(t)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	mailRepo := postgres.NewMailRepository(pool)
	attRepo := postgres.NewAttachmentRepository(pool)
	store := blobstore.NewPostgresStore(pool)
	attSvc := service.NewAttachmentService(attRepo, store, domain.AttachmentStoragePostgres)

	// Une pièce jointe stockée une fois (dédup par contenu).
	attID, err := attSvc.Store(ctx, tenant.ID, []byte("contenu partagé de la campagne"), "application/pdf")
	require.NoError(t, err)
	links := []domain.AttachmentLink{{AttachmentID: attID, Filename: "doc.pdf", Position: 0}}

	const n = 5
	mails := make([]*domain.Mail, n)
	var recipients []domain.MailRecipient
	for i := range mails {
		id, _ := uuid.NewV7()
		mails[i] = &domain.Mail{
			ID: id, TenantID: tenant.ID, FromEmail: "campagne@example.com",
			Subject: "Campagne", Status: domain.MailStatusPending,
			Priority: domain.MailPriorityDefault, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		rid, _ := uuid.NewV7()
		recipients = append(recipients, domain.MailRecipient{
			ID: rid, MailID: id, Type: domain.RecipientTo, Email: "x@example.com",
		})
	}

	require.NoError(t, mailRepo.CreateBatchWithRecipients(ctx, mails, recipients, links))

	// Une seule métadonnée, et N liens mail_attachments (un par mail).
	_, err = attRepo.GetMeta(ctx, tenant.ID, attID)
	require.NoError(t, err)
	var linkCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM mail_attachments WHERE attachment_id = $1`, attID).Scan(&linkCount))
	assert.Equal(t, n, linkCount, "un lien mail_attachments par mail")

	// Chaque mail résout la pièce jointe via GetAttachmentRefs.
	for _, m := range mails {
		refs, err := mailRepo.GetAttachmentRefs(ctx, m.ID)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		assert.Equal(t, attID, refs[0].AttachmentID)
		assert.Equal(t, "doc.pdf", refs[0].Filename)
		assert.Equal(t, "application/pdf", refs[0].ContentType)

		// Le contenu est rechargeable via le service (round-trip blob).
		content, err := attSvc.Load(ctx, tenant.ID, refs[0].AttachmentID)
		require.NoError(t, err)
		assert.Equal(t, []byte("contenu partagé de la campagne"), content)
	}

	// Un seul blob physique stocké malgré les N mails.
	var blobCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM attachment_blobs WHERE tenant_id = $1`, tenant.ID).Scan(&blobCount))
	assert.Equal(t, 1, blobCount, "le contenu n'est stocké qu'une fois (dédup)")
}

func containsAttachment(metas []domain.AttachmentMeta, id uuid.UUID) bool {
	for _, m := range metas {
		if m.ID == id {
			return true
		}
	}
	return false
}
