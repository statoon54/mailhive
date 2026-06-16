package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/adapter/postgres"
	"github.com/statoon54/mailhive/internal/domain"
)

func TestTemplateRepo_CreateAndGetByID(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	tmpl := &domain.Template{
		ID:          uuid.New(),
		TenantID:    tenant.ID,
		Name:        "Welcome",
		Slug:        "welcome-" + uuid.New().String()[:8],
		SubjectTmpl: "Hello {{.name}}",
		TextBody:    "Welcome {{.name}}!",
		HTMLBody:    "<p>Hello {{.name}}</p>",
		Variables:   map[string]string{"name": "Nom"},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tmpl))

	got, err := repo.GetByID(ctx, tenant.ID, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, tmpl.Name, got.Name)
	assert.Equal(t, tmpl.Slug, got.Slug)
	assert.Equal(t, tmpl.Variables, got.Variables)
}

func TestTemplateRepo_GetBySlug(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	slug := "slug-" + uuid.New().String()[:8]
	tmpl := &domain.Template{
		ID: uuid.New(), TenantID: tenant.ID, Name: "Test", Slug: slug,
		SubjectTmpl: "S", TextBody: "T", Variables: map[string]string{},
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, repo.Create(ctx, tmpl))

	got, err := repo.GetBySlug(ctx, tenant.ID, slug)
	require.NoError(t, err)
	assert.Equal(t, tmpl.ID, got.ID)
}

func TestTemplateRepo_GetByID_NotFound(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrTemplateNotFound)
}

func TestTemplateRepo_List(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	for i := range 3 {
		tmpl := &domain.Template{
			ID: uuid.New(), TenantID: tenant.ID, Name: "T",
			Slug:      "list-" + uuid.New().String()[:8],
			Variables: map[string]string{}, IsActive: true,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second), UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, tmpl))
	}

	list, err := repo.List(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestTemplateRepo_Update(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	tmpl := &domain.Template{
		ID: uuid.New(), TenantID: tenant.ID, Name: "Before",
		Slug: "upd-" + uuid.New().String()[:8], SubjectTmpl: "Old",
		Variables: map[string]string{}, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, tmpl))

	tmpl.Name = "After"
	tmpl.SubjectTmpl = "New {{.x}}"
	tmpl.Variables = map[string]string{"x": "value"}
	tmpl.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, tmpl))

	got, err := repo.GetByID(ctx, tenant.ID, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, map[string]string{"x": "value"}, got.Variables)
}

func TestTemplateRepo_Delete(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	tmpl := &domain.Template{
		ID: uuid.New(), TenantID: tenant.ID, Name: "Del",
		Slug: "del-" + uuid.New().String()[:8], Variables: map[string]string{},
		IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, tmpl))
	require.NoError(t, repo.Delete(ctx, tenant.ID, tmpl.ID))

	_, err := repo.GetByID(ctx, tenant.ID, tmpl.ID)
	assert.ErrorIs(t, err, domain.ErrTemplateNotFound)
}

func TestTemplateRepo_DuplicateSlug(t *testing.T) {
	pool := PGPool(t)

	repo := postgres.NewTemplateRepository(pool)
	ctx := context.Background()
	tenant := insertTenant(t, pool)

	slug := "dup-" + uuid.New().String()[:8]
	tmpl1 := &domain.Template{
		ID: uuid.New(), TenantID: tenant.ID, Name: "First", Slug: slug,
		Variables: map[string]string{}, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, tmpl1))

	tmpl2 := &domain.Template{
		ID: uuid.New(), TenantID: tenant.ID, Name: "Second", Slug: slug,
		Variables: map[string]string{}, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := repo.Create(ctx, tmpl2)
	assert.ErrorIs(t, err, domain.ErrConflict)
}
