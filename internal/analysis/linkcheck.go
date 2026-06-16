package analysis

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	htmlParser "golang.org/x/net/html"

	"github.com/statoon54/mailhive/internal/domain"
)

const (
	linkCheckTimeout     = 3 * time.Second
	linkCheckConcurrency = 10
)

// LinkChecker valide les liens trouvés dans le contenu HTML.
type LinkChecker struct {
	client *http.Client
}

// NewLinkChecker crée un nouveau vérificateur de liens.
func NewLinkChecker() *LinkChecker {
	return &LinkChecker{
		client: &http.Client{
			Timeout: linkCheckTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // ne pas suivre les redirections
			},
		},
	}
}

// Check extrait les liens href et src du HTML et les vérifie.
func (lc *LinkChecker) Check(ctx context.Context, html string) *domain.LinkCheckResult {
	links := extractLinks(html)
	if len(links) == 0 {
		return &domain.LinkCheckResult{}
	}

	// Dédupliquer les URLs
	seen := make(map[string]bool)
	var unique []rawLink
	for _, l := range links {
		if !seen[l.url] {
			seen[l.url] = true
			unique = append(unique, l)
		}
	}

	results := make([]domain.LinkStatus, len(unique))
	sem := make(chan struct{}, linkCheckConcurrency)
	var wg sync.WaitGroup

	for i, link := range unique {
		wg.Add(1)
		go func(idx int, rl rawLink) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = lc.checkLink(ctx, rl)
		}(i, link)
	}
	wg.Wait()

	var broken int
	for _, r := range results {
		if r.Status == "broken" || r.Status == "timeout" {
			broken++
		}
	}

	return &domain.LinkCheckResult{
		Links:       results,
		TotalCount:  len(results),
		BrokenCount: broken,
	}
}

type rawLink struct {
	url    string
	source string // "href" ou "src"
}

func extractLinks(htmlContent string) []rawLink {
	tokenizer := htmlParser.NewTokenizer(strings.NewReader(htmlContent))
	var links []rawLink

	for {
		tt := tokenizer.Next()
		switch tt {
		case htmlParser.ErrorToken:
			return links
		case htmlParser.StartTagToken, htmlParser.SelfClosingTagToken:
			token := tokenizer.Token()
			for _, attr := range token.Attr {
				switch attr.Key {
				case "href":
					u := strings.TrimSpace(attr.Val)
					if u != "" && !strings.HasPrefix(u, "mailto:") && !strings.HasPrefix(u, "tel:") && !strings.HasPrefix(u, "#") {
						links = append(links, rawLink{url: u, source: "href"})
					}
				case "src":
					u := strings.TrimSpace(attr.Val)
					if u != "" && !strings.HasPrefix(u, "data:") {
						links = append(links, rawLink{url: u, source: "src"})
					}
				}
			}
		}
	}
}

func (lc *LinkChecker) checkLink(ctx context.Context, rl rawLink) domain.LinkStatus {
	status := domain.LinkStatus{
		URL:    rl.url,
		Source: rl.source,
	}

	parsed, err := url.Parse(rl.url)
	if err != nil {
		status.Status = "invalid"
		status.Details = "URL invalide : " + err.Error()
		return status
	}

	// Liens relatifs
	if parsed.Host == "" {
		status.Status = "invalid"
		status.Details = "URL relative sans base"
		return status
	}

	// Détection HTTP non sécurisé
	if parsed.Scheme == "http" {
		status.Status = "insecure"
		status.Details = "Lien HTTP non sécurisé"
		return status
	}

	// Vérification HEAD
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rl.url, nil)
	if err != nil {
		status.Status = "invalid"
		status.Details = err.Error()
		return status
	}
	req.Header.Set("User-Agent", "MailHive-LinkChecker/1.0")

	resp, err := lc.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			status.Status = "timeout"
			status.Details = "Timeout de vérification"
		} else {
			status.Status = "broken"
			status.Details = err.Error()
		}
		return status
	}
	_ = resp.Body.Close()

	status.StatusCode = resp.StatusCode

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		status.Status = "ok"
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		status.Status = "redirect"
		if loc := resp.Header.Get("Location"); loc != "" {
			status.Details = "Redirige vers : " + loc
		}
	default:
		status.Status = "broken"
		status.Details = http.StatusText(resp.StatusCode)
	}

	return status
}
