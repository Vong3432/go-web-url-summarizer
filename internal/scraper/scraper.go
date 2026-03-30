package scraper

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	maxTextLength = 5000
	fetchTimeout  = 10 * time.Second
)

var multiSpace = regexp.MustCompile(`\s+`)

var httpClient = &http.Client{Timeout: fetchTimeout}

// Fetch retrieves the URL and returns its cleaned body text, truncated to 5000 chars.
func Fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; go-url-summarizer/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", url, err)
	}

	// Remove non-content nodes.
	doc.Find("script, style, noscript, head").Remove()

	text := doc.Find("body").Text()
	text = multiSpace.ReplaceAllString(strings.TrimSpace(text), " ")

	if len(text) > maxTextLength {
		text = text[:maxTextLength]
	}

	return text, nil
}
