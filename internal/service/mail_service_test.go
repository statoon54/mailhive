package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

type mailTestSetup struct {
	mailRepo   *mocks.MockMailRepo
	smtpRepo   *mocks.MockSMTPConfigRepo
	tmplRepo   *mocks.MockTemplateRepo
	tenantRepo *mocks.MockTenantRepo
	queue      *mocks.MockQueueClient
	analysis   *mocks.MockAnalysisService
	svc        *MailService
	tenantID   uuid.UUID
	smtpCfgID  uuid.UUID
}

func setupMailTest() *mailTestSetup {
	tenantID := uuid.New()
	smtpCfgID := uuid.New()

	tenantRepo := mocks.NewMockTenantRepo()
	tenantRepo.Tenants[tenantID] = &domain.Tenant{
		ID:       tenantID,
		Name:     "Test",
		Slug:     "test",
		IsActive: true,
		Settings: domain.TenantSettings{
			RateLimit:        10,
			RateBurst:        10,
			MaxDestinataires: 100,
			DefaultPriority:  domain.MailPriorityDefault,
		},
	}

	smtpRepo := mocks.NewMockSMTPConfigRepo()
	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID:        smtpCfgID,
		TenantID:  tenantID,
		IsDefault: true,
		FromEmail: "noreply@example.com",
	}

	mailRepo := mocks.NewMockMailRepo()
	tmplRepo := mocks.NewMockTemplateRepo()
	queue := &mocks.MockQueueClient{}
	analysis := &mocks.MockAnalysisService{}

	attachmentSvc := NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	svc := NewMailService(mailRepo, smtpRepo, tmplRepo, tenantRepo, queue, analysis, attachmentSvc)

	return &mailTestSetup{
		mailRepo:   mailRepo,
		smtpRepo:   smtpRepo,
		tmplRepo:   tmplRepo,
		tenantRepo: tenantRepo,
		queue:      queue,
		analysis:   analysis,
		svc:        svc,
		tenantID:   tenantID,
		smtpCfgID:  smtpCfgID,
	}
}

func TestMailService_Create_Grouped(t *testing.T) {
	s := setupMailTest()

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:       []domain.EmailAddress{{Email: "a@b.com"}, {Email: "c@d.com"}},
		Subject:  "Test",
		TextBody: "Hello",
	})
	require.NoError(t, err)
	assert.Len(t, mails, 1)
	assert.Equal(t, domain.MailStatusQueued, mails[0].Status)
	assert.Equal(t, 2, len(mails[0].Recipients))
}

func TestMailService_Create_Individuel(t *testing.T) {
	s := setupMailTest()

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:         []domain.EmailAddress{{Email: "a@b.com"}, {Email: "c@d.com"}},
		Subject:    "Test",
		TextBody:   "Hello",
		Individuel: true,
	})
	require.NoError(t, err)
	assert.Len(t, mails, 2)
	for _, m := range mails {
		assert.Equal(t, domain.MailStatusQueued, m.Status)
	}
}

func TestMailService_Create_EmptyTo(t *testing.T) {
	s := setupMailTest()

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{},
		Subject: "Test",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestMailService_Create_PriorityFromRequest(t *testing.T) {
	s := setupMailTest()
	critical := domain.MailPriorityCritical

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:       []domain.EmailAddress{{Email: "a@b.com"}},
		Subject:  "Test",
		Priority: &critical,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.MailPriorityCritical, mails[0].Priority)
}

func TestMailService_Create_PriorityFromTenant(t *testing.T) {
	s := setupMailTest()
	s.tenantRepo.Tenants[s.tenantID].Settings.DefaultPriority = domain.MailPriorityLow

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.MailPriorityLow, mails[0].Priority)
}

func TestMailService_Create_PriorityDefault(t *testing.T) {
	s := setupMailTest()

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.MailPriorityDefault, mails[0].Priority)
}

