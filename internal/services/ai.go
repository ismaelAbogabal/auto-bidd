package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ismaelfi/auto-bidd/internal/models"
)

type AIService struct {
	provider LLMProvider
}

func NewAIService(provider LLMProvider) *AIService {
	return &AIService{provider: provider}
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

	if profile.AIInstructions != "" {
		sb.WriteString("## Custom Rules\nFollow these instructions strictly:\n")
		sb.WriteString(profile.AIInstructions)
		sb.WriteString("\n\n")
	}

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

func (s *AIService) Generate(ctx context.Context, systemPrompt, userMessage string) (*GenerateBidResponse, error) {
	messages := []Message{{Role: "user", Content: userMessage}}
	text, err := s.provider.Call(ctx, systemPrompt, messages, 2048)
	if err != nil {
		return nil, err
	}
	return ParseResponse(text)
}

func (s *AIService) GenerateStream(ctx context.Context, systemPrompt, userMessage string, onText func(string)) (string, error) {
	messages := []Message{{Role: "user", Content: userMessage}}
	return s.provider.CallStream(ctx, systemPrompt, messages, 2048, onText)
}

func (s *AIService) Refine(ctx context.Context, systemPrompt string, history []models.ChatMessage, newMessage string) (*GenerateBidResponse, error) {
	messages := toMessages(history, newMessage)
	text, err := s.provider.Call(ctx, systemPrompt, messages, 2048)
	if err != nil {
		return nil, err
	}
	return ParseResponse(text)
}

func (s *AIService) RefineStream(ctx context.Context, systemPrompt string, history []models.ChatMessage, newMessage string, onText func(string)) (string, error) {
	messages := toMessages(history, newMessage)
	return s.provider.CallStream(ctx, systemPrompt, messages, 2048, onText)
}

// toMessages converts chat history + new message into provider-agnostic Messages
func toMessages(history []models.ChatMessage, newMessage string) []Message {
	messages := make([]Message, 0, len(history)+1)
	for i, msg := range history {
		messages = append(messages, Message{
			Role:            msg.Role,
			Content:         msg.Content,
			CacheBreakpoint: i == len(history)-1,
		})
	}
	messages = append(messages, Message{Role: "user", Content: newMessage})
	return messages
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
