package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/templates"
)

// MailService implémente port.MailService.
type MailService struct {
	mailRepo        port.MailRepository
	smtpRepo        port.SMTPConfigRepository
	templateRepo    port.TemplateRepository
	tenantRepo      port.TenantRepository
	queue           port.QueueClient
	analysisService port.AnalysisService
	attachments     *AttachmentService
}

// NewMailService crée un nouveau service mail.
func NewMailService(
	mailRepo port.MailRepository,
	smtpRepo port.SMTPConfigRepository,
	templateRepo port.TemplateRepository,
	tenantRepo port.TenantRepository,
	queue port.QueueClient,
	analysisService port.AnalysisService,
	attachments *AttachmentService,
) *MailService {
	return &MailService{
		mailRepo:        mailRepo,
		smtpRepo:        smtpRepo,
		templateRepo:    templateRepo,
		tenantRepo:      tenantRepo,
		queue:           queue,
		analysisService: analysisService,
		attachments:     attachments,
	}
}

// storeAttachments déduplique et stocke les pièces jointes d'une requête (une
// seule fois, même si la campagne vise N destinataires), et retourne les liens
// mail->pièce jointe à insérer pour chaque mail. Les liens ne portent pas le
// contenu : seul le BlobStore le détient.
func (s *MailService) storeAttachments(
	ctx context.Context,
	tenantID uuid.UUID,
	atts []domain.Attachment,
) ([]domain.AttachmentLink, error) {
	if len(atts) == 0 {
		return nil, nil
	}
	links := make([]domain.AttachmentLink, 0, len(atts))
	for i, att := range atts {
		content, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			return nil, fmt.Errorf("%w : pièce jointe %q : contenu base64 invalide", domain.ErrValidation, att.Filename)
		}
		id, err := s.attachments.Store(ctx, tenantID, content, att.ContentType)
		if err != nil {
			return nil, err
		}
		links = append(links, domain.AttachmentLink{
			AttachmentID: id,
			Filename:     att.Filename,
			Position:     i,
		})
	}
	return links, nil
}

// fusionnerTemplateData fusionne les données partagées avec les données spécifiques au destinataire.
// Les données spécifiques écrasent les données partagées en cas de conflit.
func fusionnerTemplateData(shared, perRecipient map[string]string) map[string]string {
	merged := make(map[string]string, len(shared)+len(perRecipient))
	maps.Copy(merged, shared)
	maps.Copy(merged, perRecipient)
	return merged
}

// creerMailParams contient les paramètres pour créer un mail unitaire.
type creerMailParams struct {
	data      map[string]string
	req       *domain.CreateMailRequest
	tenant    *domain.Tenant
	tmpl      *domain.Template
	priority  domain.MailPriority
	smtpCfgID uuid.UUID
	tenantID  uuid.UUID
	fromEmail string
	fromName  string
	toAddrs   []domain.EmailAddress
}

