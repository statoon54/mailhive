//go:build integration

package mailer_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/mailer"
	"github.com/statoon54/mailhive/internal/domain"
)

// ---------- Configuration ----------

func mailpitURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("MAILPIT_URL")
	if u == "" {
		u = "http://localhost:8025"
	}
	return u
}

func smtpHost() string {
	h := os.Getenv("MAILPIT_SMTP_HOST")
	if h == "" {
		return "localhost"
	}
	return h
}

func smtpPort() int {
	return 1025
}

func testSMTPConfig() *domain.SMTPConfig {
	return &domain.SMTPConfig{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Name:       "mailpit-test",
		Host:       smtpHost(),
		Port:       smtpPort(),
		AuthMethod: domain.AuthNone,
		TLSPolicy:  domain.TLSNone,
		FromEmail:  "test@mailhive.dev",
		FromName:   "MailHive Test",
		Charset:    domain.CharsetUTF8,
		Encoding:   domain.EncodingQP,
		IsDefault:  true,
		IsActive:   true,
	}
}

// ---------- Mailpit API helpers ----------

// mailpitMessage représente un message dans la réponse de l'API Mailpit.
type mailpitMessage struct {
	ID          string           `json:"ID"`
	From        mailpitAddress   `json:"From"`
	To          []mailpitAddress `json:"To"`
	Cc          []mailpitAddress `json:"Cc"`
	Bcc         []mailpitAddress `json:"Bcc"`
	Subject     string           `json:"Subject"`
	Tags        []string         `json:"Tags"`
	Snippet     string           `json:"Snippet"`
	Attachments int              `json:"Attachments"`
}

type mailpitAddress struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

type mailpitSearchResult struct {
	Total    int              `json:"total"`
	Messages []mailpitMessage `json:"messages"`
}

// mailpitMessageDetail contient le détail complet d'un message.
type mailpitMessageDetail struct {
	ID          string              `json:"ID"`
	From        mailpitAddress      `json:"From"`
	To          []mailpitAddress    `json:"To"`
	Cc          []mailpitAddress    `json:"Cc"`
	Bcc         []mailpitAddress    `json:"Bcc"`
	Subject     string              `json:"Subject"`
	Text        string              `json:"Text"`
	HTML        string              `json:"HTML"`
	Attachments []mailpitAttachment `json:"Attachments"`
}

type mailpitAttachment struct {
	FileName    string `json:"FileName"`
	ContentType string `json:"ContentType"`
	Size        int    `json:"Size"`
}

// deleteAllMessages purge tous les messages de Mailpit.
func deleteAllMessages(t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, mailpitURL(t)+"/api/v1/messages", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
}

// waitForMessage attend qu'un message arrive dans Mailpit (polling).
func waitForMessage(t *testing.T, subject string, timeout time.Duration) mailpitMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(mailpitURL(t) + "/api/v1/messages")
		require.NoError(t, err)

		var result mailpitSearchResult
		err = json.NewDecoder(resp.Body).Decode(&result)
		_ = resp.Body.Close()
		require.NoError(t, err)

		for _, msg := range result.Messages {
			if msg.Subject == subject {
				return msg
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("message avec sujet %q non reçu dans Mailpit après %s", subject, timeout)
	return mailpitMessage{}
}

// getMessageDetail récupère le détail complet d'un message.
func getMessageDetail(t *testing.T, id string) mailpitMessageDetail {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/message/%s", mailpitURL(t), id))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var detail mailpitMessageDetail
	err = json.NewDecoder(resp.Body).Decode(&detail)
	require.NoError(t, err)
	return detail
}

// ---------- Tests ----------

func TestIntegration_SendBasicMail(t *testing.T) {
	deleteAllMessages(t)

	sender := mailer.NewSender()
	cfg := testSMTPConfig()

	mailID, err := uuid.NewV7()
	require.NoError(t, err)

	subject := fmt.Sprintf("Test basique %s", mailID)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  cfg.TenantID,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		Subject:   subject,
		TextBody:  "Bonjour, ceci est un test en texte brut.",
		HTMLBody:  "<h1>Bonjour</h1><p>Ceci est un test en HTML.</p>",
		Recipients: []domain.MailRecipient{
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "alice@example.com", Name: "Alice"},
		},
		Status: domain.MailStatusSending,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sender.Send(ctx, cfg, mail)
	require.NoError(t, err)

	// Vérifier via l'API Mailpit
	msg := waitForMessage(t, subject, 5*time.Second)
	assert.Equal(t, subject, msg.Subject)
	assert.Equal(t, "MailHive Test", msg.From.Name)
	assert.Equal(t, "test@mailhive.dev", msg.From.Address)
	require.Len(t, msg.To, 1)
	assert.Equal(t, "alice@example.com", msg.To[0].Address)
	assert.Equal(t, "Alice", msg.To[0].Name)

	// Vérifier le contenu détaillé
	detail := getMessageDetail(t, msg.ID)
	assert.Contains(t, detail.Text, "Bonjour, ceci est un test en texte brut.")
	assert.Contains(t, detail.HTML, "<h1>Bonjour</h1>")
}

