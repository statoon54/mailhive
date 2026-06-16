package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/statoon54/mailhive/internal/compress"
	"github.com/statoon54/mailhive/internal/domain"
)

// MailRepository implémente port.MailRepository avec PostgreSQL.
type MailRepository struct {
	pool *pgxpool.Pool
}

// NewMailRepository crée un nouveau repository mail.
func NewMailRepository(pool *pgxpool.Pool) *MailRepository {
	return &MailRepository{pool: pool}
}

// execQuerier regroupe les opérations d'écriture communes à *pgxpool.Pool et à
// pgx.Tx. Cela permet de partager la logique d'insertion entre le chemin simple
// (pool) et le chemin transactionnel (tx).
type execQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

const insertMailQuery = `
	INSERT INTO mails (id, tenant_id, smtp_config_id, template_id, from_email, from_name,
		subject, text_body, html_body, template_data, status, status_message,
		attempts, scheduled_at, sent_at, task_id, priority, metadata, spam_score, tags, compressed_body, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`

// mailInsertArgs sérialise les champs JSON d'un mail et retourne les arguments
// positionnels d'insertMailQuery.
func mailInsertArgs(mail *domain.Mail) ([]any, error) {
	templateDataJSON, err := json.Marshal(mail.TemplateData)
	if err != nil {
		return nil, fmt.Errorf("erreur de sérialisation des données de template : %w", err)
	}
	metadataJSON, err := json.Marshal(mail.Metadata)
	if err != nil {
		return nil, fmt.Errorf("erreur de sérialisation des métadonnées : %w", err)
	}
	tags := mail.Tags
	if tags == nil {
		tags = []string{}
	}
	return []any{
		mail.ID, mail.TenantID, mail.SMTPConfigID, mail.TemplateID,
		mail.FromEmail, mail.FromName, mail.Subject, mail.TextBody, mail.HTMLBody,
		templateDataJSON, mail.Status, mail.StatusMessage,
		mail.Attempts, mail.ScheduledAt, mail.SentAt, mail.TaskID, mail.Priority, metadataJSON,
		mail.SpamScore, tags, mail.CompressedBody,
		mail.CreatedAt, mail.UpdatedAt,
	}, nil
}

// insertMail insère un seul mail via le querier fourni (pool ou transaction).
func insertMail(ctx context.Context, q execQuerier, mail *domain.Mail) error {
	args, err := mailInsertArgs(mail)
	if err != nil {
		return err
	}
	if _, err := q.Exec(ctx, insertMailQuery, args...); err != nil {
		return fmt.Errorf("erreur de création du mail : %w", err)
	}
	return nil
}

// batchChunkSize est la taille des chunks pour les opérations batch en base de données.
const batchChunkSize = 1000

// insertMailsBatch insère plusieurs mails par chunks via le querier fourni.
func insertMailsBatch(ctx context.Context, q execQuerier, mails []*domain.Mail) error {
	for start := 0; start < len(mails); start += batchChunkSize {
		end := min(start+batchChunkSize, len(mails))
		chunk := mails[start:end]

		batch := &pgx.Batch{}
		for _, mail := range chunk {
			args, err := mailInsertArgs(mail)
			if err != nil {
				return err
			}
			batch.Queue(insertMailQuery, args...)
		}

		br := q.SendBatch(ctx, batch)
		for range chunk {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("erreur de création batch des mails : %w", err)
			}
		}
		_ = br.Close()
	}

	return nil
}

// Create insère un nouveau mail en base de données.
func (r *MailRepository) Create(ctx context.Context, mail *domain.Mail) error {
	return insertMail(ctx, r.pool, mail)
}

// CreateBatch insère plusieurs mails en base de données par batch.
func (r *MailRepository) CreateBatch(ctx context.Context, mails []*domain.Mail) error {
	if len(mails) == 0 {
		return nil
	}
	return insertMailsBatch(ctx, r.pool, mails)
}

