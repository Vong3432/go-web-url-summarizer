package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockSummarizer implements Summarizer for testing.
type mockSummarizer struct {
	summarize func(ctx context.Context, text string) (string, error)
}

func (m *mockSummarizer) Summarize(ctx context.Context, text string) (string, error) {
	return m.summarize(ctx, text)
}

func ptr(s string) *string { return &s }

func newHandler(fetch FetchFunc, s Summarizer, maxUrlsAllowedInt int) *SummarizeHandler {
	return NewSummarizeHandler(fetch, func(_ string) Summarizer { return s }, maxUrlsAllowedInt)
}

func TestServeHTTP_MethodNotAllowed(t *testing.T) {
	h := newHandler(nil, nil, 10)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/summarize", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
}

func TestServeHTTP_InvalidBody(t *testing.T) {
	h := newHandler(nil, nil, 10)

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{bad}`},
		{"empty urls array", `{"openai_api_key":"sk-test","urls":[]}`},
		{"missing urls field", `{"openai_api_key":"sk-test"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
				bytes.NewBufferString(tc.body)))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d", rec.Code)
			}
		})
	}
}

func TestServeHTTP_MissingAPIKey(t *testing.T) {
	h := newHandler(nil, nil, 10)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
		bytes.NewBufferString(`{"urls":["https://example.com"]}`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestServeHTTP_Success(t *testing.T) {
	fetch := func(_ context.Context, url string) (string, error) {
		return "page text for " + url, nil
	}
	sum := &mockSummarizer{
		summarize: func(_ context.Context, text string) (string, error) {
			return "summary of: " + text, nil
		},
	}

	h := newHandler(fetch, sum, 10)
	body := `{"openai_api_key":"sk-test","urls":["https://a.com","https://b.com"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
		bytes.NewBufferString(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp summarizeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Summaries) != 2 {
		t.Fatalf("want 2 summaries, got %d", len(resp.Summaries))
	}
	for _, s := range resp.Summaries {
		if s.Error != nil {
			t.Errorf("url %s: unexpected error %q", s.URL, *s.Error)
		}
		if s.Summary == nil || *s.Summary == "" {
			t.Errorf("url %s: expected non-empty summary", s.URL)
		}
	}
}

func TestServeHTTP_FetchError(t *testing.T) {
	fetch := func(_ context.Context, url string) (string, error) {
		return "", errors.New("connection refused")
	}
	h := newHandler(fetch, &mockSummarizer{}, 10)

	body := `{"openai_api_key":"sk-test","urls":["https://bad.com"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
		bytes.NewBufferString(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp summarizeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Summaries) != 1 {
		t.Fatalf("want 1 result, got %d", len(resp.Summaries))
	}
	got := resp.Summaries[0]
	if got.Error == nil {
		t.Fatal("want error field set, got nil")
	}
	if got.Summary != nil {
		t.Errorf("want nil summary on error, got %q", *got.Summary)
	}
}

func TestServeHTTP_SummarizerError(t *testing.T) {
	fetch := func(_ context.Context, url string) (string, error) {
		return "some text", nil
	}
	sum := &mockSummarizer{
		summarize: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("openai rate limit")
		},
	}
	h := newHandler(fetch, sum, 10)

	body := `{"openai_api_key":"sk-test","urls":["https://ok.com"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
		bytes.NewBufferString(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp summarizeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	got := resp.Summaries[0]
	if got.Error == nil {
		t.Fatal("want error field set, got nil")
	}
	if *got.Error != "openai rate limit" {
		t.Errorf("want %q, got %q", "openai rate limit", *got.Error)
	}
}

func TestServeHTTP_MaxURLsExceeded(t *testing.T) {
	const max = 3
	h := newHandler(nil, nil, max)

	urls := make([]string, max+1)
	for i := range urls {
		urls[i] = "https://example.com"
	}
	body, _ := json.Marshal(map[string]any{"openai_api_key": "sk-test", "urls": urls})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize", bytes.NewReader(body)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestServeHTTP_MaxURLsAtLimit(t *testing.T) {
	const max = 3
	fetch := func(_ context.Context, url string) (string, error) { return "text", nil }
	sum := &mockSummarizer{
		summarize: func(_ context.Context, _ string) (string, error) { return "summary", nil },
	}
	h := newHandler(fetch, sum, max)

	urls := make([]string, max)
	for i := range urls {
		urls[i] = "https://example.com"
	}
	body, _ := json.Marshal(map[string]any{"openai_api_key": "sk-test", "urls": urls})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize", bytes.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestServeHTTP_PartialFailure(t *testing.T) {
	fetch := func(_ context.Context, url string) (string, error) {
		if url == "https://bad.com" {
			return "", errors.New("timeout")
		}
		return "text", nil
	}
	sum := &mockSummarizer{
		summarize: func(_ context.Context, text string) (string, error) {
			return "summary", nil
		},
	}
	h := newHandler(fetch, sum, 10)

	body := `{"openai_api_key":"sk-test","urls":["https://good.com","https://bad.com"]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/summarize",
		bytes.NewBufferString(body)))

	var resp summarizeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Summaries) != 2 {
		t.Fatalf("want 2 results, got %d", len(resp.Summaries))
	}

	byURL := make(map[string]urlSummary, 2)
	for _, s := range resp.Summaries {
		byURL[s.URL] = s
	}

	if good := byURL["https://good.com"]; good.Error != nil {
		t.Errorf("good.com: unexpected error %q", *good.Error)
	}
	if bad := byURL["https://bad.com"]; bad.Error == nil {
		t.Error("bad.com: expected error field, got nil")
	}
}
