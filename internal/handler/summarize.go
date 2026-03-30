package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Summarizer produces a short summary for the given text.
type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

// FetchFunc fetches the text content of a URL.
type FetchFunc func(ctx context.Context, url string) (string, error)

type summarizeRequest struct {
	OpenAIAPIKey string   `json:"openai_api_key"`
	URLs         []string `json:"urls"`
}

type urlSummary struct {
	URL     string  `json:"url"`
	Summary *string `json:"summary"`
	Error   *string `json:"error"`
}

type summarizeResponse struct {
	Summaries []urlSummary `json:"summaries"`
}

// SummarizeHandler handles POST /summarize requests.
type SummarizeHandler struct {
	fetch          FetchFunc
	newSummarizer  func(apiKey string) Summarizer
	maxUrlsAllowed int
}

// NewSummarizeHandler creates a SummarizeHandler.
func NewSummarizeHandler(fetch FetchFunc, newSummarizer func(apiKey string) Summarizer, maxUrlsAllowed int) *SummarizeHandler {
	return &SummarizeHandler{fetch: fetch, newSummarizer: newSummarizer, maxUrlsAllowed: maxUrlsAllowed}
}

// ServeHTTP handles the summarize endpoint.
//
//	@Summary		Summarize URLs
//	@Description	Scrapes each provided URL and returns a ~500-character AI-generated summary per URL. Failed URLs include an error message instead of a summary.
//	@Tags			summarizer
//	@Accept			json
//	@Produce		json
//	@Param			request	body		summarizeRequest	true	"List of URLs to summarize"
//	@Success		200		{object}	summarizeResponse
//	@Failure		400		{string}	string	"invalid request: provide a non-empty urls array"
//	@Failure		400		{string}	string	"invalid request: openai_api_key is required"
//	@Failure		405		{string}	string	"method not allowed"
//	@Router			/summarize [post]
func (h *SummarizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req summarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.URLs) == 0 {
		http.Error(w, "invalid request: provide a non-empty urls array", http.StatusBadRequest)
		return
	}

	if req.OpenAIAPIKey == "" {
		http.Error(w, "invalid request: openai_api_key is required", http.StatusBadRequest)
		return
	}

	if len(req.URLs) > h.maxUrlsAllowed {
		msg := fmt.Sprintf("invalid request: please make sure urls is not more than %d", h.maxUrlsAllowed)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	sum := h.newSummarizer(req.OpenAIAPIKey)
	results := make([]urlSummary, len(req.URLs))

	var wg sync.WaitGroup
	for i, url := range req.URLs {
		i, url := i, url
		wg.Go(func() {
			text, err := h.fetch(r.Context(), url)
			if err != nil {
				msg := err.Error()
				results[i] = urlSummary{URL: url, Error: &msg}
				return
			}
			summary, err := sum.Summarize(r.Context(), text)
			if err != nil {
				msg := err.Error()
				results[i] = urlSummary{URL: url, Error: &msg}
				return
			}
			results[i] = urlSummary{URL: url, Summary: &summary}
		})
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summarizeResponse{Summaries: results})
}

// ensure SummarizeHandler satisfies http.Handler at compile time
var _ http.Handler = (*SummarizeHandler)(nil)
