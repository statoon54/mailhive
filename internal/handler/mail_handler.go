package handler

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
	"github.com/statoon54/mailhive/internal/middleware"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/templates"
)

// MailHandler gère les endpoints de mails.
type MailHandler struct {
	mailService port.MailService
}

// NewMailHandler crée un nouveau handler mail.
func NewMailHandler(mailService port.MailService) *MailHandler {
	return &MailHandler{mailService: mailService}
}

// Create compose et met en file un mail.
func (h *MailHandler) Create(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	var req domain.CreateMailRequest
	if err := bindRequest(c, &req); err != nil {
		return err
	}

	if errs := validateRequest(&req); len(errs) > 0 {
		return validationFailed(c, errs)
	}

	// Poser les détails métier pour l'audit AVANT l'appel au service
	// pour que les destinataires soient visibles même en cas d'erreur
	middleware.SetAuditDetails(c, buildMailAuditDetails(req, 0))

	mails, err := h.mailService.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return handleError(c, err)
	}

	// Mettre à jour avec le nombre réel de mails créés
	middleware.SetAuditDetails(c, buildMailAuditDetails(req, len(mails)))

	if req.Individuel {
		ids := make([]uuid.UUID, len(mails))
		for i, m := range mails {
			ids[i] = m.ID
		}
		return accepted(c, domain.CreateMailBatchResponse{Total: len(mails), MailIDs: ids})
	}
	return accepted(c, mails[0])
}

// List liste les mails avec pagination et filtrage.
func (h *MailHandler) List(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	page, limit := paginationParams(c)
	filter := domain.MailListFilter{
		Page:  page,
		Limit: limit,
	}

	// Filtre par statut optionnel
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status := domain.MailStatus(statusStr)
		filter.Status = &status
	}

	// Filtre par tags
	if tagsStr := c.QueryParam("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
	}
	filter.TagMode = c.QueryParam("tag_mode")
	if filter.TagMode == "" {
		filter.TagMode = "and"
	}

	// Recherche textuelle
	filter.Query = c.QueryParam("q")

	result, err := h.mailService.List(c.Request().Context(), tenantID, filter)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, result)
}

// GetByID retourne le détail d'un mail avec ses destinataires.
func (h *MailHandler) GetByID(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	mail, err := h.mailService.GetByID(c.Request().Context(), tenantID, id)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, mail)
}

// DownloadAttachment renvoie le contenu d'une pièce jointe d'un mail en téléchargement.
func (h *MailHandler) DownloadAttachment(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}
	mailID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return handleError(c, domain.ErrValidation)
	}
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		return handleError(c, domain.ErrValidation)
	}

	ref, content, err := h.mailService.DownloadAttachment(c.Request().Context(), tenantID, mailID, attachmentID)
	if err != nil {
		return handleError(c, err)
	}

	contentType := ref.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", ref.Filename))
	return c.Blob(http.StatusOK, contentType, content)
}

// Cancel annule un mail en attente.
func (h *MailHandler) Cancel(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.mailService.Cancel(c.Request().Context(), tenantID, id); err != nil {
		return handleError(c, err)
	}

	return ok(c, map[string]string{"message": i18n.T(lang(c), "msg.mail_cancelled")})
}

// Retry relance un mail en échec.
func (h *MailHandler) Retry(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	id, err := parseUUID(c)
	if err != nil {
		return handleError(c, err)
	}

	if err := h.mailService.Retry(c.Request().Context(), tenantID, id); err != nil {
		return handleError(c, err)
	}

	return ok(c, map[string]string{"message": i18n.T(lang(c), "msg.mail_retried")})
}

// Stats retourne les statistiques d'envoi.
func (h *MailHandler) Stats(c *echo.Context) error {
	tenantID, err := getTenantID(c)
	if err != nil {
		return handleError(c, err)
	}

	stats, err := h.mailService.Stats(c.Request().Context(), tenantID)
	if err != nil {
		return handleError(c, err)
	}

	return ok(c, stats)
}

// StatsByTenant retourne les statistiques de mails regroupées par tenant (admin).
func (h *MailHandler) StatsByTenant(c *echo.Context) error {
	stats, err := h.mailService.StatsByTenant(c.Request().Context())
	if err != nil {
		return handleError(c, err)
	}
	return ok(c, stats)
}

// mailAuditDetails représente les détails d'audit pour un envoi de mail (JSON).
type mailAuditDetails struct {
	Destinataires []mailAuditRecipient `json:"destinataires"`
	TotalDest     int                  `json:"total_destinataires"`
	TotalMails    int                  `json:"total_mails"`
	Sujet         string               `json:"sujet"`
	SujetExemples []string             `json:"sujet_exemples,omitempty"`
	TextBody      string               `json:"text_body,omitempty"`
	HTMLBody      string               `json:"html_body,omitempty"`
}

// mailAuditRecipient représente un destinataire dans les détails d'audit.
type mailAuditRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Sujet string `json:"sujet,omitempty"`
}

// maxAuditRecipients est le nombre maximal de destinataires dans les détails d'audit.
const maxAuditRecipients = 10

// maxAuditBodyLen est la longueur maximale du corps dans les détails d'audit.
const maxAuditBodyLen = 500

// mergeTemplateData fusionne les données partagées avec les données spécifiques au destinataire.
func mergeTemplateData(shared, perRecipient map[string]string) map[string]string {
	merged := make(map[string]string, len(shared)+len(perRecipient))
	maps.Copy(merged, shared)
	maps.Copy(merged, perRecipient)
	return merged
}

// buildMailAuditDetails construit les détails d'audit JSON pour un envoi de mail.
func buildMailAuditDetails(req domain.CreateMailRequest, totalMails int) string {
	var recipients []mailAuditRecipient
	for _, addr := range req.To {
		r := mailAuditRecipient{Email: addr.Email, Name: addr.Name, Type: "to"}
		// En mode individuel, rendre le sujet avec les données fusionnées par destinataire
		if req.Individuel && req.Subject != "" && len(addr.TemplateData) > 0 {
			data := mergeTemplateData(req.TemplateData, addr.TemplateData)
			if rendered, err := templates.RenderSubject(req.Subject, data); err == nil {
				r.Sujet = rendered
			}
		}
		recipients = append(recipients, r)
	}
	for _, addr := range req.CC {
		recipients = append(recipients, mailAuditRecipient{Email: addr.Email, Name: addr.Name, Type: "cc"})
	}
	for _, addr := range req.BCC {
		recipients = append(recipients, mailAuditRecipient{Email: addr.Email, Name: addr.Name, Type: "bcc"})
	}

	totalDest := len(recipients)
	if len(recipients) > maxAuditRecipients {
		recipients = recipients[:maxAuditRecipients]
	}

	// Rendre le sujet global (mode groupé ou données partagées uniquement)
	sujet := req.Subject
	if sujet != "" && len(req.TemplateData) > 0 {
		if rendered, err := templates.RenderSubject(sujet, req.TemplateData); err == nil {
			sujet = rendered
		}
	}

	details := mailAuditDetails{
		Destinataires: recipients,
		TotalDest:     totalDest,
		TotalMails:    totalMails,
		Sujet:         sujet,
		TextBody:      truncate(req.TextBody, maxAuditBodyLen),
		HTMLBody:      truncate(req.HTMLBody, maxAuditBodyLen),
	}

	b, _ := json.Marshal(details)
	return string(b)
}

// truncate tronque une chaîne à la longueur maximale spécifiée.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
