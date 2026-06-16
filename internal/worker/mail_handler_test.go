package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

// testEncryptionKey est une clé AES-256 valide (64 hex chars = 32 bytes).
const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestMailHandler(
	mailRepo *mocks.MockMailRepo,
	smtpRepo *mocks.MockSMTPConfigRepo,
	tenantRepo *mocks.MockTenantRepo,
	tmplRepo *mocks.MockTemplateRepo,
	sender *mocks.MockMailSender,
	rateLimiter *mocks.MockRateLimiter,
) *MailHandler {
	smtpService, _ := service.NewSMTPConfigService(smtpRepo, sender, testEncryptionKey)
	cbRegistry := NewCircuitBreakerRegistry(CircuitBreakerConfig{})
	attachmentSvc := service.NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	return NewMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, smtpService, rateLimiter, attachmentSvc, cbRegistry)
}

func makeTask(t *testing.T, mailID, tenantID uuid.UUID) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(MailSendPayload{MailID: mailID, TenantID: tenantID})
	require.NoError(t, err)
	return asynq.NewTask(TypeMailSend, payload)
}

func defaultTenant(tenantID uuid.UUID) *domain.Tenant {
	return &domain.Tenant{
		ID:   tenantID,
		Name: "Test Tenant",
		Slug: "test-tenant",
		Settings: domain.TenantSettings{
			RateLimit: 100,
			RateBurst: 100,
		},
		IsActive: true,
	}
}

func TestMailHandler_Success(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Hello",
		TextBody:     "World",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{{Email: "a@b.com"}}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID:       smtpCfgID,
		TenantID: tenantID,
	}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, domain.MailStatusSent, mail.Status)
	assert.True(t, sender.Called("Send"))

	// Les transitions sending/sent sont fusionnées : une seule requête chacune.
	assert.Equal(t, 1, mailRepo.CallCount("MarkSending"))
	assert.Equal(t, 1, mailRepo.CallCount("MarkSent"))
	assert.Equal(t, 1, mail.Attempts)
}

func TestMailHandler_MailNotFound(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
	assert.False(t, sender.Called("Send"))
}

func TestMailHandler_MailCancelled(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:       mailID,
		TenantID: tenantID,
		Status:   domain.MailStatusCancelled,
	}
	mailRepo.Mails[mailID] = mail

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.NoError(t, err) // cancelled mails are silently skipped
	assert.False(t, sender.Called("Send"))
}

func TestMailHandler_RateLimited(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: false}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.ErrorIs(t, err, domain.ErrRateLimited)
	assert.False(t, sender.Called("Send"))
}

func TestMailHandler_NoSMTPConfig(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: nil, // No SMTP config
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
	assert.Equal(t, domain.MailStatusFailed, mail.Status)
}

func TestMailHandler_CircuitBreakerOpen(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	smtpService, _ := service.NewSMTPConfigService(smtpRepo, sender, testEncryptionKey)
	cbRegistry := NewCircuitBreakerRegistry(CircuitBreakerConfig{FailureThreshold: 1})
	attachmentSvc := service.NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	h := NewMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, smtpService, rateLimiter, attachmentSvc, cbRegistry)

	// Trip the circuit breaker
	cbRegistry.RecordFailure(smtpCfgID)

	task := makeTask(t, mailID, tenantID)
	err := h.HandleMailSend(context.Background(), task)
	assert.ErrorIs(t, err, domain.ErrCircuitOpen)
}

func TestMailHandler_SendPermanentError(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{
		Err: domain.NewSMTPPermanentError(fmt.Errorf("550 user not found")),
	}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID:       smtpCfgID,
		TenantID: tenantID,
	}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
	assert.Equal(t, domain.MailStatusFailed, mail.Status)
	// Should have "bounced" tag
	found := false
	for _, tag := range mail.Tags {
		if tag == "bounced" {
			found = true
		}
	}
	assert.True(t, found, "mail should have 'bounced' tag")
}

func TestMailHandler_SendTemporaryError(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{
		Err: fmt.Errorf("connection timeout"),
	}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID:       smtpCfgID,
		TenantID: tenantID,
	}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
	// In test context, GetRetryCount=0 and GetMaxRetry=0, so retried >= maxRetry-1 (0 >= -1)
	// triggers failMail. This is correct behavior: when no retries configured, fail immediately.
	assert.Equal(t, domain.MailStatusFailed, mail.Status)
}

func TestMailHandler_SMTPNotFound(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Test",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)
	// Don't add SMTP config to repo — it won't be found

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
	assert.Equal(t, domain.MailStatusFailed, mail.Status)
}

func TestMailHandler_FailMailAddBouncedTag(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:       mailID,
		TenantID: tenantID,
		Status:   domain.MailStatusQueued,
		Tags:     []string{"initial"},
	}
	mailRepo.Mails[mailID] = mail

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)

	h.failMail(context.Background(), mailID, "test error")

	assert.Equal(t, domain.MailStatusFailed, mail.Status)
	assert.Contains(t, mail.Tags, "bounced")
	assert.Contains(t, mail.Tags, "initial")
}