// creerMail crée un mail unitaire : rendu template, sauvegarde, recipients, enqueue.
func (s *MailService) creerMail(ctx context.Context, p creerMailParams) (*domain.Mail, error) {
	now := time.Now()
	mailID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	// Calculer le spam score
	spamScore := s.analysisService.ComputeSpamScore(p.req.Subject, p.req.TextBody, p.req.HTMLBody)

	// Vérifier le seuil spam du tenant
	if p.tenant != nil && p.tenant.Settings.SpamScoreThreshold != nil &&
		spamScore > *p.tenant.Settings.SpamScoreThreshold {
		if p.tenant.Settings.SpamScoreAction != nil &&
			*p.tenant.Settings.SpamScoreAction == domain.SpamScoreActionBlock {
			return nil, fmt.Errorf(
				"%w (score: %.1f, seuil: %.1f)",
				domain.ErrSpamBlocked,
				spamScore,
				*p.tenant.Settings.SpamScoreThreshold,
			)
		}
		slog.Warn("score spam au-dessus du seuil du tenant",
			"tenant_id", p.tenantID,
			"spam_score", spamScore,
			"threshold", *p.tenant.Settings.SpamScoreThreshold)
	}

	// Construire les tags automatiques + custom
	tags := buildAutoTags(p.priority, p.tmpl, p.req.Tags)

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     p.tenantID,
		SMTPConfigID: &p.smtpCfgID,
		TemplateID:   p.req.TemplateID,
		FromEmail:    p.fromEmail,
		FromName:     p.fromName,
		Subject:      p.req.Subject,
		TextBody:     p.req.TextBody,
		HTMLBody:     p.req.HTMLBody,
		TemplateData: p.data,
		Status:       domain.MailStatusPending,
		Priority:     p.priority,
		Metadata:     p.req.Metadata,
		SpamScore:    &spamScore,
		Tags:         tags,
		ScheduledAt:  p.req.ScheduledAt.TimePtr(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Stocker les pièces jointes (dédupliquées) hors transaction, puis lier.
	links, err := s.storeAttachments(ctx, p.tenantID, p.req.Attachments)
	if err != nil {
		return nil, err
	}

	// Le rendu template est différé au worker — on stocke juste les template_data fusionnées.
	// La validation des données a déjà eu lieu côté API (fail fast).

	// Construire les destinataires
	var recipients []domain.MailRecipient
	for _, addr := range p.toAddrs {
		rid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, domain.MailRecipient{
			ID:     rid,
			MailID: mail.ID,
			Type:   domain.RecipientTo,
			Email:  addr.Email,
			Name:   addr.Name,
		})
	}
	for _, addr := range p.req.CC {
		rid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, domain.MailRecipient{
			ID:     rid,
			MailID: mail.ID,
			Type:   domain.RecipientCC,
			Email:  addr.Email,
			Name:   addr.Name,
		})
	}
	for _, addr := range p.req.BCC {
		rid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, domain.MailRecipient{
			ID:     rid,
			MailID: mail.ID,
			Type:   domain.RecipientBCC,
			Email:  addr.Email,
			Name:   addr.Name,
		})
	}

	// Persister le mail, ses destinataires et ses pièces jointes de façon atomique.
	if err := s.mailRepo.CreateWithRecipients(ctx, mail, recipients, links); err != nil {
		return nil, err
	}
	mail.Recipients = recipients

	// Mettre en file d'attente
	taskID, err := s.queue.EnqueueMailSend(ctx, mail.ID, p.tenantID, p.priority, mail.ScheduledAt)
	if err != nil {
		return nil, err
	}

	if err := s.mailRepo.SetQueued(ctx, mail.ID, taskID); err != nil {
		return nil, err
	}

	mail.TaskID = taskID
	mail.Status = domain.MailStatusQueued

	return mail, nil
}

// Create crée un ou plusieurs mails selon le mode (groupé ou individuel) et les met en file d'attente.
func (s *MailService) Create(
	ctx context.Context,
	tenantID uuid.UUID,
	req domain.CreateMailRequest,
) ([]*domain.Mail, error) {
	if len(req.To) == 0 {
		return nil, domain.ErrValidation
	}

	// Résoudre la priorité : requête > tenant > default
	priority := domain.MailPriorityDefault
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Settings.DefaultPriority != "" {
		priority = tenant.Settings.DefaultPriority
	}
	if req.Priority != nil {
		if !domain.ValidMailPriorities[*req.Priority] {
			return nil, fmt.Errorf(
				"%w : priorité invalide : %s (valeurs : critical, default, low)",
				domain.ErrValidation,
				*req.Priority,
			)
		}
		priority = *req.Priority
	}

	// Enforcement MaxDestinataires
	totalDest := len(req.To) + len(req.CC) + len(req.BCC)
	if totalDest > tenant.Settings.MaxDestinataires {
		return nil, fmt.Errorf("%w : %d destinataires fournis, limite %d",
			domain.ErrValidation, totalDest, tenant.Settings.MaxDestinataires)
	}

	// Résoudre la config SMTP
	var smtpCfg *domain.SMTPConfig
	if req.SMTPConfigID != nil {
		smtpCfg, err = s.smtpRepo.GetByID(ctx, tenantID, *req.SMTPConfigID)
	} else {
		smtpCfg, err = s.smtpRepo.GetDefault(ctx, tenantID)
		if err != nil {
			err = domain.ErrSMTPConfigNotSet
		}
	}
	if err != nil {
		return nil, err
	}

	// Résoudre le from
	fromEmail := smtpCfg.FromEmail
	fromName := smtpCfg.FromName
	if req.From != nil {
		fromEmail = req.From.Email
		fromName = req.From.Name
	}

	// Résoudre le template si fourni
	var tmpl *domain.Template
	if req.TemplateID != nil {
		tmpl, err = s.templateRepo.GetByID(ctx, tenantID, *req.TemplateID)
		if err != nil {
			return nil, err
		}
	}

	// Branchement individuel / groupé
	if req.Individuel {
		return s.createIndividuel(
			ctx,
			tenantID,
			req,
			tmpl,
			smtpCfg.ID,
			fromEmail,
			fromName,
			priority,
		)
	}

	// Mode groupé (défaut) : validation template sur les données partagées
	if tmpl != nil {
		data := req.TemplateData
		if data == nil {
			data = make(map[string]string)
		}
		if err := templates.ValidateData(tmpl.Variables, data); err != nil {
			return nil, fmt.Errorf("%w : %s", domain.ErrValidation, err.Error())
		}
	}

	mail, err := s.creerMail(ctx, creerMailParams{
		tenantID:  tenantID,
		smtpCfgID: smtpCfg.ID,
		fromEmail: fromEmail,
		fromName:  fromName,
		priority:  priority,
		req:       &req,
		tmpl:      tmpl,
		data:      req.TemplateData,
		toAddrs:   req.To,
		tenant:    tenant,
	})
	if err != nil {
		return nil, err
	}

	return []*domain.Mail{mail}, nil
}

