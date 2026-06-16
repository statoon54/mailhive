package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/handler"
	"github.com/statoon54/mailhive/internal/service"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

// setupAnalysisServer crée un serveur Echo minimal pour les tests d'analyse.
func setupAnalysisServer(t *testing.T) (*echo.Echo, uuid.UUID, uuid.UUID) {
	t.Helper()

	tenantID := uuid.New()
	templateID := uuid.New()

	tmplRepo := mocks.NewMockTemplateRepo()
	tmplRepo.Templates[templateID] = TestTemplate(templateID, tenantID)

	analysisService := service.NewAnalysisService(tmplRepo)
	redisClient := RedisClient(t)
	analysisHandler := handler.NewAnalysisHandler(analysisService, redisClient)

	e := echo.New()

	// Simuler le middleware JWT en injectant le tenant_id dans le contexte
	e.POST("/api/v1/templates/:id/spam-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.SpamCheck(c)
	})
	e.POST("/api/v1/templates/:id/html-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.HTMLCheck(c)
	})
	e.POST("/api/v1/templates/:id/link-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.LinkCheck(c)
	})

	return e, tenantID, templateID
}

func TestIntegration_SpamCheck_CleanTemplate(t *testing.T) {
	e, _, templateID := setupAnalysisServer(t)

	body := SpamRequest(map[string]string{"name": "Alice"})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/spam-check", body, "")

	AssertStatus(t, rec, http.StatusOK)

	var resp struct {
		Data domain.SpamCheckResult `json:"data"`
	}
	DecodeResponse(t, rec, &resp)

	if resp.Data.Score > 2.0 {
		t.Errorf("score trop élevé pour un template propre : %.1f", resp.Data.Score)
	}
}

func TestIntegration_SpamCheck_TemplateNotFound(t *testing.T) {
	e, _, _ := setupAnalysisServer(t)

	fakeID := uuid.New()
	body := SpamRequest(map[string]string{"name": "Alice"})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+fakeID.String()+"/spam-check", body, "")

	AssertStatus(t, rec, http.StatusNotFound)
}

func TestIntegration_SpamCheck_MissingVariables(t *testing.T) {
	e, _, templateID := setupAnalysisServer(t)

	// Ne pas fournir la variable "name" requise
	body := SpamRequest(map[string]string{})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/spam-check", body, "")

	AssertStatus(t, rec, http.StatusBadRequest)
}

func TestIntegration_HTMLCheck_CleanTemplate(t *testing.T) {
	e, _, templateID := setupAnalysisServer(t)

	body := SpamRequest(map[string]string{"name": "Alice"})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/html-check", body, "")

	AssertStatus(t, rec, http.StatusOK)

	var resp struct {
		Data domain.HTMLCheckResult `json:"data"`
	}
	DecodeResponse(t, rec, &resp)

	if resp.Data.TotalCount != 0 {
		t.Errorf("un template simple ne devrait pas avoir de problèmes HTML, obtenu %d", resp.Data.TotalCount)
	}
}

func TestIntegration_HTMLCheck_ProblematicTemplate(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	tmplRepo := mocks.NewMockTemplateRepo()
	tmplRepo.Templates[templateID] = &domain.Template{
		ID:          templateID,
		TenantID:    tenantID,
		Name:        "Flexbox Template",
		Slug:        "flexbox-template",
		SubjectTmpl: "Hello",
		HTMLBody:    `<div style="display: flex;"><p>Content</p></div>`,
		Variables:   map[string]string{},
		IsActive:    true,
	}

	analysisService := service.NewAnalysisService(tmplRepo)
	redisClient := RedisClient(t)
	analysisHandler := handler.NewAnalysisHandler(analysisService, redisClient)

	e := echo.New()
	e.POST("/api/v1/templates/:id/html-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.HTMLCheck(c)
	})

	body := SpamRequest(map[string]string{})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/html-check", body, "")

	AssertStatus(t, rec, http.StatusOK)

	var resp struct {
		Data domain.HTMLCheckResult `json:"data"`
	}
	DecodeResponse(t, rec, &resp)

	if resp.Data.TotalCount == 0 {
		t.Error("flexbox devrait déclencher un problème de compatibilité")
	}
}

