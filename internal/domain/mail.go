package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MailPriority représente la priorité d'un mail dans la file d'attente.
type MailPriority string

const (
	MailPriorityCritical MailPriority = "critical"
	MailPriorityDefault  MailPriority = "default"
	MailPriorityLow      MailPriority = "low"
)

// ValidMailPriorities contient les priorités autorisées.
var ValidMailPriorities = map[MailPriority]bool{
	MailPriorityCritical: true,
	MailPriorityDefault:  true,
	MailPriorityLow:      true,
}

// MailStatus représente le statut d'un mail dans le cycle de vie.
type MailStatus string

const (
	MailStatusPending   MailStatus = "pending"
	MailStatusQueued    MailStatus = "queued"
	MailStatusSending   MailStatus = "sending"
	MailStatusSent      MailStatus = "sent"
	MailStatusFailed    MailStatus = "failed"
	MailStatusCancelled MailStatus = "cancelled"
)

// RecipientType représente le type de destinataire.
type RecipientType string

const (
	RecipientTo  RecipientType = "to"
	RecipientCC  RecipientType = "cc"
	RecipientBCC RecipientType = "bcc"
)

// Mail représente un mail à envoyer ou envoyé.
type Mail struct {
	ID             uuid.UUID         `json:"id"`
	TenantID       uuid.UUID         `json:"tenant_id"`
	SMTPConfigID   *uuid.UUID        `json:"smtp_config_id,omitempty"`
	TemplateID     *uuid.UUID        `json:"template_id,omitempty"`
	FromEmail      string            `json:"from_email"`
	FromName       string            `json:"from_name"`
	Subject        string            `json:"subject"`
	TextBody       string            `json:"text_body"`
	HTMLBody       string            `json:"html_body"`
	TemplateData   map[string]string `json:"template_data,omitempty"`
	Attachments    []Attachment      `json:"attachments,omitempty"`
	AttachmentRefs []AttachmentRef   `json:"attachment_refs,omitempty"`
	Status         MailStatus        `json:"status"`
	StatusMessage  string            `json:"status_message,omitempty"`
	Attempts       int               `json:"attempts"`
	ScheduledAt    *time.Time        `json:"scheduled_at,omitempty"`
	SentAt         *time.Time        `json:"sent_at,omitempty"`
	Priority       MailPriority      `json:"priority"`
	TaskID         string            `json:"task_id,omitempty"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
	SpamScore      *float32          `json:"spam_score,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	CompressedBody []byte            `json:"-"`
	Recipients     []MailRecipient   `json:"recipients,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Attachment représente une pièce jointe.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// MailRecipient représente un destinataire d'un mail.
type MailRecipient struct {
	ID     uuid.UUID     `json:"id"`
	MailID uuid.UUID     `json:"mail_id"`
	Type   RecipientType `json:"type"`
	Email  string        `json:"email"`
	Name   string        `json:"name,omitempty"`
}

// EmailAddress représente une adresse email avec un nom optionnel.
type EmailAddress struct {
	TemplateData map[string]string `json:"template_data,omitempty"`
	Email        string            `json:"email"                   validate:"required,email"`
	Name         string            `json:"name,omitempty"`
}

// FlexTime accepte plusieurs formats de date en JSON (RFC3339, datetime avec/sans T, avec timezone).
type FlexTime time.Time

// Formats avec timezone explicite (parsés tels quels).
var flexTimeAbsolute = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z",
}

// Formats sans timezone (parsés dans AppTimezone ou heure locale du serveur).
var flexTimeLocal = []string{
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// AppTimezone est la timezone configurée dans le branding.
// Si nil, time.Local est utilisé pour les dates sans timezone explicite.
var AppTimezone *time.Location

func (ft *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	for _, f := range flexTimeAbsolute {
		if t, err := time.Parse(f, s); err == nil {
			*ft = FlexTime(t)
			return nil
		}
	}
	loc := AppTimezone
	if loc == nil {
		loc = time.Local
	}
	for _, f := range flexTimeLocal {
		if t, err := time.ParseInLocation(f, s, loc); err == nil {
			*ft = FlexTime(t)
			return nil
		}
	}
	return fmt.Errorf(
		"format de date non reconnu : %s (attendu : RFC3339 ou AAAA-MM-JJ HH:MM:SS)",
		s,
	)
}

func (ft FlexTime) MarshalJSON() ([]byte, error) {
	return time.Time(ft).MarshalJSON()
}

// TimePtr convertit en *time.Time (nil si valeur zéro).
func (ft *FlexTime) TimePtr() *time.Time {
	if ft == nil {
		return nil
	}
	t := time.Time(*ft)
	if t.IsZero() {
		return nil
	}
	return &t
}

// CreateMailRequest contient les données pour composer un mail.
type CreateMailRequest struct {
	From         *EmailAddress     `json:"from,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
	Priority     *MailPriority     `json:"priority,omitempty"`
	ScheduledAt  *FlexTime         `json:"scheduled_at,omitempty"`
	SMTPConfigID *uuid.UUID        `json:"smtp_config_id,omitempty"`
	TemplateData map[string]string `json:"template_data,omitempty"`
	TemplateID   *uuid.UUID        `json:"template_id,omitempty"`
	HTMLBody     string            `json:"html_body,omitempty"`
	Subject      string            `json:"subject,omitempty"`
	TextBody     string            `json:"text_body,omitempty"`
	Attachments  []Attachment      `json:"attachments,omitempty"`
	BCC          []EmailAddress    `json:"bcc,omitempty"`
	CC           []EmailAddress    `json:"cc,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	To           []EmailAddress    `json:"to"                       validate:"required,min=1,dive"`
	Individuel   bool              `json:"individuel,omitempty"`
}

// CreateMailBatchResponse représente la réponse lors d'un envoi individuel (N mails créés).
type CreateMailBatchResponse struct {
	MailIDs []uuid.UUID `json:"mail_ids"`
	Total   int         `json:"total"`
}

// MailListFilter contient les critères de filtrage pour la liste des mails.
type MailListFilter struct {
	Status  *MailStatus `json:"status,omitempty"`
	Query   string      `json:"q,omitempty"`
	TagMode string      `json:"tag_mode,omitempty"` // "and" (défaut) ou "or"
	Tags    []string    `json:"tags,omitempty"`
	Limit   int         `json:"limit"`
	Page    int         `json:"page"`
}

// MailStats contient les statistiques d'envoi de mails d'un tenant.
type MailStats struct {
	Pending   int64 `json:"pending"`
	Queued    int64 `json:"queued"`
	Sending   int64 `json:"sending"`
	Sent      int64 `json:"sent"`
	Failed    int64 `json:"failed"`
	Cancelled int64 `json:"cancelled"`
	Rejected  int64 `json:"rejected"`
	Total     int64 `json:"total"`
}

// TenantMailStats contient les statistiques de mails par tenant.
type TenantMailStats struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Sent       int64  `json:"sent"`
	Pending    int64  `json:"pending"`
	Failed     int64  `json:"failed"`
	Total      int64  `json:"total"`
}

// PaginatedList représente une liste paginée générique.
type PaginatedList[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}
