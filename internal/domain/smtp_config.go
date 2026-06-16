package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuthMethod représente la méthode d'authentification SMTP.
type AuthMethod string

const (
	AuthPlain   AuthMethod = "PLAIN"
	AuthLogin   AuthMethod = "LOGIN"
	AuthCRAMMD5 AuthMethod = "CRAM-MD5"
	AuthNone    AuthMethod = "NONE"
)

// TLSPolicy représente la politique TLS pour SMTP.
type TLSPolicy string

const (
	TLSMandatory     TLSPolicy = "mandatory"
	TLSOpportunistic TLSPolicy = "opportunistic"
	TLSNone          TLSPolicy = "none"
)

// MailCharset représente le jeu de caractères du mail.
type MailCharset string

const (
	CharsetUTF8      MailCharset = "UTF-8"
	CharsetASCII     MailCharset = "US-ASCII"
	CharsetISO88591  MailCharset = "ISO-8859-1"
	CharsetISO885915 MailCharset = "ISO-8859-15"
)

// ValidMailCharsets contient les charsets autorisés.
var ValidMailCharsets = map[MailCharset]bool{
	CharsetUTF8:      true,
	CharsetASCII:     true,
	CharsetISO88591:  true,
	CharsetISO885915: true,
}

// MailEncoding représente l'encodage de transfert du contenu.
type MailEncoding string

const (
	EncodingQP     MailEncoding = "quoted-printable"
	EncodingBase64 MailEncoding = "base64"
	Encoding7Bit   MailEncoding = "7bit"
	Encoding8Bit   MailEncoding = "8bit"
)

// ValidMailEncodings contient les encodages autorisés.
var ValidMailEncodings = map[MailEncoding]bool{
	EncodingQP:     true,
	EncodingBase64: true,
	Encoding7Bit:   true,
	Encoding8Bit:   true,
}

// SMTPConfig représente une configuration SMTP d'un tenant.
type SMTPConfig struct {
	Username   *string      `json:"username,omitempty"`
	AuthMethod AuthMethod   `json:"auth_method"`
	Charset    MailCharset  `json:"charset"`
	CreatedAt  time.Time    `json:"created_at"`
	Encoding   MailEncoding `json:"encoding"`
	ID         uuid.UUID    `json:"id"`
	TenantID   uuid.UUID    `json:"tenant_id"`
	TLSPolicy  TLSPolicy    `json:"tls_policy"`
	UpdatedAt  time.Time    `json:"updated_at"`
	FromEmail  string       `json:"from_email"`
	FromName   string       `json:"from_name"`
	Host       string       `json:"host"`
	Name       string       `json:"name"`
	Password   string       `json:"-"`
	Port       int          `json:"port"`
	IsActive   bool         `json:"is_active"`
	IsDefault  bool         `json:"is_default"`
}

// CreateSMTPConfigRequest contient les données pour créer une config SMTP.
type CreateSMTPConfigRequest struct {
	Username   *string      `json:"username,omitempty"`
	AuthMethod AuthMethod   `json:"auth_method"        validate:"required,oneof=PLAIN LOGIN CRAM-MD5 NONE"`
	Charset    MailCharset  `json:"charset"`
	Encoding   MailEncoding `json:"encoding"`
	TLSPolicy  TLSPolicy    `json:"tls_policy"         validate:"required,oneof=mandatory opportunistic none"`
	FromEmail  string       `json:"from_email"         validate:"required,email"`
	FromName   string       `json:"from_name"`
	Host       string       `json:"host"               validate:"required"`
	Name       string       `json:"name"               validate:"required"`
	Password   string       `json:"password,omitempty"`
	Port       int          `json:"port"               validate:"min=1,max=65535"`
	IsDefault  bool         `json:"is_default"`
}

// UpdateSMTPConfigRequest contient les données pour modifier une config SMTP.
type UpdateSMTPConfigRequest struct {
	Name       *string       `json:"name,omitempty"`
	Host       *string       `json:"host,omitempty"`
	Port       *int          `json:"port,omitempty"`
	Username   *string       `json:"username,omitempty"`
	Password   *string       `json:"password,omitempty"`
	AuthMethod *AuthMethod   `json:"auth_method,omitempty"`
	TLSPolicy  *TLSPolicy    `json:"tls_policy,omitempty"`
	FromEmail  *string       `json:"from_email,omitempty"`
	FromName   *string       `json:"from_name,omitempty"`
	Charset    *MailCharset  `json:"charset,omitempty"`
	Encoding   *MailEncoding `json:"encoding,omitempty"`
	IsDefault  *bool         `json:"is_default,omitempty"`
	IsActive   *bool         `json:"is_active,omitempty"`
}
