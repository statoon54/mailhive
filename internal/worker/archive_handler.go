package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/i18n"
)

// archiveBatchSize est la taille des batches pour l'archivage des mails.
const archiveBatchSize = 5000

// ArchiveHandler gère l'archivage des mails terminés de plus de 90 jours.
type ArchiveHandler struct {
	pool *pgxpool.Pool
}

// NewArchiveHandler crée un nouveau ArchiveHandler.
func NewArchiveHandler(pool *pgxpool.Pool) *ArchiveHandler {
	return &ArchiveHandler{pool: pool}
}

// HandleMailArchive est le handler Asynq pour la tâche d'archivage des mails.
func (h *ArchiveHandler) HandleMailArchive(_ context.Context, _ *asynq.Task) error {
	return h.ArchiveOldMails(context.Background())
}

// ArchiveOldMails déplace les mails terminés (sent, failed, cancelled) de plus de 90 jours
// vers les tables d'archive, par batch pour limiter la charge.
func (h *ArchiveHandler) ArchiveOldMails(ctx context.Context) error {
	totalArchived := 0

	for {
		tx, err := h.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_tx"), err)
		}

		// Insérer un batch de mails dans l'archive
		insertQuery := `
			WITH to_archive AS (
				SELECT id FROM mails
				WHERE status IN ('sent', 'failed', 'cancelled')
				  AND created_at < NOW() - INTERVAL '90 days'
				LIMIT $1
				FOR UPDATE SKIP LOCKED
			)
			INSERT INTO mails_archive (
				id, tenant_id, smtp_config_id, template_id, from_email, from_name,
				subject, text_body, html_body, template_data, status,
				status_message, attempts, scheduled_at, sent_at, task_id, priority,
				metadata, created_at, updated_at, archived_at
			)
			SELECT
				m.id, m.tenant_id, m.smtp_config_id, m.template_id, m.from_email, m.from_name,
				m.subject, m.text_body, m.html_body, m.template_data, m.status,
				m.status_message, m.attempts, m.scheduled_at, m.sent_at, m.task_id, m.priority,
				m.metadata, m.created_at, m.updated_at, NOW()
			FROM mails m
			WHERE m.id IN (SELECT id FROM to_archive)
			RETURNING id`

		rows, err := tx.Query(ctx, insertQuery, archiveBatchSize)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_insert"), err)
		}

		var archivedIDs []any
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				_ = tx.Rollback(ctx)
				return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_scan"), err)
			}
			archivedIDs = append(archivedIDs, id)
		}
		rows.Close()

		if len(archivedIDs) == 0 {
			_ = tx.Rollback(ctx)
			break
		}

		// Archiver les destinataires correspondants
		archiveRecipientsQuery := `
			INSERT INTO mail_recipients_archive (id, mail_id, type, email, name)
			SELECT r.id, r.mail_id, r.type, r.email, r.name
			FROM mail_recipients r
			WHERE r.mail_id = ANY($1::uuid[])`

		_, err = tx.Exec(ctx, archiveRecipientsQuery, archivedIDs)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_recipients"), err)
		}

		// Supprimer les destinataires archivés de la table active
		deleteRecipientsQuery := `DELETE FROM mail_recipients WHERE mail_id = ANY($1::uuid[])`
		_, err = tx.Exec(ctx, deleteRecipientsQuery, archivedIDs)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_del_recip"), err)
		}

		// Supprimer les mails archivés de la table active
		deleteMailsQuery := `DELETE FROM mails WHERE id = ANY($1::uuid[])`
		_, err = tx.Exec(ctx, deleteMailsQuery, archivedIDs)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_del_mails"), err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf(i18n.T(i18n.FR, "worker.err.archive_commit"), err)
		}

		totalArchived += len(archivedIDs)
		slog.Info("lot de mails archivé", "batch_size", len(archivedIDs), "total_archived", totalArchived)

		// Si le batch est incomplet, on a fini
		if len(archivedIDs) < archiveBatchSize {
			break
		}
	}

	if totalArchived > 0 {
		slog.Info("archivage terminé", "total_archived", totalArchived)
	} else {
		slog.Info("aucun mail à archiver")
	}

	return nil
}
