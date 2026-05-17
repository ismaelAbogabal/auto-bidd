package services

import (
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
	CoverLetter    string   `json:"cover_letter"`
	EstimatedHours int      `json:"estimated_hours"`
	Reasoning      string   `json:"reasoning"`
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
Respond ONLY with valid JSON in this exact format:
{
  "cover_letter": "the proposal text",
  "estimated_hours": <number>,
  "reasoning": "brief explanation of the hours estimate",
  "qa_answers": [{"question": "...", "answer": "..."}]
}

If there are no client questions, return an empty qa_answers array.
Do not include any text outside the JSON.
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

func (s *AIService) Generate(ctx context.Context, systemPrompt, userMessage string) (*GenerateBidResponse, error) {
	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := apiResp.Content[0].Text

	// Try to extract JSON from the response (in case it's wrapped in markdown)
	text = extractJSON(text)

	var result GenerateBidResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse AI response: %w (raw: %s)", err, text)
	}

	return &result, nil
}

func (s *AIService) Refine(ctx context.Context, systemPrompt string, history []models.ChatMessage, newMessage string) (*GenerateBidResponse, error) {
	messages := make([]map[string]string, 0, len(history)+1)

	for _, msg := range history {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": newMessage,
	})

	body := map[string]any{
		"model":      s.model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages":   messages,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := extractJSON(apiResp.Content[0].Text)

	var result GenerateBidResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse AI response: %w (raw: %s)", err, text)
	}

	return &result, nil
}

// extractJSON tries to find JSON in a response that may be wrapped in markdown code blocks
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try direct parse first
	if strings.HasPrefix(s, "{") {
		return s
	}

	// Try extracting from ```json ... ```
	if idx := strings.Index(s, "```json"); idx != -1 {
		start := idx + 7
		end := strings.Index(s[start:], "```")
		if end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	// Try extracting from ``` ... ```
	if idx := strings.Index(s, "```"); idx != -1 {
		start := idx + 3
		// Skip to newline
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
