package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"

// Known base URLs for OpenAI-compatible providers
var knownBaseURLs = map[string]string{
	"openai":   "https://api.openai.com/v1/chat/completions",
	"deepseek": "https://api.deepseek.com/v1/chat/completions",
}

type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates a provider compatible with OpenAI, DeepSeek, and any
// service that implements the OpenAI chat completions API.
// If baseURL is empty, it defaults to OpenAI's endpoint.
func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIURL
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Call(ctx context.Context, systemPrompt string, messages []Message, maxTokens int) (string, error) {
	body := p.buildBody(systemPrompt, messages, maxTokens, false)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) CallStream(ctx context.Context, systemPrompt string, messages []Message, maxTokens int, onText func(string)) (string, error) {
	body := p.buildBody(systemPrompt, messages, maxTokens, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			text := chunk.Choices[0].Delta.Content
			fullText.WriteString(text)
			onText(text)
		}
	}

	if err := scanner.Err(); err != nil {
		return fullText.String(), fmt.Errorf("stream read error: %w", err)
	}

	return fullText.String(), nil
}

func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *OpenAIProvider) buildBody(systemPrompt string, messages []Message, maxTokens int, stream bool) map[string]any {
	// OpenAI format: system prompt is the first message
	msgs := make([]map[string]string, 0, len(messages)+1)
	msgs = append(msgs, map[string]string{
		"role":    "system",
		"content": systemPrompt,
	})

	for _, m := range messages {
		msgs = append(msgs, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]any{
		"model":      p.model,
		"max_tokens": maxTokens,
		"messages":   msgs,
	}
	if stream {
		body["stream"] = true
	}
	return body
}
