package analysis

import (
	"regexp"
	"strings"
	"unicode"
)

// --- Règle 1 : Ratio texte/HTML ---

type textHTMLRatioRule struct{}

func (r *textHTMLRatioRule) Name() string        { return "text_html_ratio" }
func (r *textHTMLRatioRule) Description() string { return "HTML présent sans alternative texte" }
func (r *textHTMLRatioRule) Check(subject, textBody, htmlBody string) (float32, string) {
	if htmlBody != "" && strings.TrimSpace(textBody) == "" {
		return 1.5, "Le mail contient du HTML mais pas de version texte"
	}
	return 0, ""
}

// --- Règle 2 : Mots-clés spam ---

var spamKeywords = []string{
	"act now", "action required", "apply now", "buy now", "call now",
	"click here", "click below", "deal ending", "do it today", "don't delete",
	"exclusive deal", "expire", "free", "get it now", "gift",
	"give it away", "guaranteed", "increase", "incredible deal", "limited time",
	"new customers only", "order now", "please read", "special promotion", "take action",
	"this won't last", "urgent", "what are you waiting for", "while supplies last", "winner",
	"you have been selected", "your account", "important information regarding", "verify your account",
	"congratulations", "no obligation", "risk-free", "satisfaction guaranteed",
	"double your", "earn money", "extra income", "make money", "million dollars",
	"cash bonus", "credit card", "investment", "no fees", "no cost",
	"unsecured", "lowest price", "best price", "discount", "save big",
}

type spamKeywordsRule struct{}

func (r *spamKeywordsRule) Name() string        { return "spam_keywords" }
func (r *spamKeywordsRule) Description() string { return "Mots-clés fréquemment associés au spam" }
func (r *spamKeywordsRule) Check(subject, textBody, htmlBody string) (float32, string) {
	content := strings.ToLower(subject + " " + textBody + " " + stripHTML(htmlBody))
	var found []string
	for _, kw := range spamKeywords {
		if strings.Contains(content, kw) {
			found = append(found, kw)
		}
	}
	if len(found) == 0 {
		return 0, ""
	}
	score := float32(len(found)) * 0.5
	if score > 3.0 {
		score = 3.0
	}
	return score, "Mots-clés détectés : " + strings.Join(found, ", ")
}

// --- Règle 3 : Majuscules excessives ---

type excessiveCapsRule struct{}

func (r *excessiveCapsRule) Name() string { return "excessive_caps" }
func (r *excessiveCapsRule) Description() string {
	return "Proportion excessive de majuscules dans le corps"
}
func (r *excessiveCapsRule) Check(subject, textBody, htmlBody string) (float32, string) {
	text := textBody + " " + stripHTML(htmlBody)
	if len(text) < 20 {
		return 0, ""
	}
	var upper, alpha int
	for _, c := range text {
		if unicode.IsLetter(c) {
			alpha++
			if unicode.IsUpper(c) {
				upper++
			}
		}
	}
	if alpha == 0 {
		return 0, ""
	}
	ratio := float64(upper) / float64(alpha)
	if ratio > 0.3 {
		return 1.5, "Plus de 30% de majuscules dans le contenu"
	}
	return 0, ""
}

// --- Règle 4 : Liens suspects ---