// insertMailAttachments insère les liens mail->pièce jointe (table mail_attachments)
// pour chaque mail fourni. Les mêmes liens s'appliquent à tous les mailIDs (cas du
// mode individuel : une campagne partage les mêmes pièces jointes). L'orphelinage
// se déduisant de l'absence de lien, il n'y a aucun compteur à maintenir ici.
func insertMailAttachments(ctx context.Context, q execQuerier, mailIDs []uuid.UUID, links []domain.AttachmentLink) error {
	if len(links) == 0 || len(mailIDs) == 0 {
		return nil
	}

	const linkQuery = `INSERT INTO mail_attachments (mail_id, attachment_id, filename, position) VALUES ($1, $2, $3, $4)`

	batch := &pgx.Batch{}
	for _, mailID := range mailIDs {
		for _, l := range links {
			batch.Queue(linkQuery, mailID, l.AttachmentID, l.Filename, l.Position)
		}
	}

	br := q.SendBatch(ctx, batch)
	total := len(mailIDs) * len(links)
	for range total {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("erreur d'association des pièces jointes : %w", err)
		}
	}
	return br.Close()
}

// CreateWithRecipients insère un mail, ses destinataires et ses liens de pièces
// jointes dans une seule transaction : en cas d'échec d'une étape, aucune ligne
// n'est persistée (plus de mail orphelin sans destinataires).
func (r *MailRepository) CreateWithRecipients(
	ctx context.Context,
	mail *domain.Mail,
	recipients []domain.MailRecipient,
	links []domain.AttachmentLink,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erreur d'ouverture de la transaction : %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op après un Commit réussi

	if err := insertMail(ctx, tx, mail); err != nil {
		return err
	}
	if err := insertRecipients(ctx, tx, recipients); err != nil {
		return err
	}
	if err := insertMailAttachments(ctx, tx, []uuid.UUID{mail.ID}, links); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erreur de validation de la transaction : %w", err)
	}
	return nil
}

// CreateBatchWithRecipients insère plusieurs mails, tous leurs destinataires et
// les liens de pièces jointes (partagés par tous les mails) dans une seule
// transaction (atomicité du mode individuel).
func (r *MailRepository) CreateBatchWithRecipients(
	ctx context.Context,
	mails []*domain.Mail,
	recipients []domain.MailRecipient,
	links []domain.AttachmentLink,
) error {
	if len(mails) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erreur d'ouverture de la transaction : %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op après un Commit réussi

	if err := insertMailsBatch(ctx, tx, mails); err != nil {
		return err
	}
	if err := insertRecipients(ctx, tx, recipients); err != nil {
		return err
	}
	mailIDs := make([]uuid.UUID, len(mails))
	for i, m := range mails {
		mailIDs[i] = m.ID
	}
	if err := insertMailAttachments(ctx, tx, mailIDs, links); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erreur de validation de la transaction : %w", err)
	}
	return nil
}

