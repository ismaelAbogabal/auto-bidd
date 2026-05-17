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

	"github.com/ismaelfi/auto-bidd/internal/models"
)

type AIService struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAIService(apiKey, model string) *AIService {
	return &AIService{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

type GenerateBidRequest struct {
	JobTitle       string
	JobDescription string
	Questions      []string
	BudgetMin      float64
	BudgetMax      float64
	TemplateCover  string
}

type GenerateBidResponse struct {
	CoverLetter    string      `json:"cover_letter"`
	EstimatedHours int         `json:"estimated_hours"`
	Reasoning      string      `json:"reasoning"`
	QAAnswers      []models.QA `json:"qa_answers"`
}

func (s *AIService) BuildSystemPrompt(profile *models.UserProfile, tones []models.ToneExample, portfolio []models.PortfolioItem) string {
	var sb strings.Builder

	sb.WriteString("You are a freelance proposal writer. Your job is to write winning bids for freelance projects.\n\n")

	// Writer profile
	sb.WriteString("## About the Writer\n")
	if profile.FullName != "" {
		sb.WriteString(fmt.Sprintf("- Name: %s\n", profile.FullName))
	}
	if profile.Title != "" {
		sb.WriteString(fmt.Sprintf("- Title: %s\n", profile.Title))
	}
	if len(profile.Skills) > 0 {
		sb.WriteString(fmt.Sprintf("- Skills: %s\n", strings.Join(profile.Skills, ", ")))
	}
	if profile.HourlyRate > 0 {
		sb.WriteString(fmt.Sprintf("- Hourly Rate: $%.0f/hr\n", profile.HourlyRate))
	}
	sb.WriteString("\n")

	// Tone examples
	if len(tones) > 0 {
		sb.WriteString("## Writing Style\nWrite in this tone and style. Match the voice in these examples:\n\n")
		for _, t := range tones {
			if t.Label != "" {
				sb.WriteString(fmt.Sprintf("### %s", t.Label))
				if t.Context != "" {
					sb.WriteString(fmt.Sprintf(" (context: %s)", t.Context))
				}
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("\"%s\"\n\n", t.Content))
		}
	}

	// Portfolio
	if len(portfolio) > 0 {
		sb.WriteString("## Relevant Past Work\nReference these when relevant to demonstrate experience:\n\n")
		for _, p := range portfolio {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.Title, p.Description))
			if len(p.TechStack) > 0 {
				sb.WriteString(fmt.Sprintf("  Tech: %s\n", strings.Join(p.TechStack, ", ")))
			}
			if p.Outcome != "" {
				sb.WriteString(fmt.Sprintf("  Result: %s\n", p.Outcome))
			}
		}
		sb.WriteString("\n")
	}

	// AI instructions
	if profile.AIInstructions != "" {
		sb.WriteString("## Custom Rules\nFollow these instructions strictly:\n")
		sb.WriteString(profile.AIInstructions)
		sb.WriteString("\n\n")
	}

	// Output format
	sb.WriteString(`## Output Format
Write the cover letter directly as plain text. Do not wrap it in quotes or JSON.

After the cover letter, output a separator line containing only "---META---" followed by a JSON object on the next line:

---META---
{"estimated_hours": <number>, "reasoning": "brief explanation", "qa_answers": [{"question": "...", "answer": "..."}]}

If there are no client questions, use an empty qa_answers array.
The cover letter MUST come before ---META---. Do not put any text after the JSON line.
`)

	return sb.String()
}

func (s *AIService) BuildUserMessage(req GenerateBidRequest) string {
	var sb strings.Builder

	sb.WriteString("Generate a bid for this job:\n\n")
	sb.WriteString(fmt.Sprintf("**Title:** %s\n\n", req.JobTitle))
	sb.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", req.JobDescription))

	if len(req.Questions) > 0 {
		sb.WriteString("**Client Questions:**\n")
		for i, q := range req.Questions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
		}
		sb.WriteString("\n")
	}

	if req.BudgetMin > 0 || req.BudgetMax > 0 {
		sb.WriteString(fmt.Sprintf("**Client Budget:** $%.0f - $%.0f\n\n", req.BudgetMin, req.BudgetMax))
	}

	if req.TemplateCover != "" {
		sb.WriteString("**Reference Template (adapt style but don't copy):**\n")
		sb.WriteString(req.TemplateCover)
		sb.WriteString("\n")
	}

	return sb.String()
}

