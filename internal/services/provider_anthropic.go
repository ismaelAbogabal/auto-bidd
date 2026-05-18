package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const defaultAnthropicURL = "https://api.anthropic.com/v1/messages"

type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider(apiKey, model, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = defaultAnthropicURL
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (p *AnthropicProvider) Call(ctx context.Context, systemPrompt string, messages []Message, maxTokens int) (string, error) {
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
		return "", fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}

	return apiResp.Content[0].Text, nil
}

func (p *AnthropicProvider) CallStream(ctx context.Context, systemPrompt string, messages []Message, maxTokens int, onText func(string)) (string, error) {
	body := p.buildBody(systemPrompt, messages, maxTokens, true)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("[ANTHROPIC] POST %s (model: %s, max_tokens: %d, body: %d bytes)", p.baseURL, p.model, maxTokens, len(jsonBody))

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

	log.Printf("[ANTHROPIC] response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	var eventType string
	chunks := 0

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			if eventType == "error" {
				log.Printf("[ANTHROPIC] received error event in stream")
			}
			continue
		}

		if strings.HasPrefix(line, "data: ") && eventType == "content_block_delta" {
			data := strings.TrimPrefix(line, "data: ")
			var delta struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &delta); err == nil && delta.Delta.Type == "text_delta" {
				chunks++
				fullText.WriteString(delta.Delta.Text)
				onText(delta.Delta.Text)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ANTHROPIC] scanner error after %d chunks: %v", chunks, err)
		return fullText.String(), fmt.Errorf("stream read error: %w", err)
	}

	log.Printf("[ANTHROPIC] stream done: %d chunks, %d chars", chunks, fullText.Len())
	return fullText.String(), nil
}

func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

func (p *AnthropicProvider) buildBody(systemPrompt string, messages []Message, maxTokens int, stream bool) map[string]any {
	// System prompt with cache control
	system := []map[string]any{
		{
			"type":          "text",
			"text":          systemPrompt,
			"cache_control": map[string]string{"type": "ephemeral"},
		},
	}

	// Convert messages to Anthropic format
	msgs := make([]map[string]any, len(messages))
	for i, m := range messages {
		content := []map[string]any{
			{"type": "text", "text": m.Content},
		}
		if m.CacheBreakpoint {
			content = []map[string]any{
				{
					"type":          "text",
					"text":          m.Content,
					"cache_control": map[string]string{"type": "ephemeral"},
				},
			}
		}
		msgs[i] = map[string]any{
			"role":    m.Role,
			"content": content,
		}
	}

	body := map[string]any{
		"model":      p.model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   msgs,
	}
	if stream {
		body["stream"] = true
	}
	return body
}