// createIndividuel crée N mails distincts (1 par destinataire TO) via batch inserts.
func (s *MailService) createIndividuel(
	ctx context.Context,
	tenantID uuid.UUID,
	req domain.CreateMailRequest,
	tmpl *domain.Template,
	smtpCfgID uuid.UUID,
	fromEmail, fromName string,
	priority domain.MailPriority,
) ([]*domain.Mail, error) {
	now := time.Now()
	mails := make([]*domain.Mail, 0, len(req.To))
	var allRecipients []domain.MailRecipient

	// Calculer le spam score une seule fois (contenu identique pour chaque destinataire)
	spamScore := s.analysisService.ComputeSpamScore(req.Subject, req.TextBody, req.HTMLBody)

	// Construire les tags automatiques + custom
	tags := buildAutoTags(priority, tmpl, req.Tags)

	// Stocker les pièces jointes une seule fois pour toute la campagne (dédup) :
	// les N mails partageront les mêmes liens.
	links, err := s.storeAttachments(ctx, tenantID, req.Attachments)
	if err != nil {
		return nil, err
	}

	// Phase 1 : validation et construction des objets en mémoire (légers, pas de rendu HTML)
	for i, addr := range req.To {
		data := fusionnerTemplateData(req.TemplateData, addr.TemplateData)

		// Validation template par destinataire (fail fast)
		if tmpl != nil {
			if err := templates.ValidateData(tmpl.Variables, data); err != nil {
				return nil, fmt.Errorf("%w : destinataire %d (%s) : %s",
					domain.ErrValidation, i+1, addr.Email, err.Error())
			}
		}

		mailID, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}

		mail := &domain.Mail{
			ID:           mailID,
			TenantID:     tenantID,
			SMTPConfigID: &smtpCfgID,
			TemplateID:   req.TemplateID,
			FromEmail:    fromEmail,
			FromName:     fromName,
			Subject:      req.Subject,
			TextBody:     req.TextBody,
			HTMLBody:     req.HTMLBody,
			TemplateData: data,
			Status:       domain.MailStatusPending,
			Priority:     priority,
			Metadata:     req.Metadata,
			SpamScore:    &spamScore,
			Tags:         tags,
			ScheduledAt:  req.ScheduledAt.TimePtr(),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		mails = append(mails, mail)

		// Destinataire TO
		rid, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		allRecipients = append(allRecipients, domain.MailRecipient{
			ID:     rid,
			MailID: mailID,
			Type:   domain.RecipientTo,
			Email:  addr.Email,
			Name:   addr.Name,
		})
	}

	// Phase 2 : insertion atomique des mails, destinataires et pièces jointes
	if err := s.mailRepo.CreateBatchWithRecipients(ctx, mails, allRecipients, links); err != nil {
		return nil, err
	}

	// Phase 3 : enqueue en parallèle (réduit la latence de N RTTs Redis séquentiels)
	var (
		mu          sync.Mutex
		enqueueErr  error
		mailTaskIDs = make(map[uuid.UUID]string, len(mails))
		wg          sync.WaitGroup
		semaphore   = make(chan struct{}, 50) // Limiter la concurrence à 50 goroutines
	)

	for _, mail := range mails {
		wg.Go(func() {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			mu.Lock()
			if enqueueErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			taskID, err := s.queue.EnqueueMailSend(
				ctx,
				mail.ID,
				tenantID,
				priority,
				mail.ScheduledAt,
			)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if enqueueErr == nil {
					enqueueErr = err
				}
				return
			}
			mailTaskIDs[mail.ID] = taskID
		})
	}
	wg.Wait()

	if enqueueErr != nil {
		return nil, enqueueErr
	}

	// Phase 4 : batch update task_id + status en une seule passe
	if err := s.mailRepo.SetQueuedBatch(ctx, mailTaskIDs); err != nil {
		return nil, err
	}

	// Mettre à jour les objets en mémoire pour la réponse
	for _, mail := range mails {
		mail.TaskID = mailTaskIDs[mail.ID]
		mail.Status = domain.MailStatusQueued
	}

	return mails, nil
}

