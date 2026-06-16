package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// --- MockAuthService ---

type MockAuthService struct {
	CallRecorder
	Token string
	Err   error
}

func (m *MockAuthService) GenerateToken(_ context.Context, apiKey string) (string, error) {
	m.Record("GenerateToken", apiKey)
	return m.Token, m.Err
}

func (m *MockAuthService) RefreshToken(_ context.Context, tokenString string) (string, error) {
	m.Record("RefreshToken", tokenString)
	return m.Token, m.Err
}

// --- MockTenantService ---

type MockTenantService struct {
	CallRecorder
	Tenant  *domain.Tenant
	Tenants *domain.PaginatedList[domain.Tenant]
	Err     error
}

func (m *MockTenantService) Create(_ context.Context, req domain.CreateTenantRequest) (*domain.Tenant, error) {
	m.Record("Create", req)
	return m.Tenant, m.Err
}

func (m *MockTenantService) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	m.Record("GetByID", id)
	return m.Tenant, m.Err
}

func (m *MockTenantService) List(_ context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error) {
	m.Record("List", page, limit)
	return m.Tenants, m.Err
}

func (m *MockTenantService) Update(_ context.Context, id uuid.UUID, req domain.UpdateTenantRequest) (*domain.Tenant, error) {
	m.Record("Update", id, req)
	return m.Tenant, m.Err
}

func (m *MockTenantService) Delete(_ context.Context, id uuid.UUID) error {
	m.Record("Delete", id)
	return m.Err
}

func (m *MockTenantService) RegenerateAPIKey(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	m.Record("RegenerateAPIKey", id)
	return m.Tenant, m.Err
}

// --- MockMailService ---

type MockMailService struct {
	CallRecorder
	Mails       []*domain.Mail
	Mail        *domain.Mail
	List_       *domain.PaginatedList[domain.Mail]
	Stats_      *domain.MailStats
	TenantStats []domain.TenantMailStats
	AttachRef   *domain.AttachmentRef
	AttachData  []byte
	Err         error
}

func (m *MockMailService) Create(_ context.Context, tenantID uuid.UUID, req domain.CreateMailRequest) ([]*domain.Mail, error) {
	m.Record("Create", tenantID, req)
	return m.Mails, m.Err
}

func (m *MockMailService) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Mail, error) {
	m.Record("GetByID", tenantID, id)
	return m.Mail, m.Err
}

func (m *MockMailService) DownloadAttachment(_ context.Context, tenantID, mailID, attachmentID uuid.UUID) (*domain.AttachmentRef, []byte, error) {
	m.Record("DownloadAttachment", tenantID, mailID, attachmentID)
	return m.AttachRef, m.AttachData, m.Err
}

func (m *MockMailService) List(_ context.Context, tenantID uuid.UUID, filter domain.MailListFilter) (*domain.PaginatedList[domain.Mail], error) {
	m.Record("List", tenantID, filter)
	return m.List_, m.Err
}

func (m *MockMailService) Cancel(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Cancel", tenantID, id)
	return m.Err
}

func (m *MockMailService) Retry(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Retry", tenantID, id)
	return m.Err
}

func (m *MockMailService) Stats(_ context.Context, tenantID uuid.UUID) (*domain.MailStats, error) {
	m.Record("Stats", tenantID)
	return m.Stats_, m.Err
}

func (m *MockMailService) StatsByTenant(_ context.Context) ([]domain.TenantMailStats, error) {
	m.Record("StatsByTenant")
	return m.TenantStats, m.Err
}

// --- MockTemplateService ---

type MockTemplateService struct {
	CallRecorder
	Template  *domain.Template
	Templates []domain.Template
	Preview_  *domain.PreviewTemplateResponse
	Err       error
}

func (m *MockTemplateService) Create(_ context.Context, tenantID uuid.UUID, req domain.CreateTemplateRequest) (*domain.Template, error) {
	m.Record("Create", tenantID, req)
	return m.Template, m.Err
}

func (m *MockTemplateService) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	m.Record("GetByID", tenantID, id)
	return m.Template, m.Err
}

func (m *MockTemplateService) List(_ context.Context, tenantID uuid.UUID) ([]domain.Template, error) {
	m.Record("List", tenantID)
	return m.Templates, m.Err
}

func (m *MockTemplateService) Update(_ context.Context, tenantID, id uuid.UUID, req domain.UpdateTemplateRequest) (*domain.Template, error) {
	m.Record("Update", tenantID, id, req)
	return m.Template, m.Err
}

func (m *MockTemplateService) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Delete", tenantID, id)
	return m.Err
}

func (m *MockTemplateService) Preview(_ context.Context, tenantID, id uuid.UUID, data map[string]string) (*domain.PreviewTemplateResponse, error) {
	m.Record("Preview", tenantID, id, data)
	return m.Preview_, m.Err
}

// --- MockSMTPConfigService ---

type MockSMTPConfigService struct {
	CallRecorder
	Config  *domain.SMTPConfig
	Configs []domain.SMTPConfig
	Err     error
}

