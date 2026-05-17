# 04 — Profile Management

## Purpose

The profile is the AI's "memory" of the user. Everything here feeds into the system prompt when generating bids.

## Components

### Basic Info (UserProfile)

- Full name
- Professional title (e.g., "Senior Full-Stack Developer")
- Hourly rate (decimal, used in pricing calc)
- Skills (tag list, stored as jsonb array)

### Tone Examples (ToneExample[])

User provides samples of how they write proposals. Used to make AI output match their voice.

Each example has:
- **Label** — short name ("Casual intro", "Technical pitch")
- **Content** — the actual text sample
- **Context** — when to use this tone ("for small projects", "for enterprise clients")

Minimum 1 example recommended. UI shows a prompt if none exist.

### Portfolio Items (PortfolioItem[])

Past work that the AI references to demonstrate relevant experience.

Each item has:
- **Title** — project name
- **Description** — what was built
- **Tech stack** — technologies used (tag list)
- **Outcome** — results, metrics, impact
- **URL** — optional link

The AI service selects the most relevant portfolio items per bid based on tech stack overlap with the job description.

### AI Instructions (text)

Free-form text where user defines rules for the AI:
- "Always mention my 5-year React experience"
- "Never use the word 'synergy'"
- "Keep cover letters under 200 words"
- "Always ask a clarifying question at the end"

## Page Layout

```
/profile
┌─────────────────────────────────────────────┐
│  Profile Settings                            │
├─────────────────────────────────────────────┤
│                                             │
│  [Basic Info Section]                       │
│  Name: [___________]                        │
│  Title: [___________]                       │
│  Rate: $[____]/hr                           │
│  Skills: [tag] [tag] [+ add]               │
│  [Save]                                     │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│  [Tone Examples Section]                    │
│  ┌─────────────────────────────────┐        │
│  │ "Casual intro" — "Hey! I saw..." │ [x]   │
│  └─────────────────────────────────┘        │
│  ┌─────────────────────────────────┐        │
│  │ "Technical pitch" — "With 5..."  │ [x]   │
│  └─────────────────────────────────┘        │
│  [+ Add tone example]                       │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│  [Portfolio Section]                        │
│  ┌─────────────────────────────────┐        │
│  │ E-commerce Platform              │ [x]   │
│  │ React, Node, Stripe              │       │
│  └─────────────────────────────────┘        │
│  [+ Add portfolio item]                     │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│  [AI Instructions]                          │
│  ┌─────────────────────────────────┐        │
│  │ Always mention my...            │        │
│  │ Never use...                    │        │
│  └─────────────────────────────────┘        │
│  [Save instructions]                        │
│                                             │
└─────────────────────────────────────────────┘
```

## HTMX Interactions

| Action | Request | Response |
|--------|---------|----------|
| Save basic info | `hx-put="/api/profile"` | Success alert partial |
| Add tone example | `hx-post="/api/profile/tone"` | New tone_item.html appended |
| Delete tone example | `hx-delete="/api/profile/tone/{id}"` | Item removed (hx-swap="delete") |
| Add portfolio item | `hx-post="/api/profile/portfolio"` | New portfolio_item.html appended |
| Delete portfolio item | `hx-delete="/api/profile/portfolio/{id}"` | Item removed |
| Save AI instructions | `hx-put="/api/profile/instructions"` | Success alert |

## Validation

- Full name: optional, max 255 chars
- Title: optional, max 255 chars
- Hourly rate: >= 0, max 9999.99
- Skills: max 50 items, each max 50 chars
- Tone example content: min 20 chars, max 5000 chars
- Portfolio description: max 10000 chars
- AI instructions: max 5000 chars
