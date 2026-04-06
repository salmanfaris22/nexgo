# NexGo v2.1.0 — Islands Architecture

```
  ███╗   ██╗███████╗██╗  ██╗ ██████╗  ██████╗
  ████╗  ██║██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║  ███╗██║   ██║
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║██║   ██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝
```

**Ship zero JavaScript by default. Hydrate only what needs to be interactive.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Version](https://img.shields.io/badge/Version-2.1.0-7b2ff7?style=flat-square)](#)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](#)
[![Zero Dependencies](https://img.shields.io/badge/Zero-Dependencies-00d2ff?style=flat-square)](#)

---

## What's New in v2.1.0

NexGo v2.1.0 introduces **Islands Architecture** — the same pattern used by Astro, Fresh (Deno), and Eleventy. Pages are static HTML by default. Only interactive components ("islands") ship JavaScript to the browser, with fine-grained control over **when** they hydrate.

### Why This Matters

| Metric | Next.js (React) | NexGo v2.1.0 (Islands) |
|--------|-----------------|------------------------|
| JS shipped on a static page | ~85KB (React runtime) | **0 KB** |
| JS shipped for 1 interactive button | ~85KB + component | **~2KB** (just the island) |
| Time to Interactive (blog post) | ~1.2s | **~0.1s** |
| Hydration model | Full page | Per-component |
| JS loading control | None | 5 strategies |

> A blog page with a single like button ships **~2KB** with NexGo Islands vs **~85KB** with Next.js. Every page in Next.js pays the React tax — NexGo pages are free by default.

---

## Table of Contents

1. [Quick Start](#1-quick-start)
2. [How Islands Work](#2-how-islands-work)
3. [Creating Islands](#3-creating-islands)
4. [Hydration Strategies](#4-hydration-strategies)
5. [Passing Props](#5-passing-props)
6. [Writing Island JavaScript](#6-writing-island-javascript)
7. [SSR-Only Islands](#7-ssr-only-islands)
8. [Islands in Layouts](#8-islands-in-layouts)
9. [Multiple Islands Per Page](#9-multiple-islands-per-page)
10. [Nested Data from Loaders](#10-nested-data-from-loaders)
11. [Island Runtime](#11-island-runtime)
12. [Dev Mode & HMR](#12-dev-mode--hmr)
13. [Project Structure](#13-project-structure)
14. [Configuration](#14-configuration)
15. [Template Functions Reference](#15-template-functions-reference)
16. [Architecture Deep Dive](#16-architecture-deep-dive)
17. [Performance Guide](#17-performance-guide)
18. [Migration from v2.0](#18-migration-from-v20)
19. [Examples](#19-examples)
20. [API Reference](#20-api-reference)

---

## 1. Quick Start

### Install

```bash
go install github.com/salmanfaris22/nexgo/cmd/nexgo@v2.1.0
```

### Create a Project

```bash
nexgo create my-app
cd my-app
go mod tidy
nexgo dev
```

The scaffolded project includes a working counter island out of the box.

### Project Structure

```
my-app/
├── main.go
├── nexgo.config.json
├── go.mod
├── pages/
│   ├── index.html            # Uses {{ island "counter" ... }}
│   ├── about.html
│   └── api/
│       └── hello.go
├── islands/                   # NEW in v2.1.0
│   ├── counter.html           # Server template
│   └── counter.js             # Client JS (loaded on demand)
├── layouts/
│   └── default.html
├── components/
├── static/
│   └��─ css/global.css
└── .nexgo/
```

---

## 2. How Islands Work

Islands Architecture splits your page into two types of content:

**Static HTML** — Rendered on the server, sent as pure HTML. Zero JavaScript. This is the default for everything on the page.

**Islands** — Small interactive components that need JavaScript. Each island has a server-side template (HTML) and an optional client-side script (JS). The JS is only loaded when the island needs to hydrate.

```
┌─────────────────────────────────────────────┐
│  Page (Static HTML — 0 KB JS)               │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │  Header / Navigation (static)        │   │
│  └──────────────────────────────────────┘   │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │  Article Content (static)            │   │
│  │  Rendered server-side, no JS needed  │   │
│  └──────────────────────────────────────┘   │
│                                             │
│  ┌─────────────────┐  ┌─────────────────┐   │
│  │  Like Button     │  │  Comment Form   │   │
│  │  ISLAND (~1KB)   │  │  ISLAND (~3KB)  │   │
│  │  client:visible  │  │  client:idle    │   │
│  └─────────────────┘  └─────────────────┘   │
│                                             │
│  ┌──────────────────────────────────────┐   │
│  │  Footer (static)                     │   │
│  └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

### The Flow

1. Browser requests `/blog/my-post`
2. Go server renders the **entire page** as HTML (including islands)
3. Browser receives HTML — content is visible immediately
4. Island runtime (~1KB) scans for `<nexgo-island>` elements
5. Each island hydrates based on its strategy:
   - `client:load` → immediately
   - `client:visible` → when scrolled into view
   - `client:idle` → when browser has free time
   - `client:media` → when media query matches
   - `client:none` → never (SSR only)
6. Island JS is loaded via ES module import — only the JS for that specific island

---

## 3. Creating Islands

An island is a pair of files in the `islands/` directory:

| File | Purpose | Required |
|------|---------|----------|
| `islands/name.html` | Server-side Go template | Yes |
| `islands/name.js` | Client-side JavaScript | No (SSR-only if omitted) |

### Step 1: Create the Template

```html
<!-- islands/counter.html -->
<div class="counter">
  <span class="counter-value">{{ .count }}</span>
  <button class="counter-btn" data-action="decrement">-</button>
  <button class="counter-btn" data-action="increment">+</button>
</div>
```

The template receives props as its data context. Access them with `{{ .propName }}`.

### Step 2: Create the Client JS

```javascript
// islands/counter.js
export default function init(el, props) {
  let count = props.count || 0;
  const display = el.querySelector('.counter-value');

  el.querySelectorAll('.counter-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      count += btn.dataset.action === 'increment' ? 1 : -1;
      display.textContent = count;
    });
  });
}
```

**Rules for island JS:**
- Export a `default` function or a named `init` function
- The function receives two arguments: `el` (the `<nexgo-island>` DOM element) and `props` (the serialized props object)
- The function runs once when the island hydrates
- Use vanilla JS — no framework required (but you can import one if needed)

### Step 3: Use in a Page

```html
<!-- pages/index.html -->
<h1>My Page</h1>
<p>This is static HTML. Zero JavaScript.</p>

{{ island "counter" (props "count" 0) "client:load" }}
```

---

## 4. Hydration Strategies

Hydration strategies control **when** an island's JavaScript loads and executes. Choose the right strategy to minimize the JS impact on page load.

### `client:load`

Hydrate immediately when the page loads. Use for interactive elements that are visible above the fold and needed right away.

```html
{{ island "search-bar" (props "placeholder" "Search...") "client:load" }}
```

**When to use:** Navigation menus, search bars, auth status indicators.

### `client:visible`

Hydrate when the island scrolls into the viewport (using `IntersectionObserver` with a 200px margin). Use for interactive elements below the fold.

```html
{{ island "comment-form" (props "postId" .Params.slug) "client:visible" }}
```

**When to use:** Comment sections, like buttons, share widgets, anything below the fold.

### `client:idle`

Hydrate when the browser is idle (using `requestIdleCallback`). Use for non-urgent interactive elements.

```html
{{ island "analytics-widget" (props "data" .Props.stats) "client:idle" }}
```

**When to use:** Analytics dashboards, recommendation carousels, secondary UI.

### `client:media`

Hydrate only when a CSS media query matches. Use for device-specific interactive elements.

```html
{{ island "mobile-menu" nil "client:media=(max-width:768px)" }}
```

The island JS is **never loaded** on desktop — zero bytes shipped.

**When to use:** Mobile navigation, touch-specific interfaces, responsive widgets.

### `client:none`

Never hydrate. The island is rendered on the server and sent as static HTML. No JavaScript is ever shipped for this island, even if a `.js` file exists.

```html
{{ island "sidebar" (props "user" .State.user) "client:none" }}
```

**When to use:** Content that benefits from the island template system (reusable, props-driven) but doesn't need interactivity.

### Strategy Comparison

| Strategy | JS Loaded | When | Best For |
|----------|-----------|------|----------|
| `client:load` | Immediately | Page load | Above-the-fold interactive UI |
| `client:visible` | On scroll | Enter viewport | Below-the-fold widgets |
| `client:idle` | When idle | Browser free time | Non-critical interactive UI |
| `client:media` | On match | Media query matches | Device-specific UI |
| `client:none` | Never | — | Reusable SSR-only components |

---

## 5. Passing Props

Use the `props` template function to pass data from the server to an island.

### Basic Props

```html
{{ island "greeting" (props "name" "World" "emoji" "👋") }}
```

In the island template:
```html
<!-- islands/greeting.html -->
<p>{{ .emoji }} Hello, {{ .name }}!</p>
```

In the island JS:
```javascript
// islands/greeting.js
export default function(el, props) {
  console.log(props.name);  // "World"
  console.log(props.emoji); // "👋"
}
```

### Props from Data Loaders

Pass data fetched by `RegisterDataLoader` into islands:

```go
// main.go
srv.RegisterDataLoader("/dashboard", func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
    return map[string]interface{}{
        "stats":    fetchStats(),
        "alerts":   fetchAlerts(),
        "chartData": fetchChartData(),
    }, nil
})
```

```html
<!-- pages/dashboard.html -->
<h1>Dashboard</h1>

<!-- Static stats display — no JS -->
<div class="stats-grid">
  <div>Revenue: ${{ .Props.stats.Revenue }}</div>
  <div>Users: {{ .Props.stats.Users }}</div>
</div>

<!-- Interactive chart — loads JS only when scrolled into view -->
{{ island "chart" (props "data" .Props.chartData "type" "line") "client:visible" }}

<!-- Real-time alerts — loads JS when browser is idle -->
{{ island "alert-feed" (props "alerts" .Props.alerts) "client:idle" }}
```

### Props from Route Parameters

```html
<!-- pages/blog/[slug].html -->
{{ island "comment-section" (props "postSlug" .Params.slug) "client:visible" }}
```

### Props from Global State

```html
{{ island "user-menu" (props "user" .State.user) "client:load" }}
```

### No Props

```html
{{ island "theme-toggle" }}
{{ island "scroll-to-top" nil "client:visible" }}
```

---

## 6. Writing Island JavaScript

### Basic Structure

Every island JS file should export a default function:

```javascript
// islands/my-island.js
export default function init(el, props) {
  // el    — the <nexgo-island> DOM element (contains your server-rendered HTML)
  // props — the serialized props object from the template

  // Your interactive code here
}
```

### DOM Queries

Query within the island element to avoid conflicts with other islands:

```javascript
// islands/tabs.js
export default function(el, props) {
  const buttons = el.querySelectorAll('[data-tab]');
  const panels = el.querySelectorAll('[data-panel]');

  buttons.forEach(btn => {
    btn.addEventListener('click', () => {
      const target = btn.dataset.tab;
      panels.forEach(p => p.hidden = p.dataset.panel !== target);
      buttons.forEach(b => b.classList.toggle('active', b === btn));
    });
  });
}
```

### Fetching Data

Islands can fetch data from your API routes:

```javascript
// islands/live-count.js
export default function(el, props) {
  const display = el.querySelector('.count');

  async function refresh() {
    const res = await fetch('/api/visitors/count');
    const data = await res.json();
    display.textContent = data.count;
  }

  refresh();
  setInterval(refresh, 5000);
}
```

### Using NexGo State

Islands can read and write to the global NexGo state:

```javascript
// islands/cart-button.js
export default function(el, props) {
  const badge = el.querySelector('.badge');

  // Read current state
  const cart = NexGo.getState('cart', []);
  badge.textContent = cart.length;

  // Subscribe to changes from other islands
  NexGo.subscribe(state => {
    if (state.cart) {
      badge.textContent = state.cart.length;
    }
  });

  el.querySelector('.add-btn').addEventListener('click', () => {
    const cart = NexGo.getState('cart', []);
    cart.push(props.productId);
    NexGo.setState('cart', cart);
  });
}
```

### Importing Third-Party Libraries

Since island JS uses ES modules, you can import from CDNs:

```javascript
// islands/chart.js
import Chart from 'https://cdn.jsdelivr.net/npm/chart.js@4/+esm';

export default function(el, props) {
  const canvas = el.querySelector('canvas');
  new Chart(canvas, {
    type: props.type || 'bar',
    data: props.data,
  });
}
```

### Cleanup

If your island creates intervals, event listeners on `window`/`document`, or other side effects, clean them up when the element is removed:

```javascript
// islands/live-ticker.js
export default function(el, props) {
  const interval = setInterval(fetchData, 3000);

  // MutationObserver to detect removal
  const observer = new MutationObserver(() => {
    if (!document.contains(el)) {
      clearInterval(interval);
      observer.disconnect();
    }
  });
  observer.observe(el.parentNode, { childList: true });
}
```

---

## 7. SSR-Only Islands

Create islands **without** a `.js` file for reusable server-rendered components:

```
islands/
├── product-card.html    # No .js file — always SSR only
└── pricing-table.html   # No .js file — always SSR only
```

```html
<!-- islands/product-card.html -->
<div class="product-card">
  <img src="{{ .image }}" alt="{{ .name }}" loading="lazy">
  <h3>{{ .name }}</h3>
  <p class="price">${{ .price }}</p>
  <p class="description">{{ .description }}</p>
</div>
```

Use in pages:

```html
<!-- No JS shipped — pure HTML -->
{{ range .Props.products }}
  {{ island "product-card" (props "name" .Name "price" .Price "image" .Image "description" .Desc) "client:none" }}
{{ end }}
```

You can also force `client:none` on islands that **do** have a `.js` file — useful for pages where you want the server-rendered version without interactivity:

```html
<!-- Same island, interactive on /shop, static on /catalog -->
{{ island "product-card" (props "name" .Name) "client:none" }}
```

---

## 8. Islands in Layouts

You can use islands in layout files for site-wide interactive elements:

```html
<!-- layouts/default.html -->
<!DOCTYPE html>
<html lang="en"{{ if .DevMode }} data-nexgo-dev="1"{{ end }}>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/static/css/global.css">
  <script src="/_nexgo/runtime.js" defer></script>
  <script src="/_nexgo/island-runtime.js" defer></script>
</head>
<body>
  <nav>
    {{ island "nav-menu" (props "path" .Path) "client:load" }}
  </nav>

  <main id="nexgo-root">{{ .Content }}</main>

  <footer>Built with NexGo</footer>

  {{ island "scroll-to-top" nil "client:visible" }}
</body>
</html>
```

Alternatively, use the `islandRuntime` template function to inline the runtime:

```html
<head>
  <script src="/_nexgo/runtime.js" defer></script>
  {{ islandRuntime }}
</head>
```

---

## 9. Multiple Islands Per Page

Pages can contain any number of islands, each with its own strategy:

```html
<!-- pages/dashboard.html -->
<h1>Dashboard</h1>

<!-- Loads immediately — user sees search right away -->
{{ island "search" (props "endpoint" "/api/search") "client:load" }}

<!-- Static content — no JS -->
<section class="summary">
  <h2>Today's Summary</h2>
  <p>Revenue: ${{ .Props.stats.Revenue }}</p>
</section>

<!-- Loads when scrolled to — chart is below the fold -->
{{ island "revenue-chart" (props "data" .Props.chartData) "client:visible" }}

<!-- Loads when browser is idle — not critical -->
{{ island "notification-bell" (props "count" .Props.unread) "client:idle" }}

<!-- Only loads on mobile -->
{{ island "mobile-filters" (props "categories" .Props.categories) "client:media=(max-width:768px)" }}

<!-- SSR only — no JS for this one -->
{{ island "recent-activity" (props "events" .Props.events) "client:none" }}
```

Each island loads its own JS independently. There is no shared bundle — if the user never scrolls to the chart, that JS is never downloaded.

---

## 10. Nested Data from Loaders

Use `ParallelLoader` to fetch data for multiple islands concurrently:

```go
// main.go
loader := renderer.NewParallelLoader()
loader.Add("stats", fetchStats)
loader.Add("chart", fetchChartData)
loader.Add("alerts", fetchAlerts)
loader.Add("activity", fetchActivity)

srv.RegisterDataLoader("/dashboard", loader.Execute)
```

All four data sources are fetched **in parallel** via goroutines. Total latency = slowest query, not sum of all queries.

```html
<!-- pages/dashboard.html -->
{{ island "stats-panel" (props "data" .Props.stats) "client:load" }}
{{ island "chart" (props "data" .Props.chart) "client:visible" }}
{{ island "alerts" (props "items" .Props.alerts) "client:idle" }}
{{ island "activity-feed" (props "events" .Props.activity) "client:none" }}
```

---

## 11. Island Runtime

The island runtime is a lightweight (~1KB) JavaScript file that handles island discovery and hydration.

### How It Works

1. On page load, the runtime scans for `<nexgo-island>` elements
2. For each island, it reads the `data-strategy` attribute
3. Based on the strategy, it schedules hydration:
   - **load** → runs immediately
   - **visible** → sets up an `IntersectionObserver`
   - **idle** → calls `requestIdleCallback`
   - **media** → listens to `matchMedia`
4. When hydration triggers, it dynamically imports the island's JS module
5. The JS module's default export is called with `(element, props)`

### Including the Runtime

**Option A: Script tag (recommended)**

```html
<!-- layouts/default.html -->
<script src="/_nexgo/island-runtime.js" defer></script>
```

**Option B: Inline via template function**

```html
{{ islandRuntime }}
```

### SPA Navigation

The runtime automatically re-initializes islands after SPA navigations. When NexGo's client-side router loads a new page, it fires a `nexgo:navigate` event, and the island runtime scans for new `<nexgo-island>` elements.

### No Islands? No JS.

If a page has zero `<nexgo-island>` elements, the runtime does nothing — it's a no-op. Pages without islands are pure static HTML.

---

## 12. Dev Mode & HMR

In development mode (`nexgo dev`), islands support hot reload:

- Editing `islands/counter.html` → server re-renders, browser updates via HMR
- Editing `islands/counter.js` → browser reloads the island JS module
- Adding a new island file → automatically discovered on next request

The dev server watches the `islands/` directory alongside `pages/`, `layouts/`, and `components/`.

### Dev Banner

When islands are detected, the startup banner shows them:

```
  ⚡ NexGo — Go-powered web framework

  Local:    http://localhost:3000
  Mode:     development (hot reload on)
  Pages:    ./pages/
  Islands:  3 (counter, chart, search)
  Devtools: http://localhost:3000/_nexgo/devtools
```

---

## 13. Project Structure

```
my-app/
├── main.go                    # Application entry point
├── nexgo.config.json          # Framework configuration
├── go.mod
│
├── pages/                     # File-based routes
│   ├── index.html             #   → /
│   ├── about.html             #   → /about
│   ├── blog/
│   │   ├── index.html         #   → /blog
│   │   └── [slug].html        #   → /blog/:slug
│   └── api/
│       └── hello.go           #   → /api/hello
│
├── islands/                   # Interactive island components
│   ├── counter.html           #   Server template
│   ├── counter.js             #   Client JS (ES module)
│   ├── search.html
│   ├── search.js
��   ├── product-card.html      #   SSR-only (no .js file)
│   └── chart.html
│       chart.js
│
├── layouts/                   # Page layouts
│   └── default.html
│
├── components/                # Reusable template partials
│
├── static/                    # Static assets
│   ├── css/global.css
│   └── js/
│
└── .nexgo/                    # Build output
    └── out/
```

### File Naming

| File | Purpose |
|------|---------|
| `islands/name.html` | Server-side Go template. Receives props as `.propName` |
| `islands/name.js` | Client-side ES module. Exports `default function(el, props)` |
| `islands/name.gohtml` | Alternative template extension |
| `islands/name.tmpl` | Alternative template extension |

Subdirectories are supported:

```
islands/
├── ui/
│   ├── button.html
│   └── button.js
└── charts/
    ├── bar.html
    └── bar.js
```

Use the full path in templates:

```html
{{ island "ui/button" (props "label" "Click me") }}
{{ island "charts/bar" (props "data" .Props.chart) "client:visible" }}
```

---

## 14. Configuration

Add `islandsDir` to your `nexgo.config.json`:

```json
{
  "projectName": "my-app",
  "port": 3000,
  "pagesDir": "pages",
  "staticDir": "static",
  "layoutsDir": "layouts",
  "componentsDir": "components",
  "islandsDir": "islands",
  "outputDir": ".nexgo/out",
  "hotReload": true,
  "compression": true,
  "minify": true,
  "defaultRenderMode": "ssr"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `islandsDir` | string | `"islands"` | Directory containing island templates and JS |

If `islandsDir` is not set, it defaults to `"islands"`. If the directory doesn't exist, NexGo runs normally without islands — no errors.

---

## 15. Template Functions Reference

### Island Functions (New in v2.1.0)

| Function | Usage | Description |
|----------|-------|-------------|
| `island` | `{{ island "name" }}` | Render an island with default strategy (`client:load`) |
| `island` | `{{ island "name" (props "k" "v") }}` | Render with props |
| `island` | `{{ island "name" (props "k" "v") "client:visible" }}` | Render with props and strategy |
| `island` | `{{ island "name" nil "client:idle" }}` | No props, custom strategy |
| `props` | `{{ props "key" "val" "key2" "val2" }}` | Build a props map for islands |
| `islandRuntime` | `{{ islandRuntime }}` | Inline the island hydration runtime script |

### All Template Functions

| Function | Usage | Description |
|----------|-------|-------------|
| `island` | See above | Render an interactive island |
| `props` | `{{ props "k" "v" }}` | Build props map |
| `islandRuntime` | `{{ islandRuntime }}` | Inline island runtime |
| `json` | `{{ json .Data }}` | JSON encode a value |
| `safeHTML` | `{{ safeHTML "<b>bold</b>" }}` | Render raw HTML |
| `dict` | `{{ dict "key" "val" }}` | Create a map |
| `slice` | `{{ slice 1 2 3 }}` | Create a slice |
| `asset` | `{{ asset "/static/css/app.css" }}` | Cache-busted asset URL |
| `link` | `{{ link "/about" }}` | Link to a page |
| `renderState` | `{{ renderState .State }}` | Inject state hydration |
| `times` | `{{ range times 5 }}...{{ end }}` | Iterate n times |
| `default` | `{{ default "fallback" .Value }}` | Default if nil/empty |
| `upper` | `{{ upper "hello" }}` | Uppercase |
| `lower` | `{{ lower "HELLO" }}` | Lowercase |
| `title` | `{{ title "hello world" }}` | Title case |
| `replace` | `{{ replace "hello" "l" "r" }}` | Replace substring |
| `trim` | `{{ trim "  hi  " }}` | Trim whitespace |
| `split` | `{{ split "a,b" "," }}` | Split string |
| `join` | `{{ join .Items "," }}` | Join slice |
| `add` | `{{ add 1 2 }}` | Addition |
| `sub` | `{{ sub 5 3 }}` | Subtraction |
| `mul` | `{{ mul 3 4 }}` | Multiplication |
| `div` | `{{ div 10 2 }}` | Division |

---

## 16. Architecture Deep Dive

### Server Side

```
Request → Router → Match Route → Data Loader → Renderer
                                                  ↓
                                          Render Page Template
                                                  ↓
                                    For each {{ island "name" ... }}:
                                      1. Find island template in registry
                                      2. Execute template with props
                                      3. Wrap output in <nexgo-island>
                                      4. Serialize props as data-props
                                                  ↓
                                          Wrap in Layout
                                                  ↓
                                          Send HTML Response
```

### Client Side

```
Browser receives HTML
  ↓
Content is visible immediately (no JS needed)
  ↓
Island runtime loads (~1KB)
  ↓
Scans for <nexgo-island> elements
  ↓
For each island:
  ├── client:load    → import('/_nexgo/islands/name.js') immediately
  ├── client:visible → IntersectionObserver → import when visible
  ├── client:idle    → requestIdleCallback → import when idle
  ├── client:media   → matchMedia → import when query matches
  └── client:none    → skip (no JS)
  ↓
Island JS runs: init(element, props)
  ↓
Island is interactive
```

### Generated HTML

When you write:

```html
{{ island "counter" (props "count" 5) "client:visible" }}
```

NexGo generates:

```html
<nexgo-island data-name="counter" data-strategy="client:visible" data-has-js="true" data-props='{"count":5}'>
  <div class="counter">
    <span class="counter-value">5</span>
    <button class="counter-btn" data-action="decrement">-</button>
    <button class="counter-btn" data-action="increment">+</button>
  </div>
</nexgo-island>
```

Key points:
- The **content is server-rendered** — visible before any JS loads
- `data-strategy` tells the runtime **when** to hydrate
- `data-has-js` indicates whether a JS file exists
- `data-props` carries the serialized props for client-side use
- `data-hydrated` is set after hydration to prevent double-hydration

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/_nexgo/islands/{name}.js` | Serves individual island JS files |
| `/_nexgo/island-runtime.js` | Serves the island hydration runtime |
| `/_nexgo/runtime.js` | Main NexGo client runtime (SPA router, HMR, state) |

---

## 17. Performance Guide

### Choosing the Right Strategy

```
                   ┌─────────────┐
                   │ Is it above  │
                   │  the fold?   │
                   └──────┬──────┘
                     yes/   \no
                    ↓         ↓
            ┌───────────┐  ┌─────────────┐
            │ Is it      │  │ Is it        │
            │ critical?  │  │ interactive? │
            └────┬──────┘  └──────┬──────┘
           yes/   \no        yes/   \no
          ↓         ↓       ↓         ↓
    client:load  client:idle  client:visible  client:none
```

### Bundle Size Comparison

| Scenario | Next.js | NexGo Islands |
|----------|---------|---------------|
| Static blog page | ~85KB | **0 KB** |
| Blog + like button | ~87KB | **~1.5KB** |
| Dashboard (5 widgets) | ~120KB | **~8KB** (loaded incrementally) |
| Marketing landing page | ~85KB | **0 KB** |
| E-commerce product page | ~150KB | **~5KB** (only cart + gallery) |

### Tips

1. **Default to `client:none`** — Only add JS when you actually need interactivity
2. **Use `client:visible` for below-the-fold** — Saves initial page load time
3. **Use `client:idle` for non-critical UI** — Notification bells, analytics, tooltips
4. **Use `client:media` for responsive** — Don't load mobile-only JS on desktop
5. **Keep island JS small** — Each island should do one thing well
6. **Import from CDN** — Use `import ... from 'https://cdn...'` instead of bundling libraries
7. **Avoid large props** — Don't pass entire database objects; pass only what the island needs

---

## 18. Migration from v2.0

### What Changed

| Change | Action Required |
|--------|-----------------|
| New `islands/` directory | Create the directory (or don't — it's optional) |
| New `islandsDir` config | Add to `nexgo.config.json` (defaults to `"islands"`) |
| New template functions | `island`, `props`, `islandRuntime` are available automatically |
| New endpoints | `/_nexgo/islands/*` and `/_nexgo/island-runtime.js` are served automatically |
| Layout update | Add `<script src="/_nexgo/island-runtime.js" defer></script>` to layouts that use islands |

### Steps

1. Update NexGo:
   ```bash
   go get github.com/salmanfaris22/nexgo/v2@v2.1.0
   ```

2. Create `islands/` directory:
   ```bash
   mkdir islands
   ```

3. Add the island runtime to your layout:
   ```html
   <!-- layouts/default.html -->
   <script src="/_nexgo/island-runtime.js" defer></script>
   ```

4. Start creating islands!

**No breaking changes.** All existing pages, components, and APIs work exactly as before. Islands are purely additive.

---

## 19. Examples

### Example 1: Counter

```html
<!-- islands/counter.html -->
<div class="counter">
  <span class="counter-value">{{ .count }}</span>
  <button class="counter-btn" data-action="decrement">-</button>
  <button class="counter-btn" data-action="increment">+</button>
</div>
```

```javascript
// islands/counter.js
export default function(el, props) {
  let count = props.count || 0;
  const display = el.querySelector('.counter-value');

  el.querySelectorAll('.counter-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      count += btn.dataset.action === 'increment' ? 1 : -1;
      display.textContent = count;
    });
  });
}
```

```html
<!-- pages/index.html -->
{{ island "counter" (props "count" 0) "client:load" }}
```

### Example 2: Search with API

```html
<!-- islands/search.html -->
<div class="search-box">
  <input type="text" class="search-input" placeholder="{{ .placeholder }}" />
  <div class="search-results"></div>
</div>
```

```javascript
// islands/search.js
export default function(el, props) {
  const input = el.querySelector('.search-input');
  const results = el.querySelector('.search-results');
  let timeout;

  input.addEventListener('input', () => {
    clearTimeout(timeout);
    timeout = setTimeout(async () => {
      const q = input.value.trim();
      if (!q) { results.innerHTML = ''; return; }
      const res = await fetch(`/api/search?q=${encodeURIComponent(q)}`);
      const data = await res.json();
      results.innerHTML = data.items
        .map(item => `<a href="${item.url}">${item.title}</a>`)
        .join('');
    }, 300);
  });
}
```

```html
{{ island "search" (props "placeholder" "Search articles...") "client:load" }}
```

### Example 3: Lazy-Loaded Chart

```html
<!-- islands/chart.html -->
<div class="chart-container">
  <canvas width="600" height="300"></canvas>
</div>
```

```javascript
// islands/chart.js
import Chart from 'https://cdn.jsdelivr.net/npm/chart.js@4/+esm';

export default function(el, props) {
  const canvas = el.querySelector('canvas');
  new Chart(canvas, {
    type: props.type || 'line',
    data: props.data,
    options: { responsive: true, maintainAspectRatio: false }
  });
}
```

```html
<!-- Only loads Chart.js when the user scrolls to see it -->
{{ island "chart" (props "type" "bar" "data" .Props.chartData) "client:visible" }}
```

### Example 4: Mobile-Only Menu

```html
<!-- islands/mobile-menu.html -->
<div class="mobile-menu">
  <button class="hamburger">☰</button>
  <div class="menu-panel" hidden>
    <a href="/">Home</a>
    <a href="/about">About</a>
    <a href="/blog">Blog</a>
  </div>
</div>
```

```javascript
// islands/mobile-menu.js
export default function(el) {
  const btn = el.querySelector('.hamburger');
  const panel = el.querySelector('.menu-panel');

  btn.addEventListener('click', () => {
    panel.hidden = !panel.hidden;
    btn.textContent = panel.hidden ? '☰' : '✕';
  });
}
```

```html
<!-- JS never loads on desktop — zero bytes -->
{{ island "mobile-menu" nil "client:media=(max-width:768px)" }}
```

### Example 5: Product Card (SSR Only)

```html
<!-- islands/product-card.html -->
<div class="product-card">
  <img src="{{ .image }}" alt="{{ .name }}" loading="lazy" width="300" height="200">
  <h3>{{ .name }}</h3>
  <p class="price">${{ .price }}</p>
  {{ if .sale }}<span class="badge">Sale</span>{{ end }}
</div>
```

```html
<!-- No JS file for product-card — pure SSR -->
<!-- Reusable across pages with different data -->

<!-- pages/shop.html -->
<div class="product-grid">
  {{ range .Props.products }}
    {{ island "product-card" (props "name" .Name "price" .Price "image" .Image "sale" .OnSale) "client:none" }}
  {{ end }}
</div>
```

---

## 20. API Reference

### Go API

#### `islands.NewRegistry(dir string, funcMap template.FuncMap) *Registry`

Creates a new island registry that will scan the given directory.

#### `(*Registry) Scan() error`

Discovers all `.html` and `.js` files in the islands directory. Call this on startup and after file changes.

#### `(*Registry) Get(name string) (*Island, bool)`

Returns an island by name.

#### `(*Registry) GetJS(name string) ([]byte, bool)`

Returns the raw JS source for an island.

#### `(*Registry) Names() []string`

Returns all registered island names.

#### `(*Registry) Render(name string, props map[string]interface{}, strategy string) template.HTML`

Renders an island server-side and wraps it in a `<nexgo-island>` element.

#### `islands.RuntimeJS() string`

Returns the client-side island hydration runtime as a JavaScript string.

### Template API

| Function | Signature | Returns |
|----------|-----------|---------|
| `island` | `island(name string, args ...interface{})` | `template.HTML` |
| `props` | `props(pairs ...interface{})` | `map[string]interface{}` |
| `islandRuntime` | `islandRuntime()` | `template.HTML` |

### HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/_nexgo/islands/{name}.js` | Serves island JS files |
| GET | `/_nexgo/island-runtime.js` | Serves the hydration runtime |

### Island Struct

```go
type Island struct {
    Name     string             // Island name (matches filename)
    Template *template.Template // Compiled server-side template
    HasJS    bool               // Whether a .js file exists
    JSPath   string             // Absolute path to the .js file
}
```

### Hydration Strategies

| Constant | Value | Client Behavior |
|----------|-------|-----------------|
| `StrategyLoad` | `"client:load"` | Hydrate immediately |
| `StrategyVisible` | `"client:visible"` | `IntersectionObserver` (200px margin) |
| `StrategyIdle` | `"client:idle"` | `requestIdleCallback` (fallback: 200ms timeout) |
| `StrategyMedia` | `"client:media"` | `matchMedia` — format: `client:media=(query)` |
| `StrategyNone` | `"client:none"` | No hydration, no JS shipped |

---

## Changelog

### v2.1.0 (2026-04-06)

**New: Islands Architecture**

- Added `pkg/islands` package with full island registry, rendering, and hydration
- Added `island`, `props`, and `islandRuntime` template functions
- Added `/_nexgo/islands/{name}.js` endpoint for serving island JS
- Added `/_nexgo/island-runtime.js` endpoint for serving hydration runtime
- Added `islandsDir` config option (default: `"islands"`)
- Added `islands/` directory to project scaffold with counter example
- Added HMR support for island files
- Added 11 unit tests for the islands package
- **No breaking changes** — fully backwards compatible with v2.0.x

### v2.0.3

- Environment variables, session management, JWT auth, CSRF, rate limiting
- i18n, image optimization, WebSocket, health checks, structured logging
- Metrics, database, migrations, plugin system, server actions
- Error pages, testing utilities, middleware, deployment adapters
- Redis cache, asset pipeline, ORM, cluster mode

### v1.0.5

- File-based routing, SSR, SSG, HMR, API routes
- Layouts, components, template engine, state management
- Middleware, static files, DevTools panel

---

Built with Go. Zero dependencies. Ship less JavaScript.
