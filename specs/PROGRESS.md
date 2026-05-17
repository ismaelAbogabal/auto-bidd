# Auto-Bidder — Progress Tracker

## Phase 1: Foundation

| Task | Status | Notes |
|------|--------|-------|
| Project scaffold (go mod, folders, Makefile) | [ ] | |
| Docker Compose (PostgreSQL) | [ ] | |
| Config loading (.env) | [ ] | |
| GORM connection + auto-migrate | [ ] | |
| Hot reload (air) setup | [ ] | |
| Static file serving (HTMX, Alpine.js, CSS) | [ ] | |
| Base HTML templates (layouts, nav) | [ ] | |

## Phase 2: Auth

| Task | Status | Notes |
|------|--------|-------|
| User model + migration | [ ] | |
| Register handler (bcrypt hash) | [ ] | |
| Login handler (session cookie) | [ ] | |
| Logout handler | [ ] | |
| Auth middleware (protect routes) | [ ] | |
| Login page (HTMX form) | [ ] | |
| Register page (HTMX form) | [ ] | |

## Phase 3: Profile

| Task | Status | Notes |
|------|--------|-------|
| UserProfile model | [ ] | |
| ToneExample model | [ ] | |
| PortfolioItem model | [ ] | |
| Profile page (view + edit) | [ ] | |
| Tone examples CRUD (HTMX partials) | [ ] | |
| Portfolio items CRUD (HTMX partials) | [ ] | |
| AI instructions editor | [ ] | |
| Skills tag input (Alpine.js) | [ ] | |

## Phase 4: Bid Builder

| Task | Status | Notes |
|------|--------|-------|
| Bid model | [ ] | |
| ChatMessage model | [ ] | |
| AI service (Claude API client) | [ ] | |
| Prompt builder (system prompt assembly) | [ ] | |
| Bid generation endpoint | [ ] | |
| SSE streaming response | [ ] | |
| Bid builder page (job input form) | [ ] | |
| Generated output display | [ ] | |
| Manual edit mode (Alpine.js toggle) | [ ] | |
| Chat refinement (HTMX + SSE) | [ ] | |
| Pricing display (hours x rate) | [ ] | |
| Q&A answers rendering | [ ] | |
| Bid list page | [ ] | |
| Bid detail page | [ ] | |
| Status update (draft/submitted/won/lost) | [ ] | |

## Phase 5: Templates

| Task | Status | Notes |
|------|--------|-------|
| Template model | [ ] | |
| Save winning bid as template | [ ] | |
| Template list page | [ ] | |
| Use template when generating new bid | [ ] | |
| Template win/use count tracking | [ ] | |

## Phase 6: Analytics

| Task | Status | Notes |
|------|--------|-------|
| Win rate calculation | [ ] | |
| Average pricing stats | [ ] | |
| Win rate by price bracket | [ ] | |
| Trends over time (chart data) | [ ] | |
| Template effectiveness | [ ] | |
| Analytics page (HTMX + Alpine charts) | [ ] | |

## Phase 7: Polish

| Task | Status | Notes |
|------|--------|-------|
| Error handling (flash messages) | [ ] | |
| Form validation (server + client) | [ ] | |
| Loading states (HTMX indicators) | [ ] | |
| Empty states | [ ] | |
| Responsive design | [ ] | |
| Rate limiting | [ ] | |
