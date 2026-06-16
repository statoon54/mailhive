package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/i18n"
)

// lang extrait la langue depuis l'en-tête Accept-Language de la requête.
func lang(c *echo.Context) i18n.Lang {
	return i18n.DetectLang(c.Request().Header.Get("Accept-Language"))
}

// bindRequest lie le body JSON à la struct et retourne une erreur explicite si le JSON est invalide.
func bindRequest(c *echo.Context, req any) error {
	l := lang(c)

	body, readErr := io.ReadAll(c.Request().Body)
	if readErr != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: i18n.T(l, "err.read_body"),
		})
	}

	err := json.Unmarshal(body, req)
	if err == nil {
		return nil
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: i18n.T(l, "err.invalid_json"),
			Fields: []FieldValidationError{
				{Field: "json", Message: jsonContextAt(l, body, syntaxErr.Offset)},
			},
		})
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = i18n.T(l, "err.request_body")
		}
		expected := i18n.TypeName(l, typeErr.Type.Name())
		received := i18n.TypeName(l, typeErr.Value)
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: i18n.T(l, "err.invalid_json"),
			Fields: []FieldValidationError{
				{Field: toSnakeCase(field), Message: fmt.Sprintf(i18n.T(l, "err.expected_got"), expected, received)},
			},
		})
	}

	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: i18n.T(l, "err.invalid_json"),
	})
}

// jsonContextAt extrait un extrait du JSON autour de la position d'erreur.
func jsonContextAt(l i18n.Lang, body []byte, offset int64) string {
	pos := min(int(offset), len(body))
	start := max(pos-20, 0)
	end := min(pos+20, len(body))
	extract := strings.TrimSpace(string(body[start:end]))
	return fmt.Sprintf(i18n.T(l, "err.syntax_near"), extract)
}

// validate est l'instance du validateur de structs.
var validate = validator.New(validator.WithRequiredStructEnabled())

// APIResponse représente une réponse JSON standard de l'API.
type APIResponse struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

// ErrorResponse représente une réponse d'erreur.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Fields  []FieldValidationError `json:"fields,omitempty"`
	Success bool                   `json:"success"`
}

// FieldValidationError représente une erreur de validation sur un champ.
type FieldValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// success retourne une réponse JSON de succès.
func success(c *echo.Context, code int, data any) error {
	return c.JSON(code, APIResponse{
		Success: true,
		Data:    data,
	})
}

// created retourne une réponse 201 Created.
func created(c *echo.Context, data any) error {
	return success(c, http.StatusCreated, data)
}

// accepted retourne une réponse 202 Accepted.
func accepted(c *echo.Context, data any) error {
	return success(c, http.StatusAccepted, data)
}

// ok retourne une réponse 200 OK.
func ok(c *echo.Context, data any) error {
	return success(c, http.StatusOK, data)
}