// Generate calls the API without streaming (used as fallback)
func (s *AIService) Generate(ctx context.Context, systemPrompt, userMessage string) (*GenerateBidResponse, error) {
	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"system": []map[string]any{
			{
				"type":          "text",
				"text":          systemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	}

	text, err := s.callAPI(ctx, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse(text)
}

// GenerateStream calls the API with streaming, calling onText for each text chunk.
// Returns the full accumulated text.
func (s *AIService) GenerateStream(ctx context.Context, systemPrompt, userMessage string, onText func(string)) (string, error) {
	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"stream":     true,
		"system": []map[string]any{
			{
				"type":          "text",
				"text":          systemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	}

	return s.callStreamAPI(ctx, body, onText)
}

// Refine calls the API without streaming
func (s *AIService) Refine(ctx context.Context, systemPrompt string, history []models.ChatMessage, newMessage string) (*GenerateBidResponse, error) {
	messages := buildMessages(history, newMessage)

	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"system": []map[string]any{
			{
				"type":          "text",
				"text":          systemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": messages,
	}

	text, err := s.callAPI(ctx, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse(text)
}

// RefineStream calls the API with streaming
func (s *AIService) RefineStream(ctx context.Context, systemPrompt string, history []models.ChatMessage, newMessage string, onText func(string)) (string, error) {
	messages := buildMessages(history, newMessage)

	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"stream":     true,
		"system": []map[string]any{
			{
				"type":          "text",
				"text":          systemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": messages,
	}

	return s.callStreamAPI(ctx, body, onText)
}

// buildMessages constructs the messages array with cache control on the last history message
func buildMessages(history []models.ChatMessage, newMessage string) []map[string]any {
	messages := make([]map[string]any, 0, len(history)+1)

	for i, msg := range history {
		content := []map[string]any{
			{"type": "text", "text": msg.Content},
		}
		if i == len(history)-1 {
			content = []map[string]any{
				{
					"type":          "text",
					"text":          msg.Content,
					"cache_control": map[string]string{"type": "ephemeral"},
				},
			}
		}
		messages = append(messages, map[string]any{
			"role":    msg.Role,
			"content": content,
		})
	}

	messages = append(messages, map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "text", "text": newMessage},
		},
	})

	return messages
}

// callAPI makes a non-streaming request and returns the response text
func (s *AIService) callAPI(ctx context.Context, body map[string]any) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
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
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Content[0].Text, nil
}

// callStreamAPI makes a streaming request, calling onText for each text delta
func (s *AIService) callStreamAPI(ctx context.Context, body map[string]any, onText func(string)) (string, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	var eventType string

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for large SSE data lines
	scanner.Buffer(make([]byte, 64*1024), 256*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
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
				fullText.WriteString(delta.Delta.Text)
				onText(delta.Delta.Text)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fullText.String(), fmt.Errorf("stream read error: %w", err)
	}

	return fullText.String(), nil
}

// ParseResponse parses the cover letter + ---META--- format
func ParseResponse(text string) (*GenerateBidResponse, error) {
	text = strings.TrimSpace(text)

	// Try legacy JSON format first (backward compat)
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "```") {
		jsonText := extractJSON(text)
		var result GenerateBidResponse
		if err := json.Unmarshal([]byte(jsonText), &result); err == nil {
			return &result, nil
		}
	}

	// Parse text + ---META--- format
	result := &GenerateBidResponse{}

	parts := strings.SplitN(text, "---META---", 2)
	result.CoverLetter = strings.TrimSpace(parts[0])

	if len(parts) == 2 {
		metaText := strings.TrimSpace(parts[1])
		metaText = extractJSON(metaText)
		var meta struct {
			EstimatedHours int         `json:"estimated_hours"`
			Reasoning      string      `json:"reasoning"`
			QAAnswers      []models.QA `json:"qa_answers"`
		}
		if err := json.Unmarshal([]byte(metaText), &meta); err == nil {
			result.EstimatedHours = meta.EstimatedHours
			result.Reasoning = meta.Reasoning
			result.QAAnswers = meta.QAAnswers
		}
	}

	if result.CoverLetter == "" {
		return nil, fmt.Errorf("empty cover letter in response")
	}

	return result, nil
}

// extractJSON tries to find JSON in a response that may be wrapped in markdown code blocks
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "{") {
		return s
	}

	if idx := strings.Index(s, "```json"); idx != -1 {
		start := idx + 7
		end := strings.Index(s[start:], "```")
		if end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	if idx := strings.Index(s, "```"); idx != -1 {
		start := idx + 3
		if nl := strings.Index(s[start:], "\n"); nl != -1 {
			start = start + nl + 1
		}
		end := strings.Index(s[start:], "```")
		if end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	return s
}
