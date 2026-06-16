package test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/analysis"
	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

// --- Tests ---

func TestMailService_CreateWithTags(t *testing.T) {
	tenantID := uuid.New()
	smtpCfgID := uuid.New()

	tenant := TestTenant(tenantID)

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID:        smtpCfgID,
		TenantID:  tenantID,
		IsDefault: true,
		FromEmail: "noreply@example.com",
	}
	tenantRepo := mocks.NewMockTenantRepo()
	tenantRepo.Tenants[tenantID] = tenant
	tmplRepo := mocks.NewMockTemplateRepo()
	queueClient := &mocks.MockQueueClient{}

	analysisService := service.NewAnalysisService(tmplRepo)
	attachmentService := service.NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	mailService := service.NewMailService(mailRepo, smtpRepo, tmplRepo, tenantRepo, queueClient, analysisService, attachmentService)

	req := domain.CreateMailRequest{
		To:       []domain.EmailAddress{{Email: "user@example.com"}},
		Subject:  "Hello",
		TextBody: "Normal content. Unsubscribe here.",
		Tags:     []string{"custom-tag", "important"},
	}

	mails, err := mailService.Create(context.Background(), tenantID, req)
	if err != nil {
		t.Fatalf("erreur de création du mail : %v", err)
	}

	if len(mails) != 1 {
		t.Fatalf("attendu 1 mail, obtenu %d", len(mails))
	}

	mail := mails[0]

	// Vérifier le spam score
	if mail.SpamScore == nil {
		t.Fatal("le spam score ne devrait pas être nil")
	}

	// Vérifier les tags (custom uniquement car priorité = default)
	found := map[string]bool{}
	for _, tag := range mail.Tags {
		found[tag] = true
	}
	if !found["custom-tag"] {
		t.Error("le tag 'custom-tag' devrait être présent")
	}
	if !found["important"] {
		t.Error("le tag 'important' devrait être présent")
	}
}

func TestMailService_CreateWithPriorityTag(t *testing.T) {
	tenantID := uuid.New()
	smtpCfgID := uuid.New()

	tenant := TestTenant(tenantID)

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID: smtpCfgID, TenantID: tenantID, IsDefault: true, FromEmail: "noreply@example.com",
	}
	tenantRepo := mocks.NewMockTenantRepo()
	tenantRepo.Tenants[tenantID] = tenant
	tmplRepo := mocks.NewMockTemplateRepo()
	queueClient := &mocks.MockQueueClient{}

	analysisService := service.NewAnalysisService(tmplRepo)
	attachmentService := service.NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	mailService := service.NewMailService(mailRepo, smtpRepo, tmplRepo, tenantRepo, queueClient, analysisService, attachmentService)

	critical := domain.MailPriorityCritical
	req := domain.CreateMailRequest{
		To:       []domain.EmailAddress{{Email: "user@example.com"}},
		Subject:  "Urgent",
		TextBody: "Important message. Unsubscribe link.",
		Priority: &critical,
	}

	mails, err := mailService.Create(context.Background(), tenantID, req)
	if err != nil {
		t.Fatalf("erreur : %v", err)
	}

	found := false
	for _, tag := range mails[0].Tags {
		if tag == "priority:critical" {
			found = true
		}
	}
	if !found {
		t.Errorf("le tag 'priority:critical' devrait être présent, tags: %v", mails[0].Tags)
	}
}

