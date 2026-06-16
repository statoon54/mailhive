package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/statoon54/mailhive/internal/compress"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/templates"
)

// sendTimeout est le délai maximal pour l'envoi d'un mail via SMTP.
const sendTimeout = 30 * time.Second

// defaultLang est la langue par défaut utilisée avant le chargement du tenant.
const defaultLang = i18n.FR

// MailHandler traite les tâches d'envoi de mail.
type MailHandler struct {
	mailRepo     port.MailRepository
	smtpRepo     port.SMTPConfigRepository
	tenantRepo   port.TenantRepository
	templateRepo port.TemplateRepository
	sender       port.MailSender
	smtpService  *service.SMTPConfigService
	rateLimiter  port.RateLimiter
	attachments  *service.AttachmentService
	tmplCache    *ttlCache[uuid.UUID, *templates.Compiled] // templates pré-compilés
	tenantCache  *ttlCache[uuid.UUID, *domain.Tenant]      // settings tenant
	smtpCache    *ttlCache[smtpCacheKey, *domain.SMTPConfig]
	cbRegistry   *CircuitBreakerRegistry
}

// smtpCacheKey est la clé du cache SMTP : {tenantID, configID}.
// Un tableau (et non un struct) évite un faux positif du linter "unused"
// sur des champs nommés lus uniquement via la comparaison de clé de map.
type smtpCacheKey = [2]uuid.UUID

// NewMailHandler crée un nouveau handler de tâches mail.
func NewMailHandler(
	mailRepo port.MailRepository,
	smtpRepo port.SMTPConfigRepository,
	tenantRepo port.TenantRepository,
	templateRepo port.TemplateRepository,
	sender port.MailSender,
	smtpService *service.SMTPConfigService,
	rateLimiter port.RateLimiter,
	attachments *service.AttachmentService,
	cbRegistry *CircuitBreakerRegistry,
) *MailHandler {
	return &MailHandler{
		mailRepo:     mailRepo,
		smtpRepo:     smtpRepo,
		tenantRepo:   tenantRepo,
		templateRepo: templateRepo,
		sender:       sender,
		smtpService:  smtpService,
		rateLimiter:  rateLimiter,
		attachments:  attachments,
		tmplCache:    newTTLCache[uuid.UUID, *templates.Compiled](defaultTemplateCacheTTL),
		tenantCache:  newTTLCache[uuid.UUID, *domain.Tenant](defaultConfigCacheTTL),
		smtpCache:    newTTLCache[smtpCacheKey, *domain.SMTPConfig](defaultConfigCacheTTL),
		cbRegistry:   cbRegistry,
	}
}

