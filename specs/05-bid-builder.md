# 05 — Bid Builder

## Purpose

The core feature. User inputs a job description, AI generates a complete bid proposal, user refines via chat or manual editing.

## Generation Flow

```
1. User pastes job description + optional questions
2. User clicks "Generate"
3. Server builds system prompt from profile data
4. Server calls Claude API (streaming)
5. Response streams to browser via SSE
6. Final output: cover letter + hours estimate + total price + Q&A answers
7. User can edit manually or chat to refine
```

## Input Form

```
┌─────────────────────────────────────────────┐
│  New Bid                                     │
├─────────────────────────────────────────────┤
│  Job Title: [________________________]       │
│                                             │
│  Job Description:                           │
│  ┌─────────────────────────────────────┐    │
│  │ (paste full job post here)          │    │
│  │                                     │    │
│  └─────────────────────────────────────┘    │
│                                             │
│  Client Questions (one per line):           │
│  ┌─────────────────────────────────────┐    │
│  │ What is your experience with...     │    │
│  │ How would you approach...           │    │
│  └─────────────────────────────────────┘    │
│                                             │
│  Budget Range: $[____] - $[____]            │
│  Platform: [dropdown: generic/upwork/other] │
│                                             │
│  Template: [None ▾] (optional)              │
│                                             │
│  [Generate Bid]                             │
└─────────────────────────────────────────────┘
```

## AI Request

### System Prompt Assembly

```
You are a freelance proposal writer for {profile.full_name}.

## About the Writer
- Title: {profile.title}
- Skills: {profile.skills joined}
- Hourly Rate: ${profile.hourly_rate}/hr

## Writing Style
Write in this tone and style:
{for each tone_example:}
  Example ({tone.label}, context: {tone.context}):
  "{tone.content}"

## Relevant Past Work
{for each relevant portfolio_item (max 3, selected by tech overlap):}
  - {item.title}: {item.description}
    Tech: {item.tech_stack}
    Result: {item.outcome}

## Rules
{profile.ai_instructions}

## Output Format
Respond in JSON:
{
  "cover_letter": "the proposal text",
  "estimated_hours": number,
  "reasoning": "why this estimate",
  "qa_answers": [{"question": "...", "answer": "..."}]
}
```

### User Message

```
Generate a bid for this job:

Title: {job_title}
Description: {job_description}

Client Questions:
{questions, numbered}

Client Budget: ${min} - ${max}

{if template:}
Base your response on this winning template style:
{template.cover_letter_template}
```

## Output Display

```
┌──────────────────────┬──────────────────────────────┐
│   Chat Panel         │   Bid Output                 │
│                      │                              │
│                      │   Cover Letter               │
│                      │   ┌────────────────────────┐ │
│                      │   │ Hey! I noticed you...  │ │
│                      │   │                        │ │
│   [AI messages]      │   │ [Edit] [Copy]          │ │
│   [User messages]    │   └────────────────────────┘ │
│                      │                              │
│                      │   Pricing                    │
│                      │   ┌────────────────────────┐ │
│                      │   │ Hours: [40]            │ │
│                      │   │ Rate:  $75/hr          │ │
│                      │   │ Total: $3,000          │ │
│                      │   └────────────────────────┘ │
│                      │                              │
│                      │   Q&A Answers                │
│                      │   ┌────────────────────────┐ │
│   ┌──────────────┐   │   │ Q: What experience...  │ │
│   │ Make shorter  │   │   │ A: I have built...     │ │
│   └──────────────┘   │   └────────────────────────┘ │
│   [Send]             │                              │
│                      │   [Submit] [Save Template]   │
├──────────────────────┴──────────────────────────────┤
│   Status: Draft  |  Created: 2 min ago              │
└─────────────────────────────────────────────────────┘
```

## Chat Refinement

User sends a message like:
- "Make the cover letter shorter"
- "Add more emphasis on my React experience"
- "Change the estimate to 60 hours"
- "Make the tone more professional"

### Chat Request

```
POST /api/bids/{id}/chat
Body: {"message": "Make it shorter"}

1. Load bid + chat history + profile
2. Build messages array:
   - System: same prompt as generation
   - Assistant: current bid state (JSON)
   - ...previous chat messages...
   - User: new refinement request
3. Call Claude API (streaming)
4. Parse response → update bid fields
5. Stream updated HTML partial back via SSE
6. Save chat messages to DB
```

## Manual Editing

- Alpine.js toggles between view/edit mode
- Edit mode shows textareas for cover letter and Q&A answers
- Hours field is always editable (recalculates total on change)
- Save via `hx-put="/api/bids/{id}"`

## Status Management

```
Draft → Submitted (user marks as sent to client)
Submitted → Won | Lost | Withdrawn

PATCH /api/bids/{id}/status
Body: {"status": "won"}

If status = "won":
  → prompt to save as template
  → increment template.win_count if bid used a template
```

## SSE Streaming

```go
// Handler sets headers for SSE
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

// Stream AI response as HTML chunks
for chunk := range aiStream {
    fmt.Fprintf(w, "data: <div hx-swap-oob=\"true\" id=\"bid-output\">%s</div>\n\n", rendered)
    flusher.Flush()
}
```

HTMX client:
```html
<div hx-ext="sse" sse-connect="/api/bids/{id}/stream" sse-swap="message">
</div>
```
