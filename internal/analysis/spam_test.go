package analysis_test

import (
	"testing"

	"github.com/statoon54/mailhive/internal/analysis"
)

func newChecker() *analysis.SpamChecker {
	return analysis.NewSpamChecker()
}

func TestSpamChecker_CleanEmail(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"Votre commande est confirmée",
		"Bonjour, votre commande #1234 est confirmée. Pour vous désinscrire, cliquez ici.",
		"<html><body><p>Bonjour, votre commande #1234 est confirmée.</p><a href='https://example.com/unsubscribe'>Se désinscrire</a></body></html>",
	)
	if result.Score > 1.0 {
		t.Errorf("score trop élevé pour un email propre : %.1f (règles: %+v)", result.Score, result.Rules)
	}
	if !result.Pass {
		t.Error("un email propre devrait passer")
	}
}

func TestSpamChecker_SpamKeywords(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"FREE GIFT - ACT NOW! Limited time offer!",
		"Click here to claim your free gift now! This is a limited time exclusive deal.",
		"",
	)
	var keywordsScore float32
	for _, r := range result.Rules {
		if r.Name == "spam_keywords" {
			keywordsScore = r.Score
		}
	}
	if keywordsScore == 0 {
		t.Error("la règle spam_keywords aurait dû se déclencher")
	}
	if result.Score < 2.0 {
		t.Errorf("score attendu >= 2.0, obtenu %.1f", result.Score)
	}
}

func TestSpamChecker_ExcessiveCaps(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"info",
		"THIS IS A VERY IMPORTANT MESSAGE THAT YOU SHOULD READ IMMEDIATELY BECAUSE IT CONTAINS VITAL INFORMATION",
		"",
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "excessive_caps" {
			found = true
		}
	}
	if !found {
		t.Error("la règle excessive_caps aurait dû se déclencher")
	}
}

func TestSpamChecker_NoTextBody(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"Newsletter",
		"",
		"<html><body><p>Contenu HTML uniquement</p></body></html>",
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "text_html_ratio" {
			found = true
		}
	}
	if !found {
		t.Error("la règle text_html_ratio aurait dû se déclencher (HTML sans texte)")
	}
}

func TestSpamChecker_SuspiciousLinks(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"Check this out",
		"Visit http://192.168.1.1/page and https://bit.ly/abc123",
		"",
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "suspicious_links" {
			found = true
		}
	}
	if !found {
		t.Error("la règle suspicious_links aurait dû se déclencher")
	}
}

func TestSpamChecker_AllCapsSubject(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"URGENT IMPORTANT MESSAGE",
		"Please read this carefully. Unsubscribe here.",
		"",
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "all_caps_subject" {
			found = true
		}
	}
	if !found {
		t.Error("la règle all_caps_subject aurait dû se déclencher")
	}
}

func TestSpamChecker_HiddenText(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"Hello",
		"Normal text. Unsubscribe link.",
		`<html><body><p>Visible text</p><span style="display: none">Hidden spam keywords</span></body></html>`,
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "hidden_text" {
			found = true
		}
	}
	if !found {
		t.Error("la règle hidden_text aurait dû se déclencher")
	}
}

func TestSpamChecker_ExcessivePunctuation(t *testing.T) {
	sc := newChecker()
	result := sc.Check(
		"SALE!!! Don't miss out???",
		"Great deals. Unsubscribe here.",
		"",
	)
	var found bool
	for _, r := range result.Rules {
		if r.Name == "excessive_punctuation" {
			found = true
		}
	}
	if !found {
		t.Error("la règle excessive_punctuation aurait dû se déclencher")
	}
}

func TestSpamChecker_MaxScoreCapped(t *testing.T) {
	sc := newChecker()
	// Email cumulant toutes les règles
	result := sc.Check(
		"FREE MONEY!!! ACT NOW??? CLICK HERE!!!",
		"",
		`<html><body>
			<a href="http://192.168.1.1/page">https://example.com</a>
			<a href="https://bit.ly/x">click here</a>
			<span style="display:none; font-size:0">HIDDEN HIDDEN HIDDEN</span>
			EARN MONEY FREE GIFT LIMITED TIME EXCLUSIVE DEAL GUARANTEED WINNER CONGRATULATIONS
		</body></html>`,
	)
	if result.Score > 10.0 {
		t.Errorf("le score devrait être cappé à 10.0, obtenu %.1f", result.Score)
	}
}

func TestSpamChecker_WithThreshold_Block(t *testing.T) {
	sc := newChecker()
	threshold := float32(1.0)
	result := sc.CheckWithThreshold(
		"FREE MONEY ACT NOW!!!",
		"",
		"<html><body>spam content</body></html>",
		&threshold,
	)
	if result.Pass {
		t.Errorf("le mail aurait dû échouer avec seuil=1.0 et score=%.1f", result.Score)
	}
}

func TestSpamChecker_WithThreshold_Pass(t *testing.T) {
	sc := newChecker()
	threshold := float32(9.0)
	result := sc.CheckWithThreshold(
		"Bonjour",
		"Contenu normal. Pour se désinscrire, cliquez ici.",
		"<html><body><p>Contenu normal</p></body></html>",
		&threshold,
	)
	if !result.Pass {
		t.Errorf("le mail aurait dû passer avec seuil=9.0 et score=%.1f", result.Score)
	}
}
