package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// AttachmentService gère les pièces jointes dédupliquées : il calcule le hash du
// contenu, déduplique via la métadonnée (tenant_id, sha256) et délègue le stockage
// des octets au BlobStore. Le BlobStore reste « bête » ; toute la logique de
// déduplication, de comptage de références et de cohérence DB/blob vit ici.
type AttachmentService struct {
	repo    port.AttachmentRepository
	blobs   port.BlobStore
	storage string // label du backend pour les nouvelles lignes : "postgres" | "s3"
}

// NewAttachmentService crée un service de pièces jointes. storage est l'étiquette
// du backend du BlobStore fourni (domain.AttachmentStoragePostgres ou ...S3),
// enregistrée sur les nouvelles métadonnées.
func NewAttachmentService(repo port.AttachmentRepository, blobs port.BlobStore, storage string) *AttachmentService {
	return &AttachmentService{repo: repo, blobs: blobs, storage: storage}
}

// Store déduplique et stocke une pièce jointe, et retourne l'identifiant de sa
// métadonnée. Si le contenu existe déjà pour ce tenant, aucune réécriture du blob
// n'a lieu (déduplication). Ordre : métadonnée d'abord (upsert atomique), puis blob
// — un blob manquant est réparable et sera ramassé/réessayé.
//
// L'orphelinage est déterminé par l'absence de lien mail_attachments (voir
// CollectOrphans), pas par un compteur : il n'y a donc rien à incrémenter ici.
func (s *AttachmentService) Store(
	ctx context.Context,
	tenantID uuid.UUID,
	content []byte,
	contentType string,
) (uuid.UUID, error) {
	sum := sha256.Sum256(content)
	hash := sum[:]

	id, created, err := s.repo.UpsertMeta(ctx, domain.AttachmentMeta{
		TenantID:    tenantID,
		SHA256:      hash,
		Size:        int64(len(content)),
		ContentType: contentType,
		Storage:     s.storage,
	})
	if err != nil {
		return uuid.Nil, err
	}

	if created {
		if err := s.blobs.Put(ctx, tenantID, hash, content); err != nil {
			return uuid.Nil, fmt.Errorf("erreur de stockage du contenu de la pièce jointe : %w", err)
		}
	}
	return id, nil
}

// Load récupère le contenu d'une pièce jointe (métadonnée puis blob).
func (s *AttachmentService) Load(ctx context.Context, tenantID, attachmentID uuid.UUID) ([]byte, error) {
	meta, err := s.repo.GetMeta(ctx, tenantID, attachmentID)
	if err != nil {
		return nil, err
	}
	return s.blobs.Get(ctx, tenantID, meta.SHA256)
}

// CollectOrphans supprime les pièces jointes sans aucun lien mail_attachments,
// créées avant olderThan. Le délai de grâce (olderThan) évite de ramasser un
// contenu qu'une campagne en cours de création référence. Retourne le nombre de
// pièces jointes effectivement supprimées.
//
// Ordre : métadonnée d'abord (suppression conditionnelle atomique), puis blob —
// l'inverse de l'écriture. On ne supprime le blob que si DeleteOrphanMeta a bien
// retiré la ligne ; si une campagne a ré-référencé le contenu entre ListOrphans
// et ici, la métadonnée (et donc le blob) sont préservés. Si la suppression du
// blob échoue après celle de la métadonnée, le pire cas est un blob orphelin
// (gaspillage de stockage, inoffensif) plutôt qu'un lien pointant un blob absent
// (envoi cassé).
func (s *AttachmentService) CollectOrphans(ctx context.Context, olderThan time.Time, limit int) (int, error) {
	orphans, err := s.repo.ListOrphans(ctx, olderThan, limit)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, m := range orphans {
		removed, err := s.repo.DeleteOrphanMeta(ctx, m.ID)
		if err != nil {
			return deleted, fmt.Errorf("erreur de suppression de la métadonnée orpheline : %w", err)
		}
		if !removed {
			continue // ré-référencée entre-temps : on la garde
		}
		if err := s.blobs.Delete(ctx, m.TenantID, m.SHA256); err != nil {
			return deleted, fmt.Errorf("erreur de suppression du blob orphelin : %w", err)
		}
		deleted++
	}
	return deleted, nil
}
