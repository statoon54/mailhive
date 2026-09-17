package templates

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/statoon54/mailhive/internal/domain"
)

//go:embed *.tmpl
var templatesFS embed.FS

// defaultTextTmpl et defaultHTMLTmpl sont les templates par défaut pré-chargés.
var (
	defaultTextTmpl *template.Template
	defaultHTMLTmpl *template.Template
)

func init() {
	var err error
	defaultTextTmpl, err = template.ParseFS(templatesFS, "default_text.tmpl")
	if err != nil {
		panic(fmt.Sprintf("erreur de chargement du template texte par défaut : %v", err))
	}
	defaultHTMLTmpl, err = template.ParseFS(templatesFS, "default_html.tmpl")
	if err != nil {
		panic(fmt.Sprintf("erreur de chargement du template HTML par défaut : %v", err))
	}
}

// Noms des champs porteurs d'un template, tels qu'exposés par l'API. Ils
// servent à désigner le champ fautif dans les réponses d'erreur.
const (
	FieldSubject = "subject_tmpl"
	FieldText    = "text_body"
	FieldHTML    = "html_body"
)

// Error signale un template que l'on n'a pas pu parser ou exécuter : syntaxe
// incorrecte, ou variable référencée d'une manière impossible à résoudre.
//
// Elle enveloppe domain.ErrValidation : la faute est dans la donnée fournie par
// l'utilisateur, pas dans le serveur, et l'API doit répondre 400 et non 500.
type Error struct {
	Field  string // FieldSubject, FieldText ou FieldHTML
	Detail string // message brut de text/template
}

// Error retourne le message d'erreur, champ fautif compris.
func (e *Error) Error() string {
	return fmt.Sprintf("%s : %s", e.Field, e.Detail)
}

// Unwrap rattache l'erreur à domain.ErrValidation.
func (e *Error) Unwrap() error {
	return domain.ErrValidation
}

// newError construit une *Error en retirant le préfixe que text/template ajoute
// à ses messages (« template: html:1: ... »), inutile pour l'utilisateur.
func newError(field string, err error) *Error {
	detail := err.Error()
	if _, after, found := strings.Cut(detail, ": "); found {
		detail = after
	}
	return &Error{Field: field, Detail: detail}
}

// Validate vérifie que les trois templates d'un modèle de mail sont
// syntaxiquement corrects. Les champs vides sont ignorés.
func Validate(subjectTmpl, textBody, htmlBody string) error {
	_, err := Compile(subjectTmpl, textBody, htmlBody)
	return err
}

// ValidateData vérifie que toutes les variables déclarées dans le template sont fournies dans data.
func ValidateData(variables map[string]string, data map[string]string) error {
	var missing []string
	for key := range variables {
		if _, ok := data[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("variables manquantes dans template_data : %s", strings.Join(missing, ", "))
	}
	return nil
}

// Compiled contient les templates pré-compilés (subject, text, html) prêts à être exécutés
// avec différentes données sans re-parser. Thread-safe : *template.Template.Execute est concurrent-safe.
type Compiled struct {
	subject *template.Template
	text    *template.Template
	html    *template.Template
}

// Compile pré-parse les 3 templates d'un coup. Retourne une *Error si un template est invalide.
func Compile(subjectTmpl, textBody, htmlBody string) (*Compiled, error) {
	c := &Compiled{}
	var err error

	if subjectTmpl != "" {
		c.subject, err = template.New(FieldSubject).Parse(subjectTmpl)
		if err != nil {
			return nil, newError(FieldSubject, err)
		}
	}
	if textBody != "" {
		c.text, err = template.New(FieldText).Parse(textBody)
		if err != nil {
			return nil, newError(FieldText, err)
		}
	}
	if htmlBody != "" {
		c.html, err = template.New(FieldHTML).Parse(htmlBody)
		if err != nil {
			return nil, newError(FieldHTML, err)
		}
	}

	return c, nil
}

// RenderSubject exécute le sujet pré-compilé avec les données fournies.
func (c *Compiled) RenderSubject(data map[string]string) (string, error) {
	return execute(c.subject, FieldSubject, data)
}

// RenderText exécute le template texte pré-compilé avec les données fournies.
func (c *Compiled) RenderText(data map[string]string) (string, error) {
	return execute(c.text, FieldText, data)
}

// RenderHTML exécute le template HTML pré-compilé avec les données fournies.
func (c *Compiled) RenderHTML(data map[string]string) (string, error) {
	return execute(c.html, FieldHTML, data)
}

// execute exécute un template déjà parsé ; un template nil rend une chaîne vide.
func execute(tmpl *template.Template, field string, data map[string]string) (string, error) {
	if tmpl == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", newError(field, err)
	}
	return buf.String(), nil
}

// parseAndExecute parse puis exécute un template à usage unique.
func parseAndExecute(tmplStr, field string, data map[string]string) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	tmpl, err := template.New(field).Parse(tmplStr)
	if err != nil {
		return "", newError(field, err)
	}
	return execute(tmpl, field, data)
}

// RenderText rend un template texte avec les données fournies.
func RenderText(tmplStr string, data map[string]string) (string, error) {
	return parseAndExecute(tmplStr, FieldText, data)
}

// RenderHTML rend un template HTML avec les données fournies.
func RenderHTML(tmplStr string, data map[string]string) (string, error) {
	return parseAndExecute(tmplStr, FieldHTML, data)
}

// RenderSubject rend le sujet d'un template avec les données fournies.
func RenderSubject(subjectTmpl string, data map[string]string) (string, error) {
	return parseAndExecute(subjectTmpl, FieldSubject, data)
}

// RenderDefaultText rend le template texte par défaut.
func RenderDefaultText(body string) (string, error) {
	var buf bytes.Buffer
	if err := defaultTextTmpl.Execute(&buf, map[string]string{"Body": body}); err != nil {
		return "", fmt.Errorf("erreur de rendu du template texte par défaut : %w", err)
	}
	return buf.String(), nil
}

// RenderDefaultHTML rend le template HTML par défaut.
func RenderDefaultHTML(body string) (string, error) {
	var buf bytes.Buffer
	if err := defaultHTMLTmpl.Execute(&buf, map[string]string{"Body": body}); err != nil {
		return "", fmt.Errorf("erreur de rendu du template HTML par défaut : %w", err)
	}
	return buf.String(), nil
}
