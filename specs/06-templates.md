# 06 — Templates

## Purpose

Save winning bids as reusable templates. When generating future bids, use a template as a style/structure reference to replicate what works.

## Creating a Template

Triggered when:
1. User marks a bid as "won" → prompt: "Save as template?"
2. User clicks "Save as Template" on any bid

### Data Captured

```
Template {
  name:                  user-provided label
  source_bid_id:         the bid it came from
  cover_letter_template: the cover letter text from that bid
  tags:                  auto-extracted from job description tech stack
  win_count:             starts at 1 if created from a "won" bid
  use_count:             0
}
```

## Using a Template

On the bid builder form, user can select a template from a dropdown.

When selected:
- Template's cover_letter_template is passed to the AI as a style reference
- AI is instructed: "Base your response on this winning template style"
- The AI adapts the template to the new job, not copy/paste
- `template_id` is set on the new bid for tracking

When the new bid wins → `template.win_count++`

## Template List Page

```
/templates
┌─────────────────────────────────────────────┐
│  My Templates                                │
├─────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────┐ │
│  │ "Short & Punchy"                        │ │
│  │ Tags: [react] [frontend]                │ │
│  │ Wins: 5 | Used: 12 | Win rate: 42%     │ │
│  │ [View] [Delete]                         │ │
│  └─────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────┐ │
│  │ "Technical Deep Dive"                   │ │
│  │ Tags: [backend] [architecture]          │ │
│  │ Wins: 3 | Used: 7 | Win rate: 43%      │ │
│  │ [View] [Delete]                         │ │
│  └─────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

## Endpoints

| Method | Path | Action |
|--------|------|--------|
| GET | /templates | List page |
| POST | /api/bids/{id}/template | Create template from bid |
| DELETE | /api/templates/{id} | Delete template |
| GET | /api/templates/{id} | View template detail (partial) |

## Template Effectiveness (feeds into analytics)

```
effectiveness = win_count / use_count * 100

Tracked per template:
- How many bids used this template
- How many of those bids won
- Win rate compared to bids without templates
```
