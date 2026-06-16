package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/statoon54/mailhive/internal/config"
)

// LLMService gère la génération de contenu via un fournisseur LLM.
type LLMService struct {
	cfg    config.LLMConfig
	client *http.Client
}

// NewLLMService crée un nouveau service LLM.
func NewLLMService(cfg config.LLMConfig) *LLMService {
	return &LLMService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateRequest représente une requête de génération de contenu.
type GenerateRequest struct {
	Prompt   string `json:"prompt"`
	Language string `json:"language"`
}

// GenerateResponse représente la réponse de génération de contenu.
type GenerateResponse struct {
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

const systemPrompt = `Tu es un expert en rédaction d'emails HTML professionnels et visuellement attractifs.
Tu génères UNIQUEMENT le contenu HTML du body d'un email (pas de <html>, <head>, <body>, juste le contenu intérieur).

RÈGLES DE STRUCTURE :
- Enveloppe tout le contenu dans un <table> principal centré (max-width: 600px) avec cellpadding et cellspacing.
- Utilise des <table> imbriquées pour la mise en page (colonnes, sections), PAS de <div>.
- Chaque section (en-tête, contenu, pied) doit être un <tr> distinct.

RÈGLES DE STYLE :
- TOUS les styles doivent être inline (attribut style="...").
- Utilise des couleurs professionnelles : arrière-plan de section (#f8f9fa, #ffffff), texte (#333333), accents (#4f46e5, #2563eb).
- Applique du padding généreux (20px-40px) et des marges via cellpadding.
- Les titres doivent avoir une taille visible (font-size: 24px-28px pour h1, 20px pour h2) et une couleur contrastée.
- Les paragraphes doivent avoir line-height: 1.6, font-size: 16px, color: #555555.
- Cree des boutons d'appel à l'action avec un <a> stylé : background-color, couleur blanche, padding: 12px 24px, border-radius: 6px, text-decoration: none, display: inline-block.
- Ajoute des séparateurs visuels (<hr> ou bordure de cellule) entre les sections.

BALISES AUTORISÉES : <table>, <tr>, <td>, <th>, <h1>-<h3>, <p>, <strong>, <em>, <a>, <ul>, <ol>, <li>, <br>, <hr>, <img>, <span>.
Ne génère PAS de CSS externe, de <style>, de classes CSS ni de <div>.
Réponds UNIQUEMENT avec le HTML, sans explication, sans commentaire, sans bloc markdown.`

const systemPromptEN = `You are an expert in writing professional, visually appealing HTML emails.
You generate ONLY the HTML content of an email body (no <html>, <head>, <body>, just the inner content).

STRUCTURE RULES:
- Wrap all content in a centered main <table> (max-width: 600px) with cellpadding and cellspacing.
- Use nested <table> elements for layout (columns, sections), NOT <div>.
- Each section (header, content, footer) must be a separate <tr>.

STYLE RULES:
- ALL styles must be inline (style="..." attribute).
- Use professional colors: section backgrounds (#f8f9fa, #ffffff), text (#333333), accents (#4f46e5, #2563eb).
- Apply generous padding (20px-40px) and margins via cellpadding.
- Headings must have visible size (font-size: 24px-28px for h1, 20px for h2) and contrasting color.
- Paragraphs must have line-height: 1.6, font-size: 16px, color: #555555.
- Create call-to-action buttons with a styled <a>: background-color, white color, padding: 12px 24px, border-radius: 6px, text-decoration: none, display: inline-block.
- Add visual separators (<hr> or cell borders) between sections.

ALLOWED TAGS: <table>, <tr>, <td>, <th>, <h1>-<h3>, <p>, <strong>, <em>, <a>, <ul>, <ol>, <li>, <br>, <hr>, <img>, <span>.
Do NOT generate external CSS, <style> tags, CSS classes, or <div>.
Reply ONLY with the HTML, no explanation, no comment, no markdown code block.`

// Generate génère du contenu HTML pour un email à partir d'un prompt.
func (s *LLMService) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	sys := systemPrompt
	if req.Language == "en" {
		sys = systemPromptEN
	}

	switch s.cfg.Provider {
	case "ollama":
		return s.generateOllama(ctx, sys, req.Prompt)
	case "openai":
		return s.generateOpenAI(ctx, sys, req.Prompt)
	default:
		return nil, fmt.Errorf("fournisseur LLM non supporté : %s", s.cfg.Provider)
	}
}

// Enabled retourne true si le service LLM est configuré.
func (s *LLMService) Enabled() bool {
	return s.cfg.BaseURL != ""
}

// ollamaRequest représente une requête vers l'API Ollama.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

// ollamaResponse représente la réponse de l'API Ollama.
type ollamaResponse struct {
	Response string `json:"response"`
}

func (s *LLMService) generateOllama(
	ctx context.Context,
	system, prompt string,
) (*GenerateResponse, error) {
	body, err := json.Marshal(ollamaRequest{
		Model:  s.cfg.Model,
		Prompt: prompt,
		System: system,
		Stream: false,
	})
	if err != nil {
		return nil, fmt.Errorf("erreur de sérialisation : %w", err)
	}

	url := strings.TrimRight(s.cfg.BaseURL, "/") + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erreur de création de la requête : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur d'appel à Ollama : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erreur Ollama (HTTP %d) : %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("erreur de décodage de la réponse Ollama : %w", err)
	}

	html := extractHTML(ollamaResp.Response)
	return &GenerateResponse{
		HTMLBody: html,
		TextBody: stripHTML(html),
	}, nil
}

