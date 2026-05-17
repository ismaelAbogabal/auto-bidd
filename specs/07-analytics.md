# 07 — Analytics

## Purpose

Help users understand what's working so they can improve their bid strategy over time.

## Metrics

### Overview Stats

| Metric | Calculation |
|--------|-------------|
| Total bids | COUNT(bids) |
| Win rate | won / (won + lost) * 100 |
| Avg bid price | AVG(total_price) |
| Avg estimated hours | AVG(estimated_hours) |
| Avg response time | AVG(submitted_at - created_at) |
| Total revenue (won) | SUM(total_price) WHERE status=won |

### Win Rate by Price Bracket

```
< $500
$500 - $2,000
$2,000 - $5,000
$5,000 - $10,000
$10,000+
```

### Win Rate by Skill/Tag

Join bid's job description keywords with user's skills to see which skill areas have the best win rate.

### Trends Over Time

Monthly aggregates:
- Bids submitted
- Win rate
- Average price
- Revenue

### Template Effectiveness

| Template | Uses | Wins | Win Rate |
|----------|------|------|----------|
| Short & Punchy | 12 | 5 | 42% |
| Technical Deep | 7 | 3 | 43% |
| No template | 20 | 4 | 20% |

## Page Layout

```
/analytics
┌─────────────────────────────────────────────┐
│  Analytics                                   │
├─────────────────────────────────────────────┤
│                                             │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐      │
│  │  45  │ │ 38%  │ │$2.4k │ │$18k  │      │
│  │ Bids │ │ Win  │ │ Avg  │ │ Rev  │      │
│  └──────┘ └──────┘ └──────┘ └──────┘      │
│                                             │
│  [Trends Chart — line chart over months]    │
│                                             │
│  ┌─────────────────┐ ┌─────────────────┐   │
│  │ By Price Range   │ │ By Skill        │   │
│  │ <$500:  50%     │ │ React: 45%      │   │
│  │ $500-2k: 35%   │ │ Node:  38%      │   │
│  │ $2k-5k: 25%    │ │ Go:    55%      │   │
│  └─────────────────┘ └─────────────────┘   │
│                                             │
│  [Template Effectiveness Table]             │
│                                             │
└─────────────────────────────────────────────┘
```

## Implementation

### Endpoints

| Method | Path | Returns |
|--------|------|---------|
| GET | /analytics | Full page |
| GET | /api/analytics/overview | Stats cards partial |
| GET | /api/analytics/trends | JSON (for Alpine.js chart) |
| GET | /api/analytics/by-price | Price bracket partial |
| GET | /api/analytics/by-skill | Skill breakdown partial |
| GET | /api/analytics/templates | Template table partial |

### Query Filters

All endpoints accept:
- `period`: 30d, 90d, 6m, 1y, all (default: 90d)
- Applied via HTMX dropdown that triggers re-fetch

### Charts

Use a lightweight chart library compatible with Alpine.js:
- Option A: Chart.js (loaded from CDN, Alpine component wraps it)
- Option B: Simple CSS bar charts (no JS dependency)

Start with CSS bar charts, upgrade to Chart.js if needed.

## Data Freshness

Analytics are computed on-demand (no pre-aggregation). Queries are fast enough for single-user data volumes. If performance becomes an issue, add materialized views later.
