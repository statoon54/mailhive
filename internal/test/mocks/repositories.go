package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
)

// --- MockTenantRepo ---

type MockTenantRepo struct {
	CallRecorder
	mu      sync.RWMutex
	Tenants map[uuid.UUID]*domain.Tenant
	Err     error // error injection
}

func NewMockTenantRepo() *MockTenantRepo {
	return &MockTenantRepo{Tenants: make(map[uuid.UUID]*domain.Tenant)}
}

func (m *MockTenantRepo) Create(_ context.Context, t *domain.Tenant) error {
	m.Record("Create", t)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tenants[t.ID] = t
	return nil
}

func (m *MockTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	m.Record("GetByID", id)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.Tenants[id]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	return t, nil
}

func (m *MockTenantRepo) GetByAPIKey(_ context.Context, apiKey string) (*domain.Tenant, error) {
	m.Record("GetByAPIKey", apiKey)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.Tenants {
		if t.APIKey == apiKey {
			return t, nil
		}
	}
	return nil, domain.ErrTenantNotFound
}

func (m *MockTenantRepo) GetBySlug(_ context.Context, slug string) (*domain.Tenant, error) {
	m.Record("GetBySlug", slug)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.Tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, domain.ErrTenantNotFound
}

func (m *MockTenantRepo) List(_ context.Context, page, limit int) (*domain.PaginatedList[domain.Tenant], error) {
	m.Record("List", page, limit)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var items []domain.Tenant
	for _, t := range m.Tenants {
		items = append(items, *t)
	}
	return &domain.PaginatedList[domain.Tenant]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       page,
		Limit:      limit,
		TotalPages: 1,
	}, nil
}

func (m *MockTenantRepo) Update(_ context.Context, t *domain.Tenant) error {
	m.Record("Update", t)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tenants[t.ID] = t
	return nil
}

func (m *MockTenantRepo) UpdateAPIKey(_ context.Context, id uuid.UUID, newKey string) error {
	m.Record("UpdateAPIKey", id, newKey)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.Tenants[id]; ok {
		t.APIKey = newKey
	}
	return nil
}

func (m *MockTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.Record("Delete", id)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Tenants, id)
	return nil
}

// --- MockMailRepo ---

type MockMailRepo struct {
	CallRecorder
	mu              sync.RWMutex
	Mails           map[uuid.UUID]*domain.Mail
	Recipients      map[uuid.UUID][]domain.MailRecipient
	AttachmentLinks map[uuid.UUID][]domain.AttachmentLink
	Err             error
}

func NewMockMailRepo() *MockMailRepo {
	return &MockMailRepo{
		Mails:           make(map[uuid.UUID]*domain.Mail),
		Recipients:      make(map[uuid.UUID][]domain.MailRecipient),
		AttachmentLinks: make(map[uuid.UUID][]domain.AttachmentLink),
	}
}

func (m *MockMailRepo) Create(_ context.Context, mail *domain.Mail) error {
	m.Record("Create", mail)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Mails[mail.ID] = mail
	return nil
}

func (m *MockMailRepo) CreateBatch(_ context.Context, mails []*domain.Mail) error {
	m.Record("CreateBatch", mails)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mail := range mails {
		m.Mails[mail.ID] = mail
	}
	return nil
}

func (m *MockMailRepo) CreateWithRecipients(_ context.Context, mail *domain.Mail, recipients []domain.MailRecipient, links []domain.AttachmentLink) error {
	m.Record("CreateWithRecipients", mail, recipients, links)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Mails[mail.ID] = mail
	for _, r := range recipients {
		m.Recipients[r.MailID] = append(m.Recipients[r.MailID], r)
	}
	m.AttachmentLinks[mail.ID] = append(m.AttachmentLinks[mail.ID], links...)
	return nil
}

