# Auto-Bidder

AI-powered freelance bid/proposal generator. Define your profile, paste a job description, and get a personalized cover letter in seconds — streamed in real-time.

## Features

- **AI Bid Generation** — generates tailored cover letters with pricing estimates and Q&A answers
- **Real-time Streaming** — cover letter appears word-by-word via SSE as the AI writes it
- **Chat Refinement** — refine bids through conversation ("make it shorter", "add more React experience")
- **Profile & Tone** — define your skills, portfolio, and writing style so every bid sounds like you
- **Templates** — save winning bids as reusable templates with win rate tracking
- **Analytics** — win rate, pricing trends, template effectiveness, and monthly breakdowns
- **Multi-Provider LLM** — supports Anthropic (Claude), OpenAI, DeepSeek, Ollama, and any OpenAI-compatible API

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go, Chi router, GORM |
| Database | PostgreSQL |
| Frontend | Server-rendered HTML, HTMX, Alpine.js, Tailwind CSS |
| AI | Anthropic Claude / OpenAI-compatible APIs |

## Quick Start

### Prerequisites

- Go 1.21+
- Docker (for PostgreSQL)
- An API key from Anthropic, OpenAI, or any compatible provider

### Setup

```bash
# Clone
git clone https://github.com/ismaelAbogabal/auto-bidd.git
cd auto-bidd

# Start PostgreSQL
make docker-up

# Configure
cp .env.example .env
# Edit .env and set your LLM_API_KEY

# Run
make dev
```

Open [http://localhost:8080](http://localhost:8080), register an account, and start generating bids.

## Configuration

All config is via environment variables (`.env` file):

```bash
# Database
DATABASE_URL=postgres://autobidd:autobidd@localhost:5432/autobidd?sslmode=disable

# LLM Provider: "anthropic" or "openai"
LLM_PROVIDER=anthropic
LLM_API_KEY=sk-your-key-here
LLM_MODEL=claude-sonnet-4-20250514

# Optional: custom API endpoint for OpenAI-compatible services
# LLM_BASE_URL=https://api.deepseek.com/v1/chat/completions
```

### Provider Examples

| Provider | LLM_PROVIDER | LLM_MODEL | LLM_BASE_URL |
|----------|-------------|-----------|-------------|
| Anthropic | `anthropic` | `claude-sonnet-4-20250514` | _(default)_ |
| OpenAI | `openai` | `gpt-4o` | _(default)_ |
| DeepSeek | `openai` | `deepseek-chat` | `https://api.deepseek.com/v1/chat/completions` |
| Ollama | `openai` | `llama3` | `http://localhost:11434/v1/chat/completions` |
| Together AI | `openai` | `meta-llama/Llama-3-70b-chat-hf` | `https://api.together.xyz/v1/chat/completions` |

## Commands

```bash
make dev          # Start with hot reload (requires air)
make run          # Start without hot reload
make build        # Build binary to bin/autobidd
make docker-up    # Start PostgreSQL
make docker-down  # Stop PostgreSQL
```

## How It Works

1. **Set up your profile** — Add your name, skills, hourly rate, tone examples, and portfolio
2. **Paste a job** — Enter the job title, description, client questions, and budget
3. **Generate** — AI writes a personalized cover letter streamed in real-time
4. **Refine** — Chat with the AI to adjust tone, length, or emphasis
5. **Track** — Mark bids as submitted/won/lost, view analytics over time
6. **Reuse** — Save winning bids as templates for similar future jobs

## Project Structure

```
cmd/server/         Entry point
internal/
  config/           Environment config
  database/         GORM connection + migrations
  handlers/         HTTP handlers
  middleware/       Auth, logging, rate limiting
  models/           Database models
  services/         Business logic (AI, auth, bids, analytics)
  views/            Template renderer
  router/           Route definitions
templates/
  layouts/          base, app (sidebar), auth (centered card)
  components/       Shared components (sidebar, portfolio_item, etc.)
  pages/            Full pages (define title + content)
  partials/         HTMX partials (alert, bid_stream, etc.)
static/js/          HTMX, Alpine.js, app.js
```

## License

MIT
