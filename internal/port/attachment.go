package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// AttachmentRepository persiste les métadonnées des pièces jointes dédupliquées
// (table attachments). Le contenu binaire lui-même est géré séparément par le
// BlobStore. L'orphelinage est déterminé à la volée (absence de lien
// mail_attachments) plutôt que par un compteur dénormalisé, qui dériverait lors
// des suppressions en cascade (archivage des mails).
type AttachmentRepository interface {
	// UpsertMeta insère la métadonnée si elle n'existe pas (déduplication par
	// (tenant_id, sha256)) et retourne son id. created vaut true si la ligne
	// vient d'être créée — auquel cas l'appelant doit écrire le blob.
	UpsertMeta(ctx context.Context, meta domain.AttachmentMeta) (id uuid.UUID, created bool, err error)
	// GetMeta retourne les métadonnées d'une pièce jointe d'un tenant.
	GetMeta(ctx context.Context, tenantID, attachmentID uuid.UUID) (*domain.AttachmentMeta, error)
	// ListOrphans retourne les métadonnées sans aucun lien mail_attachments, créées
	// avant olderThan, limitées à limit lignes (balayage GC).
	ListOrphans(ctx context.Context, olderThan time.Time, limit int) ([]domain.AttachmentMeta, error)
	// DeleteOrphanMeta supprime la métadonnée seulement si elle n'a (toujours)
	// aucun lien mail_attachments, en une opération atomique : si un lien est
	// apparu depuis le balayage (campagne ré-référençant le même contenu), rien
	// n'est supprimé. deleted indique si une ligne a effectivement été retirée —
	// l'appelant ne doit supprimer le blob que dans ce cas.
	DeleteOrphanMeta(ctx context.Context, attachmentID uuid.UUID) (deleted bool, err error)
}
