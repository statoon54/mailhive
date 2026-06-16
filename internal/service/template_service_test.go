package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

func TestTemplateService_Create_Slug(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)
	tenantID := uuid.New()

	tmpl, err := svc.Create(context.Background(), tenantID, domain.CreateTemplateRequest{
		Name:        "Mon Template Français",
		SubjectTmpl: "Hello {{.name}}",
	})
	require.NoError(t, err)
	assert.Equal(t, "mon-template-francais", tmpl.Slug)
	assert.True(t, tmpl.IsActive)
	assert.NotNil(t, tmpl.Variables)
}

func TestTemplateService_Create_HTMLSanitized(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)

	tmpl, err := svc.Create(context.Background(), uuid.New(), domain.CreateTemplateRequest{
		Name:        "Test",
		SubjectTmpl: "Hello",
		HTMLBody:    `<p>Hello</p><script>alert('xss')</script>`,
	})
	require.NoError(t, err)
	assert.NotContains(t, tmpl.HTMLBody, "<script>")
	assert.Contains(t, tmpl.HTMLBody, "<p>Hello</p>")
}

func TestTemplateService_Create_TemplateVarsPreserved(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)

	tmpl, err := svc.Create(context.Background(), uuid.New(), domain.CreateTemplateRequest{
		Name:        "Test",
		SubjectTmpl: "Hello {{.name}}",
		HTMLBody:    `<a href="{{.link}}">Click</a>`,
		Variables:   map[string]string{"name": "Nom", "link": "URL"},
	})
	require.NoError(t, err)
	assert.Contains(t, tmpl.HTMLBody, "{{.link}}")
	assert.Equal(t, map[string]string{"name": "Nom", "link": "URL"}, tmpl.Variables)
}

func TestTemplateService_Update_Partial(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)
	tenantID := uuid.New()
	tmplID := uuid.New()
	repo.Templates[tmplID] = &domain.Template{
		ID:       tmplID,
		TenantID: tenantID,
		Name:     "Old Name",
		Slug:     "old-slug",
		HTMLBody: "<p>Old</p>",
	}

	newName := "New Name"
	tmpl, err := svc.Update(context.Background(), tenantID, tmplID, domain.UpdateTemplateRequest{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "New Name", tmpl.Name)
	assert.Equal(t, "old-slug", tmpl.Slug)
}

func TestTemplateService_Update_HTMLReSanitized(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)
	tenantID := uuid.New()
	tmplID := uuid.New()
	repo.Templates[tmplID] = &domain.Template{
		ID: tmplID, TenantID: tenantID, HTMLBody: "<p>Clean</p>",
	}

	newHTML := `<p>New</p><script>alert('xss')</script>`
	tmpl, err := svc.Update(context.Background(), tenantID, tmplID, domain.UpdateTemplateRequest{
		HTMLBody: &newHTML,
	})
	require.NoError(t, err)
	assert.NotContains(t, tmpl.HTMLBody, "<script>")
}

func TestTemplateService_Preview(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)
	tenantID := uuid.New()
	tmplID := uuid.New()
	repo.Templates[tmplID] = &domain.Template{
		ID:          tmplID,
		TenantID:    tenantID,
		SubjectTmpl: "Hello {{.name}}",
		TextBody:    "Welcome {{.name}}",
		HTMLBody:    "<p>Welcome {{.name}}</p>",
		Variables:   map[string]string{"name": "Nom"},
	}

	preview, err := svc.Preview(context.Background(), tenantID, tmplID, map[string]string{"name": "Alice"})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice", preview.Subject)
	assert.Contains(t, preview.TextBody, "Welcome Alice")
	assert.Contains(t, preview.HTMLBody, "Welcome Alice")
}

func TestTemplateService_Delete(t *testing.T) {
	repo := mocks.NewMockTemplateRepo()
	svc := NewTemplateService(repo)
	tenantID := uuid.New()
	tmplID := uuid.New()
	repo.Templates[tmplID] = &domain.Template{ID: tmplID, TenantID: tenantID}

	err := svc.Delete(context.Background(), tenantID, tmplID)
	require.NoError(t, err)
	assert.True(t, repo.Called("Delete"))
}
