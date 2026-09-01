package domain

import (
	json "encoding/json/v2"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ces tests figent la forme JSON exacte du contrat d'API public, désormais
// produite par encoding/json/v2. Toute différence de sortie doit être une
// décision explicite — et donc une mise à jour consciente de `want` — jamais un
// effet de bord silencieux.
//
// Ce qui a changé lors de la migration, et que ces cas verrouillent :
//   - les slices et maps nulles sortent en `[]` et `{}` au lieu de `null` ;
//   - `status_code` et `individuel` restent omis à zéro grâce à `omitzero`,
//     `omitempty` ne l'assurant plus pour les nombres et booléens ;
//   - les champs pointeurs restent omis quand ils sont nuls, et présents dès
//     qu'ils sont renseignés — y compris vers une chaîne vide, ce qu'`omitempty`
//     aurait silencieusement supprimé en v2.

// jsonContractCase décrit une valeur et la sortie JSON attendue pour elle.
type jsonContractCase struct {
	value any
	name  string
	want  string
}

// fixedTime est une date figée : le contrat ne doit pas dépendre de l'horloge.
var fixedTime = time.Date(2026, 3, 25, 10, 30, 0, 0, time.UTC)

// fixedUUID est un UUID figé, pour la même raison.
var fixedUUID = uuid.MustParse("018f3a2b-0000-7000-8000-000000000001")

// TestJSONContract_Analysis fige les réponses des endpoints d'analyse.
//
// Les champs `rules`, `issues`, `clients` et `links` n'ont pas d'`omitempty` :
// ce sont les collections dont la sérialisation d'une valeur nulle change entre
// les deux moteurs.
func TestJSONContract_Analysis(t *testing.T) {
	runJSONContract(t, []jsonContractCase{
		{
			name:  "SpamCheckResult sans règle déclenchée",
			value: SpamCheckResult{},
			want:  `{"rules":[],"max_score":0,"score":0,"pass":false}`,
		},
		{
			name:  "HTMLCheckResult sans problème",
			value: HTMLCheckResult{},
			want:  `{"issues":[],"total_count":0}`,
		},
		{
			name:  "HTMLCompatIssue sans client concerné",
			value: HTMLCompatIssue{},
			want:  `{"selector":"","property":"","description":"","severity":"","clients":[]}`,
		},
		{
			name:  "LinkCheckResult sans lien",
			value: LinkCheckResult{},
			want:  `{"links":[],"total_count":0,"broken_count":0}`,
		},
		{
			// status_code vaut 0 : il doit rester absent. C'est précisément le
			// champ qu'`omitempty` cesserait d'omettre en v2 sans passage à
			// `omitzero`.
			name:  "LinkStatus sans code HTTP",
			value: LinkStatus{},
			want:  `{"source":"","status":"","url":""}`,
		},
		{
			name:  "LinkStatus avec code HTTP",
			value: LinkStatus{Source: "href", Status: "ok", URL: "https://exemple.test", StatusCode: 200},
			want:  `{"source":"href","status":"ok","url":"https://exemple.test","status_code":200}`,
		},
	})
}

// TestJSONContract_Template fige les réponses des endpoints de templates.
func TestJSONContract_Template(t *testing.T) {
	runJSONContract(t, []jsonContractCase{
		{
			name:  "Template sans variable",
			value: Template{},
			want: `{"variables":{},"created_at":"0001-01-01T00:00:00Z",` +
				`"id":"00000000-0000-0000-0000-000000000000",` +
				`"tenant_id":"00000000-0000-0000-0000-000000000000",` +
				`"updated_at":"0001-01-01T00:00:00Z","html_body":"","name":"","slug":"",` +
				`"subject_tmpl":"","text_body":"","is_active":false}`,
		},
		{
			name:  "PreviewTemplateRequest sans donnée",
			value: PreviewTemplateRequest{},
			want:  `{"data":{}}`,
		},
	})
}

// TestJSONContract_Mail fige les réponses des endpoints de mails.
func TestJSONContract_Mail(t *testing.T) {
	runJSONContract(t, []jsonContractCase{
		{
			name: "Mail minimal",
			value: Mail{
				ID:        fixedUUID,
				CreatedAt: fixedTime,
				UpdatedAt: fixedTime,
				Status:    MailStatusPending,
				Priority:  MailPriorityDefault,
			},
			want: `{"created_at":"2026-03-25T10:30:00Z",` +
				`"id":"018f3a2b-0000-7000-8000-000000000001","priority":"default",` +
				`"status":"pending","tenant_id":"00000000-0000-0000-0000-000000000000",` +
				`"updated_at":"2026-03-25T10:30:00Z","from_email":"","from_name":"",` +
				`"html_body":"","subject":"","text_body":"","attempts":0}`,
		},
		{
			// `individuel` vaut false : il doit rester absent. Second champ que
			// `omitempty` cesserait d'omettre en v2.
			name: "CreateMailRequest non individuel",
			value: CreateMailRequest{
				To:      []EmailAddress{{Email: "destinataire@exemple.test"}},
				Subject: "Sujet",
			},
			want: `{"subject":"Sujet","to":[{"email":"destinataire@exemple.test"}]}`,
		},
		{
			name:  "MailRecipient sans nom",
			value: MailRecipient{},
			want: `{"id":"00000000-0000-0000-0000-000000000000",` +
				`"mail_id":"00000000-0000-0000-0000-000000000000","type":"","email":""}`,
		},
		{
			name:  "EmailAddress sans nom",
			value: EmailAddress{},
			want:  `{"email":""}`,
		},
		{
			name:  "PaginatedList vide",
			value: PaginatedList[Mail]{},
			want:  `{"items":[],"total":0,"page":0,"limit":0,"total_pages":0}`,
		},
	})
}

// TestJSONContract_NullablePointers verrouille la sémantique des champs
// pointeurs des requêtes de modification partielle.
//
// La distinction porte tout le sens de ces endpoints : un pointeur nul signifie
// « ne pas toucher au champ », un pointeur vers la valeur zéro signifie « mettre
// à vide ». En v2, `omitempty` aurait confondu les deux pour les chaînes — d'où
// `omitzero` sur tous les champs pointeurs.
func TestJSONContract_NullablePointers(t *testing.T) {
	vide := ""
	faux := false

	runJSONContract(t, []jsonContractCase{
		{
			name:  "aucun champ à modifier",
			value: UpdateTemplateRequest{},
			want:  `{}`,
		},
		{
			name:  "mise à vide explicite d'une chaîne",
			value: UpdateTemplateRequest{Name: &vide},
			want:  `{"name":""}`,
		},
		{
			name:  "désactivation explicite d'un booléen",
			value: UpdateTemplateRequest{IsActive: &faux},
			want:  `{"is_active":false}`,
		},
	})
}

// TestJSONContract_RoundTrip vérifie que les requêtes entrantes se décodent
// toujours correctement, y compris les marshaleurs maison de FlexTime.
func TestJSONContract_RoundTrip(t *testing.T) {
	t.Run("FlexTime accepte les formats documentés", func(t *testing.T) {
		for _, raw := range []string{
			`"2026-03-25T10:30:00Z"`,
			`"2026-03-25T10:30:00"`,
			`"2026-03-25 10:30:00"`,
			`"2026-03-25"`,
		} {
			var ft FlexTime
			require.NoError(t, json.Unmarshal([]byte(raw), &ft), "entrée %s", raw)
			assert.False(t, time.Time(ft).IsZero(), "entrée %s", raw)
		}
	})

	t.Run("CreateMailRequest se décode complètement", func(t *testing.T) {
		const body = `{
			"subject": "Sujet",
			"to": [{"email": "destinataire@exemple.test", "name": "Destinataire"}],
			"cc": [{"email": "copie@exemple.test"}],
			"individuel": true,
			"tags": ["facture"],
			"template_data": {"nom": "Ada"},
			"scheduled_at": "2026-03-25T10:30:00Z"
		}`
		var req CreateMailRequest
		require.NoError(t, json.Unmarshal([]byte(body), &req))

		assert.Equal(t, "Sujet", req.Subject)
		require.Len(t, req.To, 1)
		assert.Equal(t, "destinataire@exemple.test", req.To[0].Email)
		assert.Equal(t, "Destinataire", req.To[0].Name)
		require.Len(t, req.CC, 1)
		assert.True(t, req.Individuel)
		assert.Equal(t, []string{"facture"}, req.Tags)
		assert.Equal(t, map[string]string{"nom": "Ada"}, req.TemplateData)
		require.NotNil(t, req.ScheduledAt)
		assert.Equal(t, fixedTime, *req.ScheduledAt.TimePtr())
	})
}

// runJSONContract sérialise chaque cas et compare la sortie octet pour octet.
func runJSONContract(t *testing.T, cases []jsonContractCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.value)
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}
