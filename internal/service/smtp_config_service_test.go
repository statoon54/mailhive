package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

const validEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestSMTPConfigService_New_Valid(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, err := NewSMTPConfigService(repo, sender, validEncryptionKey)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestSMTPConfigService_New_InvalidKey(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	_, err := NewSMTPConfigService(repo, sender, "too-short")
	require.Error(t, err)
}

func TestSMTPConfigService_Create_EncryptsPassword(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)
	tenantID := uuid.New()

	cfg, err := svc.Create(context.Background(), tenantID, domain.CreateSMTPConfigRequest{
		Name:       "Test",
		Host:       "smtp.example.com",
		Port:       587,
		Password:   "secret123",
		AuthMethod: domain.AuthPlain,
		TLSPolicy:  domain.TLSOpportunistic,
		FromEmail:  "test@example.com",
	})
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", cfg.Password)
	assert.NotEmpty(t, cfg.Password)
}

func TestSMTPConfigService_Create_IsDefaultClearsPrevious(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)
	tenantID := uuid.New()

	existingID := uuid.New()
	repo.Configs[existingID] = &domain.SMTPConfig{
		ID: existingID, TenantID: tenantID, IsDefault: true,
	}

	_, err := svc.Create(context.Background(), tenantID, domain.CreateSMTPConfigRequest{
		Name:       "New Default",
		Host:       "smtp.example.com",
		Port:       587,
		AuthMethod: domain.AuthPlain,
		TLSPolicy:  domain.TLSOpportunistic,
		FromEmail:  "test@example.com",
		IsDefault:  true,
	})
	require.NoError(t, err)
	assert.True(t, repo.Called("ClearDefault"))
}

func TestSMTPConfigService_Create_CharsetDefault(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)

	cfg, err := svc.Create(context.Background(), uuid.New(), domain.CreateSMTPConfigRequest{
		Name:       "Test",
		Host:       "smtp.example.com",
		Port:       587,
		AuthMethod: domain.AuthPlain,
		TLSPolicy:  domain.TLSOpportunistic,
		FromEmail:  "test@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.CharsetUTF8, cfg.Charset)
	assert.Equal(t, domain.EncodingQP, cfg.Encoding)
}

func TestSMTPConfigService_Create_InvalidCharset(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateSMTPConfigRequest{
		Name:       "Test",
		Host:       "smtp.example.com",
		Port:       587,
		AuthMethod: domain.AuthPlain,
		TLSPolicy:  domain.TLSOpportunistic,
		FromEmail:  "test@example.com",
		Charset:    "INVALID",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestSMTPConfigService_Update_Partial(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)
	tenantID := uuid.New()
	cfgID := uuid.New()
	repo.Configs[cfgID] = &domain.SMTPConfig{
		ID:       cfgID,
		TenantID: tenantID,
		Name:     "Old Name",
		Host:     "old.host.com",
		Port:     25,
	}

	newName := "New Name"
	cfg, err := svc.Update(context.Background(), tenantID, cfgID, domain.UpdateSMTPConfigRequest{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", cfg.Name)
	assert.Equal(t, "old.host.com", cfg.Host)
}

func TestSMTPConfigService_Update_ReEncryptsPassword(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)
	tenantID := uuid.New()
	cfgID := uuid.New()
	repo.Configs[cfgID] = &domain.SMTPConfig{
		ID: cfgID, TenantID: tenantID, Password: "old-encrypted",
	}

	newPwd := "new-password"
	cfg, err := svc.Update(context.Background(), tenantID, cfgID, domain.UpdateSMTPConfigRequest{
		Password: &newPwd,
	})
	require.NoError(t, err)
	assert.NotEqual(t, "new-password", cfg.Password)
	assert.NotEqual(t, "old-encrypted", cfg.Password)
}

func TestSMTPConfigService_EncryptDecrypt_RoundTrip(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)

	encrypted, err := svc.encrypt("my-secret-password")
	require.NoError(t, err)
	assert.NotEqual(t, "my-secret-password", encrypted)

	decrypted, err := svc.decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "my-secret-password", decrypted)
}

func TestSMTPConfigService_DecryptPassword(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)

	encrypted, _ := svc.encrypt("test123")
	decrypted, err := svc.DecryptPassword(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "test123", decrypted)
}

func TestSMTPConfigService_Test_CallsSender(t *testing.T) {
	repo := mocks.NewMockSMTPConfigRepo()
	sender := &mocks.MockMailSender{}
	svc, _ := NewSMTPConfigService(repo, sender, validEncryptionKey)
	tenantID := uuid.New()
	cfgID := uuid.New()
	repo.Configs[cfgID] = &domain.SMTPConfig{
		ID:        cfgID,
		TenantID:  tenantID,
		FromEmail: "test@example.com",
	}

	err := svc.Test(context.Background(), tenantID, cfgID)
	require.NoError(t, err)
	assert.True(t, sender.Called("Send"))
}
