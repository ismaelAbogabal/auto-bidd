# 09 — Frontend Patterns

## Template Rendering

### Go Template Structure

```
templates/
├── layouts/
│   ├── base.html        ← <html>, <head>, HTMX/Alpine scripts, <body>
│   ├── app.html         ← authenticated: nav + sidebar + {{block "content"}}
│   └── auth.html        ← unauthenticated: centered card + {{block "content"}}
├── pages/               ← define "content" block, full page
├── partials/            ← HTML fragments returned for HTMX swaps
└── components/          ← reusable pieces ({{template "navbar" .}})
```

### Rendering Helper

```go
type ViewRenderer struct {
    templates map[string]*template.Template
}

func (v *ViewRenderer) Page(w http.ResponseWriter, name string, data any)    // full page
func (v *ViewRenderer) Partial(w http.ResponseWriter, name string, data any) // fragment only
```

Handlers check `HX-Request` header to decide full page vs partial.

## HTMX Patterns Used

### Pattern 1: Form Submit → Swap Response

```html
<form hx-post="/api/profile" hx-target="#alert-area" hx-swap="innerHTML">
  ...
  <button type="submit">Save</button>
</form>
<div id="alert-area"></div>
```

### Pattern 2: Delete Item → Remove from DOM

```html
<div id="tone-{id}" class="tone-item">
  <span>{{.Label}}</span>
  <button hx-delete="/api/profile/tone/{{.ID}}"
          hx-target="#tone-{{.ID}}"
          hx-swap="delete">×</button>
</div>
```

### Pattern 3: Append to List

```html
<div id="tone-list">
  {{range .ToneExamples}}
    {{template "tone_item" .}}
  {{end}}
</div>

<form hx-post="/api/profile/tone" hx-target="#tone-list" hx-swap="beforeend">
  ...
</form>
```

### Pattern 4: SSE Streaming

```html
<div hx-ext="sse"
     sse-connect="/api/bids/{{.ID}}/stream"
     sse-swap="message"
     hx-target="#bid-output">
  <div id="bid-output">
    <!-- streamed content lands here -->
  </div>
</div>
```

### Pattern 5: Tabs with HTMX

```html
<div class="tabs">
  <button hx-get="/api/analytics/overview" hx-target="#tab-content" class="active">Overview</button>
  <button hx-get="/api/analytics/trends" hx-target="#tab-content">Trends</button>
</div>
<div id="tab-content">
  <!-- loaded content -->
</div>
```

## Alpine.js Patterns Used

### Pattern 1: Edit/View Toggle

```html
<div x-data="{ editing: false }">
  <div x-show="!editing">
    <p>{{.CoverLetter}}</p>
    <button @click="editing = true">Edit</button>
  </div>
  <div x-show="editing">
    <textarea>{{.CoverLetter}}</textarea>
    <button @click="editing = false">Cancel</button>
    <button @click="$refs.form.submit(); editing = false">Save</button>
  </div>
</div>
```

### Pattern 2: Tag Input

```html
<div x-data="{ skills: {{.Skills | json}}, newSkill: '' }">
  <div class="flex flex-wrap gap-1">
    <template x-for="(skill, i) in skills" :key="i">
      <span class="tag">
        <span x-text="skill"></span>
        <button @click="skills.splice(i, 1)">×</button>
      </span>
    </template>
  </div>
  <input x-model="newSkill"
         @keydown.enter.prevent="if(newSkill.trim()) { skills.push(newSkill.trim()); newSkill='' }">
  <input type="hidden" name="skills" :value="JSON.stringify(skills)">
</div>
```

### Pattern 3: Confirmation Modal

```html
<div x-data="{ open: false, action: '' }">
  <button @click="open = true; action = '/api/bids/123'">Delete</button>

  <div x-show="open" class="modal">
    <p>Are you sure?</p>
    <button @click="htmx.ajax('DELETE', action, {}); open = false">Yes</button>
    <button @click="open = false">Cancel</button>
  </div>
</div>
```

### Pattern 4: Price Calculator

```html
<div x-data="{ hours: {{.EstimatedHours}}, rate: {{.HourlyRate}} }">
  <input type="number" x-model="hours" name="estimated_hours">
  <span x-text="'$' + rate + '/hr'"></span>
  <strong x-text="'$' + (hours * rate).toLocaleString()"></strong>
</div>
```

## CSS Framework

Tailwind CSS via CDN in development:
```html
<script src="https://cdn.tailwindcss.com"></script>
```

For production: build step with Tailwind CLI to generate minimal CSS.

## Page Load Strategy

1. Initial page load → full HTML (SSR, fast TTFB)
2. Subsequent interactions → HTMX partial swaps (no full reload)
3. Client state → Alpine.js (edit modes, calculators, modals)
4. No SPA routing, no client-side bundle, no build step in dev