func (m *MockMailRepo) CreateBatchWithRecipients(_ context.Context, mails []*domain.Mail, recipients []domain.MailRecipient, links []domain.AttachmentLink) error {
	m.Record("CreateBatchWithRecipients", mails, recipients, links)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mail := range mails {
		m.Mails[mail.ID] = mail
		m.AttachmentLinks[mail.ID] = append(m.AttachmentLinks[mail.ID], links...)
	}
	for _, r := range recipients {
		m.Recipients[r.MailID] = append(m.Recipients[r.MailID], r)
	}
	return nil
}

func (m *MockMailRepo) GetAttachmentRefs(_ context.Context, mailID uuid.UUID) ([]domain.AttachmentRef, error) {
	m.Record("GetAttachmentRefs", mailID)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	links := m.AttachmentLinks[mailID]
	refs := make([]domain.AttachmentRef, 0, len(links))
	for _, l := range links {
		refs = append(refs, domain.AttachmentRef{
			AttachmentID: l.AttachmentID,
			Filename:     l.Filename,
		})
	}
	return refs, nil
}

func (m *MockMailRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Mail, error) {
	m.Record("GetByID", tenantID, id)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	mail, ok := m.Mails[id]
	if !ok || mail.TenantID != tenantID {
		return nil, domain.ErrMailNotFound
	}
	return mail, nil
}

func (m *MockMailRepo) List(_ context.Context, tenantID uuid.UUID, filter domain.MailListFilter) (*domain.PaginatedList[domain.Mail], error) {
	m.Record("List", tenantID, filter)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var items []domain.Mail
	for _, mail := range m.Mails {
		if mail.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && mail.Status != *filter.Status {
			continue
		}
		items = append(items, *mail)
	}
	return &domain.PaginatedList[domain.Mail]{
		Items:      items,
		Total:      int64(len(items)),
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: 1,
	}, nil
}

func (m *MockMailRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.MailStatus, message string) error {
	m.Record("UpdateStatus", id, status, message)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.Status = status
		mail.StatusMessage = message
	}
	return nil
}

func (m *MockMailRepo) UpdateStatuses(_ context.Context, ids []uuid.UUID, status domain.MailStatus, message string) error {
	m.Record("UpdateStatuses", ids, status, message)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if mail, ok := m.Mails[id]; ok {
			mail.Status = status
			mail.StatusMessage = message
		}
	}
	return nil
}

func (m *MockMailRepo) SetQueued(_ context.Context, id uuid.UUID, taskID string) error {
	m.Record("SetQueued", id, taskID)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.TaskID = taskID
		mail.Status = domain.MailStatusQueued
	}
	return nil
}

func (m *MockMailRepo) SetQueuedBatch(_ context.Context, mailTaskIDs map[uuid.UUID]string) error {
	m.Record("SetQueuedBatch", mailTaskIDs)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, taskID := range mailTaskIDs {
		if mail, ok := m.Mails[id]; ok {
			mail.TaskID = taskID
			mail.Status = domain.MailStatusQueued
		}
	}
	return nil
}

func (m *MockMailRepo) UpdateTaskID(_ context.Context, id uuid.UUID, taskID string) error {
	m.Record("UpdateTaskID", id, taskID)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.TaskID = taskID
	}
	return nil
}

func (m *MockMailRepo) UpdateTaskIDs(_ context.Context, mailTaskIDs map[uuid.UUID]string) error {
	m.Record("UpdateTaskIDs", mailTaskIDs)
	return nil
}

func (m *MockMailRepo) MarkSending(_ context.Context, id uuid.UUID) error {
	m.Record("MarkSending", id)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.Status = domain.MailStatusSending
		mail.StatusMessage = ""
		mail.Attempts++
	}
	return nil
}

func (m *MockMailRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	m.Record("MarkSent", id)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.Status = domain.MailStatusSent
		mail.StatusMessage = ""
		now := time.Now()
		mail.SentAt = &now
	}
	return nil
}

func (m *MockMailRepo) Stats(_ context.Context, _ uuid.UUID) (*domain.MailStats, error) {
	m.Record("Stats")
	return &domain.MailStats{}, nil
}

func (m *MockMailRepo) StatsByTenant(_ context.Context) ([]domain.TenantMailStats, error) {
	m.Record("StatsByTenant")
	return nil, nil
}

