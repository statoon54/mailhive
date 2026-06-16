package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/port"
)

// bodyCapture capture le body écrit dans la réponse tout en le transmettant au client.
type bodyCapture struct {
	buf bytes.Buffer
	http.ResponseWriter
	code int
}

// WriteHeader capture le code de statut HTTP avant de le transmettre.
func (r *bodyCapture) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// Write capture le corps de la réponse avant de le transmettre.
func (r *bodyCapture) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

// SetAuditDetails permet aux handlers de poser des détails métier pour l'audit.
func SetAuditDetails(c *echo.Context, details string) {
	c.Set("audit_details", details)
}

// AuditMiddleware enregistre les actions d'écriture dans le journal d'audit.
func AuditMiddleware(auditService port.AuditLogService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			method := c.Request().Method
			// Ignorer les lectures avant d'instrumenter
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				return next(c)
			}

			// Envelopper le ResponseWriter pour capturer le status code et le body
			rec := &bodyCapture{ResponseWriter: c.Response(), code: http.StatusOK}
			c.SetResponse(rec)

			// Exécuter le handler
			err := next(c)

			// Récupérer le tenant_id depuis le contexte
			tenantIDStr, _ := c.Get("tenant_id").(string)
			tenantID, parseErr := uuid.Parse(tenantIDStr)
			if parseErr != nil {
				return err
			}

			path := c.Request().URL.Path
			statusCode := rec.code

			// Déduire l'action depuis la méthode HTTP
			action := methodToAction(method)
			// Déduire le type de ressource et l'ID depuis le path
			resourceType, resourceID := parseResourceFromPath(path)

			// Surcharger l'action pour les routes spéciales
			if strings.HasSuffix(path, "/test") {
				action = "test"
			} else if strings.Contains(path, "/cancel") {
				action = "cancel"
			} else if strings.Contains(path, "/retry") {
				action = "retry"
			} else if strings.Contains(path, "/preview") {
				action = "preview"
			}

			// Déterminer le statut et extraire le message d'erreur
			status := "success"
			errorMessage := ""
			if statusCode >= 400 {
				status = "error"
				errorMessage = extractErrorMessage(rec.buf.Bytes())
			}

			// Récupérer les détails métier posés par le handler
			details, _ := c.Get("audit_details").(string)

			auditService.Log(
				tenantID,
				action,
				resourceType,
				resourceID,
				status,
				statusCode,
				errorMessage,
				details,
				method,
				path,
			)

			return err
		}
	}
}

// extractErrorMessage extrait le champ "error" d'une réponse JSON d'erreur.
func extractErrorMessage(body []byte) string {
	var resp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &resp) == nil && resp.Error != "" {
		return resp.Error
	}
	return ""
}

// methodToAction convertit la méthode HTTP en action d'audit.
func methodToAction(method string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(method)
	}
}

// parseResourceFromPath extrait le type de ressource et l'ID depuis le path.
func parseResourceFromPath(path string) (resourceType, resourceID string) {
	// Retirer le préfixe /api/v1 et /admin
	trimmed := strings.TrimPrefix(path, "/api/v1")
	trimmed = strings.TrimPrefix(trimmed, "/admin")

	// Retirer les suffixes d'action
	for _, suffix := range []string{"/test", "/cancel", "/retry", "/preview"} {
		trimmed = strings.TrimSuffix(trimmed, suffix)
	}

	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 {
		return "unknown", ""
	}

	// Mapper le segment de path au type de ressource
	resourceType = pathToResourceType(parts[0])

	// L'ID est le second segment s'il existe
	if len(parts) >= 2 {
		resourceID = parts[1]
	}

	return resourceType, resourceID
}

// pathToResourceType convertit un segment de path en type de ressource.
func pathToResourceType(segment string) string {
	switch segment {
	case "templates":
		return "template"
	case "smtp-configs":
		return "smtp_config"
	case "mails":
		return "mail"
	case "branding":
		return "branding"
	case "tenants":
		return "tenant"
	default:
		return segment
	}
}
