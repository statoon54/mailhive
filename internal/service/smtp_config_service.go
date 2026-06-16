package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
)

// SMTPConfigService implémente port.SMTPConfigService.
type SMTPConfigService struct {
	repo          port.SMTPConfigRepository
	sender        port.MailSender
	encryptionKey []byte
}

// NewSMTPConfigService crée un nouveau service config SMTP.
func NewSMTPConfigService(
	repo port.SMTPConfigRepository,
	sender port.MailSender,
	encryptionKey string,
) (*SMTPConfigService, error) {
	key, err := hex.DecodeString(encryptionKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf(
			"la clé de chiffrement doit être une chaîne hexadécimale de 64 caractères (32 octets)",
		)
	}
	return &SMTPConfigService{
		repo:          repo,
		sender:        sender,
		encryptionKey: key,
	}, nil
}

// Create crée une nouvelle configuration SMTP avec chiffrement du mot de passe.
func (s *SMTPConfigService) Create(
	ctx context.Context,
	tenantID uuid.UUID,
	req domain.CreateSMTPConfigRequest,
) (*domain.SMTPConfig, error) {
	now := time.Now()

	// Chiffrer le mot de passe si fourni
	encryptedPwd := ""
	if req.Password != "" {
		var err error
		encryptedPwd, err = s.encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("erreur de chiffrement du mot de passe : %w", err)
		}
	}

	// Si cette config est la config par défaut, retirer le défaut des autres
	if req.IsDefault {
		if err := s.repo.ClearDefault(ctx, tenantID); err != nil {
			return nil, err
		}
	}

	// Defaults et validation charset/encoding
	charset := req.Charset
	if charset == "" {
		charset = domain.CharsetUTF8
	} else if !domain.ValidMailCharsets[charset] {
		return nil, fmt.Errorf("%w : charset invalide", domain.ErrValidation)
	}
	encoding := req.Encoding
	if encoding == "" {
		encoding = domain.EncodingQP
	} else if !domain.ValidMailEncodings[encoding] {
		return nil, fmt.Errorf("%w : encodage invalide", domain.ErrValidation)
	}

	cfg := &domain.SMTPConfig{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		Password:   encryptedPwd,
		AuthMethod: req.AuthMethod,
		TLSPolicy:  req.TLSPolicy,
		FromEmail:  req.FromEmail,
		FromName:   req.FromName,
		Charset:    charset,
		Encoding:   encoding,
		IsDefault:  req.IsDefault,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetByID retourne une configuration SMTP par son identifiant.
func (s *SMTPConfigService) GetByID(
	ctx context.Context,
	tenantID, id uuid.UUID,
) (*domain.SMTPConfig, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retourne toutes les configurations SMTP d'un tenant.
func (s *SMTPConfigService) List(
	ctx context.Context,
	tenantID uuid.UUID,
) ([]domain.SMTPConfig, error) {
	return s.repo.List(ctx, tenantID)
}

// validateUpdateRequest valide les champs optionnels de la requête de mise à jour SMTP.
func validateUpdateRequest(req domain.UpdateSMTPConfigRequest) error {
	if req.Name != nil && *req.Name == "" {
		return fmt.Errorf("%w : le nom est requis", domain.ErrValidation)
	}
	if req.Host != nil && *req.Host == "" {
		return fmt.Errorf("%w : l'hôte est requis", domain.ErrValidation)
	}
	if req.Port != nil && (*req.Port < 1 || *req.Port > 65535) {
		return fmt.Errorf("%w : le port doit être entre 1 et 65535", domain.ErrValidation)
	}
	if req.AuthMethod != nil {
		switch *req.AuthMethod {
		case domain.AuthPlain, domain.AuthLogin, domain.AuthCRAMMD5, domain.AuthNone:
		default:
			return fmt.Errorf("%w : méthode d'authentification invalide", domain.ErrValidation)
		}
	}
	if req.TLSPolicy != nil {
		switch *req.TLSPolicy {
		case domain.TLSMandatory, domain.TLSOpportunistic, domain.TLSNone:
		default:
			return fmt.Errorf("%w : politique TLS invalide", domain.ErrValidation)
		}
	}
	if req.FromEmail != nil && *req.FromEmail == "" {
		return fmt.Errorf("%w : l'email expéditeur est requis", domain.ErrValidation)
	}
	if req.Charset != nil && !domain.ValidMailCharsets[*req.Charset] {
		return fmt.Errorf("%w : charset invalide", domain.ErrValidation)
	}
	if req.Encoding != nil && !domain.ValidMailEncodings[*req.Encoding] {
		return fmt.Errorf("%w : encodage invalide", domain.ErrValidation)
	}
	return nil
}

// Update met à jour une configuration SMTP existante.
func (s *SMTPConfigService) Update(
	ctx context.Context,
	tenantID, id uuid.UUID,
	req domain.UpdateSMTPConfigRequest,
) (*domain.SMTPConfig, error) {
	if err := validateUpdateRequest(req); err != nil {
		return nil, err
	}

	cfg, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		cfg.Name = *req.Name
	}
	if req.Host != nil {
		cfg.Host = *req.Host
	}
	if req.Port != nil {
		cfg.Port = *req.Port
	}
	if req.Username != nil {
		cfg.Username = req.Username
	}
	if req.Password != nil && *req.Password != "" {
		encrypted, err := s.encrypt(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("erreur de chiffrement du mot de passe : %w", err)
		}
		cfg.Password = encrypted
	}
	if req.AuthMethod != nil {
		cfg.AuthMethod = *req.AuthMethod
	}
	if req.TLSPolicy != nil {
		cfg.TLSPolicy = *req.TLSPolicy
	}
	if req.FromEmail != nil {
		cfg.FromEmail = *req.FromEmail
	}
	if req.FromName != nil {
		cfg.FromName = *req.FromName
	}
	if req.Charset != nil {
		cfg.Charset = *req.Charset
	}
	if req.Encoding != nil {
		cfg.Encoding = *req.Encoding
	}
	if req.IsDefault != nil && *req.IsDefault {
		if err := s.repo.ClearDefault(ctx, tenantID); err != nil {
			return nil, err
		}
		cfg.IsDefault = true
	}
	if req.IsActive != nil {
		cfg.IsActive = *req.IsActive
	}
	cfg.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Delete supprime une configuration SMTP par son identifiant.
func (s *SMTPConfigService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// Test envoie un mail de test via la configuration SMTP spécifiée.
func (s *SMTPConfigService) Test(ctx context.Context, tenantID, id uuid.UUID) error {
	cfg, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Déchiffrer le mot de passe pour le test
	if cfg.Password != "" {
		decrypted, err := s.decrypt(cfg.Password)
		if err != nil {
			return fmt.Errorf("erreur de déchiffrement du mot de passe : %w", err)
		}
		cfg.Password = decrypted
	}

	// Créer un mail de test
	testMail := &domain.Mail{
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		Subject:   "Test de configuration SMTP - MailHive",
		TextBody:  "Ceci est un mail de test envoyé depuis MailHive pour vérifier la configuration SMTP.",
		Recipients: []domain.MailRecipient{
			{Email: cfg.FromEmail, Type: domain.RecipientTo, Name: cfg.FromName},
		},
	}

	return s.sender.Send(ctx, cfg, testMail)
}

// DecryptPassword déchiffre le mot de passe d'une config SMTP.
func (s *SMTPConfigService) DecryptPassword(encrypted string) (string, error) {
	return s.decrypt(encrypted)
}

// encrypt chiffre une chaîne en AES-GCM et retourne le résultat en hexadécimal.
func (s *SMTPConfigService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decrypt déchiffre une chaîne hexadécimale chiffrée en AES-GCM.
func (s *SMTPConfigService) decrypt(ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("texte chiffré trop court")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