func (m *MockMailRepo) CreateRecipients(_ context.Context, recipients []domain.MailRecipient) error {
	m.Record("CreateRecipients", recipients)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range recipients {
		m.Recipients[r.MailID] = append(m.Recipients[r.MailID], r)
	}
	return nil
}

func (m *MockMailRepo) GetRecipients(_ context.Context, mailID uuid.UUID) ([]domain.MailRecipient, error) {
	m.Record("GetRecipients", mailID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Recipients[mailID], nil
}

func (m *MockMailRepo) GetRecipientsByMailIDs(_ context.Context, mailIDs []uuid.UUID) (map[uuid.UUID][]domain.MailRecipient, error) {
	m.Record("GetRecipientsByMailIDs", mailIDs)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[uuid.UUID][]domain.MailRecipient, len(mailIDs))
	for _, id := range mailIDs {
		if recs, ok := m.Recipients[id]; ok {
			result[id] = recs
		}
	}
	return result, nil
}

func (m *MockMailRepo) AddTags(_ context.Context, id uuid.UUID, tags []string) error {
	m.Record("AddTags", id, tags)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		existing := make(map[string]bool)
		for _, t := range mail.Tags {
			existing[t] = true
		}
		for _, t := range tags {
			if !existing[t] {
				mail.Tags = append(mail.Tags, t)
			}
		}
	}
	return nil
}

func (m *MockMailRepo) ClearBodies(_ context.Context, id uuid.UUID, compressedBody []byte) error {
	m.Record("ClearBodies", id, compressedBody)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail, ok := m.Mails[id]; ok {
		mail.TextBody = ""
		mail.HTMLBody = ""
		mail.CompressedBody = compressedBody
	}
	return nil
}

// --- MockSMTPConfigRepo ---

type MockSMTPConfigRepo struct {
	CallRecorder
	mu      sync.RWMutex
	Configs map[uuid.UUID]*domain.SMTPConfig
	Err     error
}

func NewMockSMTPConfigRepo() *MockSMTPConfigRepo {
	return &MockSMTPConfigRepo{Configs: make(map[uuid.UUID]*domain.SMTPConfig)}
}

func (m *MockSMTPConfigRepo) Create(_ context.Context, cfg *domain.SMTPConfig) error {
	m.Record("Create", cfg)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Configs[cfg.ID] = cfg
	return nil
}

func (m *MockSMTPConfigRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.SMTPConfig, error) {
	m.Record("GetByID", tenantID, id)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.Configs[id]
	if !ok {
		return nil, domain.ErrSMTPConfigNotFound
	}
	return cfg, nil
}

func (m *MockSMTPConfigRepo) GetDefault(_ context.Context, tenantID uuid.UUID) (*domain.SMTPConfig, error) {
	m.Record("GetDefault", tenantID)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cfg := range m.Configs {
		if cfg.TenantID == tenantID && cfg.IsDefault {
			return cfg, nil
		}
	}
	return nil, domain.ErrSMTPConfigNotSet
}

func (m *MockSMTPConfigRepo) List(_ context.Context, tenantID uuid.UUID) ([]domain.SMTPConfig, error) {
	m.Record("List", tenantID)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.SMTPConfig
	for _, cfg := range m.Configs {
		if cfg.TenantID == tenantID {
			result = append(result, *cfg)
		}
	}
	return result, nil
}

func (m *MockSMTPConfigRepo) Update(_ context.Context, cfg *domain.SMTPConfig) error {
	m.Record("Update", cfg)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Configs[cfg.ID] = cfg
	return nil
}

func (m *MockSMTPConfigRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Delete", tenantID, id)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Configs, id)
	return nil
}

func (m *MockSMTPConfigRepo) ClearDefault(_ context.Context, tenantID uuid.UUID) error {
	m.Record("ClearDefault", tenantID)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cfg := range m.Configs {
		if cfg.TenantID == tenantID {
			cfg.IsDefault = false
		}
	}
	return nil
}

// --- MockTemplateRepo ---