func TestIntegration_LinkCheck_RateLimit(t *testing.T) {
	e, _, templateID := setupAnalysisServer(t)

	body := SpamRequest(map[string]string{"name": "Alice"})
	path := "/api/v1/templates/" + templateID.String() + "/link-check"

	// Premier appel — devrait réussir
	rec1 := DoRequest(e, http.MethodPost, path, body, "")
	AssertStatus(t, rec1, http.StatusOK)

	// Deuxième appel immédiat — devrait être rate limité
	rec2 := DoRequest(e, http.MethodPost, path, body, "")
	AssertStatus(t, rec2, http.StatusTooManyRequests)
}

func TestIntegration_LinkCheck_Results(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	tmplRepo := mocks.NewMockTemplateRepo()
	tmplRepo.Templates[templateID] = &domain.Template{
		ID:          templateID,
		TenantID:    tenantID,
		Name:        "Links Template",
		Slug:        "links-template",
		SubjectTmpl: "Links",
		HTMLBody:    `<a href="http://example.com">Link</a><a href="/relative">Rel</a>`,
		Variables:   map[string]string{},
		IsActive:    true,
	}

	analysisService := service.NewAnalysisService(tmplRepo)
	redisClient := RedisClient(t)
	analysisHandler := handler.NewAnalysisHandler(analysisService, redisClient)

	e := echo.New()
	e.POST("/api/v1/templates/:id/link-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.LinkCheck(c)
	})

	body := SpamRequest(map[string]string{})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/link-check", body, "")

	AssertStatus(t, rec, http.StatusOK)

	var resp struct {
		Data domain.LinkCheckResult `json:"data"`
	}
	DecodeResponse(t, rec, &resp)

	if resp.Data.TotalCount != 2 {
		t.Errorf("attendu 2 liens, obtenu %d", resp.Data.TotalCount)
	}

	// Vérifier les statuts
	statuses := map[string]bool{}
	for _, l := range resp.Data.Links {
		statuses[l.Status] = true
	}
	if !statuses["insecure"] {
		t.Error("http://example.com devrait être marqué 'insecure'")
	}
	if !statuses["invalid"] {
		t.Error("/relative devrait être marqué 'invalid'")
	}
}

func TestIntegration_SpamCheck_SpamContent(t *testing.T) {
	tenantID := uuid.New()
	templateID := uuid.New()

	tmplRepo := mocks.NewMockTemplateRepo()
	tmplRepo.Templates[templateID] = &domain.Template{
		ID:          templateID,
		TenantID:    tenantID,
		Name:        "Spam Template",
		Slug:        "spam-template",
		SubjectTmpl: "FREE MONEY!!! ACT NOW???",
		TextBody:    "",
		HTMLBody: `<html><body>
			<a href="http://192.168.1.1/page">https://legit.com</a>
			<span style="display:none">Hidden text</span>
			CLICK HERE FOR FREE GIFT LIMITED TIME OFFER
		</body></html>`,
		Variables: map[string]string{},
		IsActive:  true,
	}

	analysisService := service.NewAnalysisService(tmplRepo)
	redisClient := RedisClient(t)
	analysisHandler := handler.NewAnalysisHandler(analysisService, redisClient)

	e := echo.New()
	e.POST("/api/v1/templates/:id/spam-check", func(c *echo.Context) error {
		c.Set("tenant_id", tenantID.String())
		return analysisHandler.SpamCheck(c)
	})

	body := SpamRequest(map[string]string{})
	rec := DoRequest(e, http.MethodPost, "/api/v1/templates/"+templateID.String()+"/spam-check", body, "")

	AssertStatus(t, rec, http.StatusOK)

	var resp struct {
		Data domain.SpamCheckResult `json:"data"`
	}
	DecodeResponse(t, rec, &resp)

	if resp.Data.Score < 5.0 {
		t.Errorf("contenu spam devrait avoir score >= 5.0, obtenu %.1f", resp.Data.Score)
	}

	if len(resp.Data.Rules) < 3 {
		t.Errorf("attendu >= 3 règles déclenchées, obtenu %d", len(resp.Data.Rules))
	}

	// Vérifier les noms des règles
	ruleNames := map[string]bool{}
	for _, r := range resp.Data.Rules {
		ruleNames[r.Name] = true
	}

	expectedRules := []string{"text_html_ratio", "spam_keywords", "suspicious_links", "hidden_text"}
	for _, name := range expectedRules {
		if !ruleNames[name] {
			t.Errorf("règle '%s' non déclenchée", name)
		}
	}

	// Sérialiser le résultat pour vérifier le format JSON
	b, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("erreur de sérialisation : %v", err)
	}
	if len(b) == 0 {
		t.Error("la sérialisation JSON ne devrait pas être vide")
	}
}