// openaiRequest représente une requête vers une API compatible OpenAI.
type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (s *LLMService) generateOpenAI(
	ctx context.Context,
	system, prompt string,
) (*GenerateResponse, error) {
	body, err := json.Marshal(openaiRequest{
		Model: s.cfg.Model,
		Messages: []openaiMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("erreur de sérialisation : %w", err)
	}

	url := strings.TrimRight(s.cfg.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erreur de création de la requête : %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur d'appel au LLM : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erreur LLM (HTTP %d) : %s", resp.StatusCode, string(respBody))
	}

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("erreur de décodage de la réponse : %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("réponse vide du LLM")
	}

	html := extractHTML(openaiResp.Choices[0].Message.Content)
	return &GenerateResponse{
		HTMLBody: html,
		TextBody: stripHTML(html),
	}, nil
}

// extractHTML nettoie la réponse LLM en extrayant le HTML.
func extractHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	// Retirer les blocs de code markdown ```html ... ```
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		var result []string
		inBlock := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock || !strings.HasPrefix(raw, "```") {
				result = append(result, line)
			}
		}
		if len(result) > 0 {
			return strings.TrimSpace(strings.Join(result, "\n"))
		}
	}
	return raw
}

// stripHTML produit une version texte brut à partir du HTML.
func stripHTML(html string) string {
	replacer := strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"<p>", "", "</p>", "\n",
		"<h1>", "\n", "</h1>", "\n",
		"<h2>", "\n", "</h2>", "\n",
		"<h3>", "\n", "</h3>", "\n",
		"<li>", "- ", "</li>", "\n",
		"<ul>", "", "</ul>", "\n",
		"<ol>", "", "</ol>", "\n",
		"<tr>", "", "</tr>", "\n",
		"<td>", " ", "</td>", "",
		"<th>", " ", "</th>", "",
		"<hr>", "\n---\n", "<hr/>", "\n---\n", "<hr />", "\n---\n",
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
	)
	text := replacer.Replace(html)
	// Supprimer les balises restantes
	var result strings.Builder
	inTag := false
	for _, r := range text {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Condenser les lignes blanches multiples en une seule
	lines := strings.Split(result.String(), "\n")
	var condensed []string
	prevEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !prevEmpty {
				condensed = append(condensed, "")
			}
			prevEmpty = true
		} else {
			condensed = append(condensed, trimmed)
			prevEmpty = false
		}
	}
	return strings.TrimSpace(strings.Join(condensed, "\n"))
}