// GetAttachmentRefs retourne les pièces jointes liées à un mail (jointure
// mail_attachments + attachments), pour reconstituer le contenu côté worker.
func (r *MailRepository) GetAttachmentRefs(ctx context.Context, mailID uuid.UUID) ([]domain.AttachmentRef, error) {
	query := `
		SELECT ma.attachment_id, ma.filename, a.sha256, a.content_type, a.size_bytes, a.storage
		FROM mail_attachments ma
		JOIN attachments a ON a.id = ma.attachment_id
		WHERE ma.mail_id = $1
		ORDER BY ma.position`

	rows, err := r.pool.Query(ctx, query, mailID)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des pièces jointes du mail : %w", err)
	}
	defer rows.Close()

	var refs []domain.AttachmentRef
	for rows.Next() {
		var ref domain.AttachmentRef
		if err := rows.Scan(
			&ref.AttachmentID, &ref.Filename, &ref.SHA256, &ref.ContentType, &ref.Size, &ref.Storage,
		); err != nil {
			return nil, fmt.Errorf("erreur de lecture d'une pièce jointe du mail : %w", err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur de parcours des pièces jointes du mail : %w", err)
	}
	return refs, nil
}

// GetByID retourne un mail par son identifiant et tenant.
func (r *MailRepository) GetByID(
	ctx context.Context,
	tenantID, id uuid.UUID,
) (*domain.Mail, error) {
	query := `
		SELECT id, tenant_id, smtp_config_id, template_id, from_email, from_name,
			subject, text_body, html_body, template_data, status, status_message,
			attempts, scheduled_at, sent_at, task_id, priority, metadata, spam_score, tags, compressed_body, created_at, updated_at
		FROM mails WHERE id = $1 AND tenant_id = $2`

	row := r.pool.QueryRow(ctx, query, id, tenantID)
	return r.scanMail(row)
}

// List retourne la liste paginée des mails d'un tenant.
func (r *MailRepository) List(
	ctx context.Context,
	tenantID uuid.UUID,
	filter domain.MailListFilter,
) (*domain.PaginatedList[domain.Mail], error) {
	offset := (filter.Page - 1) * filter.Limit

	// Clause WHERE partagée entre le comptage et le listage : source unique de
	// vérité (évite la dérive entre les deux requêtes).
	where, args := buildMailListWhere(tenantID, filter)

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM mails m`+where, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("erreur de comptage des mails : %w", err)
	}

	listQuery := `
		SELECT m.id, m.tenant_id, m.smtp_config_id, m.template_id, m.from_email, m.from_name,
			m.subject, m.text_body, m.html_body, m.template_data, m.status, m.status_message,
			m.attempts, m.scheduled_at, m.sent_at, m.task_id, m.priority, m.metadata, m.spam_score, m.tags, m.compressed_body, m.created_at, m.updated_at
		FROM mails m` + where +
		fmt.Sprintf(` ORDER BY m.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	listArgs := append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des mails : %w", err)
	}
	defer rows.Close()

	var mails []domain.Mail
	for rows.Next() {
		m, err := r.scanMail(rows)
		if err != nil {
			return nil, err
		}
		mails = append(mails, *m)
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &domain.PaginatedList[domain.Mail]{
		Items:      mails,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// buildMailListWhere construit la clause WHERE (et ses arguments positionnels)
// partagée par les requêtes de comptage et de listage des mails d'un tenant.
// Le premier argument ($1) est toujours le tenant_id.
func buildMailListWhere(tenantID uuid.UUID, filter domain.MailListFilter) (string, []any) {
	where := ` WHERE m.tenant_id = $1`
	args := []any{tenantID}
	idx := 2

	if filter.Status != nil {
		where += fmt.Sprintf(` AND m.status = $%d`, idx)
		args = append(args, *filter.Status)
		idx++
	}
	if len(filter.Tags) > 0 {
		if filter.TagMode == "or" {
			where += fmt.Sprintf(` AND m.tags && $%d`, idx)
		} else {
			where += fmt.Sprintf(` AND m.tags @> $%d`, idx)
		}
		args = append(args, filter.Tags)
		idx++
	}
	if filter.Query != "" {
		// ILIKE '%x%' : rendu sargable par les index GIN trigramme (pg_trgm)
		// sur mails.subject et mail_recipients.email.
		where += fmt.Sprintf(` AND (m.subject ILIKE '%%' || $%d || '%%' OR EXISTS (SELECT 1 FROM mail_recipients mr WHERE mr.mail_id = m.id AND mr.email ILIKE '%%' || $%d || '%%'))`, idx, idx)
		args = append(args, filter.Query)
	}

	return where, args
}

// SetQueued met à jour le statut d'un mail en "queued" avec l'identifiant de tâche.
func (r *MailRepository) SetQueued(ctx context.Context, id uuid.UUID, taskID string) error {
	query := `UPDATE mails SET task_id = $2, status = 'queued', status_message = '', updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, taskID)
	if err != nil {
		return fmt.Errorf("erreur de mise en file d'attente du mail : %w", err)
	}
	return nil
}

// SetQueuedBatch met à jour le statut de plusieurs mails en "queued" par batch.
func (r *MailRepository) SetQueuedBatch(
	ctx context.Context,
	mailTaskIDs map[uuid.UUID]string,
) error {
	if len(mailTaskIDs) == 0 {
		return nil
	}

	query := `UPDATE mails SET task_id = $2, status = 'queued', status_message = '', updated_at = NOW() WHERE id = $1`

	type entry struct {
		id     uuid.UUID
		taskID string
	}
	entries := make([]entry, 0, len(mailTaskIDs))
	for id, taskID := range mailTaskIDs {
		entries = append(entries, entry{id, taskID})
	}

	for start := 0; start < len(entries); start += batchChunkSize {
		end := min(start+batchChunkSize, len(entries))
		chunk := entries[start:end]

		batch := &pgx.Batch{}
		for _, e := range chunk {
			batch.Queue(query, e.id, e.taskID)
		}

		br := r.pool.SendBatch(ctx, batch)
		for range chunk {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("erreur de mise en file d'attente batch : %w", err)
			}
		}
		_ = br.Close()
	}

	return nil
}

// UpdateStatus met à jour le statut d'un mail.
func (r *MailRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.MailStatus,
	message string,
) error {
	query := `UPDATE mails SET status = $2, status_message = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, status, message)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour du statut : %w", err)
	}
	return nil
}

// UpdateTaskID met à jour l'identifiant de tâche d'un mail.
func (r *MailRepository) UpdateTaskID(ctx context.Context, id uuid.UUID, taskID string) error {
	query := `UPDATE mails SET task_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, taskID)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour du task_id : %w", err)
	}
	return nil
}

// UpdateTaskIDs met à jour les identifiants de tâche de plusieurs mails par batch.
func (r *MailRepository) UpdateTaskIDs(
	ctx context.Context,
	mailTaskIDs map[uuid.UUID]string,
) error {
	if len(mailTaskIDs) == 0 {
		return nil
	}

	query := `UPDATE mails SET task_id = $2, updated_at = NOW() WHERE id = $1`

	// Collecter les entrées pour itérer par chunks
	type entry struct {
		id     uuid.UUID
		taskID string
	}
	entries := make([]entry, 0, len(mailTaskIDs))
	for id, taskID := range mailTaskIDs {
		entries = append(entries, entry{id, taskID})
	}

	for start := 0; start < len(entries); start += batchChunkSize {
		end := min(start+batchChunkSize, len(entries))
		chunk := entries[start:end]

		batch := &pgx.Batch{}
		for _, e := range chunk {
			batch.Queue(query, e.id, e.taskID)
		}

		br := r.pool.SendBatch(ctx, batch)
		for range chunk {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("erreur de mise à jour batch des task_ids : %w", err)
			}
		}
		_ = br.Close()
	}

	return nil
}

// UpdateStatuses met à jour le statut de plusieurs mails par batch.
func (r *MailRepository) UpdateStatuses(
	ctx context.Context,
	ids []uuid.UUID,
	status domain.MailStatus,
	message string,
) error {
	if len(ids) == 0 {
		return nil
	}

	query := `UPDATE mails SET status = $2, status_message = $3, updated_at = NOW() WHERE id = $1`

	for start := 0; start < len(ids); start += batchChunkSize {
		end := min(start+batchChunkSize, len(ids))
		chunk := ids[start:end]

		batch := &pgx.Batch{}
		for _, id := range chunk {
			batch.Queue(query, id, status, message)
		}

		br := r.pool.SendBatch(ctx, batch)
		for range chunk {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("erreur de mise à jour batch des statuts : %w", err)
			}
		}
		_ = br.Close()
	}

	return nil
}

// MarkSending passe un mail en "sending" et incrémente son compteur de tentatives
// en une seule requête (statut + status_message + attempts).
func (r *MailRepository) MarkSending(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE mails SET status = 'sending', status_message = '', attempts = attempts + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erreur de passage en envoi : %w", err)
	}
	return nil
}

// MarkSent passe un mail en "sent" et enregistre sa date d'envoi en une seule requête
// (statut + status_message + sent_at).
func (r *MailRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE mails SET status = 'sent', status_message = '', sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("erreur de mise à jour d'envoi : %w", err)
	}
	return nil
}

// Stats retourne les statistiques d'envoi d'un tenant sur les 30 derniers jours.
func (r *MailRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*domain.MailStats, error) {
	// Borner les stats aux 30 derniers jours pour éviter les full-scan sur millions de lignes.
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'sending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM mails WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '30 days'`

	var stats domain.MailStats
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&stats.Pending, &stats.Queued, &stats.Sending,
		&stats.Sent, &stats.Failed, &stats.Cancelled, &stats.Total,
	)
	if err != nil {
		return nil, fmt.Errorf("erreur de récupération des statistiques : %w", err)
	}

	// Compter les requêtes de création de mail rejetées (erreurs de validation) — 30 derniers jours.
	rejectedQuery := `
		SELECT COUNT(*) FROM audit_logs
		WHERE tenant_id = $1
		  AND resource_type = 'mail'
		  AND action = 'create'
		  AND status_code >= 400
		  AND created_at > NOW() - INTERVAL '30 days'`

	_ = r.pool.QueryRow(ctx, rejectedQuery, tenantID).Scan(&stats.Rejected)

	return &stats, nil
}

