package analysis_test

import (
	"testing"

	"github.com/statoon54/mailhive/internal/analysis"
)

func newHTMLChecker() *analysis.HTMLCompatChecker {
	return analysis.NewHTMLCompatChecker()
}

func TestHTMLCompat_Flexbox(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<div style="display: flex; justify-content: center;"><p>Contenu</p></div>`)
	found := false
	for _, issue := range result.Issues {
		if issue.Property == "display: flex" {
			found = true
			if issue.Severity != "error" {
				t.Errorf("flexbox devrait être severity error, obtenu %s", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("flexbox devrait être détecté comme problème")
	}
}

func TestHTMLCompat_Grid(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<div style="display: grid; grid-template-columns: 1fr 1fr;"><p>A</p><p>B</p></div>`)
	found := false
	for _, issue := range result.Issues {
		if issue.Property == "display: grid" {
			found = true
		}
	}
	if !found {
		t.Error("CSS Grid devrait être détecté comme problème")
	}
}

func TestHTMLCompat_CleanTableLayout(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`
		<table width="600" cellpadding="0" cellspacing="0">
			<tr><td style="padding: 20px; font-family: Arial, sans-serif;">
				<h1>Bonjour</h1>
				<p>Ceci est un email compatible.</p>
			</td></tr>
		</table>
	`)
	if result.TotalCount > 0 {
		t.Errorf("un layout table propre ne devrait pas avoir de problèmes, obtenu %d: %+v",
			result.TotalCount, result.Issues)
	}
}

func TestHTMLCompat_VideoTag(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<video src="https://example.com/video.mp4" controls></video>`)
	found := false
	for _, issue := range result.Issues {
		if issue.Property == "<video>" {
			found = true
		}
	}
	if !found {
		t.Error("<video> devrait être détecté comme problème")
	}
}

func TestHTMLCompat_CSSAnimation(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<style>@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
		.anim { animation: fadeIn 1s; }</style>`)
	if result.TotalCount < 2 {
		t.Errorf("@keyframes et animation devraient être détectés, obtenu %d problèmes", result.TotalCount)
	}
}

func TestHTMLCompat_RemUnits(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<p style="font-size: 1.5rem; margin: 2vh;">Texte</p>`)
	found := false
	for _, issue := range result.Issues {
		if issue.Property == "rem/vh/vw" {
			found = true
		}
	}
	if !found {
		t.Error("les unités rem/vh devraient être détectées comme problème")
	}
}

func TestHTMLCompat_FormTag(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check(`<form action="/submit"><input type="text" name="email"><button type="submit">OK</button></form>`)
	found := false
	for _, issue := range result.Issues {
		if issue.Property == "<form>" {
			found = true
		}
	}
	if !found {
		t.Error("<form> devrait être détecté comme problème")
	}
}

func TestHTMLCompat_EmptyHTML(t *testing.T) {
	hc := newHTMLChecker()
	result := hc.Check("")
	if result.TotalCount != 0 {
		t.Error("un HTML vide ne devrait pas avoir de problèmes")
	}
}
