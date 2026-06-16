package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/statoon54/mailhive/internal/domain"
)

// LogSender implémente port.MailSender en loguant les mails au lieu de les envoyer via SMTP.
type LogSender struct{}

// NewLogSender crée un nouveau sender de simulation.
func NewLogSender() *LogSender {
	return &LogSender{}
}

// Send simule l'envoi d'un mail en affichant les détails dans les logs.
func (s *LogSender) Send(_ context.Context, cfg *domain.SMTPConfig, mail *domain.Mail) error {
	// Simuler un léger délai pour un comportement réaliste dans les queues
	time.Sleep(100 * time.Millisecond)

	// Collecter les destinataires par type
	var to, cc, bcc []string
	for _, r := range mail.Recipients {
		addr := fmt.Sprintf("%s <%s>", r.Name, r.Email)
		switch r.Type {
		case domain.RecipientTo:
			to = append(to, addr)
		case domain.RecipientCC:
			cc = append(cc, addr)
		case domain.RecipientBCC:
			bcc = append(bcc, addr)
		}
	}

	// Aperçu du corps (max 200 caractères)
	body := mail.HTMLBody
	if body == "" {
		body = mail.TextBody
	}
	if len(body) > 200 {
		body = body[:200] + "..."
	}

	// Pièces jointes
	var attachments []string
	for _, att := range mail.Attachments {
		attachments = append(attachments, fmt.Sprintf("%s (%s)", att.Filename, att.ContentType))
	}

	fields := []any{
		"from", fmt.Sprintf("%s <%s>", cfg.FromName, mail.FromEmail),
		"to", strings.Join(to, ", "),
		"subject", mail.Subject,
		"body_preview", body,
		"smtp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}
	if len(cc) > 0 {
		fields = append(fields, "cc", strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		fields = append(fields, "bcc", strings.Join(bcc, ", "))
	}
	if len(attachments) > 0 {
		fields = append(fields, "attachments", strings.Join(attachments, ", "))
	}
	slog.Info("mail simulé (non envoyé)", fields...)

	return nil
}
