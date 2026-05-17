# 08 — AI Service

## Overview

Wraps Claude API calls. Responsible for building prompts from user data and parsing structured responses.

## Claude API Integration

### Client

Direct HTTP calls to `https://api.anthropic.com/v1/messages` (or use anthropic-sdk-go if stable).

```go
type AIService struct {
    apiKey     string
    httpClient *http.Client
    model      string // "claude-sonnet-4-20250514"
}
```

### Model Selection

- Default: `claude-sonnet-4-20250514` (good balance of speed/quality for proposals)
- Configurable via env var `CLAUDE_MODEL`

## Prompt Building

### System Prompt Builder

```go
func (s *AIService) BuildSystemPrompt(profile *models.UserProfile, tones []models.ToneExample, portfolio []models.PortfolioItem) string
```

Assembles from:
1. Role definition
2. Writer profile (name, title, skills, rate)
3. Tone examples (all of them)
4. Relevant portfolio items (filtered by tech overlap, max 3)
5. AI instructions (user's custom rules)
6. Output format specification

### Portfolio Relevance Filtering

```go
func selectRelevantPortfolio(items []models.PortfolioItem, jobDescription string) []models.PortfolioItem
```

Simple keyword overlap between portfolio tech_stack and job description. No embedding search needed at this scale.

## Request Types

### Generate Bid

```go
type GenerateBidRequest struct {
    JobTitle       string
    JobDescription string
    Questions      []string
    BudgetMin      float64
    BudgetMax      float64
    TemplateCover  string // optional, from template
}

type GenerateBidResponse struct {
    CoverLetter    string `json:"cover_letter"`
    EstimatedHours int    `json:"estimated_hours"`
    Reasoning      string `json:"reasoning"`
    QAAnswers      []QA   `json:"qa_answers"`
}
```

### Refine Bid (Chat)

```go
type RefineBidRequest struct {
    CurrentBid  *models.Bid
    ChatHistory []models.ChatMessage
    NewMessage  string
}
```

Response format same as GenerateBidResponse. AI returns the full updated bid each time.

## Streaming

### SSE Flow

```go
func (s *AIService) GenerateStream(ctx context.Context, req GenerateBidRequest, profile *UserProfile, ...) (<-chan string, error) {
    // 1. Build system prompt
    // 2. Build user message
    // 3. Call Claude API with stream=true
    // 4. Return channel that emits response chunks
    // 5. Caller (handler) formats chunks as SSE events
}
```

### Parsing Streamed JSON

Since AI returns JSON, we accumulate the full response, then parse once complete. During streaming, show a "generating..." indicator. When done, parse JSON and render the full output partial.

Alternative: Ask AI to respond in sections (cover letter first, then pricing, then Q&A) and render each section as it completes.

## Error Handling

| Error | Action |
|-------|--------|
| API rate limit (429) | Retry with backoff, show "busy" to user |
| Invalid response (bad JSON) | Retry once, then show raw text + error |
| Context too long | Trim portfolio items, retry |
| Network timeout | Show error, allow retry button |

## Token Budget

Rough estimates per bid generation:
- System prompt: ~1500 tokens
- User message: ~500-2000 tokens (depends on job description)
- Response: ~500-1500 tokens
- Total: ~3000-5000 tokens per generation

Chat refinements are similar but include history (~500 tokens per previous turn).

## Rate Limiting (future)

Track per-user:
- Generations per day (limit: 50?)
- Chat messages per bid (limit: 20?)

Not implemented in v1, but schema allows adding counters later.
