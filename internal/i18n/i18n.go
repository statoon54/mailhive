package i18n

import "strings"

// Lang représente une langue supportée.
type Lang string

const (
	FR Lang = "fr"
	EN Lang = "en"
)

// messages contient les traductions par langue.
var messages = map[Lang]map[string]string{
	FR: {
		// bindRequest
		"err.read_body":       "Impossible de lire le corps de la requête",
		"err.invalid_json":    "Format JSON invalide",
		"err.request_body":    "corps de la requête",
		"err.syntax_near":     "Erreur de syntaxe près de : ...%s...",
		"err.expected_got":    "Attendu %s, reçu %s",
		"err.validation":      "Données invalides",
		"err.internal":        "Erreur interne du serveur",
		"err.generation":      "Erreur de génération : %s",
		"err.prompt_required": "Le prompt est requis",
		"err.body_required":   "Au moins un corps (text_body ou html_body) est requis",

		// handleError
		"err.template_not_found":      "Template introuvable",
		"err.smtp_config_not_found":   "Configuration SMTP introuvable",
		"err.mail_not_found":          "Mail introuvable",
		"err.tenant_not_found":        "Tenant introuvable",
		"err.not_found":               "Ressource introuvable",
		"err.conflict":                "Conflit : la ressource existe déjà",
		"err.unauthorized":            "Authentification requise",
		"err.forbidden":               "Accès interdit",
		"err.invalid_api_key":         "Clé API invalide",
		"err.tenant_inactive":         "Tenant désactivé",
		"err.smtp_not_set":            "Aucune configuration SMTP par défaut",
		"err.mail_not_pending":        "Le mail n'est pas en attente",
		"err.mail_not_failed":         "Le mail n'est pas en échec",
		"err.rate_limited":            "Trop de requêtes",
		"err.link_check_rate_limited": "Vérification de liens limitée à 1 par minute",
		"err.spam_blocked":            "Mail bloqué : score spam trop élevé",

		// goTypeToFrench / validation
		"type.string":  "chaîne de caractères",
		"type.int":     "entier",
		"type.float64": "nombre décimal",
		"type.bool":    "booléen",

		// tagMessages
		"validation.required":  "Ce champ est requis",
		"validation.email":     "L'adresse email n'est pas valide",
		"validation.min_slice": "Au moins %s élément(s) requis",
		"validation.min":       "La valeur minimale est %s",
		"validation.max":       "La valeur maximale est %s",
		"validation.oneof":     "Valeur autorisée : %s",
		"validation.failed":    "Validation échouée : %s",

		// messages de succès
		"msg.logo_updated":   "Logo mis à jour",
		"msg.mail_cancelled": "Mail annulé",
		"msg.mail_retried":   "Mail relancé",
		"msg.smtp_test_ok":   "Test SMTP réussi",

		// auth
		"err.api_key_required": "La clé API est requise",
		"err.token_required":   "Le token est requis",

		// queues
		"err.queue_fetch": "Impossible de récupérer les queues",

		// middleware
		"err.invalid_token":     "Token invalide",
		"err.invalid_claims":    "Claims invalides",
		"err.invalid_tenant_id": "Tenant ID invalide",
		"err.tenant_expired":    "Tenant inexistant, veuillez vous reconnecter",
		"err.admin_only":        "Accès réservé aux administrateurs",

		// worker — mail_handler
		"worker.err.deserialize":       "erreur de désérialisation du payload : %w",
		"worker.err.load_mail":         "erreur de chargement du mail : %w",
		"worker.err.load_recipients":   "erreur de chargement des destinataires : %w",
		"worker.err.load_attachments":  "erreur de chargement des pièces jointes : %w",
		"worker.err.load_tenant":       "erreur de chargement du tenant pour le rate limit : %w",
		"worker.err.rate_limiter":      "erreur du rate limiter : %w",
		"worker.err.smtp_send":         "erreur d'envoi SMTP : %w",
		"worker.fail.template_invalid": "template introuvable ou invalide",
		"worker.fail.subject_render":   "erreur de rendu du sujet",
		"worker.fail.text_render":      "erreur de rendu du texte",
		"worker.fail.html_render":      "erreur de rendu du HTML",
		"worker.fail.no_smtp":          "aucune config SMTP définie",
		"worker.fail.smtp_not_found":   "configuration SMTP introuvable",
		"worker.fail.smtp_decrypt":     "erreur de déchiffrement du mot de passe SMTP",
		"worker.err.no_smtp":           "aucune config SMTP pour le mail %s",

		// worker — queue_client
		"worker.err.create_task": "erreur de création de la tâche : %w",
		"worker.err.enqueue":     "erreur de mise en file d'attente : %w",

		// worker — archive_handler
		"worker.err.archive_tx":         "erreur de début de transaction d'archivage : %w",
		"worker.err.archive_insert":     "erreur d'insertion dans l'archive : %w",
		"worker.err.archive_scan":       "erreur de scan des IDs archivés : %w",
		"worker.err.archive_recipients": "erreur d'archivage des destinataires : %w",
		"worker.err.archive_del_recip":  "erreur de suppression des destinataires archivés : %w",
		"worker.err.archive_del_mails":  "erreur de suppression des mails archivés : %w",
		"worker.err.archive_commit":     "erreur de commit d'archivage : %w",

		// worker — partition_handler
		"worker.err.partition_migrate": "migration default → %s : %w",
		"worker.err.partition_create":  "création partition %s : %w",
	},
	EN: {
		// bindRequest
		"err.read_body":       "Unable to read request body",
		"err.invalid_json":    "Invalid JSON format",
		"err.request_body":    "request body",
		"err.syntax_near":     "Syntax error near: ...%s...",
		"err.expected_got":    "Expected %s, got %s",
		"err.validation":      "Invalid data",
		"err.internal":        "Internal server error",
		"err.generation":      "Generation error: %s",
		"err.prompt_required": "Prompt is required",
		"err.body_required":   "At least one body (text_body or html_body) is required",

		// handleError
		"err.template_not_found":      "Template not found",
		"err.smtp_config_not_found":   "SMTP configuration not found",
		"err.mail_not_found":          "Mail not found",
		"err.tenant_not_found":        "Tenant not found",
		"err.not_found":               "Resource not found",
		"err.conflict":                "Conflict: resource already exists",
		"err.unauthorized":            "Authentication required",
		"err.forbidden":               "Access denied",
		"err.invalid_api_key":         "Invalid API key",
		"err.tenant_inactive":         "Tenant is disabled",
		"err.smtp_not_set":            "No default SMTP configuration",
		"err.mail_not_pending":        "Mail is not pending",
		"err.mail_not_failed":         "Mail is not in failed state",
		"err.rate_limited":            "Too many requests",
		"err.link_check_rate_limited": "Link check limited to 1 per minute",
		"err.spam_blocked":            "Mail blocked: spam score too high",

		// goType
		"type.string":  "string",
		"type.int":     "integer",
		"type.float64": "decimal number",
		"type.bool":    "boolean",

		// tagMessages
		"validation.required":  "This field is required",
		"validation.email":     "Invalid email address",
		"validation.min_slice": "At least %s element(s) required",
		"validation.min":       "Minimum value is %s",
		"validation.max":       "Maximum value is %s",
		"validation.oneof":     "Allowed values: %s",
		"validation.failed":    "Validation failed: %s",

		// success messages
		"msg.logo_updated":   "Logo updated",
		"msg.mail_cancelled": "Mail cancelled",
		"msg.mail_retried":   "Mail retried",
		"msg.smtp_test_ok":   "SMTP test successful",

		// auth
		"err.api_key_required": "API key is required",
		"err.token_required":   "Token is required",

		// queues
		"err.queue_fetch": "Unable to fetch queues",

		// middleware
		"err.invalid_token":     "Invalid token",
		"err.invalid_claims":    "Invalid claims",
		"err.invalid_tenant_id": "Invalid tenant ID",
		"err.tenant_expired":    "Tenant not found, please reconnect",
		"err.admin_only":        "Admin access only",

		// worker — mail_handler
		"worker.err.deserialize":       "payload deserialization error: %w",
		"worker.err.load_mail":         "error loading mail: %w",
		"worker.err.load_recipients":   "error loading recipients: %w",
		"worker.err.load_attachments":  "error loading attachments: %w",
		"worker.err.load_tenant":       "error loading tenant for rate limit: %w",
		"worker.err.rate_limiter":      "rate limiter error: %w",
		"worker.err.smtp_send":         "SMTP send error: %w",
		"worker.fail.template_invalid": "template not found or invalid",
		"worker.fail.subject_render":   "subject rendering error",
		"worker.fail.text_render":      "text rendering error",
		"worker.fail.html_render":      "HTML rendering error",
		"worker.fail.no_smtp":          "no SMTP configuration defined",
		"worker.fail.smtp_not_found":   "SMTP configuration not found",
		"worker.fail.smtp_decrypt":     "SMTP password decryption error",
		"worker.err.no_smtp":           "no SMTP configuration for mail %s",

		// worker — queue_client
		"worker.err.create_task": "task creation error: %w",
		"worker.err.enqueue":     "enqueue error: %w",

		// worker — archive_handler
		"worker.err.archive_tx":         "archive transaction start error: %w",
		"worker.err.archive_insert":     "archive insert error: %w",
		"worker.err.archive_scan":       "archived IDs scan error: %w",
		"worker.err.archive_recipients": "recipients archiving error: %w",
		"worker.err.archive_del_recip":  "archived recipients deletion error: %w",
		"worker.err.archive_del_mails":  "archived mails deletion error: %w",
		"worker.err.archive_commit":     "archive commit error: %w",

		// worker — partition_handler
		"worker.err.partition_migrate": "default → %s migration: %w",
		"worker.err.partition_create":  "partition %s creation: %w",
	},
}

// T retourne le message traduit pour la clé et la langue données.
func T(lang Lang, key string) string {
	if msgs, ok := messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	// Fallback vers le français.
	if msgs, ok := messages[FR]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	return key
}

// TypeName retourne le libellé traduit d'un type Go.
func TypeName(lang Lang, goType string) string {
	key := "type." + goType
	translated := T(lang, key)
	if translated == key {
		return goType
	}
	return translated
}

// DetectLang détermine la langue depuis l'en-tête Accept-Language.
func DetectLang(acceptLanguage string) Lang {
	al := strings.ToLower(acceptLanguage)
	for part := range strings.SplitSeq(al, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if strings.HasPrefix(tag, "en") {
			return EN
		}
		if strings.HasPrefix(tag, "fr") {
			return FR
		}
	}
	return FR
}
