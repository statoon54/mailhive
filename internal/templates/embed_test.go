package templates_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/statoon54/mailhive/internal/domain"
	"github.com/statoon54/mailhive/internal/templates"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		subject   string
		text      string
		html      string
		wantField string // "" : aucune erreur attendue
	}{
		{
			name:    "template valide",
			subject: "Bonjour {{.Prenom}}",
			text:    "Salut {{.Prenom}}",
			html:    "<p>Bonjour {{.Prenom}}</p>",
		},
		{
			name: "champs vides",
		},
		{
			name:      "action non fermée dans le sujet",
			subject:   "Bonjour {{.Prenom}",
			wantField: templates.FieldSubject,
		},
		{
			name:      "variable sans point dans le texte",
			text:      "Salut {{Prenom}}",
			wantField: templates.FieldText,
		},
		{
			name:      "end orphelin dans le HTML",
			html:      "<p>{{end}}</p>",
			wantField: templates.FieldHTML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := templates.Validate(tt.subject, tt.text, tt.html)

			if tt.wantField == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrValidation,
				"une erreur de syntaxe doit produire un 400, pas un 500")

			tmplErr, ok := errors.AsType[*templates.Error](err)
			require.True(t, ok, "l'erreur doit être un *templates.Error")
			assert.Equal(t, tt.wantField, tmplErr.Field)
			assert.NotEmpty(t, tmplErr.Detail)
		})
	}
}

func TestCompiled_RenderHTML_ChampImbriqueSurUneChaine(t *testing.T) {
	// Erreur d'exécution et non de parsing : {{.User.Nom}} parse correctement,
	// mais .User vaut une chaîne dans une map[string]string.
	compiled, err := templates.Compile("", "", "<p>{{.User.Nom}}</p>")
	require.NoError(t, err)

	_, err = compiled.RenderHTML(map[string]string{"User": "x"})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
	tmplErr, ok := errors.AsType[*templates.Error](err)
	require.True(t, ok)
	assert.Equal(t, templates.FieldHTML, tmplErr.Field)
}

func TestError_MessageNormalise(t *testing.T) {
	// text/template préfixe ses messages de « template: <nom>:<ligne>: » et, pour
	// une erreur d'exécution, répète le nom dans « executing "<nom>" at ». Le
	// champ étant déjà porté par Field, ces répétitions sont retirées.
	tests := []struct {
		name       string
		run        func() error
		wantLine   int
		wantDetail string
	}{
		{
			name:       "erreur de parsing",
			run:        func() error { return templates.Validate("", "", "<p>{{.Prenom}</p>") },
			wantLine:   1,
			wantDetail: "bad character U+007D '}'",
		},
		{
			name:       "erreur de parsing sur la deuxième ligne",
			run:        func() error { return templates.Validate("", "", "<p>\n{{.Prenom}\n</p>") },
			wantLine:   2,
			wantDetail: "bad character U+007D '}'",
		},
		{
			name: "erreur d'exécution",
			run: func() error {
				compiled, err := templates.Compile("", "", "<p>{{.Prenom.Nom}}</p>")
				if err != nil {
					return err
				}
				_, err = compiled.RenderHTML(map[string]string{"Prenom": "Jean"})
				return err
			},
			wantLine:   1,
			wantDetail: "<.Prenom.Nom>: can't evaluate field Nom in type string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			require.Error(t, err)

			tmplErr, ok := errors.AsType[*templates.Error](err)
			require.True(t, ok)
			assert.Equal(t, templates.FieldHTML, tmplErr.Field)
			assert.Equal(t, tt.wantLine, tmplErr.Line)
			assert.Equal(t, tt.wantDetail, tmplErr.Detail)
			assert.NotContains(t, tmplErr.Detail, templates.FieldHTML,
				"le nom du champ ne doit pas être répété dans le détail")
		})
	}
}
