package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// RateLimiter définit les opérations de rate limiting par tenant.
type RateLimiter interface {
	// Allow vérifie si une requête est autorisée pour le tenant donné.
	// rateLimit est le nombre de tokens par seconde, burst est la capacité maximale du bucket.
	Allow(ctx context.Context, tenantID uuid.UUID, rateLimit float64, burst int) (bool, error)
}

// AuthService définit les opérations d'authentification.
type AuthService interface {
	GenerateToken(ctx context.Context, apiKey string) (string, error)
	RefreshToken(ctx context.Context, tokenString string) (string, error)
}

// TenantService définit les opérations métier sur les tenants.
type TenantService interface {
	Create(ctx context.Context, req domain.CreateTenantRequest) (*domain.Tenant, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	List(ctx context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		req domain.UpdateTenantRequest,
	) (*domain.Tenant, error)
	Delete(ctx context.Context, id uuid.UUID) error
	RegenerateAPIKey(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
}

// MailService définit les opérations métier sur les mails.
type MailService interface {
	Create(
		ctx context.Context,
		tenantID uuid.UUID,
		req domain.CreateMailRequest,
	) ([]*domain.Mail, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Mail, error)
	// DownloadAttachment retourne les métadonnées et le contenu d'une pièce jointe
	// d'un mail, après vérification de l'appartenance au tenant et du lien.
	DownloadAttachment(ctx context.Context, tenantID, mailID, attachmentID uuid.UUID) (*domain.AttachmentRef, []byte, error)
	List(
		ctx context.Context,
		tenantID uuid.UUID,
		filter domain.MailListFilter,
	) (*domain.PaginatedList[domain.Mail], error)
	Cancel(ctx context.Context, tenantID, id uuid.UUID) error
	Retry(ctx context.Context, tenantID, id uuid.UUID) error
	Stats(ctx context.Context, tenantID uuid.UUID) (*domain.MailStats, error)
	StatsByTenant(ctx context.Context) ([]domain.TenantMailStats, error)
}

// TemplateService définit les opérations métier sur les templates.
type TemplateService interface {
	Create(
		ctx context.Context,
		tenantID uuid.UUID,
		req domain.CreateTemplateRequest,
	) (*domain.Template, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Template, error)
	Update(
		ctx context.Context,
		tenantID, id uuid.UUID,
		req domain.UpdateTemplateRequest,
	) (*domain.Template, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Preview(
		ctx context.Context,
		tenantID, id uuid.UUID,
		data map[string]string,
	) (*domain.PreviewTemplateResponse, error)
}

// SMTPConfigService définit les opérations métier sur les configs SMTP.
type SMTPConfigService interface {
	Create(
		ctx context.Context,
		tenantID uuid.UUID,
		req domain.CreateSMTPConfigRequest,
	) (*domain.SMTPConfig, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.SMTPConfig, error)
	Update(
		ctx context.Context,
		tenantID, id uuid.UUID,
		req domain.UpdateSMTPConfigRequest,
	) (*domain.SMTPConfig, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Test(ctx context.Context, tenantID, id uuid.UUID) error
}

// BrandingService définit les opérations métier sur le branding.
type BrandingService interface {
	Get(ctx context.Context) (*domain.AppBranding, error)
	Update(ctx context.Context, req domain.UpdateBrandingRequest) (*domain.AppBranding, error)
	UploadLogo(ctx context.Context, data []byte, contentType string) error
	GetLogo(ctx context.Context) (*domain.LogoData, error)
}

// AuditLogService définit les opérations métier sur les logs d'audit.
type AuditLogService interface {
	Log(
		tenantID uuid.UUID,
		action, resourceType, resourceID, status string,
		statusCode int,
		errorMessage, details string,
		method, path string,
	)
	List(
		ctx context.Context,
		filter domain.AuditLogFilter,
	) (*domain.PaginatedList[domain.AuditLog], error)
}

// AnalysisService définit les opérations d'analyse de templates.
type AnalysisService interface {
	SpamCheck(ctx context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.SpamCheckResult, error)
	HTMLCheck(ctx context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.HTMLCheckResult, error)
	LinkCheck(ctx context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.LinkCheckResult, error)
	ComputeSpamScore(subject, textBody, htmlBody string) float32
}

// MailSender définit l'interface d'envoi de mails via SMTP.
type MailSender interface {
	Send(ctx context.Context, cfg *domain.SMTPConfig, mail *domain.Mail) error
}

// QueueClient définit l'interface de mise en file d'attente.
type QueueClient interface {
	EnqueueMailSend(
		ctx context.Context,
		mailID, tenantID uuid.UUID,
		priority domain.MailPriority,
		scheduledAt *time.Time,
	) (string, error)
	DeleteTask(queue, taskID string) error
}
