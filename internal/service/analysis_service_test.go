package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/templates"
	"github.com/statoon54/mailhive/internal/test/mocks"
)

// storeBrokenTemplate écrit un template invalide directement dans le dépôt,
// comme ceux déjà présents en base avant que Create/Update ne les refusent.
func storeBrokenTemplate(t *testing.T, html string) (*mocks.MockTemplateRepo, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := mocks.NewMockTemplateRepo()
	tenantID, tmplID := uuid.New(), uuid.New()
	repo.Templates[tmplID] = &domain.Template{
		ID:          tmplID,
		TenantID:    tenantID,
		Name:        "Cassé",
		SubjectTmpl: "Bonjour",
		HTMLBody:    html,
		Variables:   map[string]string{},
	}
	return repo, tenantID, tmplID
}

func TestAnalysisService_TemplateCasse(t *testing.T) {
	checks := map[string]func(*AnalysisService, context.Context, uuid.UUID, uuid.UUID, map[string]string) error{
		"SpamCheck": func(s *AnalysisService, ctx context.Context, tenant, id uuid.UUID, data map[string]string) error {
			_, err := s.SpamCheck(ctx, tenant, id, data)
			return err
		},
		"HTMLCheck": func(s *AnalysisService, ctx context.Context, tenant, id uuid.UUID, data map[string]string) error {
			_, err := s.HTMLCheck(ctx, tenant, id, data)
			return err
		},
		"LinkCheck": func(s *AnalysisService, ctx context.Context, tenant, id uuid.UUID, data map[string]string) error {
			_, err := s.LinkCheck(ctx, tenant, id, data)
			return err
		},
	}

	bodies := map[string]struct {
		html string
		data map[string]string
	}{
		// Refusé au parsing : accolade fermante manquante.
		"syntaxe invalide": {html: "<p>{{.Prenom}</p>"},
		// Parse correctement mais échoue à l'exécution : .Prenom est une chaîne,
		// elle n'a pas de champ .Nom. Une clé absente, elle, rend « <no value> »
		// sans erreur — d'où la donnée fournie ici.
		"rendu impossible": {html: "<p>{{.Prenom.Nom}}</p>", data: map[string]string{"Prenom": "Jean"}},
	}

	for checkName, run := range checks {
		for bodyName, body := range bodies {
			t.Run(checkName+"/"+bodyName, func(t *testing.T) {
				repo, tenantID, tmplID := storeBrokenTemplate(t, body.html)
				svc := NewAnalysisService(repo)

				err := run(svc, context.Background(), tenantID, tmplID, body.data)

				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrValidation,
					"un template cassé est une erreur de données (400), pas une erreur serveur (500)")
				tmplErr, ok := errors.AsType[*templates.Error](err)
				require.True(t, ok, "l'erreur doit désigner le champ fautif")
				assert.Equal(t, templates.FieldHTML, tmplErr.Field)
			})
		}
	}
}

func TestTemplateService_Preview_TemplateCasse(t *testing.T) {
	repo, tenantID, tmplID := storeBrokenTemplate(t, "<p>{{.Prenom}</p>")
	svc := NewTemplateService(repo)

	_, err := svc.Preview(context.Background(), tenantID, tmplID, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}