// StatsByTenant retourne les statistiques de mails regroupées par tenant (30 derniers jours).
func (r *MailRepository) StatsByTenant(ctx context.Context) ([]domain.TenantMailStats, error) {
	query := `
		SELECT t.id, t.name,
			COALESCE(SUM(CASE WHEN m.status = 'sent' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN m.status IN ('pending', 'queued', 'sending') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN m.status = 'failed' THEN 1 ELSE 0 END), 0),
			COUNT(m.id)
		FROM tenants t
		LEFT JOIN mails m ON m.tenant_id = t.id AND m.created_at > NOW() - INTERVAL '30 days'
		WHERE t.is_active = true
		GROUP BY t.id, t.name
		ORDER BY COUNT(m.id) DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erreur de récupération des stats par tenant : %w", err)
	}
	defer rows.Close()

	var result []domain.TenantMailStats
	for rows.Next() {
		var s domain.TenantMailStats
		if err := rows.Scan(&s.TenantID, &s.TenantName, &s.Sent, &s.Pending, &s.Failed, &s.Total); err != nil {
			return nil, fmt.Errorf("erreur de scan stats par tenant : %w", err)
		}
		result = append(result, s)
	}
	return result, nil
}

// insertRecipients insère des destinataires par chunks via le querier fourni.
func insertRecipients(ctx context.Context, q execQuerier, recipients []domain.MailRecipient) error {
	const query = `INSERT INTO mail_recipients (id, mail_id, type, email, name) VALUES ($1, $2, $3, $4, $5)`

	for start := 0; start < len(recipients); start += batchChunkSize {
		end := min(start+batchChunkSize, len(recipients))
		chunk := recipients[start:end]

		batch := &pgx.Batch{}
		for _, rec := range chunk {
			batch.Queue(query, rec.ID, rec.MailID, rec.Type, rec.Email, rec.Name)
		}

		br := q.SendBatch(ctx, batch)
		for range chunk {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("erreur de création des destinataires : %w", err)
			}
		}
		_ = br.Close()
	}

	return nil
}

// CreateRecipients insère les destinataires d'un mail par batch.
func (r *MailRepository) CreateRecipients(
	ctx context.Context,
	recipients []domain.MailRecipient,
) error {
	if len(recipients) == 0 {
		return nil
	}
	return insertRecipients(ctx, r.pool, recipients)
}

// GetRecipients retourne les destinataires d'un mail.
func (r *MailRepository) GetRecipients(
	ctx context.Context,
	mailID uuid.UUID,
) ([]domain.MailRecipient, error) {
	query := `SELECT id, mail_id, type, email, name FROM mail_recipients WHERE mail_id = $1 ORDER BY type, email`

	rows, err := r.pool.Query(ctx, query, mailID)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des destinataires : %w", err)
	}
	defer rows.Close()

	var recipients []domain.MailRecipient
	for rows.Next() {
		var rec domain.MailRecipient
		if err := rows.Scan(&rec.ID, &rec.MailID, &rec.Type, &rec.Email, &rec.Name); err != nil {
			return nil, fmt.Errorf("erreur de lecture du destinataire : %w", err)
		}
		recipients = append(recipients, rec)
	}
	return recipients, nil
}

// GetRecipientsByMailIDs retourne les destinataires de plusieurs mails en une seule requête,
// regroupés par mail_id. Évite le problème N+1 lors du listage paginé.
func (r *MailRepository) GetRecipientsByMailIDs(
	ctx context.Context,
	mailIDs []uuid.UUID,
) (map[uuid.UUID][]domain.MailRecipient, error) {
	result := make(map[uuid.UUID][]domain.MailRecipient, len(mailIDs))
	if len(mailIDs) == 0 {
		return result, nil
	}

	query := `SELECT id, mail_id, type, email, name FROM mail_recipients
		WHERE mail_id = ANY($1) ORDER BY mail_id, type, email`

	rows, err := r.pool.Query(ctx, query, mailIDs)
	if err != nil {
		return nil, fmt.Errorf("erreur de listage des destinataires par lot : %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rec domain.MailRecipient
		if err := rows.Scan(&rec.ID, &rec.MailID, &rec.Type, &rec.Email, &rec.Name); err != nil {
			return nil, fmt.Errorf("erreur de lecture du destinataire : %w", err)
		}
		result[rec.MailID] = append(result[rec.MailID], rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur de parcours des destinataires : %w", err)
	}
	return result, nil
}

// scannable est une interface commune à pgx.Row et pgx.Rows pour le scan de mails.
type scannable interface {
	Scan(dest ...any) error
}

// scanMail scanne une ligne de résultat en Mail.
func (r *MailRepository) scanMail(s scannable) (*domain.Mail, error) {
	var m domain.Mail
	var templateDataJSON, metadataJSON []byte

	err := s.Scan(
		&m.ID, &m.TenantID, &m.SMTPConfigID, &m.TemplateID,
		&m.FromEmail, &m.FromName, &m.Subject, &m.TextBody, &m.HTMLBody,
		&templateDataJSON, &m.Status, &m.StatusMessage,
		&m.Attempts, &m.ScheduledAt, &m.SentAt, &m.TaskID, &m.Priority, &metadataJSON,
		&m.SpamScore, &m.Tags, &m.CompressedBody,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMailNotFound
		}
		return nil, fmt.Errorf("erreur de lecture du mail : %w", err)
	}

	if len(templateDataJSON) > 0 {
		if err := json.Unmarshal(templateDataJSON, &m.TemplateData); err != nil {
			return nil, fmt.Errorf("erreur de désérialisation des données de template : %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, fmt.Errorf("erreur de désérialisation des métadonnées : %w", err)
		}
	}

	// Restaurer les corps depuis la version compressée si les champs texte sont vides.
	if m.TextBody == "" && m.HTMLBody == "" && len(m.CompressedBody) > 0 {
		textBody, htmlBody, err := compress.DecompressBody(m.CompressedBody)
		if err != nil {
			return nil, fmt.Errorf("erreur de décompression du corps : %w", err)
		}
		m.TextBody = textBody
		m.HTMLBody = htmlBody
	}

	return &m, nil
}

// AddTags ajoute des tags à un mail sans duplication.
func (r *MailRepository) AddTags(ctx context.Context, id uuid.UUID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	query := `UPDATE mails SET tags = (SELECT array_agg(DISTINCT t) FROM unnest(tags || $2::text[]) t), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, tags)
	if err != nil {
		return fmt.Errorf("erreur d'ajout des tags : %w", err)
	}
	return nil
}

// ClearBodies vide les corps texte/HTML d'un mail et stocke éventuellement la version compressée.
func (r *MailRepository) ClearBodies(ctx context.Context, id uuid.UUID, compressedBody []byte) error {
	query := `UPDATE mails SET text_body = '', html_body = '', compressed_body = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, compressedBody)
	if err != nil {
		return fmt.Errorf("erreur de purge du corps du mail : %w", err)
	}
	return nil
}
