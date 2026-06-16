package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/domain"
)

// AttachmentRepository implémente port.AttachmentRepository avec PostgreSQL.
type AttachmentRepository struct {
	pool *pgxpool.Pool
}

// NewAttachmentRepository crée un nouveau repository de pièces jointes.
func NewAttachmentRepository(pool *pgxpool.Pool) *AttachmentRepository {
	return &AttachmentRepository{pool: pool}
}

// UpsertMeta insère la métadonnée si absente (dédup par (tenant_id, sha256)).
// L'idiome ON CONFLICT DO UPDATE (no-op) + RETURNING (xmax = 0) permet de
// récupérer l'id existant ou nouvellement créé, et de savoir lequel des deux,
// en une seule requête.
func (r *AttachmentRepository) UpsertMeta(
	ctx context.Context,
	meta domain.AttachmentMeta,
) (uuid.UUID, bool, error) {
	query := `
		INSERT INTO attachments (tenant_id, sha256, size_bytes, content_type, storage)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, sha256) DO UPDATE SET tenant_id = attachments.tenant_id
		RETURNING id, (xmax = 0) AS created`

	var id uuid.UUID
	var created bool
	err := r.pool.QueryRow(ctx, query,
		meta.TenantID, meta.SHA256, meta.Size, meta.ContentType, meta.Storage,
	).Scan(&id, &created)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("erreur d'upsert de la pièce jointe : %w", err)
	}
	return id, created, nil
}

// GetMeta retourne les métadonnées d'une pièce jointe d'un tenant.
func (r *AttachmentRepository) GetMeta(
	ctx context.Context,
	tenantID, attachmentID uuid.UUID,
) (*domain.AttachmentMeta, error) {
	query := `
		SELECT id, tenant_id, sha256, size_bytes, content_type, storage
		FROM attachments WHERE tenant_id = $1 AND id = $2`

	var m domain.AttachmentMeta
	err := r.pool.QueryRow(ctx, query, tenantID, attachmentID).Scan(
		&m.ID, &m.TenantID, &m.SHA256, &m.Size, &m.ContentType, &m.Storage,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAttachmentNotFound
		}
		return nil, fmt.Errorf("erreur de lecture de la pièce jointe : %w", err)
	}
	return &m, nil
}

// ListOrphans retourne les métadonnées sans aucun lien mail_attachments, créées
// avant olderThan. L'orphelinage est calculé à la volée (NOT EXISTS) : pas de
// compteur dénormalisé à maintenir, donc rien ne peut dériver lors des
// suppressions en cascade (archivage des mails).
func (r *AttachmentRepository) ListOrphans(
	ctx context.Context,
	olderThan time.Time,
	limit int,
) ([]domain.AttachmentMeta, error) {
	query := `
		SELECT a.id, a.tenant_id, a.sha256, a.size_bytes, a.content_type, a.storage
		FROM attachments a
		WHERE a.created_at < $1
		  AND NOT EXISTS (SELECT 1 FROM mail_attachments ma WHERE ma.attachment_id = a.id)
		ORDER BY a.created_at
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, olderThan, limit)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des pièces jointes orphelines : %w", err)
	}
	defer rows.Close()

	var result []domain.AttachmentMeta
	for rows.Next() {
		var m domain.AttachmentMeta
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.SHA256, &m.Size, &m.ContentType, &m.Storage,
		); err != nil {
			return nil, fmt.Errorf("erreur de lecture d'une pièce jointe orpheline : %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur de parcours des pièces jointes orphelines : %w", err)
	}
	return result, nil
}

// DeleteOrphanMeta supprime la métadonnée seulement si elle n'a toujours aucun
// lien mail_attachments. Le garde NOT EXISTS rend l'opération atomique vis-à-vis
// d'une campagne concurrente qui ré-référencerait le même contenu entre le
// balayage (ListOrphans) et ici : dans ce cas la ligne n'est pas supprimée et le
// blob est préservé (sinon le DELETE échouerait d'ailleurs sur la FK). Retourne
// true si une ligne a effectivement été supprimée.
func (r *AttachmentRepository) DeleteOrphanMeta(ctx context.Context, attachmentID uuid.UUID) (bool, error) {
	query := `
		DELETE FROM attachments a
		WHERE a.id = $1
		  AND NOT EXISTS (SELECT 1 FROM mail_attachments ma WHERE ma.attachment_id = a.id)`
	tag, err := r.pool.Exec(ctx, query, attachmentID)
	if err != nil {
		return false, fmt.Errorf("erreur de suppression de la pièce jointe : %w", err)
	}
	return tag.RowsAffected() > 0, nil
}