// HandleMailSend traite une tâche d'envoi de mail.
func (h *MailHandler) HandleMailSend(ctx context.Context, task *asynq.Task) error {
	var payload MailSendPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf(i18n.T(defaultLang, "worker.err.deserialize"), err)
	}

	slog.Info("traitement du mail", "mail_id", payload.MailID, "tenant_id", payload.TenantID)

	// Charger le mail
	mail, err := h.mailRepo.GetByID(ctx, payload.TenantID, payload.MailID)
	if err != nil {
		return fmt.Errorf(i18n.T(defaultLang, "worker.err.load_mail"), err)
	}

	// Vérifier que le mail est toujours envoyable
	if mail.Status == domain.MailStatusCancelled {
		slog.Info("mail annulé, ignoré", "mail_id", payload.MailID)
		return nil
	}

	// Charger les destinataires
	recipients, err := h.mailRepo.GetRecipients(ctx, mail.ID)
	if err != nil {
		return fmt.Errorf(i18n.T(defaultLang, "worker.err.load_recipients"), err)
	}
	mail.Recipients = recipients

	// Charger les pièces jointes dédupliquées et reconstituer mail.Attachments
	// (le chemin legacy — attachments inline en JSONB — reste pris en charge).
	if err := h.loadAttachments(ctx, payload.TenantID, mail); err != nil {
		return fmt.Errorf(i18n.T(defaultLang, "worker.err.load_attachments"), err)
	}

	// Rendu template si nécessaire : chaque champ vide est rendu indépendamment
	// (permet de fournir un subject dans la requête tout en utilisant le corps du template)
	if mail.TemplateID != nil && (mail.Subject == "" || mail.TextBody == "" || mail.HTMLBody == "") {
		compiled, err := h.getCompiledTemplate(ctx, payload.TenantID, *mail.TemplateID)
		if err != nil {
			h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.template_invalid"))
			return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
		}

		data := mail.TemplateData
		if data == nil {
			data = make(map[string]string)
		}

		if mail.Subject == "" {
			subject, err := compiled.RenderSubject(data)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.subject_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.Subject = subject
		}

		if mail.TextBody == "" {
			textBody, err := compiled.RenderText(data)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.text_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.TextBody = textBody
		}

		if mail.HTMLBody == "" {
			htmlBody, err := compiled.RenderHTML(data)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.html_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.HTMLBody = htmlBody
		}
	}

	// Rendre subject/text/html comme templates Go si template_data est fourni (sans template_id)
	if len(mail.TemplateData) > 0 {
		if mail.Subject != "" {
			rendered, err := templates.RenderSubject(mail.Subject, mail.TemplateData)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.subject_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.Subject = rendered
		}
		if mail.TextBody != "" {
			rendered, err := templates.RenderText(mail.TextBody, mail.TemplateData)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.text_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.TextBody = rendered
		}
		if mail.HTMLBody != "" {
			rendered, err := templates.RenderHTML(mail.HTMLBody, mail.TemplateData)
			if err != nil {
				h.failMail(ctx, mail.ID, i18n.T(defaultLang, "worker.fail.html_render"))
				return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
			}
			mail.HTMLBody = rendered
		}
	}

	// Vérifier le rate limiter du tenant (via Redis)
	tenant, err := h.getTenant(ctx, payload.TenantID)
	if err != nil {
		return fmt.Errorf(i18n.T(defaultLang, "worker.err.load_tenant"), err)
	}
	tLang := tenant.Settings.Lang()

	allowed, err := h.rateLimiter.Allow(
		ctx,
		payload.TenantID,
		tenant.Settings.RateLimit,
		tenant.Settings.RateBurst,
	)
	if err != nil {
		return fmt.Errorf(i18n.T(tLang, "worker.err.rate_limiter"), err)
	}
	if !allowed {
		slog.Warn("rate limit atteint", "tenant_id", payload.TenantID)
		return domain.ErrRateLimited
	}

	// Charger la config SMTP
	if mail.SMTPConfigID == nil {
		h.failMail(ctx, mail.ID, i18n.T(tLang, "worker.fail.no_smtp"))
		return fmt.Errorf("%w : "+i18n.T(tLang, "worker.err.no_smtp"), asynq.SkipRetry, mail.ID)
	}

	smtpConfigID := *mail.SMTPConfigID

	// Vérifier le circuit breaker
	if !h.cbRegistry.Allow(smtpConfigID) {
		slog.Warn("circuit breaker ouvert", "smtp_config_id", smtpConfigID)
		return domain.ErrCircuitOpen
	}

	smtpCfg, err := h.getSMTPConfig(ctx, payload.TenantID, smtpConfigID)
	if err != nil {
		h.failMail(ctx, mail.ID, i18n.T(tLang, "worker.fail.smtp_not_found"))
		return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
	}

	// Déchiffrer le mot de passe
	if smtpCfg.Password != "" {
		decrypted, err := h.smtpService.DecryptPassword(smtpCfg.Password)
		if err != nil {
			h.failMail(ctx, mail.ID, i18n.T(tLang, "worker.fail.smtp_decrypt"))
			return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
		}
		smtpCfg.Password = decrypted
	}

	// Passer en "sending" et incrémenter les tentatives en une seule requête.
	if err := h.mailRepo.MarkSending(ctx, mail.ID); err != nil {
		return err
	}

	// Envoyer le mail avec timeout
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	if err := h.sender.Send(sendCtx, smtpCfg, mail); err != nil {
		h.cbRegistry.RecordFailure(smtpConfigID)

		// Erreur permanente → fail immédiat sans retry
		var permErr *domain.SMTPPermanentError
		if errors.As(err, &permErr) {
			h.failMail(ctx, mail.ID, err.Error())
			return fmt.Errorf("%w : %w", asynq.SkipRetry, err)
		}

		// Erreur temporaire → vérifier si c'est le dernier retry
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)
		if retried > 0 {
			_ = h.mailRepo.AddTags(ctx, mail.ID, []string{fmt.Sprintf("retry-%d", retried)})
		}
		if retried >= maxRetry-1 {
			h.failMail(ctx, mail.ID, err.Error())
		} else {
			// Remettre en pending pour le prochain retry
			_ = h.mailRepo.UpdateStatus(ctx, mail.ID, domain.MailStatusPending, "")
		}

		return fmt.Errorf(i18n.T(tLang, "worker.err.smtp_send"), err)
	}

	// Succès → reset circuit breaker
	h.cbRegistry.RecordSuccess(smtpConfigID)

	// Passer en "sent" et enregistrer sent_at en une seule requête.
	if err := h.mailRepo.MarkSent(ctx, mail.ID); err != nil {
		return err
	}

	// Purger ou compresser les corps selon le paramètre store_body du tenant.
	var compressedBody []byte
	if tenant.Settings.StoreBody {
		compressed, err := compress.CompressBody(mail.TextBody, mail.HTMLBody)
		if err != nil {
			slog.Error("échec de compression du corps", "mail_id", mail.ID, "err", err)
		} else {
			compressedBody = compressed
		}
	}
	if err := h.mailRepo.ClearBodies(ctx, mail.ID, compressedBody); err != nil {
		slog.Error("échec de purge du corps", "mail_id", mail.ID, "err", err)
	}

	slog.Info("mail envoyé", "mail_id", mail.ID)
	return nil
}

