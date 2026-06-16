package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// TenantRepository définit les opérations de persistance des tenants.
type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	List(ctx context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	UpdateAPIKey(ctx context.Context, id uuid.UUID, newKey string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SMTPConfigRepository définit les opérations de persistance des configs SMTP.
type SMTPConfigRepository interface {
	Create(ctx context.Context, cfg *domain.SMTPConfig) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error)
	GetDefault(ctx context.Context, tenantID uuid.UUID) (*domain.SMTPConfig, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.SMTPConfig, error)
	Update(ctx context.Context, cfg *domain.SMTPConfig) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ClearDefault(ctx context.Context, tenantID uuid.UUID) error
}

// MailRepository définit les opérations de persistance des mails.
type MailRepository interface {
	Create(ctx context.Context, mail *domain.Mail) error
	CreateBatch(ctx context.Context, mails []*domain.Mail) error
	CreateWithRecipients(ctx context.Context, mail *domain.Mail, recipients []domain.MailRecipient, links []domain.AttachmentLink) error
	CreateBatchWithRecipients(ctx context.Context, mails []*domain.Mail, recipients []domain.MailRecipient, links []domain.AttachmentLink) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Mail, error)
	GetAttachmentRefs(ctx context.Context, mailID uuid.UUID) ([]domain.AttachmentRef, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.MailListFilter) (*domain.PaginatedList[domain.Mail], error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MailStatus, message string) error
	UpdateStatuses(ctx context.Context, ids []uuid.UUID, status domain.MailStatus, message string) error
	SetQueued(ctx context.Context, id uuid.UUID, taskID string) error
	SetQueuedBatch(ctx context.Context, mailTaskIDs map[uuid.UUID]string) error
	UpdateTaskID(ctx context.Context, id uuid.UUID, taskID string) error
	UpdateTaskIDs(ctx context.Context, mailTaskIDs map[uuid.UUID]string) error
	MarkSending(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID) error
	Stats(ctx context.Context, tenantID uuid.UUID) (*domain.MailStats, error)
	StatsByTenant(ctx context.Context) ([]domain.TenantMailStats, error)
	CreateRecipients(ctx context.Context, recipients []domain.MailRecipient) error
	GetRecipients(ctx context.Context, mailID uuid.UUID) ([]domain.MailRecipient, error)
	GetRecipientsByMailIDs(ctx context.Context, mailIDs []uuid.UUID) (map[uuid.UUID][]domain.MailRecipient, error)
	AddTags(ctx context.Context, id uuid.UUID, tags []string) error
	ClearBodies(ctx context.Context, id uuid.UUID, compressedBody []byte) error
}

// BrandingRepository définit les opérations de persistance du branding.
type BrandingRepository interface {
	Get(ctx context.Context) (*domain.AppBranding, error)
	Update(ctx context.Context, branding *domain.AppBranding) error
	UpdateLogo(ctx context.Context, data []byte, contentType string) error
	GetLogo(ctx context.Context) (*domain.LogoData, error)
}

// TemplateRepository définit les opérations de persistance des templates.
type TemplateRepository interface {
	Create(ctx context.Context, tmpl *domain.Template) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Template, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Template, error)
	Update(ctx context.Context, tmpl *domain.Template) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

// AuditLogRepository définit les opérations de persistance des logs d'audit.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error)
}