func (m *MockSMTPConfigService) Create(_ context.Context, tenantID uuid.UUID, req domain.CreateSMTPConfigRequest) (*domain.SMTPConfig, error) {
	m.Record("Create", tenantID, req)
	return m.Config, m.Err
}

func (m *MockSMTPConfigService) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error) {
	m.Record("GetByID", tenantID, id)
	return m.Config, m.Err
}

func (m *MockSMTPConfigService) List(_ context.Context, tenantID uuid.UUID) ([]domain.SMTPConfig, error) {
	m.Record("List", tenantID)
	return m.Configs, m.Err
}

func (m *MockSMTPConfigService) Update(_ context.Context, tenantID, id uuid.UUID, req domain.UpdateSMTPConfigRequest) (*domain.SMTPConfig, error) {
	m.Record("Update", tenantID, id, req)
	return m.Config, m.Err
}

func (m *MockSMTPConfigService) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Delete", tenantID, id)
	return m.Err
}

func (m *MockSMTPConfigService) Test(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Test", tenantID, id)
	return m.Err
}

// --- MockBrandingService ---

type MockBrandingService struct {
	CallRecorder
	Branding *domain.AppBranding
	Logo     *domain.LogoData
	Err      error
}

func (m *MockBrandingService) Get(_ context.Context) (*domain.AppBranding, error) {
	m.Record("Get")
	return m.Branding, m.Err
}

func (m *MockBrandingService) Update(_ context.Context, req domain.UpdateBrandingRequest) (*domain.AppBranding, error) {
	m.Record("Update", req)
	return m.Branding, m.Err
}

func (m *MockBrandingService) UploadLogo(_ context.Context, data []byte, contentType string) error {
	m.Record("UploadLogo", data, contentType)
	return m.Err
}

func (m *MockBrandingService) GetLogo(_ context.Context) (*domain.LogoData, error) {
	m.Record("GetLogo")
	return m.Logo, m.Err
}

// --- MockAuditLogService ---

type MockAuditLogService struct {
	CallRecorder
	List_ *domain.PaginatedList[domain.AuditLog]
	Err   error
}

func (m *MockAuditLogService) Log(tenantID uuid.UUID, action, resourceType, resourceID, status string, statusCode int, errorMessage, details string, method, path string) {
	m.Record("Log", tenantID, action, resourceType, resourceID, status, statusCode, errorMessage, details, method, path)
}

func (m *MockAuditLogService) List(_ context.Context, filter domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	m.Record("List", filter)
	return m.List_, m.Err
}

// --- MockAnalysisService ---

type MockAnalysisService struct {
	CallRecorder
	SpamResult *domain.SpamCheckResult
	HTMLResult *domain.HTMLCheckResult
	LinkResult *domain.LinkCheckResult
	SpamScore  float32
	Err        error
}

func (m *MockAnalysisService) SpamCheck(_ context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.SpamCheckResult, error) {
	m.Record("SpamCheck", tenantID, templateID, data)
	return m.SpamResult, m.Err
}

func (m *MockAnalysisService) HTMLCheck(_ context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.HTMLCheckResult, error) {
	m.Record("HTMLCheck", tenantID, templateID, data)
	return m.HTMLResult, m.Err
}

func (m *MockAnalysisService) LinkCheck(_ context.Context, tenantID, templateID uuid.UUID, data map[string]string) (*domain.LinkCheckResult, error) {
	m.Record("LinkCheck", tenantID, templateID, data)
	return m.LinkResult, m.Err
}

func (m *MockAnalysisService) ComputeSpamScore(subject, textBody, htmlBody string) float32 {
	m.Record("ComputeSpamScore", subject, textBody, htmlBody)
	return m.SpamScore
}

// --- MockRateLimiter ---

type MockRateLimiter struct {
	CallRecorder
	Allowed bool
	Err     error
}

func (m *MockRateLimiter) Allow(_ context.Context, tenantID uuid.UUID, rateLimit float64, burst int) (bool, error) {
	m.Record("Allow", tenantID, rateLimit, burst)
	if m.Err != nil {
		return false, m.Err
	}
	return m.Allowed, nil
}

// --- MockMailSender ---

type MockMailSender struct {
	CallRecorder
	Err error
}

func (m *MockMailSender) Send(_ context.Context, cfg *domain.SMTPConfig, mail *domain.Mail) error {
	m.Record("Send", cfg, mail)
	return m.Err
}

// --- MockQueueClient ---

type MockQueueClient struct {
	CallRecorder
	TaskID string
	Err    error
}

func (m *MockQueueClient) EnqueueMailSend(_ context.Context, mailID, tenantID uuid.UUID, priority domain.MailPriority, scheduledAt *time.Time) (string, error) {
	m.Record("EnqueueMailSend", mailID, tenantID, priority, scheduledAt)
	if m.Err != nil {
		return "", m.Err
	}
	taskID := m.TaskID
	if taskID == "" {
		taskID = "task-" + mailID.String()
	}
	return taskID, nil
}

func (m *MockQueueClient) DeleteTask(queue, taskID string) error {
	m.Record("DeleteTask", queue, taskID)
	return m.Err
}