func TestMailHandler_Success_ClearsBodies(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Hello",
		TextBody:     "World",
		HTMLBody:     "<p>World</p>",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{{Email: "a@b.com"}}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{ID: smtpCfgID, TenantID: tenantID}
	tenant := defaultTenant(tenantID)
	tenant.Settings.StoreBody = false // défaut : pas de copie
	tenantRepo.Tenants[tenantID] = tenant

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, domain.MailStatusSent, mail.Status)
	// Les corps doivent être purgés
	assert.Empty(t, mail.TextBody)
	assert.Empty(t, mail.HTMLBody)
	assert.Nil(t, mail.CompressedBody)
	assert.True(t, mailRepo.Called("ClearBodies"))
}

func TestMailHandler_Success_StoresCompressedBody(t *testing.T) {
	tenantID := uuid.New()
	mailID, _ := uuid.NewV7()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	mail := &domain.Mail{
		ID:           mailID,
		TenantID:     tenantID,
		SMTPConfigID: &smtpCfgID,
		Subject:      "Hello",
		TextBody:     "World",
		HTMLBody:     "<p>World</p>",
		Status:       domain.MailStatusQueued,
	}
	mailRepo.Mails[mailID] = mail
	mailRepo.Recipients[mailID] = []domain.MailRecipient{{Email: "a@b.com"}}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{ID: smtpCfgID, TenantID: tenantID}
	tenant := defaultTenant(tenantID)
	tenant.Settings.StoreBody = true // copie compressée activée
	tenantRepo.Tenants[tenantID] = tenant

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := makeTask(t, mailID, tenantID)

	err := h.HandleMailSend(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, domain.MailStatusSent, mail.Status)
	// Les corps texte doivent être purgés mais la version compressée doit exister
	assert.Empty(t, mail.TextBody)
	assert.Empty(t, mail.HTMLBody)
	assert.NotNil(t, mail.CompressedBody)
	assert.True(t, mailRepo.Called("ClearBodies"))
}

func TestMailHandler_InvalidPayload(t *testing.T) {
	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)
	task := asynq.NewTask(TypeMailSend, []byte("invalid json"))

	err := h.HandleMailSend(context.Background(), task)
	assert.Error(t, err)
}

func TestMailHandler_GetCompiledTemplate_ReloadsAfterTTL(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	tmplRepo := mocks.NewMockTemplateRepo()
	tmplRepo.Templates[templateID] = &domain.Template{
		ID:          templateID,
		TenantID:    tenantID,
		SubjectTmpl: "v1",
		TextBody:    "body",
		HTMLBody:    "<p>body</p>",
	}

	h := newTestMailHandler(
		mocks.NewMockMailRepo(), mocks.NewMockSMTPConfigRepo(), mocks.NewMockTenantRepo(),
		tmplRepo, &mocks.MockMailSender{}, &mocks.MockRateLimiter{Allowed: true},
	)

	// Horloge contrôlée pour piloter l'expiration du cache.
	now := time.Now()
	h.tmplCache.now = func() time.Time { return now }

	ctx := context.Background()
	render := func() string {
		c, err := h.getCompiledTemplate(ctx, tenantID, templateID)
		require.NoError(t, err)
		s, err := c.RenderSubject(map[string]string{})
		require.NoError(t, err)
		return s
	}

	assert.Equal(t, "v1", render())

	// Édition du template côté API.
	tmplRepo.Templates[templateID].SubjectTmpl = "v2"

	// Avant le TTL : version cachée toujours servie.
	assert.Equal(t, "v1", render(), "doit servir la version cachée avant le TTL")

	// Après le TTL : rechargement depuis le repo.
	now = now.Add(defaultTemplateCacheTTL + time.Second)
	assert.Equal(t, "v2", render(), "doit recharger après expiration du TTL")
}

func TestMailHandler_CachesTenantAndSMTPConfig(t *testing.T) {
	tenantID := uuid.New()
	smtpCfgID := uuid.New()

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	tenantRepo := mocks.NewMockTenantRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	sender := &mocks.MockMailSender{}
	rateLimiter := &mocks.MockRateLimiter{Allowed: true}

	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{ID: smtpCfgID, TenantID: tenantID}
	tenantRepo.Tenants[tenantID] = defaultTenant(tenantID)

	h := newTestMailHandler(mailRepo, smtpRepo, tenantRepo, tmplRepo, sender, rateLimiter)

	// Deux mails du même tenant/config SMTP, traités successivement.
	for i := 0; i < 2; i++ {
		id, _ := uuid.NewV7()
		mailRepo.Mails[id] = &domain.Mail{
			ID: id, TenantID: tenantID, SMTPConfigID: &smtpCfgID,
			Subject: "Hello", TextBody: "World", Status: domain.MailStatusQueued,
		}
		mailRepo.Recipients[id] = []domain.MailRecipient{{Email: "a@b.com"}}
		require.NoError(t, h.HandleMailSend(context.Background(), makeTask(t, id, tenantID)))
	}

	// Tenant et config SMTP chargés une seule fois malgré deux envois (cache TTL).
	assert.Equal(t, 1, tenantRepo.CallCount("GetByID"), "tenant doit être mis en cache")
	assert.Equal(t, 1, smtpRepo.CallCount("GetByID"), "config SMTP doit être mise en cache")
}
