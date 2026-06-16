package port

import (
	"context"

	"github.com/google/uuid"
)

// BlobStore stocke et récupère le contenu binaire des pièces jointes, adressé
// par le SHA-256 du contenu et scopé au tenant. C'est un magasin « bête » : il
// ne connaît ni la déduplication ni le comptage de références (gérés par le
// service applicatif), il ne fait que persister et relire des octets.
//
// Implémentations : adaptateur postgres (table attachment_blobs) et adaptateur
// s3 (compatible SeaweedFS / MinIO / S3 / R2).
type BlobStore interface {
	// Put stocke le contenu sous (tenantID, sha256). Idempotent : si la clé
	// existe déjà, l'appel réussit sans réécrire.
	Put(ctx context.Context, tenantID uuid.UUID, sha256, content []byte) error
	// Get retourne le contenu stocké sous (tenantID, sha256).
	Get(ctx context.Context, tenantID uuid.UUID, sha256 []byte) ([]byte, error)
	// Delete supprime le contenu sous (tenantID, sha256). Idempotent : l'absence
	// n'est pas une erreur.
	Delete(ctx context.Context, tenantID uuid.UUID, sha256 []byte) error
	// Exists indique si un contenu est présent sous (tenantID, sha256).
	Exists(ctx context.Context, tenantID uuid.UUID, sha256 []byte) (bool, error)
}
