# Auto-Bidder — Progress Tracker

## Phase 1: Foundation

| Task | Status | Notes |
|------|--------|-------|
| Project scaffold (go mod, folders, Makefile) | [x] | |
| Docker Compose (PostgreSQL) | [x] | |
| Config loading (.env) | [x] | |
| GORM connection + auto-migrate | [x] | |
| Hot reload (air) setup | [x] | |
| Static file serving (HTMX, Alpine.js, CSS) | [x] | |
| Base HTML templates (layouts, nav) | [x] | Refactored to layout-based rendering (app/auth) |

## Phase 2: Auth

| Task | Status | Notes |
|------|--------|-------|
| User model + migration | [x] | |
| Register handler (bcrypt hash) | [x] | |
| Login handler (session cookie) | [x] | |
| Logout handler | [x] | |
| Auth middleware (protect routes) | [x] | |
| Login page (HTMX form) | [x] | |
| Register page (HTMX form) | [x] | |

## Phase 3: Profile

| Task | Status | Notes |
|------|--------|-------|
| UserProfile model | [x] | |
| ToneExample model | [x] | |
| PortfolioItem model | [x] | |
| Profile page (view + edit) | [x] | |
| Tone examples CRUD (HTMX partials) | [x] | |
| Portfolio items CRUD (HTMX partials) | [x] | |
| AI instructions editor | [x] | |
| Skills tag input (Alpine.js) | [x] | |

## Phase 4: Bid Builder

| Task | Status | Notes |
|------|--------|-------|
| Bid model | [x] | |
| ChatMessage model | [x] | |
| AI service (Claude API client) | [x] | |
| Prompt builder (system prompt assembly) | [x] | |
| Bid generation endpoint | [x] | |
| SSE streaming response | [x] | Bid generation + chat refinement |
| Bid builder page (job input form) | [x] | |
| Generated output display | [x] | |
| Manual edit mode (Alpine.js toggle) | [x] | |
| Chat refinement (HTMX + SSE) | [x] | Full SSE streaming |
| Pricing display (hours x rate) | [x] | |
| Q&A answers rendering | [x] | |
| Bid list page | [x] | |
| Bid detail page | [x] | |
| Status update (draft/submitted/won/lost) | [x] | |

## Phase 5: Templates

| Task | Status | Notes |
|------|--------|-------|
| Template model | [x] | |
| Save winning bid as template | [x] | |
| Template list page | [x] | |
| Use template when generating new bid | [x] | |
| Template win/use count tracking | [x] | |

## Phase 6: Analytics

| Task | Status | Notes |
|------|--------|-------|
| Win rate calculation | [x] | |
| Average pricing stats | [x] | |
| Win rate by price bracket | [x] | CSS bar charts |
| Trends over time (chart data) | [x] | Monthly aggregates with bar chart |
| Template effectiveness | [x] | Includes "no template" baseline |
| Analytics page (HTMX + Alpine charts) | [x] | Period filter (30d/90d/6m/1y/all) |

## Phase 7: Polish

| Task | Status | Notes |
|------|--------|-------|
| Error handling (flash messages) | [x] | Auto-dismiss alerts with icons, dismissible |
| Form validation (server + client) | [x] | Client-side password match, min length, disabled submit |
| Loading states (HTMX indicators) | [x] | Bid generation + chat refinement |
| Empty states | [x] | Bid list + templates |
| Responsive design | [x] | Collapsible sidebar, mobile header, responsive grids |
| Rate limiting | [x] | Auth: 5/min, AI endpoints: 10/min |