func TestIntegration_SendMailWithMultipleRecipients(t *testing.T) {
	deleteAllMessages(t)

	sender := mailer.NewSender()
	cfg := testSMTPConfig()

	mailID, err := uuid.NewV7()
	require.NoError(t, err)

	subject := fmt.Sprintf("Test multi-destinataires %s", mailID)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  cfg.TenantID,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		Subject:   subject,
		TextBody:  "Mail avec TO, CC et BCC.",
		Recipients: []domain.MailRecipient{
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "to1@example.com", Name: "To 1"},
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "to2@example.com", Name: "To 2"},
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientCC, Email: "cc@example.com", Name: "CC"},
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientBCC, Email: "bcc@example.com", Name: "BCC"},
		},
		Status: domain.MailStatusSending,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sender.Send(ctx, cfg, mail)
	require.NoError(t, err)

	msg := waitForMessage(t, subject, 5*time.Second)
	assert.Len(t, msg.To, 2)
	assert.Equal(t, "to1@example.com", msg.To[0].Address)
	assert.Equal(t, "to2@example.com", msg.To[1].Address)
	assert.Len(t, msg.Cc, 1)
	assert.Equal(t, "cc@example.com", msg.Cc[0].Address)
	// BCC n'apparaît pas dans les headers — Mailpit le capture séparément
	assert.Len(t, msg.Bcc, 1)
	assert.Equal(t, "bcc@example.com", msg.Bcc[0].Address)
}

func TestIntegration_SendMailWithAttachment(t *testing.T) {
	deleteAllMessages(t)

	sender := mailer.NewSender()
	cfg := testSMTPConfig()

	mailID, err := uuid.NewV7()
	require.NoError(t, err)

	fileContent := "Contenu du fichier de test.\nLigne 2."
	b64Content := base64.StdEncoding.EncodeToString([]byte(fileContent))

	subject := fmt.Sprintf("Test pièce jointe %s", mailID)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  cfg.TenantID,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		Subject:   subject,
		TextBody:  "Mail avec pièce jointe.",
		Attachments: []domain.Attachment{
			{
				Filename:    "test.txt",
				Content:     b64Content,
				ContentType: "text/plain",
			},
		},
		Recipients: []domain.MailRecipient{
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "bob@example.com", Name: "Bob"},
		},
		Status: domain.MailStatusSending,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sender.Send(ctx, cfg, mail)
	require.NoError(t, err)

	msg := waitForMessage(t, subject, 5*time.Second)
	assert.Equal(t, 1, msg.Attachments)

	detail := getMessageDetail(t, msg.ID)
	require.Len(t, detail.Attachments, 1)
	assert.Equal(t, "test.txt", detail.Attachments[0].FileName)
	assert.Equal(t, "text/plain", detail.Attachments[0].ContentType)
}

func TestIntegration_SendHTMLOnlyMail(t *testing.T) {
	deleteAllMessages(t)

	sender := mailer.NewSender()
	cfg := testSMTPConfig()

	mailID, err := uuid.NewV7()
	require.NoError(t, err)

	subject := fmt.Sprintf("Test HTML seul %s", mailID)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  cfg.TenantID,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		Subject:   subject,
		HTMLBody:  "<p>Uniquement du HTML, pas de texte brut.</p>",
		Recipients: []domain.MailRecipient{
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "html@example.com", Name: "HTML User"},
		},
		Status: domain.MailStatusSending,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sender.Send(ctx, cfg, mail)
	require.NoError(t, err)

	msg := waitForMessage(t, subject, 5*time.Second)
	detail := getMessageDetail(t, msg.ID)
	assert.Contains(t, detail.HTML, "Uniquement du HTML")
	assert.Empty(t, detail.Text)
}

func TestIntegration_SendMailWithUTF8Characters(t *testing.T) {
	deleteAllMessages(t)

	sender := mailer.NewSender()
	cfg := testSMTPConfig()

	mailID, err := uuid.NewV7()
	require.NoError(t, err)

	subject := fmt.Sprintf("Test UTF-8 éàü ñ %s", mailID)
	mail := &domain.Mail{
		ID:        mailID,
		TenantID:  cfg.TenantID,
		FromEmail: cfg.FromEmail,
		FromName:  "Éric Müller",
		Subject:   subject,
		TextBody:  "Caractères spéciaux : é à ü ñ ö ß 你好 🎉",
		HTMLBody:  "<p>Caractères spéciaux : é à ü ñ ö ß 你好 🎉</p>",
		Recipients: []domain.MailRecipient{
			{ID: uuid.New(), MailID: mailID, Type: domain.RecipientTo, Email: "utf8@example.com", Name: "Héloïse"},
		},
		Status: domain.MailStatusSending,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sender.Send(ctx, cfg, mail)
	require.NoError(t, err)

	msg := waitForMessage(t, subject, 5*time.Second)
	assert.Equal(t, "Éric Müller", msg.From.Name)

	detail := getMessageDetail(t, msg.ID)
	assert.Contains(t, detail.Text, "你好")
	assert.Contains(t, detail.HTML, "🎉")
}