// loadAttachments reconstitue mail.Attachments à partir des pièces jointes
// dédupliquées (table mail_attachments + BlobStore), juste avant l'envoi SMTP.
func (h *MailHandler) loadAttachments(ctx context.Context, tenantID uuid.UUID, mail *domain.Mail) error {
	refs, err := h.mailRepo.GetAttachmentRefs(ctx, mail.ID)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return nil
	}
	atts := make([]domain.Attachment, 0, len(refs))
	for _, ref := range refs {
		content, err := h.attachments.Load(ctx, tenantID, ref.AttachmentID)
		if err != nil {
			return err
		}
		atts = append(atts, domain.Attachment{
			Filename:    ref.Filename,
			Content:     base64.StdEncoding.EncodeToString(content),
			ContentType: ref.ContentType,
		})
	}
	mail.Attachments = atts
	return nil
}

// failMail met à jour le statut d'un mail en échec et ajoute le tag "bounced".
// Le message d'échec (déjà traduit par l'appelant) est stocké en base et affiché dans l'UI.
func (h *MailHandler) failMail(ctx context.Context, mailID uuid.UUID, message string) {
	if err := h.mailRepo.UpdateStatus(ctx, mailID, domain.MailStatusFailed, message); err != nil {
		slog.Error("échec de mise à jour du statut en échec", "mail_id", mailID, "err", err)
	}
	if err := h.mailRepo.AddTags(ctx, mailID, []string{"bounced"}); err != nil {
		slog.Error("échec d'ajout du tag bounced", "mail_id", mailID, "err", err)
	}
}

// getTenant retourne le tenant depuis le cache (TTL court) ou le charge.
// Le tenant est utilisé en lecture seule par le worker : le pointeur cached est partagé.
func (h *MailHandler) getTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error) {
	if t, ok := h.tenantCache.get(tenantID); ok {
		return t, nil
	}
	t, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	h.tenantCache.set(tenantID, t)
	return t, nil
}

// getSMTPConfig retourne une COPIE de la config SMTP depuis le cache (TTL court) ou la charge.
// On renvoie toujours une copie car l'appelant déchiffre le mot de passe en place :
// muter l'objet partagé corromprait le cache (déchiffrement répété).
func (h *MailHandler) getSMTPConfig(ctx context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error) {
	key := smtpCacheKey{tenantID, id}
	if c, ok := h.smtpCache.get(key); ok {
		cp := *c
		return &cp, nil
	}
	c, err := h.smtpRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	h.smtpCache.set(key, c)
	cp := *c
	return &cp, nil
}

// getCompiledTemplate retourne un template pré-compilé depuis le cache, ou le charge et le compile.
func (h *MailHandler) getCompiledTemplate(
	ctx context.Context,
	tenantID, templateID uuid.UUID,
) (*templates.Compiled, error) {
	if cached, ok := h.tmplCache.get(templateID); ok {
		return cached, nil
	}

	tmpl, err := h.templateRepo.GetByID(ctx, tenantID, templateID)
	if err != nil {
		return nil, err
	}

	compiled, err := templates.Compile(tmpl.SubjectTmpl, tmpl.TextBody, tmpl.HTMLBody)
	if err != nil {
		return nil, err
	}

	h.tmplCache.set(templateID, compiled)
	return compiled, nil
}
