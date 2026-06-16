package templates

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
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

// Compile pré-parse les 3 templates d'un coup. Retourne une erreur si un template est invalide.
func Compile(subjectTmpl, textBody, htmlBody string) (*Compiled, error) {
	c := &Compiled{}
	var err error

	if subjectTmpl != "" {
		c.subject, err = template.New("subject").Parse(subjectTmpl)
		if err != nil {
			return nil, fmt.Errorf("erreur de parsing du sujet : %w", err)
		}
	}
	if textBody != "" {
		c.text, err = template.New("text").Parse(textBody)
		if err != nil {
			return nil, fmt.Errorf("erreur de parsing du template texte : %w", err)
		}
	}
	if htmlBody != "" {
		c.html, err = template.New("html").Parse(htmlBody)
		if err != nil {
			return nil, fmt.Errorf("erreur de parsing du template HTML : %w", err)
		}
	}

	return c, nil
}

// RenderSubject exécute le sujet pré-compilé avec les données fournies.
func (c *Compiled) RenderSubject(data map[string]string) (string, error) {
	if c.subject == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := c.subject.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du sujet : %w", err)
	}
	return buf.String(), nil
}

// RenderText exécute le template texte pré-compilé avec les données fournies.
func (c *Compiled) RenderText(data map[string]string) (string, error) {
	if c.text == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := c.text.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du template texte : %w", err)
	}
	return buf.String(), nil
}

// RenderHTML exécute le template HTML pré-compilé avec les données fournies.
func (c *Compiled) RenderHTML(data map[string]string) (string, error) {
	if c.html == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := c.html.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du template HTML : %w", err)
	}
	return buf.String(), nil
}

// RenderText rend un template texte avec les données fournies.
func RenderText(tmplStr string, data map[string]string) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	tmpl, err := template.New("text").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing du template texte : %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du template texte : %w", err)
	}
	return buf.String(), nil
}

// RenderHTML rend un template HTML avec les données fournies.
func RenderHTML(tmplStr string, data map[string]string) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	tmpl, err := template.New("html").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing du template HTML : %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du template HTML : %w", err)
	}
	return buf.String(), nil
}

// RenderSubject rend le sujet d'un template avec les données fournies.
func RenderSubject(subjectTmpl string, data map[string]string) (string, error) {
	if subjectTmpl == "" {
		return "", nil
	}
	tmpl, err := template.New("subject").Parse(subjectTmpl)
	if err != nil {
		return "", fmt.Errorf("erreur de parsing du sujet : %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("erreur de rendu du sujet : %w", err)
	}
	return buf.String(), nil
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