// GetByID retourne un mail par son identifiant avec ses destinataires.
func (s *MailService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Mail, error) {
	mail, err := s.mailRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	recipients, err := s.mailRepo.GetRecipients(ctx, mail.ID)
	if err != nil {
		return nil, err
	}
	mail.Recipients = recipients

	// Métadonnées des pièces jointes (sans le contenu) pour l'affichage UI.
	refs, err := s.mailRepo.GetAttachmentRefs(ctx, mail.ID)
	if err != nil {
		return nil, err
	}
	mail.AttachmentRefs = refs

	return mail, nil
}

// DownloadAttachment retourne les métadonnées et le contenu d'une pièce jointe
// d'un mail. Vérifie que le mail appartient au tenant et que la pièce jointe lui
// est bien liée (empêche le téléchargement d'un blob d'un autre mail/tenant).
func (s *MailService) DownloadAttachment(
	ctx context.Context,
	tenantID, mailID, attachmentID uuid.UUID,
) (*domain.AttachmentRef, []byte, error) {
	// Appartenance au tenant.
	if _, err := s.mailRepo.GetByID(ctx, tenantID, mailID); err != nil {
		return nil, nil, err
	}

	refs, err := s.mailRepo.GetAttachmentRefs(ctx, mailID)
	if err != nil {
		return nil, nil, err
	}
	var ref *domain.AttachmentRef
	for i := range refs {
		if refs[i].AttachmentID == attachmentID {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		return nil, nil, domain.ErrAttachmentNotFound
	}

	content, err := s.attachments.Load(ctx, tenantID, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	return ref, content, nil
}

// List retourne la liste paginée des mails d'un tenant avec filtres.
func (s *MailService) List(
	ctx context.Context,
	tenantID uuid.UUID,
	filter domain.MailListFilter,
) (*domain.PaginatedList[domain.Mail], error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	result, err := s.mailRepo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return result, nil
	}

	// Charger les destinataires de tous les mails en une seule requête (évite le N+1).
	mailIDs := make([]uuid.UUID, len(result.Items))
	for i := range result.Items {
		mailIDs[i] = result.Items[i].ID
	}
	byMailID, err := s.mailRepo.GetRecipientsByMailIDs(ctx, mailIDs)
	if err != nil {
		return nil, err
	}
	for i := range result.Items {
		result.Items[i].Recipients = byMailID[result.Items[i].ID]
	}
	return result, nil
}

// Cancel annule un mail en attente ou en file d'attente.
func (s *MailService) Cancel(ctx context.Context, tenantID, id uuid.UUID) error {
	mail, err := s.mailRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if mail.Status != domain.MailStatusPending && mail.Status != domain.MailStatusQueued {
		return domain.ErrMailNotPending
	}

	if mail.TaskID != "" {
		if err := s.queue.DeleteTask(string(mail.Priority), mail.TaskID); err != nil {
			slog.Error("échec de suppression de la tâche Asynq",
				"task_id", mail.TaskID, "queue", mail.Priority, "err", err)
		}
	}

	return s.mailRepo.UpdateStatus(ctx, id, domain.MailStatusCancelled, "annulé par l'utilisateur")
}

// Retry relance l'envoi d'un mail en échec en le remettant en file d'attente.
func (s *MailService) Retry(ctx context.Context, tenantID, id uuid.UUID) error {
	mail, err := s.mailRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if mail.Status != domain.MailStatusFailed {
		return domain.ErrMailNotFailed
	}

	// Remettre en file d'attente avec la priorité d'origine (pas de scheduled_at lors d'un retry)
	taskID, err := s.queue.EnqueueMailSend(ctx, mail.ID, tenantID, mail.Priority, nil)
	if err != nil {
		return err
	}

	if err := s.mailRepo.UpdateTaskID(ctx, id, taskID); err != nil {
		return err
	}
	return s.mailRepo.UpdateStatus(ctx, id, domain.MailStatusQueued, "relance manuelle")
}

// Stats retourne les statistiques d'envoi de mails d'un tenant.
func (s *MailService) Stats(ctx context.Context, tenantID uuid.UUID) (*domain.MailStats, error) {
	return s.mailRepo.Stats(ctx, tenantID)
}

// StatsByTenant retourne les statistiques de mails regroupées par tenant.
func (s *MailService) StatsByTenant(ctx context.Context) ([]domain.TenantMailStats, error) {
	return s.mailRepo.StatsByTenant(ctx)
}

// buildAutoTags construit la liste de tags (automatiques + custom).
func buildAutoTags(priority domain.MailPriority, tmpl *domain.Template, customTags []string) []string {
	var tags []string

	// Tag de priorité (sauf "default" pour éviter le bruit)
	if priority != domain.MailPriorityDefault {
		tags = append(tags, "priority:"+string(priority))
	}

	// Tag de template
	if tmpl != nil {
		tags = append(tags, "template:"+tmpl.Slug)
	}

	// Tags custom de la requête
	tags = append(tags, customTags...)

	if tags == nil {
		tags = []string{}
	}
	return tags
}
