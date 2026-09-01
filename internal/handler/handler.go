package handler

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
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

	err := json.Unmarshal(body, req, decodeOptions)
	if err == nil {
		return nil
	}

	// Erreur de grammaire : le document n'est pas du JSON valide.
	if syntaxErr, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: i18n.T(l, "err.invalid_json"),
			Fields: []FieldValidationError{
				{Field: "json", Message: jsonContextAt(l, body, syntaxErr.ByteOffset)},
			},
		})
	}

	// Erreur de sens : le JSON est valide mais ne correspond pas au type Go.
	if semErr, ok := errors.AsType[*json.SemanticError](err); ok {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:  i18n.T(l, "err.invalid_json"),
			Fields: []FieldValidationError{semanticFieldError(l, semErr)},
		})
	}

	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: i18n.T(l, "err.invalid_json"),
	})
}

// semanticFieldError traduit une erreur sémantique json/v2 en erreur de champ.
//
// La v2 ne fournit plus le nom du champ Go ni une étiquette de type prête à
// l'emploi : le nom vient du JSON Pointer (« /to/0/email ») et les types se
// lisent sur GoType et JSONKind.
func semanticFieldError(l i18n.Lang, semErr *json.SemanticError) FieldValidationError {
	field := pointerField(semErr.JSONPointer)
	if field == "" {
		field = i18n.T(l, "err.request_body")
	}

	expected := i18n.TypeName(l, goTypeLabel(semErr.GoType))
	received := i18n.TypeName(l, jsonKindLabel(semErr.JSONKind))

	return FieldValidationError{
		Field:   field,
		Message: fmt.Sprintf(i18n.T(l, "err.expected_got"), expected, received),
	}
}

// pointerField extrait le nom du champ fautif d'un JSON Pointer RFC 6901.
// Les indices de tableau sont ignorés : « /to/0/email » donne « email ».
func pointerField(ptr jsontext.Pointer) string {
	field := ""
	for token := range ptr.Tokens() {
		if _, err := strconv.Atoi(token); err == nil {
			continue // indice de tableau, pas un nom de champ
		}
		field = token
	}
	return field
}

// goTypeLabel nomme un type Go pour l'affichage.
//
// reflect.Type.Name() est vide pour les types composites (slice, map, pointeur),
// ce qui produisait auparavant des messages tronqués du genre « Attendu , reçu
// chaîne de caractères ». On retombe sur la catégorie du type.
func goTypeLabel(t reflect.Type) string {
	if t == nil {
		return ""
	}
	if name := t.Name(); name != "" {
		return name
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return "slice"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Pointer:
		return goTypeLabel(t.Elem())
	default:
		return t.Kind().String()
	}
}

// jsonKindLabel nomme la valeur JSON reçue, dans le vocabulaire déjà traduit
// par i18n (« string », « number », « bool », « slice », « object »).
func jsonKindLabel(k jsontext.Kind) string {
	// Kind est le premier octet du symbole de la grammaire ('0' pour tous les
	// nombres). Sa méthode String() renvoie « true »/« false » séparément et le
	// symbole brut pour les tableaux et objets : on mappe donc l'octet.
	switch k {
	case '"':
		return "string"
	case '0':
		return "number"
	case 't', 'f':
		return "bool"
	case '[':
		return "slice"
	case '{':
		return "object"
	case 'n':
		return "null"
	default:
		return ""
	}
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