func TestMailService_Create_InvalidPriority(t *testing.T) {
	s := setupMailTest()
	invalid := domain.MailPriority("urgent")

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:       []domain.EmailAddress{{Email: "a@b.com"}},
		Subject:  "Test",
		Priority: &invalid,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestMailService_Create_MaxDestinataires(t *testing.T) {
	s := setupMailTest()
	s.tenantRepo.Tenants[s.tenantID].Settings.MaxDestinataires = 2

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}, {Email: "c@d.com"}, {Email: "e@f.com"}},
		Subject: "Test",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
	assert.Contains(t, err.Error(), "destinataires")
}

func TestMailService_Create_ExplicitSMTP(t *testing.T) {
	s := setupMailTest()
	customSMTPID := uuid.New()
	s.smtpRepo.Configs[customSMTPID] = &domain.SMTPConfig{
		ID:        customSMTPID,
		TenantID:  s.tenantID,
		FromEmail: "custom@example.com",
	}

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		SMTPConfigID: &customSMTPID,
		To:           []domain.EmailAddress{{Email: "a@b.com"}},
		Subject:      "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, customSMTPID, *mails[0].SMTPConfigID)
}

func TestMailService_Create_DefaultSMTP(t *testing.T) {
	s := setupMailTest()

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "Test",
	})
	require.NoError(t, err)
	assert.Equal(t, s.smtpCfgID, *mails[0].SMTPConfigID)
}

func TestMailService_Create_NoSMTP(t *testing.T) {
	s := setupMailTest()
	// Remove all SMTP configs
	for k := range s.smtpRepo.Configs {
		delete(s.smtpRepo.Configs, k)
	}

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "Test",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSMTPConfigNotSet)
}

func TestMailService_Create_SpamBlocked(t *testing.T) {
	s := setupMailTest()
	threshold := float32(1.0)
	action := domain.SpamScoreActionBlock
	s.tenantRepo.Tenants[s.tenantID].Settings.SpamScoreThreshold = &threshold
	s.tenantRepo.Tenants[s.tenantID].Settings.SpamScoreAction = &action
	s.analysis.SpamScore = 5.0

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "SPAM",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSpamBlocked))
}

func TestMailService_Create_SpamWarning(t *testing.T) {
	s := setupMailTest()
	threshold := float32(1.0)
	action := domain.SpamScoreActionWarn
	s.tenantRepo.Tenants[s.tenantID].Settings.SpamScoreThreshold = &threshold
	s.tenantRepo.Tenants[s.tenantID].Settings.SpamScoreAction = &action
	s.analysis.SpamScore = 5.0

	mails, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "a@b.com"}},
		Subject: "Test",
	})
	require.NoError(t, err)
	assert.Len(t, mails, 1)
}

