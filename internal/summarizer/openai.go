package summarizer

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const systemPrompt = `You are a concise summarizer. Given the text content of a web page,
write a summary that is no longer than 500 characters. Focus on the main topic and key points.
Return only the summary text with no additional commentary.`

// Summarizer calls the OpenAI API to produce short summaries.
type Summarizer struct {
	client openai.Client
}

// NewSummarizer creates a Summarizer using the provided API key.
func NewSummarizer(apiKey string) *Summarizer {
	return &Summarizer{client: openai.NewClient(option.WithAPIKey(apiKey))}
}

// Summarize returns a ~500-character summary of the provided text.
func (s *Summarizer) Summarize(ctx context.Context, text string) (string, error) {
	resp, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(text),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