type MockTemplateRepo struct {
	CallRecorder
	mu        sync.RWMutex
	Templates map[uuid.UUID]*domain.Template
	Err       error
}

func NewMockTemplateRepo() *MockTemplateRepo {
	return &MockTemplateRepo{Templates: make(map[uuid.UUID]*domain.Template)}
}

func (m *MockTemplateRepo) Create(_ context.Context, tmpl *domain.Template) error {
	m.Record("Create", tmpl)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Templates[tmpl.ID] = tmpl
	return nil
}

func (m *MockTemplateRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	m.Record("GetByID", tenantID, id)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	tmpl, ok := m.Templates[id]
	if !ok || tmpl.TenantID != tenantID {
		return nil, domain.ErrTemplateNotFound
	}
	return tmpl, nil
}

func (m *MockTemplateRepo) GetBySlug(_ context.Context, tenantID uuid.UUID, slug string) (*domain.Template, error) {
	m.Record("GetBySlug", tenantID, slug)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, tmpl := range m.Templates {
		if tmpl.TenantID == tenantID && tmpl.Slug == slug {
			return tmpl, nil
		}
	}
	return nil, domain.ErrTemplateNotFound
}

func (m *MockTemplateRepo) List(_ context.Context, tenantID uuid.UUID) ([]domain.Template, error) {
	m.Record("List", tenantID)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.Template
	for _, tmpl := range m.Templates {
		if tmpl.TenantID == tenantID {
			result = append(result, *tmpl)
		}
	}
	return result, nil
}

func (m *MockTemplateRepo) Update(_ context.Context, tmpl *domain.Template) error {
	m.Record("Update", tmpl)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Templates[tmpl.ID] = tmpl
	return nil
}

func (m *MockTemplateRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	m.Record("Delete", tenantID, id)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Templates, id)
	return nil
}

// --- MockBrandingRepo ---

type MockBrandingRepo struct {
	CallRecorder
	Branding *domain.AppBranding
	Logo     *domain.LogoData
	Err      error
}

func NewMockBrandingRepo() *MockBrandingRepo {
	return &MockBrandingRepo{
		Branding: &domain.AppBranding{
			AppTitle:    "MailHive",
			AppSubtitle: "Mail API",
			Timezone:    "Europe/Paris",
		},
	}
}

func (m *MockBrandingRepo) Get(_ context.Context) (*domain.AppBranding, error) {
	m.Record("Get")
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Branding, nil
}

func (m *MockBrandingRepo) Update(_ context.Context, branding *domain.AppBranding) error {
	m.Record("Update", branding)
	if m.Err != nil {
		return m.Err
	}
	m.Branding = branding
	return nil
}

func (m *MockBrandingRepo) UpdateLogo(_ context.Context, data []byte, contentType string) error {
	m.Record("UpdateLogo", data, contentType)
	if m.Err != nil {
		return m.Err
	}
	m.Logo = &domain.LogoData{Data: data, ContentType: contentType}
	return nil
}

func (m *MockBrandingRepo) GetLogo(_ context.Context) (*domain.LogoData, error) {
	m.Record("GetLogo")
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Logo, nil
}

// --- MockAuditLogRepo ---

type MockAuditLogRepo struct {
	CallRecorder
	mu   sync.Mutex
	Logs []domain.AuditLog
	Err  error
}

func NewMockAuditLogRepo() *MockAuditLogRepo {
	return &MockAuditLogRepo{}
}

func (m *MockAuditLogRepo) Create(_ context.Context, log *domain.AuditLog) error {
	m.Record("Create", log)
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Logs = append(m.Logs, *log)
	return nil
}

func (m *MockAuditLogRepo) List(_ context.Context, filter domain.AuditLogFilter) (*domain.PaginatedList[domain.AuditLog], error) {
	m.Record("List", filter)
	if m.Err != nil {
		return nil, m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return &domain.PaginatedList[domain.AuditLog]{
		Items:      m.Logs,
		Total:      int64(len(m.Logs)),
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: 1,
	}, nil
}