func TestMailService_Create_TemplateValidation(t *testing.T) {
	s := setupMailTest()
	tmplID := uuid.New()
	s.tmplRepo.Templates[tmplID] = &domain.Template{
		ID:        tmplID,
		TenantID:  s.tenantID,
		Variables: map[string]string{"name": "Name"},
	}

	_, err := s.svc.Create(context.Background(), s.tenantID, domain.CreateMailRequest{
		TemplateID: &tmplID,
		To:         []domain.EmailAddress{{Email: "a@b.com"}},
		Subject:    "Test",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestMailService_Cancel_Success(t *testing.T) {
	s := setupMailTest()
	mailID, _ := uuid.NewV7()
	s.mailRepo.Mails[mailID] = &domain.Mail{
		ID:       mailID,
		TenantID: s.tenantID,
		Status:   domain.MailStatusPending,
		TaskID:   "task-123",
		Priority: domain.MailPriorityDefault,
	}

	err := s.svc.Cancel(context.Background(), s.tenantID, mailID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusCancelled, s.mailRepo.Mails[mailID].Status)
}

func TestMailService_Cancel_NotPending(t *testing.T) {
	s := setupMailTest()
	mailID, _ := uuid.NewV7()
	s.mailRepo.Mails[mailID] = &domain.Mail{
		ID:       mailID,
		TenantID: s.tenantID,
		Status:   domain.MailStatusSent,
	}

	err := s.svc.Cancel(context.Background(), s.tenantID, mailID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrMailNotPending)
}

func TestMailService_Retry_Success(t *testing.T) {
	s := setupMailTest()
	mailID, _ := uuid.NewV7()
	s.mailRepo.Mails[mailID] = &domain.Mail{
		ID:       mailID,
		TenantID: s.tenantID,
		Status:   domain.MailStatusFailed,
		Priority: domain.MailPriorityDefault,
	}

	err := s.svc.Retry(context.Background(), s.tenantID, mailID)
	require.NoError(t, err)
	assert.Equal(t, domain.MailStatusQueued, s.mailRepo.Mails[mailID].Status)
}

func TestMailService_Retry_NotFailed(t *testing.T) {
	s := setupMailTest()
	mailID, _ := uuid.NewV7()
	s.mailRepo.Mails[mailID] = &domain.Mail{
		ID:       mailID,
		TenantID: s.tenantID,
		Status:   domain.MailStatusSent,
	}

	err := s.svc.Retry(context.Background(), s.tenantID, mailID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrMailNotFailed)
}

func TestMailService_GetByID(t *testing.T) {
	s := setupMailTest()
	mailID, _ := uuid.NewV7()
	s.mailRepo.Mails[mailID] = &domain.Mail{
		ID:       mailID,
		TenantID: s.tenantID,
		Subject:  "Test",
	}

	mail, err := s.svc.GetByID(context.Background(), s.tenantID, mailID)
	require.NoError(t, err)
	assert.Equal(t, "Test", mail.Subject)
}

func TestMailService_List(t *testing.T) {
	s := setupMailTest()
	for i := 0; i < 3; i++ {
		id, _ := uuid.NewV7()
		s.mailRepo.Mails[id] = &domain.Mail{ID: id, TenantID: s.tenantID}
	}

	list, err := s.svc.List(context.Background(), s.tenantID, domain.MailListFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(3), list.Total)
}

func TestMailService_List_RecipientsBatchedNoNPlusOne(t *testing.T) {
	s := setupMailTest()

	// 3 mails, chacun avec ses propres destinataires.
	for range 3 {
		id, _ := uuid.NewV7()
		s.mailRepo.Mails[id] = &domain.Mail{ID: id, TenantID: s.tenantID}
		rid, _ := uuid.NewV7()
		s.mailRepo.Recipients[id] = []domain.MailRecipient{
			{ID: rid, MailID: id, Type: domain.RecipientTo, Email: "to@example.com"},
		}
	}

	list, err := s.svc.List(context.Background(), s.tenantID, domain.MailListFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	require.Len(t, list.Items, 3)

	// Les destinataires sont correctement assemblés sur chaque mail.
	for _, item := range list.Items {
		require.Len(t, item.Recipients, 1, "mail %s should have its recipient", item.ID)
		assert.Equal(t, item.ID, item.Recipients[0].MailID)
	}

	// Pas de N+1 : un seul appel batch, aucun appel unitaire GetRecipients.
	assert.Equal(t, 1, s.mailRepo.CallCount("GetRecipientsByMailIDs"))
	assert.Equal(t, 0, s.mailRepo.CallCount("GetRecipients"))
}

func TestBuildAutoTags(t *testing.T) {
	tests := []struct {
		name     string
		priority domain.MailPriority
		tmpl     *domain.Template
		custom   []string
		expected []string
	}{
		{"default priority no tmpl", domain.MailPriorityDefault, nil, nil, []string{}},
		{"critical priority", domain.MailPriorityCritical, nil, nil, []string{"priority:critical"}},
		{"low priority", domain.MailPriorityLow, nil, nil, []string{"priority:low"}},
		{"with template", domain.MailPriorityDefault, &domain.Template{Slug: "welcome"}, nil, []string{"template:welcome"}},
		{"with custom", domain.MailPriorityDefault, nil, []string{"vip"}, []string{"vip"}},
		{"all combined", domain.MailPriorityCritical, &domain.Template{Slug: "alert"}, []string{"urgent"}, []string{"priority:critical", "template:alert", "urgent"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := buildAutoTags(tt.priority, tt.tmpl, tt.custom)
			assert.Equal(t, tt.expected, tags)
		})
	}
}
