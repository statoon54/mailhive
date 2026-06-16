package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	gomail "github.com/wneessen/go-mail"

	"github.com/statoon54/mailhive/internal/domain"
)

// Sender implémente port.MailSender avec go-mail.
type Sender struct{}

// NewSender crée un nouveau sender SMTP.
func NewSender() *Sender {
	return &Sender{}
}

// toGoMailCharset convertit un MailCharset domain en charset go-mail.
func toGoMailCharset(c domain.MailCharset) gomail.Charset {
	switch c {
	case domain.CharsetASCII:
		return gomail.CharsetASCII
	case domain.CharsetISO88591:
		return gomail.CharsetISO88591
	case domain.CharsetISO885915:
		return gomail.CharsetISO885915
	default:
		return gomail.CharsetUTF8
	}
}

// toGoMailEncoding convertit un MailEncoding domain en encoding go-mail.
func toGoMailEncoding(e domain.MailEncoding) gomail.Encoding {
	switch e {
	case domain.EncodingBase64:
		return gomail.EncodingB64
	case domain.Encoding7Bit:
		return gomail.EncodingUSASCII
	case domain.Encoding8Bit:
		return gomail.NoEncoding
	default:
		return gomail.EncodingQP
	}
}

// Send envoie un mail via SMTP en utilisant go-mail.
func (s *Sender) Send(ctx context.Context, cfg *domain.SMTPConfig, mail *domain.Mail) error {
	msg := gomail.NewMsg(
		gomail.WithCharset(toGoMailCharset(cfg.Charset)),
		gomail.WithEncoding(toGoMailEncoding(cfg.Encoding)),
	)

	// Expéditeur
	if err := msg.FromFormat(mail.FromName, mail.FromEmail); err != nil {
		return fmt.Errorf("erreur de configuration de l'expéditeur : %w", err)
	}

	// Destinataires
	for _, r := range mail.Recipients {
		switch r.Type {
		case domain.RecipientTo:
			if err := msg.AddToFormat(r.Name, r.Email); err != nil {
				return fmt.Errorf("erreur d'ajout du destinataire TO : %w", err)
			}
		case domain.RecipientCC:
			if err := msg.AddCcFormat(r.Name, r.Email); err != nil {
				return fmt.Errorf("erreur d'ajout du destinataire CC : %w", err)
			}
		case domain.RecipientBCC:
			if err := msg.AddBccFormat(r.Name, r.Email); err != nil {
				return fmt.Errorf("erreur d'ajout du destinataire BCC : %w", err)
			}
		}
	}

	// Sujet
	msg.Subject(mail.Subject)

	// Corps texte et HTML
	if mail.TextBody != "" {
		msg.SetBodyString(gomail.TypeTextPlain, mail.TextBody)
	}
	if mail.HTMLBody != "" {
		if mail.TextBody != "" {
			msg.AddAlternativeString(gomail.TypeTextHTML, mail.HTMLBody)
		} else {
			msg.SetBodyString(gomail.TypeTextHTML, mail.HTMLBody)
		}
	}

	// Pièces jointes
	for _, att := range mail.Attachments {
		data, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			return fmt.Errorf("erreur de décodage de la pièce jointe %s : %w", att.Filename, err)
		}
		reader := bytes.NewReader(data)
		if err := msg.AttachReader(att.Filename, reader, gomail.WithFileContentType(gomail.ContentType(att.ContentType))); err != nil {
			return fmt.Errorf("erreur d'ajout de la pièce jointe %s : %w", att.Filename, err)
		}
	}

	// Configuration du client SMTP
	var opts []gomail.Option
	opts = append(opts, gomail.WithPort(cfg.Port))

	// Politique TLS
	switch cfg.TLSPolicy {
	case domain.TLSMandatory:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	case domain.TLSOpportunistic:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSOpportunistic))
	case domain.TLSNone:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	}

	// Authentification
	if cfg.AuthMethod != domain.AuthNone && cfg.Username != nil && cfg.Password != "" {
		switch cfg.AuthMethod {
		case domain.AuthPlain:
			opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain))
		case domain.AuthLogin:
			opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthLogin))
		case domain.AuthCRAMMD5:
			opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthCramMD5))
		}
		opts = append(opts, gomail.WithUsername(*cfg.Username))
		opts = append(opts, gomail.WithPassword(cfg.Password))
	}

	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("erreur de création du client SMTP : %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("erreur d'envoi du mail : %w", err)
	}

	return nil
}
