# 01 — Architecture

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.23+ |
| Router | chi/v5 |
| ORM | GORM |
| Database | PostgreSQL 16 |
| Frontend | Server-rendered Go templates |
| Interactivity | HTMX 2.x + Alpine.js 3.x |
| CSS | Tailwind CSS (CDN for dev, bundled for prod) |
| AI | Claude API (anthropic-sdk-go or raw HTTP) |

## Request Flow

```
Browser
  │
  ├── Full page load: GET /bids → handler renders full HTML (layout + page)
  │
  ├── HTMX request: POST /api/bids → handler renders HTML fragment (partial)
  │   (HX-Request header present)
  │
  └── SSE stream: GET /api/bids/{id}/stream → handler streams HTML chunks
```

## Project Layout

```
cmd/server/main.go          → entry point, wire dependencies
internal/config/            → env/config parsing
internal/database/          → GORM init, migrations
internal/models/            → GORM structs
internal/handlers/          → HTTP handlers (one file per domain)
internal/services/          → business logic layer
internal/middleware/        → auth, logging, recovery
internal/router/            → route registration
internal/views/             → template rendering helpers
templates/                  → HTML templates (layouts, pages, partials, components)
static/                     → CSS, JS, images
migrations/                 → raw SQL (optional, GORM auto-migrate primary)
specs/                      → this folder
```

## Dependency Injection

Simple struct-based DI. No framework.

```go
type App struct {
    Config   *config.Config
    DB       *gorm.DB
    Auth     *services.AuthService
    AI       *services.AIService
    Bids     *services.BidService
    Analytics *services.AnalyticsService
}
```

`main.go` builds the App, passes it to router, handlers receive what they need.

## Error Handling

- Handlers return rendered error partials for HTMX requests
- Full page errors use a shared error template
- Flash messages via cookie (set → read → clear pattern)
- AI errors surface as user-friendly messages in the bid builder

## Environment Variables

```
DATABASE_URL=postgres://user:pass@localhost:5432/autobidd?sslmode=disable
CLAUDE_API_KEY=sk-ant-...
SESSION_SECRET=random-32-byte-string
PORT=8080
ENV=development
```
