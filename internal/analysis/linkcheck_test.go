package analysis_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/statoon54/mailhive/internal/analysis"
)

func TestLinkChecker_ValidLinks(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	lc := analysis.NewLinkChecker()
	// Le LinkChecker utilise son propre client, on teste avec un serveur HTTPS réel
	// Comme c'est un TLS test server avec cert auto-signé, on teste plutôt le parsing
	html := fmt.Sprintf(`<a href="%s/page">Link</a>`, srv.URL)
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 1 {
		t.Errorf("attendu 1 lien, obtenu %d", result.TotalCount)
	}
}

func TestLinkChecker_BrokenLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	lc := analysis.NewLinkChecker()
	html := fmt.Sprintf(`<a href="%s/missing">Link</a>`, srv.URL)
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 1 {
		t.Fatalf("attendu 1 lien, obtenu %d", result.TotalCount)
	}
	// HTTP lien -> insecure détecté en premier (scheme http)
	if result.Links[0].Status != "insecure" {
		t.Errorf("statut attendu 'insecure' pour HTTP, obtenu '%s'", result.Links[0].Status)
	}
}

func TestLinkChecker_InsecureLinks(t *testing.T) {
	lc := analysis.NewLinkChecker()
	html := `<a href="http://example.com/page">Link</a>`
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 1 {
		t.Fatalf("attendu 1 lien, obtenu %d", result.TotalCount)
	}
	if result.Links[0].Status != "insecure" {
		t.Errorf("statut attendu 'insecure', obtenu '%s'", result.Links[0].Status)
	}
}

func TestLinkChecker_RelativeLinks(t *testing.T) {
	lc := analysis.NewLinkChecker()
	html := `<a href="/relative/path">Link</a><img src="/images/logo.png">`
	result := lc.Check(context.Background(), html)

	for _, link := range result.Links {
		if link.Status != "invalid" {
			t.Errorf("lien relatif '%s' devrait être 'invalid', obtenu '%s'", link.URL, link.Status)
		}
	}
}

func TestLinkChecker_RedirectLinks(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://example.com/final")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()

	lc := analysis.NewLinkChecker()
	html := fmt.Sprintf(`<a href="%s/redirect">Link</a>`, srv.URL)
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 1 {
		t.Fatalf("attendu 1 lien, obtenu %d", result.TotalCount)
	}
}

func TestLinkChecker_TimeoutLinks(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	lc := analysis.NewLinkChecker()
	html := fmt.Sprintf(`<a href="%s/slow">Link</a>`, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	result := lc.Check(ctx, html)

	if result.TotalCount != 1 {
		t.Fatalf("attendu 1 lien, obtenu %d", result.TotalCount)
	}
}

func TestLinkChecker_MailtoAndDataIgnored(t *testing.T) {
	lc := analysis.NewLinkChecker()
	html := `
		<a href="mailto:test@example.com">Email</a>
		<a href="tel:+33123456789">Appeler</a>
		<a href="#section">Ancre</a>
		<img src="data:image/png;base64,iVBOR...">
	`
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 0 {
		t.Errorf("mailto, tel, ancre et data URIs devraient être ignorés, obtenu %d liens", result.TotalCount)
	}
}

func TestLinkChecker_Deduplication(t *testing.T) {
	lc := analysis.NewLinkChecker()
	html := `
		<a href="http://example.com">Link 1</a>
		<a href="http://example.com">Link 2</a>
		<a href="http://example.com">Link 3</a>
	`
	result := lc.Check(context.Background(), html)

	if result.TotalCount != 1 {
		t.Errorf("les URLs dupliquées devraient être dédupliquées, obtenu %d liens", result.TotalCount)
	}
}

func TestLinkChecker_EmptyHTML(t *testing.T) {
	lc := analysis.NewLinkChecker()
	result := lc.Check(context.Background(), "")

	if result.TotalCount != 0 {
		t.Errorf("un HTML vide ne devrait pas avoir de liens, obtenu %d", result.TotalCount)
	}
}

func TestLinkChecker_MixedSources(t *testing.T) {
	lc := analysis.NewLinkChecker()
	html := `
		<a href="http://example.com/page">Link</a>
		<img src="http://example.com/image.png">
	`
	result := lc.Check(context.Background(), html)

	// Les 2 URLs sont identiques en host mais différentes en path
	if result.TotalCount != 2 {
		t.Errorf("attendu 2 liens (href + src), obtenu %d", result.TotalCount)
	}

	sources := map[string]bool{}
	for _, l := range result.Links {
		sources[l.Source] = true
	}
	if !sources["href"] || !sources["src"] {
		t.Error("les sources href et src devraient être présentes")
	}
}