func TestMailService_SpamBlocked(t *testing.T) {
	tenantID := uuid.New()
	smtpCfgID := uuid.New()

	threshold := float32(1.0)
	action := domain.SpamScoreActionBlock
	tenant := TestTenant(tenantID)
	tenant.Settings.SpamScoreThreshold = &threshold
	tenant.Settings.SpamScoreAction = &action

	mailRepo := mocks.NewMockMailRepo()
	smtpRepo := mocks.NewMockSMTPConfigRepo()
	smtpRepo.Configs[smtpCfgID] = &domain.SMTPConfig{
		ID: smtpCfgID, TenantID: tenantID, IsDefault: true, FromEmail: "noreply@example.com",
	}
	tenantRepo := mocks.NewMockTenantRepo()
	tenantRepo.Tenants[tenantID] = tenant
	tmplRepo := mocks.NewMockTemplateRepo()
	queueClient := &mocks.MockQueueClient{}

	analysisService := service.NewAnalysisService(tmplRepo)
	attachmentService := service.NewAttachmentService(mocks.NewMockAttachmentRepo(), mocks.NewMockBlobStore(), domain.AttachmentStoragePostgres)
	mailService := service.NewMailService(mailRepo, smtpRepo, tmplRepo, tenantRepo, queueClient, analysisService, attachmentService)

	req := domain.CreateMailRequest{
		To:      []domain.EmailAddress{{Email: "user@example.com"}},
		Subject: "FREE MONEY!!! ACT NOW!!! CLICK HERE!!!",
		HTMLBody: `<html><body>
			<span style="display:none">hidden</span>
			CLICK HERE FREE GIFT LIMITED TIME
		</body></html>`,
	}

	_, err := mailService.Create(context.Background(), tenantID, req)
	if err == nil {
		t.Fatal("le mail aurait dû être bloqué par le seuil spam")
	}
	if !isSpamBlocked(err) {
		t.Errorf("erreur attendue ErrSpamBlocked, obtenue : %v", err)
	}
}

func isSpamBlocked(err error) bool {
	return err != nil && (err.Error() == domain.ErrSpamBlocked.Error() ||
		len(err.Error()) > len(domain.ErrSpamBlocked.Error()) &&
			err.Error()[:len(domain.ErrSpamBlocked.Error())] == domain.ErrSpamBlocked.Error())
}

func TestMailService_AddTagsOnMock(t *testing.T) {
	mailRepo := mocks.NewMockMailRepo()
	id, _ := uuid.NewV7()
	mail := &domain.Mail{
		ID:       id,
		TenantID: uuid.New(),
		Tags:     []string{"initial"},
	}
	mailRepo.Mails[id] = mail

	err := mailRepo.AddTags(context.Background(), id, []string{"bounced", "retry-1"})
	if err != nil {
		t.Fatalf("erreur AddTags : %v", err)
	}

	if len(mail.Tags) != 3 {
		t.Errorf("attendu 3 tags, obtenu %d: %v", len(mail.Tags), mail.Tags)
	}

	// Pas de doublons
	err = mailRepo.AddTags(context.Background(), id, []string{"bounced"})
	if err != nil {
		t.Fatalf("erreur AddTags (dedup) : %v", err)
	}
	if len(mail.Tags) != 3 {
		t.Errorf("doublons non dédupliqués, attendu 3 tags, obtenu %d: %v", len(mail.Tags), mail.Tags)
	}
}

func TestSpamScore_ComputeViaService(t *testing.T) {
	tmplRepo := mocks.NewMockTemplateRepo()
	svc := service.NewAnalysisService(tmplRepo)

	// Email propre
	score := svc.ComputeSpamScore("Hello", "Normal email content. Unsubscribe here.", "<p>Normal</p>")
	if score > 1.0 {
		t.Errorf("score trop élevé pour un email propre : %.1f", score)
	}

	// Email spam
	score = svc.ComputeSpamScore("FREE MONEY!!!", "", `<span style="display:none">hidden</span>`)
	if score < 3.0 {
		t.Errorf("score trop bas pour un email spam : %.1f", score)
	}
}

func TestBuildAutoTags(t *testing.T) {
	// Utilisation indirecte via MailService.Create — on vérifie que le spam checker fonctionne
	sc := analysis.NewSpamChecker()

	// Test que le checker retourne bien un résultat valide
	result := sc.Check("Normal subject", "Normal body. Unsubscribe here.", "<p>Normal HTML</p>")
	if result.MaxScore != 10.0 {
		t.Errorf("MaxScore devrait être 10.0, obtenu %.1f", result.MaxScore)
	}
	if result.Score < 0 {
		t.Error("le score ne devrait pas être négatif")
	}
}