var (
	reIPURL       = regexp.MustCompile(`https?://\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	reShortener   = regexp.MustCompile(`(?i)https?://(bit\.ly|tinyurl\.com|t\.co|goo\.gl|ow\.ly|is\.gd|buff\.ly|rebrand\.ly|cutt\.ly)/`)
	reHrefContent = regexp.MustCompile(`(?i)<a\s[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
)

type suspiciousLinksRule struct{}

func (r *suspiciousLinksRule) Name() string { return "suspicious_links" }
func (r *suspiciousLinksRule) Description() string {
	return "Liens suspects (IP, raccourcisseurs, texte trompeur)"
}
func (r *suspiciousLinksRule) Check(subject, textBody, htmlBody string) (float32, string) {
	content := textBody + " " + htmlBody
	var issues []string

	if reIPURL.MatchString(content) {
		issues = append(issues, "URL avec adresse IP")
	}
	if reShortener.MatchString(content) {
		issues = append(issues, "Raccourcisseur d'URL")
	}

	// Texte d'ancre différent du href (phishing)
	matches := reHrefContent.FindAllStringSubmatch(htmlBody, -1)
	for _, m := range matches {
		href := strings.ToLower(m[1])
		text := strings.ToLower(strings.TrimSpace(m[2]))
		if strings.HasPrefix(text, "http") && !strings.Contains(href, text) {
			issues = append(issues, "Texte d'ancre ne correspond pas au lien")
			break
		}
	}

	if len(issues) == 0 {
		return 0, ""
	}
	score := float32(len(issues)) * 1.0
	if score > 2.0 {
		score = 2.0
	}
	return score, strings.Join(issues, " ; ")
}

// --- Règle 5 : Absence de lien de désinscription ---

type missingUnsubscribeRule struct{}

func (r *missingUnsubscribeRule) Name() string        { return "missing_unsubscribe" }
func (r *missingUnsubscribeRule) Description() string { return "Absence de lien de désinscription" }
func (r *missingUnsubscribeRule) Check(subject, textBody, htmlBody string) (float32, string) {
	content := strings.ToLower(htmlBody + " " + textBody)
	keywords := []string{"unsubscribe", "désinscri", "se désabonner", "opt-out", "optout", "désinscrire"}
	for _, kw := range keywords {
		if strings.Contains(content, kw) {
			return 0, ""
		}
	}
	return 0.5, "Aucun lien de désinscription détecté"
}

// --- Règle 6 : Ponctuation excessive dans le sujet ---

type excessivePunctuationRule struct{}

func (r *excessivePunctuationRule) Name() string        { return "excessive_punctuation" }
func (r *excessivePunctuationRule) Description() string { return "Ponctuation excessive dans le sujet" }
func (r *excessivePunctuationRule) Check(subject, textBody, htmlBody string) (float32, string) {
	if strings.Contains(subject, "!!!") || strings.Contains(subject, "???") || strings.Contains(subject, "!!!") {
		return 1.0, "Ponctuation excessive dans le sujet : !!!, ???"
	}
	excl := strings.Count(subject, "!")
	quest := strings.Count(subject, "?")
	if excl+quest >= 3 {
		return 1.0, "Trop de ! et ? dans le sujet"
	}
	return 0, ""
}

// --- Règle 7 : Sujet entièrement en majuscules ---

type allCapsSubjectRule struct{}

func (r *allCapsSubjectRule) Name() string        { return "all_caps_subject" }
func (r *allCapsSubjectRule) Description() string { return "Sujet entièrement en majuscules" }
func (r *allCapsSubjectRule) Check(subject, textBody, htmlBody string) (float32, string) {
	if len(subject) < 5 {
		return 0, ""
	}
	upper := strings.ToUpper(subject)
	if subject == upper && subject != strings.ToLower(subject) {
		return 2.0, "Le sujet est entièrement en majuscules"
	}
	return 0, ""
}

// --- Règle 8 : Texte caché ---

var reHiddenCSS = regexp.MustCompile(`(?i)(display\s*:\s*none|font-size\s*:\s*0|visibility\s*:\s*hidden)`)

type hiddenTextRule struct{}

func (r *hiddenTextRule) Name() string { return "hidden_text" }
func (r *hiddenTextRule) Description() string {
	return "Texte caché via CSS (display:none, font-size:0)"
}
func (r *hiddenTextRule) Check(subject, textBody, htmlBody string) (float32, string) {
	if reHiddenCSS.MatchString(htmlBody) {
		return 1.5, "CSS masquant du contenu détecté"
	}
	return 0, ""
}

// --- Utilitaire ---

var reHTMLTag = regexp.MustCompile(`<[^>]*>`)

// stripHTML supprime les balises HTML pour l'analyse du texte brut.
func stripHTML(html string) string {
	return reHTMLTag.ReplaceAllString(html, " ")
}
