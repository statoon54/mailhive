package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/port"
	"github.com/statoon54/mailhive/internal/templates"
)

// htmlPolicy est la politique de sanitisation HTML pour les templates email.
var htmlPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style", "class", "align").Globally()
	p.AllowAttrs("width", "height").OnElements("img", "table", "td", "th")
	p.AllowElements("table", "thead", "tbody", "tfoot", "tr", "td", "th", "caption", "colgroup", "col")
	p.AllowAttrs("colspan", "rowspan").OnElements("td", "th")
	p.AllowAttrs("target").OnElements("a")
	p.AllowStyles("color", "background-color", "font-size", "font-family", "text-align",
		"margin", "padding", "border", "width", "height", "max-width", "line-height").Globally()
	return p
}()

// TemplateService implémente port.TemplateService.
type TemplateService struct {
	repo port.TemplateRepository
}

// NewTemplateService crée un nouveau service template.
func NewTemplateService(repo port.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

// goTmplRe détecte les placeholders Go template {{.xxx}} et {{.xxx | func}}.
var goTmplRe = regexp.MustCompile(`\{\{[^}]+\}\}`)

// sanitizeHTML nettoie le HTML tout en préservant les placeholders Go template.
// bluemonday URL-encode les {{ }} dans les attributs href/src, ce qui casse le rendu.
func sanitizeHTML(html string) string {
	// Remplacer les placeholders par des tokens uniques avant sanitisation
	placeholders := goTmplRe.FindAllString(html, -1)
	protected := html
	tokens := make([]string, len(placeholders))
	for i, ph := range placeholders {
		token := "TMPLPH" + strings.Repeat("X", i+1) + "END"
		tokens[i] = token
		protected = strings.Replace(protected, ph, token, 1)
	}

	sanitized := htmlPolicy.Sanitize(protected)

	// Restaurer les placeholders
	for i, token := range tokens {
		sanitized = strings.Replace(sanitized, token, placeholders[i], 1)
	}
	return sanitized
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify génère un slug à partir d'une chaîne (supprime les accents, met en minuscules).
func slugify(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = strings.ToLower(result)
	result = slugRe.ReplaceAllString(result, "-")
	return strings.Trim(result, "-")
}

// Create crée un nouveau template avec génération automatique du slug si absent.
func (s *TemplateService) Create(ctx context.Context, tenantID uuid.UUID, req domain.CreateTemplateRequest) (*domain.Template, error) {
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}

	now := time.Now()
	tmpl := &domain.Template{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        req.Slug,
		SubjectTmpl: req.SubjectTmpl,
		TextBody:    req.TextBody,
		HTMLBody:    sanitizeHTML(req.HTMLBody),
		Variables:   req.Variables,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if tmpl.Variables == nil {
		tmpl.Variables = make(map[string]string)
	}

	if err := s.repo.Create(ctx, tmpl); err != nil {
		return nil, err
	}
	return tmpl, nil
}

// GetByID retourne un template par son identifiant.
func (s *TemplateService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

// List retourne tous les templates d'un tenant.
func (s *TemplateService) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Template, error) {
	return s.repo.List(ctx, tenantID)
}

// Update met à jour les champs modifiables d'un template existant.
func (s *TemplateService) Update(ctx context.Context, tenantID, id uuid.UUID, req domain.UpdateTemplateRequest) (*domain.Template, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		tmpl.Name = *req.Name
	}
	if req.Slug != nil {
		tmpl.Slug = *req.Slug
	}
	if req.SubjectTmpl != nil {
		tmpl.SubjectTmpl = *req.SubjectTmpl
	}
	if req.TextBody != nil {
		tmpl.TextBody = *req.TextBody
	}
	if req.HTMLBody != nil {
		sanitized := sanitizeHTML(*req.HTMLBody)
		tmpl.HTMLBody = sanitized
	}
	if req.Variables != nil {
		tmpl.Variables = req.Variables
	}
	if req.IsActive != nil {
		tmpl.IsActive = *req.IsActive
	}
	tmpl.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tmpl); err != nil {
		return nil, err
	}
	return tmpl, nil
}

// Delete supprime un template par son identifiant.
func (s *TemplateService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

// Preview retourne un aperçu du template rendu avec les données fournies.
func (s *TemplateService) Preview(ctx context.Context, tenantID, id uuid.UUID, data map[string]string) (*domain.PreviewTemplateResponse, error) {
	tmpl, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	subject, err := templates.RenderSubject(tmpl.SubjectTmpl, data)
	if err != nil {
		return nil, err
	}

	textBody, err := templates.RenderText(tmpl.TextBody, data)
	if err != nil {
		return nil, err
	}

	htmlBody, err := templates.RenderHTML(tmpl.HTMLBody, data)
	if err != nil {
		return nil, err
	}

	return &domain.PreviewTemplateResponse{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}, nil
}
