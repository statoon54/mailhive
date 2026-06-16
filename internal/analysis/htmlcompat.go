package analysis

import (
	"regexp"
	"strings"

	"github.com/statoon54/mailhive/internal/domain"
)

// htmlCompatRule est une règle de compatibilité HTML email.
type htmlCompatRule struct {
	pattern     *regexp.Regexp
	property    string
	description string
	severity    string
	clients     []string
}

// HTMLCompatChecker analyse le HTML pour la compatibilité avec les clients email.
type HTMLCompatChecker struct {
	rules []htmlCompatRule
}

// NewHTMLCompatChecker crée un vérificateur avec les règles caniemail intégrées.
func NewHTMLCompatChecker() *HTMLCompatChecker {
	return &HTMLCompatChecker{
		rules: builtinHTMLRules(),
	}
}

func builtinHTMLRules() []htmlCompatRule {
	return []htmlCompatRule{
		{
			pattern:     regexp.MustCompile(`(?i)display\s*:\s*flex`),
			property:    "display: flex",
			description: "Flexbox non supporté par Outlook",
			severity:    "error",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)display\s*:\s*grid`),
			property:    "display: grid",
			description: "CSS Grid non supporté par Outlook, support partiel Gmail",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)position\s*:\s*(absolute|fixed|sticky)`),
			property:    "position: absolute/fixed/sticky",
			description: "Positionnement CSS non supporté par la plupart des clients",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail", "Yahoo"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)@media\s`),
			property:    "@media",
			description: "Media queries avec support limité sur Outlook",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)background-image\s*:`),
			property:    "background-image",
			description: "Images de fond avec support partiel sur Outlook",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)border-radius\s*:`),
			property:    "border-radius",
			description: "Coins arrondis non supportés par Outlook < 2019",
			severity:    "warning",
			clients:     []string{"Outlook < 2019"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)box-shadow\s*:`),
			property:    "box-shadow",
			description: "Ombres non supportées par Outlook",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)<\s*video[\s>]`),
			property:    "<video>",
			description: "Balise vidéo non supportée par la plupart des clients",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)<\s*audio[\s>]`),
			property:    "<audio>",
			description: "Balise audio non supportée par la plupart des clients",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)<\s*form[\s>]`),
			property:    "<form>",
			description: "Formulaires supprimés par Gmail",
			severity:    "error",
			clients:     []string{"Gmail"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)@keyframes\s`),
			property:    "@keyframes",
			description: "Animations CSS non supportées par la plupart des clients",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail", "Yahoo"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)(animation|animation-name)\s*:`),
			property:    "animation",
			description: "Propriétés d'animation non supportées par la plupart des clients",
			severity:    "error",
			clients:     []string{"Outlook", "Gmail", "Yahoo"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)\d+(\.\d+)?(rem|vh|vw)\b`),
			property:    "rem/vh/vw",
			description: "Unités CSS relatives non supportées par Outlook",
			severity:    "error",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)<\s*svg[\s>]`),
			property:    "<svg>",
			description: "SVG inline avec support partiel sur Outlook",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)max-width\s*:`),
			property:    "max-width",
			description: "max-width ignoré par Outlook sur certains éléments",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
		{
			pattern:     regexp.MustCompile(`(?i)margin\s*:\s*(0\s+)?auto`),
			property:    "margin: auto",
			description: "Centrage via margin auto non fiable sur Outlook",
			severity:    "warning",
			clients:     []string{"Outlook"},
		},
	}
}

// Check analyse le HTML rendu et retourne les problèmes de compatibilité.
func (hc *HTMLCompatChecker) Check(html string) *domain.HTMLCheckResult {
	result := &domain.HTMLCheckResult{}
	if strings.TrimSpace(html) == "" {
		return result
	}

	for _, rule := range hc.rules {
		locs := rule.pattern.FindAllStringIndex(html, -1)
		if len(locs) > 0 {
			result.Issues = append(result.Issues, domain.HTMLCompatIssue{
				Property:    rule.property,
				Description: rule.description,
				Severity:    rule.severity,
				Clients:     rule.clients,
			})
		}
	}

	result.TotalCount = len(result.Issues)
	return result
}