// handleError traduit une erreur domaine en réponse HTTP appropriée.
func handleError(c *echo.Context, err error) error {
	l := lang(c)
	switch {
	case errors.Is(err, domain.ErrTemplateNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.template_not_found")})
	case errors.Is(err, domain.ErrSMTPConfigNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.smtp_config_not_found")})
	case errors.Is(err, domain.ErrMailNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.mail_not_found")})
	case errors.Is(err, domain.ErrAttachmentNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.not_found")})
	case errors.Is(err, domain.ErrTenantNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.tenant_not_found")})
	case errors.Is(err, domain.ErrNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: i18n.T(l, "err.not_found")})
	case errors.Is(err, domain.ErrConflict):
		return c.JSON(http.StatusConflict, ErrorResponse{Error: i18n.T(l, "err.conflict")})
	case errors.Is(err, domain.ErrUnauthorized):
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: i18n.T(l, "err.unauthorized")})
	case errors.Is(err, domain.ErrForbidden):
		return c.JSON(http.StatusForbidden, ErrorResponse{Error: i18n.T(l, "err.forbidden")})
	case errors.Is(err, domain.ErrValidation):
		msg := i18n.T(l, "err.validation")
		if err.Error() != domain.ErrValidation.Error() {
			msg = err.Error()
		}
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: msg})
	case errors.Is(err, domain.ErrInvalidAPIKey):
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: i18n.T(l, "err.invalid_api_key")})
	case errors.Is(err, domain.ErrTenantInactive):
		return c.JSON(http.StatusForbidden, ErrorResponse{Error: i18n.T(l, "err.tenant_inactive")})
	case errors.Is(err, domain.ErrSMTPConfigNotSet):
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: i18n.T(l, "err.smtp_not_set")})
	case errors.Is(err, domain.ErrMailNotPending):
		return c.JSON(http.StatusConflict, ErrorResponse{Error: i18n.T(l, "err.mail_not_pending")})
	case errors.Is(err, domain.ErrMailNotFailed):
		return c.JSON(http.StatusConflict, ErrorResponse{Error: i18n.T(l, "err.mail_not_failed")})
	case errors.Is(err, domain.ErrRateLimited):
		return c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: i18n.T(l, "err.rate_limited")})
	case errors.Is(err, domain.ErrSpamBlocked):
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrLinkCheckRateLimit):
		return c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: i18n.T(l, "err.link_check_rate_limited")})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: i18n.T(l, "err.internal")})
	}
}

// parseUUID parse l'UUID du paramètre de route "id".
func parseUUID(c *echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, domain.ErrValidation
	}
	return id, nil
}

// getTenantID récupère le tenant_id depuis le contexte (défini par le middleware).
func getTenantID(c *echo.Context) (uuid.UUID, error) {
	tenantIDStr, ok := c.Get("tenant_id").(string)
	if !ok || tenantIDStr == "" {
		return uuid.Nil, domain.ErrUnauthorized
	}
	id, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return id, nil
}

// translateFieldError traduit une FieldError selon la langue.
func translateFieldError(l i18n.Lang, fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return i18n.T(l, "validation.required")
	case "email":
		return i18n.T(l, "validation.email")
	case "min":
		if fe.Kind().String() == "slice" {
			return fmt.Sprintf(i18n.T(l, "validation.min_slice"), fe.Param())
		}
		return fmt.Sprintf(i18n.T(l, "validation.min"), fe.Param())
	case "max":
		return fmt.Sprintf(i18n.T(l, "validation.max"), fe.Param())
	case "oneof":
		return fmt.Sprintf(i18n.T(l, "validation.oneof"), fe.Param())
	default:
		return fmt.Sprintf(i18n.T(l, "validation.failed"), fe.Tag())
	}
}

// toSnakeCase convertit un nom de champ PascalCase en snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// validateRequest valide une struct et retourne les erreurs par champ.
func validateRequestL(l i18n.Lang, req any) []FieldValidationError {
	err := validate.Struct(req)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return nil
	}
	errs := make([]FieldValidationError, 0, len(ve))
	for _, fe := range ve {
		errs = append(errs, FieldValidationError{
			Field:   toSnakeCase(fe.Field()),
			Message: translateFieldError(l, fe),
		})
	}
	return errs
}

// validateRequest valide une struct (utilise le français par défaut, pour compatibilité).
func validateRequest(req any) []FieldValidationError {
	return validateRequestL(i18n.FR, req)
}

// validationFailedL retourne une réponse 400 avec les erreurs de validation traduites.
func validationFailedL(c *echo.Context, l i18n.Lang, errs []FieldValidationError) error {
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:  i18n.T(l, "err.validation"),
		Fields: errs,
	})
}

// validationFailed retourne une réponse 400 avec les erreurs de validation.
func validationFailed(c *echo.Context, errs []FieldValidationError) error {
	return validationFailedL(c, lang(c), errs)
}

// paginationParams extrait les paramètres de pagination depuis la query string.
func paginationParams(c *echo.Context) (page, limit int) {
	page, _ = strconv.Atoi(c.QueryParam("page"))
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
