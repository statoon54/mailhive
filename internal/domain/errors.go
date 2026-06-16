package domain

import "errors"

// Erreurs métier communes.
var (
	ErrNotFound           = errors.New("ressource introuvable")
	ErrTemplateNotFound   = errors.New("template introuvable")
	ErrSMTPConfigNotFound = errors.New("configuration SMTP introuvable")
	ErrMailNotFound       = errors.New("mail introuvable")
	ErrTenantNotFound     = errors.New("tenant introuvable")
	ErrConflict           = errors.New("conflit : la ressource existe déjà")
	ErrUnauthorized       = errors.New("authentification requise")
	ErrForbidden          = errors.New("accès interdit")
	ErrValidation         = errors.New("données invalides")
	ErrInvalidAPIKey      = errors.New("clé API invalide")
	ErrTenantInactive     = errors.New("le tenant est désactivé")
	ErrSMTPConfigNotSet   = errors.New("aucune configuration SMTP par défaut")
	ErrMailNotPending     = errors.New("le mail n'est pas en attente")
	ErrMailNotFailed      = errors.New("le mail n'est pas en échec")
	ErrRateLimited        = errors.New("trop de requêtes, réessayez plus tard")
	ErrCircuitOpen        = errors.New("circuit breaker ouvert : SMTP temporairement indisponible")
	ErrSMTPTemporary      = errors.New("erreur SMTP temporaire")
	ErrSpamBlocked        = errors.New("mail bloqué : score spam trop élevé")
	ErrLinkCheckRateLimit = errors.New("vérification de liens limitée à 1 par minute")
	ErrAttachmentNotFound = errors.New("pièce jointe introuvable")
)

// SMTPPermanentError représente une erreur SMTP permanente (config invalide, auth échouée).
// Ces erreurs ne doivent pas être retryées.
type SMTPPermanentError struct {
	Cause error
}

// Error retourne le message de l'erreur SMTP permanente.
func (e *SMTPPermanentError) Error() string {
	return "erreur SMTP permanente : " + e.Cause.Error()
}

// Unwrap retourne l'erreur sous-jacente.
func (e *SMTPPermanentError) Unwrap() error {
	return e.Cause
}

// NewSMTPPermanentError crée une erreur SMTP permanente.
func NewSMTPPermanentError(cause error) *SMTPPermanentError {
	return &SMTPPermanentError{Cause: cause}
}
