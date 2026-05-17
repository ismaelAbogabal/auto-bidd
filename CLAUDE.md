# Auto-Bidder — Agent Instructions

## What this project is

AI-powered freelance bid/proposal generator. Users define their profile (skills, tone, portfolio), paste a job description, and the app generates a personalized cover letter via Claude API. Bids can be refined through a chat interface, tracked by status, and saved as reusable templates. Analytics show win rates and pricing trends.

## Tech stack

- **Backend:** Go 1.26, Chi router, GORM + PostgreSQL
- **Frontend:** Server-rendered HTML templates, HTMX, Alpine.js, Tailwind CSS (CDN)
- **AI:** Claude API (Anthropic) with prompt caching and SSE streaming
- **Dev tools:** Air (hot reload), Docker Compose (Postgres)

## Commands

```bash
make dev          # Start dev server with hot reload (air)
make run          # Run without hot reload
make build        # Build binary to bin/autobidd
make docker-up    # Start PostgreSQL
make docker-down  # Stop PostgreSQL
go build ./...    # Check compilation
go test ./...     # Run tests
```

## Project structure

```
cmd/server/main.go          # Entry point
internal/
  config/config.go           # Env config loading
  database/database.go       # GORM connection + auto-migrate
  middleware/                 # Auth, logging, recovery, rate limiting
  models/                    # GORM models (User, Bid, Profile, Template, etc.)
  services/                  # Business logic (AI, auth, bids, analytics)
  handlers/                  # HTTP handlers
  views/render.go            # Template renderer
  router/router.go           # Route definitions
templates/
  layouts/                   # base.html, app.html (sidebar), auth.html (centered card)
  components/                # sidebar.html
  pages/                     # Full pages (define "title" + "content" blocks)
  partials/                  # HTMX partials (alert, bid_stream, etc.)
static/js/                   # htmx.min.js, alpine.min.js
specs/                       # Design docs and progress tracker
```

## Template system

Templates use a **layout-based composition** pattern:

- `renderer.Page(w, "layout", "page", data)` — layout is `"app"` or `"auth"`
- Each page only defines `{{define "title"}}` and `{{define "content"}}`
- The renderer compiles: `base.html` + specific layout + components + partials + page
- **No page should define `{{define "body"}}`** — that's the layout's job
- Partials are available in both pages and standalone `renderer.Partial()` calls

## AI integration

- System prompt is built from user profile, tone examples, portfolio, and custom instructions
- Output format: plain text cover letter + `---META---` separator + JSON metadata
- Streaming uses Claude's SSE API → forwarded to browser via EventSource
- Prompt caching enabled on system prompt and last chat history message
- Non-streaming fallback methods exist for backward compat

## Key conventions

- All handlers receive `*views.Renderer` and call `Page()` or `Partial()`
- HTMX handles most interactions — forms POST to `/api/*`, partials swap in
- Alpine.js handles client-side state (tags input, edit toggles, streaming)
- Rate limiting: auth endpoints 5/min, AI endpoints 10/min per IP
- Session-based auth with cookie, bcrypt passwords

## Environment

Copy `.env.example` to `.env` and fill in `CLAUDE_API_KEY`. Run `make docker-up` before starting.

## What NOT to do

- Don't add layout wrappers (sidebar, auth card) inside page templates — use the layout system
- Don't parse all layouts for every page — the renderer includes only the relevant layout
- Don't use `{{template "base" .}}` at the top of page files — it's unnecessary
- Don't return full JSON from Claude for streaming — use the text + `---META---` format
