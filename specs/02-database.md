# 02 — Database Schema

## Entity Relationship

```
User 1──1 UserProfile
User 1──N ToneExample
User 1──N PortfolioItem
User 1──N Bid
User 1──N Template
Bid  1──N ChatMessage
Bid  N──1 Template (optional)
Template N──1 Bid (source_bid)
```

## Tables

### users

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK, default gen_random_uuid() |
| email | varchar(255) | unique, not null |
| password_hash | varchar(255) | not null |
| created_at | timestamptz | not null |
| updated_at | timestamptz | not null |

### user_profiles

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id), unique, not null |
| full_name | varchar(255) | |
| title | varchar(255) | |
| hourly_rate | decimal(10,2) | default 0 |
| ai_instructions | text | |
| skills | jsonb | default '[]' |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### tone_examples

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id), not null |
| label | varchar(255) | |
| content | text | not null |
| context | text | description of when this tone is used |
| created_at | timestamptz | |

### portfolio_items

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id), not null |
| title | varchar(255) | not null |
| description | text | |
| tech_stack | jsonb | default '[]' |
| outcome | text | |
| url | varchar(500) | |
| created_at | timestamptz | |

### bids

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id), not null |
| template_id | uuid | FK templates(id), nullable |
| job_title | varchar(500) | not null |
| job_description | text | not null |
| job_budget_min | decimal(10,2) | |
| job_budget_max | decimal(10,2) | |
| cover_letter | text | |
| estimated_hours | int | |
| hourly_rate | decimal(10,2) | |
| total_price | decimal(10,2) | |
| qa_answers | jsonb | default '[]' |
| status | varchar(20) | default 'draft' |
| platform | varchar(100) | |
| submitted_at | timestamptz | nullable |
| created_at | timestamptz | |
| updated_at | timestamptz | |

**Status values:** draft, submitted, won, lost, withdrawn

**QA answers format:**
```json
[
  {"question": "What is your experience with X?", "answer": "I have 5 years..."}
]
```

### chat_messages

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| bid_id | uuid | FK bids(id), not null |
| role | varchar(20) | not null (user, assistant) |
| content | text | not null |
| created_at | timestamptz | |

### templates

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | PK |
| user_id | uuid | FK users(id), not null |
| source_bid_id | uuid | FK bids(id) |
| name | varchar(255) | not null |
| cover_letter_template | text | |
| tags | jsonb | default '[]' |
| win_count | int | default 0 |
| use_count | int | default 0 |
| created_at | timestamptz | |

## Indexes

```sql
CREATE INDEX idx_bids_user_id ON bids(user_id);
CREATE INDEX idx_bids_status ON bids(user_id, status);
CREATE INDEX idx_bids_created_at ON bids(user_id, created_at DESC);
CREATE INDEX idx_chat_messages_bid_id ON chat_messages(bid_id);
CREATE INDEX idx_templates_user_id ON templates(user_id);
```

## GORM Notes

- Use `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"` for IDs
- Enable uuid-ossp or use pgcrypto extension for gen_random_uuid()
- Use `gorm:"serializer:json"` for jsonb fields mapped to Go slices/structs
- AutoMigrate in dev, versioned migrations for prod
