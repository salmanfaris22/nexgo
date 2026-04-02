
<p align="center">
  <pre align="center">
  ███╗   ██╗███████╗██╗  ██╗ ██████╗  ██████╗ 
  ████╗  ██║██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║  ███╗██║   ██║
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║██║   ██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝
  </pre>
</p>


INSTALL CODE go install github.com/salmanfaris22/nexgo/cmd/nexgo@v1.0.5

✅ FIX PATH (VERY IMPORTANT)

Run:

export PATH=$PATH:$(go env GOPATH)/bin
Make it permanent (recommended)
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
🔁 Now test again
nexgo



<h3 align="center">The Go-Powered Web Framework Inspired by Next.js</h3>

<p align="center">
  <strong>File-based routing | Server-side rendering | Hot reload | API routes | Single binary deploy</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.22+">
  <img src="https://img.shields.io/badge/Version-1.0.0-7b2ff7?style=flat-square" alt="Version 1.0.0">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Zero-Dependencies-00d2ff?style=flat-square" alt="Zero Dependencies">
</p>

---

## What is NexGo?

NexGo brings the **Next.js developer experience** to the **Go ecosystem**. Build full-stack web applications with file-based routing, server-side rendering, API routes, and hot reload -- all compiled into a **single binary** with **zero runtime dependencies**.

```
50,000+ requests/sec  |  ~10MB binary  |  No Node.js  |  No npm  |  No runtime
```

---

## Table of Contents

- [Quick Start](#quick-start)
- [CLI Commands](#cli-commands)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [Core Concepts](#core-concepts)
  - [File-Based Routing](#file-based-routing)
  - [Layouts & Components](#layouts--components)
  - [Template Engine](#template-engine)
  - [API Routes](#api-routes)
  - [Data Loaders](#data-loaders)
  - [Middleware](#middleware)
  - [Static Files](#static-files)
- [Development Mode](#development-mode)
  - [Hot Reload (HMR)](#hot-reload-hmr)
  - [DevTools Panel](#devtools-panel)
  - [Dev Endpoints](#dev-endpoints)
- [Building for Production](#building-for-production)
- [Configuration](#configuration)
- [Template Functions Reference](#template-functions-reference)
- [API Helpers Reference](#api-helpers-reference)
- [Technology Stack](#technology-stack)
- [How It Works Internally](#how-it-works-internally)

---

## Quick Start

### Install

```bash
go install github.com/nexgo/nexgo/cmd/nexgo@latest
```

### Create a New Project

```bash
nexgo create my-app
cd my-app
go mod tidy
nexgo dev
```

Open [http://localhost:3000](http://localhost:3000) -- your app is live with hot reload.

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `nexgo create <name>` | Scaffold a new NexGo project |
| `nexgo dev` | Start dev server with hot reload |
| `nexgo build` | Static site generation (SSG) |
| `nexgo start` | Start production server |
| `nexgo version` | Print version |
| `nexgo help` | Show help |

### Options

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--port` | `-p` | Override server port | `3000` |
| `--help` | `-h` | Show help | -- |

### Examples

```bash
# Create project
nexgo create blog-app

# Development with custom port
nexgo dev --port 8080

# Build static output
nexgo build

# Production server
nexgo start --port 4000
```

---

## Project Structure

Running `nexgo create my-app` generates this structure:

```
my-app/
├── main.go                  # Application entry point
├── nexgo.config.json        # Framework configuration
├── go.mod                   # Go module definition
│
├── pages/                   # File-based routes (auto-discovered)
│   ├── index.html           #   → /
│   ├── about.html           #   → /about
│   ├── blog/
│   │   ├── index.html       #   → /blog
│   │   └── [slug].html      #   → /blog/:slug (dynamic)
│   └── api/
│       └── hello.go         #   → /api/hello (API endpoint)
│
├── layouts/                 # Page layouts (wraps page content)
│   └── default.html         # Default layout for all pages
│
├── components/              # Reusable template partials
│
├── static/                  # Static assets (served at /static/)
│   ├── css/
│   │   └── global.css       # Global styles
│   └── js/
│
└── .nexgo/                  # Build output (generated)
    └── out/                 # Static site output
```

---

## Architecture

### High-Level Overview

```
                    ┌──────────────────────────────────────────────┐
                    │                  CLI (cmd/nexgo)              │
                    │         create | dev | build | start          │
                    └───────────┬──────────────┬───────────────────┘
                                │              │
                    ┌───────────▼──────┐  ┌────▼──────────────┐
                    │     Server       │  │     Builder        │
                    │  (pkg/server)    │  │  (pkg/builder)     │
                    │  HTTP + HMR      │  │  Static Generation │
                    └──┬───┬───┬───┬──┘  └──┬───┬────────────┘
                       │   │   │   │        │   │
          ┌────────────┘   │   │   └────┐   │   │
          ▼                ▼   ▼        ▼   ▼   ▼
   ┌──────────┐   ┌────────────────┐  ┌──────────┐
   │  Router   │   │   Renderer     │  │  Config   │
   │(pkg/router)│  │ (pkg/renderer) │  │(pkg/config)│
   │File-based │   │ Template Engine│  │  JSON     │
   │ matching  │   │ + Layouts      │  │  loader   │
   └──────────┘   └────────────────┘  └──────────┘
          │
   ┌──────┴──────────────┐
   ▼                     ▼
┌──────────┐     ┌──────────────┐
│Middleware │     │   Watcher     │
│(pkg/      │     │ (pkg/watcher) │
│middleware)│     │  File polling  │
└──────────┘     └──────────────┘
                         │
                  ┌──────▼──────┐
                  │   DevTools   │
                  │(pkg/devtools)│
                  │  Debug panel │
                  └─────────────┘
```

### Package Dependency Graph

```
cmd/nexgo/main.go
├── pkg/config           # Configuration loading
└── pkg/builder          # Static site generation
    ├── pkg/config
    ├── pkg/router       # Route discovery
    └── pkg/renderer     # Template rendering

pkg/server               # HTTP server
├── pkg/config
├── pkg/router
├── pkg/renderer
├── pkg/middleware        # Request middleware
├── pkg/watcher          # File change detection
└── pkg/devtools         # Dev panel UI

pkg/api                  # API route helpers (standalone)
```

### Source Files

| File | Lines | Purpose |
|------|-------|---------|
| `cmd/nexgo/main.go` | ~359 | CLI entry point, project scaffolding |
| `pkg/config/config.go` | ~98 | Config loading with defaults |
| `pkg/router/router.go` | ~324 | File-based route matching |
| `pkg/router/context.go` | ~31 | Request context helpers |
| `pkg/renderer/renderer.go` | ~430 | Template engine with layouts |
| `pkg/server/server.go` | ~378 | HTTP server, HMR, runtime JS |
| `pkg/middleware/middleware.go` | ~138 | Logger, CORS, Gzip, Security |
| `pkg/watcher/watcher.go` | ~154 | Polling-based file watcher |
| `pkg/builder/builder.go` | ~160 | Static site generation |
| `pkg/devtools/penel.go` | ~691 | DevTools panel UI |
| `pkg/api/helpers.go` | ~176 | JSON response, routing helpers |
| **Total** | **~3,130** | |

---

## Core Concepts

### File-Based Routing

Drop HTML files in `pages/` and NexGo creates routes automatically. No manual registration needed.

| File Path | URL Pattern | Example URL |
|-----------|-------------|-------------|
| `pages/index.html` | `/` | `/` |
| `pages/about.html` | `/about` | `/about` |
| `pages/blog/index.html` | `/blog` | `/blog` |
| `pages/blog/[slug].html` | `/blog/:slug` | `/blog/my-post` |
| `pages/docs/[...path].html` | `/docs/*` | `/docs/api/auth/login` |
| `pages/api/users.go` | `/api/users` | `/api/users` |

#### Dynamic Routes

Use `[param]` syntax for dynamic segments:

```
pages/
├── users/
│   ├── [id].html          → /users/123
│   └── [id]/
│       └── posts.html     → /users/123/posts
└── [...catchall].html     → /any/path/here
```

Access parameters in templates:

```html
<h1>User {{ .Params.id }}</h1>
```

#### Route Priority

Routes are matched by specificity (higher priority first):

1. Exact static routes (`/about`)
2. Dynamic segments (`/blog/[slug]`)
3. Catch-all routes (`/[...rest]`)

---

### Layouts & Components

#### Layouts

Layouts wrap page content. Place them in the `layouts/` directory:

```html
<!-- layouts/default.html -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ .Title }}</title>
  <link rel="stylesheet" href="/static/css/global.css">
  <script src="/_nexgo/runtime.js" defer></script>
</head>
<body>
  <nav>
    <a href="/">Home</a>
    <a href="/about">About</a>
  </nav>
  <main id="nexgo-root">{{ .Content }}</main>
  <footer>Built with NexGo</footer>
</body>
</html>
```

**Layout Detection** -- NexGo walks up the directory tree to find the nearest layout:

```
pages/blog/my-post.html
  → looks for layouts/blog.html
  → falls back to layouts/default.html
```

#### Components

Reusable template partials stored in `components/`:

```html
<!-- components/card.html -->
<div class="card">
  <h3>{{ .Title }}</h3>
  <p>{{ .Body }}</p>
</div>
```

Include in pages:

```html
{{ template "card" (dict "Title" "Hello" "Body" "World") }}
```

---

### Template Engine

NexGo uses Go's `html/template` package. Every page template receives a `PageData` object:

```go
type PageData struct {
    Title        string                 // Page title
    Description  string                 // Meta description
    Keywords     string                 // Meta keywords
    OGImage      string                 // Open Graph image
    Canonical    string                 // Canonical URL
    Path         string                 // Current URL path
    Params       map[string]string      // Route parameters
    Query        map[string]string      // Query string values
    Props        map[string]interface{} // Data from DataLoader
    NexGoVersion string                 // Framework version
    DevMode      bool                   // Development mode flag
    BuildID      string                 // Cache-busting ID
}
```

#### Usage in Templates

```html
<h1>{{ .Title }}</h1>
<p>You are at: {{ .Path }}</p>

{{ if .DevMode }}
  <p>Running in development mode</p>
{{ end }}

{{ range $key, $value := .Params }}
  <p>{{ $key }}: {{ $value }}</p>
{{ end }}
```

---

### API Routes

Create Go files in `pages/api/` to build REST endpoints. Handlers register themselves via `init()`:

```go
// pages/api/hello.go
package api

import (
    "net/http"
    "github.com/nexgo/nexgo/pkg/api"
    "github.com/nexgo/nexgo/pkg/router"
)

func init() {
    router.RegisterAPI("/api/hello", Hello)
}

func Hello(w http.ResponseWriter, r *http.Request) {
    api.JSON(w, map[string]interface{}{
        "message": "Hello from NexGo!",
        "method":  r.Method,
    })
}
```

#### Method-Based Routing

```go
func Users(w http.ResponseWriter, r *http.Request) {
    api.Route(w, r, api.Methods{
        "GET":  listUsers,
        "POST": createUser,
    })
}
```

#### Accessing Route Parameters

```go
func GetUser(w http.ResponseWriter, r *http.Request) {
    id := router.Param(r, "id")
    api.JSON(w, map[string]interface{}{"id": id})
}
```

---

### Data Loaders

Data loaders work like Next.js `getServerSideProps` -- fetch data on the server before rendering:

```go
// main.go
srv.RegisterDataLoader("/blog/[slug]", func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
    slug := params["slug"]
    post, err := db.GetPost(slug)
    if err != nil {
        return nil, err
    }
    return map[string]interface{}{
        "post": post,
    }, nil
})
```

Access in templates via `.Props`:

```html
<!-- pages/blog/[slug].html -->
<article>
  <h1>{{ .Props.post.Title }}</h1>
  <p>{{ .Props.post.Body }}</p>
</article>
```

---

### Middleware

NexGo includes built-in middleware applied in this order:

```
Request → Recover → Logger → SecurityHeaders → [Gzip] → Handler → Response
```

| Middleware | Description |
|-----------|-------------|
| `Recover` | Catches panics and returns 500 |
| `Logger` | Logs method, path, status, and duration |
| `SecurityHeaders` | Adds `X-Content-Type-Options`, `X-Frame-Options`, etc. |
| `Gzip` | Compresses responses (optional, enabled via config) |
| `CORS(origins...)` | Configurable CORS headers |
| `Cache(maxAge)` | Sets `Cache-Control` headers |

#### Chaining Middleware

```go
handler := middleware.Chain(
    middleware.Recover,
    middleware.Logger,
    middleware.SecurityHeaders,
)(yourHandler)
```

---

### Static Files

Place files in `static/` and they're served at `/static/`:

```
static/
├── css/global.css    → /static/css/global.css
├── js/app.js         → /static/js/app.js
└── images/logo.png   → /static/images/logo.png
```

Use the `asset` template function for cache-busted URLs:

```html
<link rel="stylesheet" href="{{ asset "/static/css/global.css" }}">
```

---

## Development Mode

### Hot Reload (HMR)

Run `nexgo dev` to start the dev server with **automatic hot reload**.

```bash
nexgo dev
# [NexGo] Dev server → http://localhost:3000
# [NexGo] Hot reload enabled. Press Ctrl+C to stop.
```

**How it works:**

1. A **file watcher** polls `pages/`, `layouts/`, and `components/` every **500ms**
2. When a file changes, the server **recompiles all templates** and **re-scans routes**
3. A reload message is broadcast to all connected browsers via **Server-Sent Events** (SSE)
4. The browser receives the event and **automatically reloads the page**

**Keyboard shortcuts:**

| Key | Action |
|-----|--------|
| `Ctrl+C` | Stop the dev server |

### DevTools Panel

In dev mode, visit [http://localhost:3000/_nexgo/devtools](http://localhost:3000/_nexgo/devtools) for a built-in debug panel:

| Tab | What it shows |
|-----|---------------|
| **Routes** | All discovered routes with type badges (PAGE / API) |
| **Requests** | Live request log with method, status, path, duration |
| **Logs** | Server logs, HMR events, errors |
| **Performance** | Total requests, avg response time, error rate, active routes |
| **Config** | Current server and build configuration |

The panel includes a **Reload** button and an **HMR connection indicator** (green = connected, red = reconnecting).

### Dev Endpoints

These endpoints are only available in development mode:

| Endpoint | Description |
|----------|-------------|
| `/_nexgo/devtools` | DevTools panel UI |
| `/_nexgo/hmr` | SSE stream for hot reload |
| `/_nexgo/routes` | JSON list of all registered routes |
| `/_nexgo/reload` | Manually trigger template reload |
| `/_nexgo/runtime.js` | Client-side runtime (router, HMR, prefetch) |

---

## Building for Production

### Static Site Generation (SSG)

```bash
nexgo build
```

This will:

1. Scan all page routes
2. Render each page to static HTML
3. Copy static assets
4. Output everything to `.nexgo/out/`

**Output structure:**

```
.nexgo/out/
├── index.html
├── about/
│   └── index.html
├── blog/
│   └── index.html
└── static/
    ├── css/global.css
    └── js/
```

### Production Server

```bash
nexgo start
# [NexGo] Production server → http://localhost:3000
```

Or with a custom port:

```bash
nexgo start --port 4000
```

Production mode disables:
- Hot reload / HMR
- DevTools panel
- Debug endpoints

Production mode enables:
- Gzip compression (if configured)
- Cache-Control headers
- Security headers

### Deploying

NexGo compiles to a **single binary** -- deploy anywhere Go runs:

```bash
# Build the binary
go build -o myapp ./main.go

# Deploy -- just copy the binary + pages/ + static/ + layouts/
scp myapp server:/opt/myapp/
scp -r pages/ layouts/ static/ nexgo.config.json server:/opt/myapp/

# Run
ssh server '/opt/myapp/myapp'
```

---

## Configuration

Create `nexgo.config.json` in your project root:

```json
{
  "projectName": "my-app",
  "port": 3000,
  "host": "localhost",
  "pagesDir": "pages",
  "staticDir": "static",
  "layoutsDir": "layouts",
  "componentsDir": "components",
  "outputDir": ".nexgo/out",
  "hotReload": true,
  "compression": true,
  "minify": true,
  "sourceMap": false,
  "baseURL": "",
  "trailingSlash": false,
  "defaultRenderMode": "ssr",
  "cacheControl": "public, max-age=31536000"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `projectName` | string | `"nexgo-app"` | Project identifier |
| `port` | int | `3000` | Server port |
| `host` | string | `"localhost"` | Server host |
| `pagesDir` | string | `"pages"` | Pages directory |
| `staticDir` | string | `"static"` | Static assets directory |
| `layoutsDir` | string | `"layouts"` | Layouts directory |
| `componentsDir` | string | `"components"` | Components directory |
| `outputDir` | string | `".nexgo/out"` | SSG build output |
| `hotReload` | bool | `true` | Enable HMR in dev mode |
| `compression` | bool | `true` | Enable gzip compression |
| `minify` | bool | `true` | Minify HTML output |
| `sourceMap` | bool | `false` | Generate source maps |
| `baseURL` | string | `""` | Base URL for links |
| `trailingSlash` | bool | `false` | Add trailing slashes to routes |
| `defaultRenderMode` | string | `"ssr"` | Default render mode: `ssr`, `ssg`, or `spa` |
| `cacheControl` | string | `"public, max-age=31536000"` | Cache-Control header value |

---

## Template Functions Reference

These functions are available in all templates (pages, layouts, components):

### String Functions

| Function | Usage | Description |
|----------|-------|-------------|
| `upper` | `{{ upper "hello" }}` | Convert to uppercase |
| `lower` | `{{ lower "HELLO" }}` | Convert to lowercase |
| `title` | `{{ title "hello world" }}` | Title case |
| `replace` | `{{ replace "hello" "l" "r" }}` | Replace substring |
| `trim` | `{{ trim "  hi  " }}` | Trim whitespace |
| `split` | `{{ split "a,b,c" "," }}` | Split into slice |
| `join` | `{{ join .Items "," }}` | Join slice into string |

### Math Functions

| Function | Usage | Description |
|----------|-------|-------------|
| `add` | `{{ add 1 2 }}` | Addition |
| `sub` | `{{ sub 5 3 }}` | Subtraction |
| `mul` | `{{ mul 3 4 }}` | Multiplication |
| `div` | `{{ div 10 2 }}` | Division |

### Utility Functions

| Function | Usage | Description |
|----------|-------|-------------|
| `json` | `{{ json .Data }}` | JSON encode a value |
| `safeHTML` | `{{ safeHTML "<b>bold</b>" }}` | Render as raw HTML |
| `dict` | `{{ dict "key" "val" }}` | Create a key-value map |
| `slice` | `{{ slice 1 2 3 }}` | Create a slice |
| `asset` | `{{ asset "/static/css/app.css" }}` | Cache-busted asset URL |
| `link` | `{{ link "/about" }}` | Link to a page |
| `times` | `{{ range times 5 }}...{{ end }}` | Iterate n times |
| `default` | `{{ default "fallback" .Value }}` | Default if nil/empty |

---

## API Helpers Reference

Import `github.com/nexgo/nexgo/pkg/api` in your API route handlers:

### Response Helpers

```go
api.JSON(w, data)                // 200 + JSON
api.JSONStatus(w, 201, data)     // Custom status + JSON
api.Error(w, 400, "bad input")   // JSON error response
```

### HTTP Status Shortcuts

```go
api.BadRequest(w, "invalid id")  // 400
api.NotFound(w, "not found")     // 404
api.Unauthorized(w)              // 401
api.Forbidden(w)                 // 403
api.InternalError(w, err)        // 500 (logs the error)
```

### Request Parsing

```go
var input CreateUserInput
if !api.Decode(w, r, &input) {
    return // Decode writes 400 on failure
}
// input is now populated
```

- Enforces **4MB** body size limit
- Disallows unknown JSON fields
- Returns `false` and writes error response on failure

### Method Guard

```go
if !api.MethodGuard(w, r, "GET", "POST") {
    return // Writes 405 with Allow header
}
```

### Method Router

```go
api.Route(w, r, api.Methods{
    "GET":    handleList,
    "POST":   handleCreate,
    "DELETE": handleDelete,
})
```

### Pagination

```go
page, limit := api.Paginate(r) // from ?page=2&limit=10
// Defaults: page=1, limit=20
// Limit range: 1-100
```

---

## Technology Stack

| Layer | Technology |
|-------|-----------|
| **Language** | Go 1.22+ |
| **Template Engine** | Go `html/template` (stdlib) |
| **HTTP Server** | Go `net/http` (stdlib) |
| **File Watching** | Custom polling-based (no external deps) |
| **Hot Reload** | Server-Sent Events (SSE) |
| **Client Runtime** | Vanilla JavaScript (embedded) |
| **Styling** | CSS (Google Fonts: Outfit + JetBrains Mono) |
| **Dependencies** | **Zero** -- stdlib only |

---

## How It Works Internally

### Development Server Flow

```
nexgo dev
    │
    ▼
config.Load(".")            ← reads nexgo.config.json
    │
    ▼
server.New(cfg)              ← creates Server instance
    │
    ▼
server.Start(ctx)
    ├── router.Scan()        ← walks pages/ dir, builds route table
    ├── renderer.LoadAll()   ← compiles layouts, components, pages
    ├── watcher.Start()      ← starts polling pages/, layouts/, components/
    └── http.ListenAndServe  ← binds to :3000
         │
         ├── GET /about
         │   ├── router.Match("/about")     → pages/about.html
         │   ├── renderer.RenderPage()      → execute template
         │   │   ├── dataLoader (if any)    → fetch server data
         │   │   ├── render page template   → HTML fragment
         │   │   └── wrap in layout         → full HTML document
         │   └── write response
         │
         ├── GET /api/hello
         │   ├── router.Match("/api/hello") → RouteTypeAPI
         │   └── execute handler(w, r)      → JSON response
         │
         └── File change detected
             ├── renderer.Reload()          → recompile templates
             ├── router.Scan()              → rediscover routes
             └── broadcastHMR()             → SSE → browser reloads
```

### Static Build Flow

```
nexgo build
    │
    ▼
config.Load(".")
    │
    ▼
builder.New(cfg)
    │
    ▼
builder.Build()
    ├── router.Scan()                ← discover all routes
    ├── renderer.LoadAll()           ← compile templates
    ├── for each page route:
    │   ├── create fake http.Request
    │   ├── renderer.RenderPage()    → HTML string
    │   └── write to .nexgo/out/     → /about → about/index.html
    ├── copyStatic()                 → static/ → .nexgo/out/static/
    └── return BuildResult           → pages built, files copied, duration
```

### Client-Side Runtime

The embedded JavaScript runtime (`/_nexgo/runtime.js`) provides:

| Feature | How It Works |
|---------|-------------|
| **Client Router** | Intercepts `<a>` clicks, uses `history.pushState`, fetches new content via `fetch()`, updates `#nexgo-root` |
| **HMR Client** | Connects to `/_nexgo/hmr` SSE stream, reloads page on `{"type":"reload"}` message |
| **Prefetch** | On link hover, injects `<link rel="prefetch">` for faster navigation |
| **Lazy Loading** | Uses `IntersectionObserver` for images with `data-src` attribute |

The runtime dispatches a `nexgo:ready` event on DOM load and is available globally as `window.NexGo`.

---

## Render Modes

| Mode | Description | When to Use |
|------|-------------|-------------|
| **SSR** (Server-Side Rendering) | Renders HTML on every request | Dynamic content, personalized pages |
| **SSG** (Static Site Generation) | Pre-renders HTML at build time | Blogs, docs, marketing pages |
| **SPA** (Single Page Application) | Client-side rendering only | Highly interactive UIs |

Set the default in `nexgo.config.json`:

```json
{
  "defaultRenderMode": "ssr"
}
```

---

## Example: Full Blog Setup

```
my-blog/
├── main.go
├── nexgo.config.json
├── pages/
│   ├── index.html
│   ├── blog/
│   │   ├── index.html         # Blog listing
│   │   └── [slug].html        # Individual post
│   └── api/
│       └── posts.go           # Posts API
├── layouts/
│   └── default.html
├── components/
│   └── post-card.html
└── static/
    └── css/global.css
```

**Register a data loader** in `main.go`:

```go
srv.RegisterDataLoader("/blog/[slug]", func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
    post := getPostBySlug(params["slug"])
    return map[string]interface{}{"post": post}, nil
})
```

**Template** (`pages/blog/[slug].html`):

```html
<article class="page-content">
  <h1>{{ .Props.post.Title }}</h1>
  <time>{{ .Props.post.Date }}</time>
  {{ safeHTML .Props.post.Content }}
</article>
```

**API route** (`pages/api/posts.go`):

```go
func init() { router.RegisterAPI("/api/posts", Posts) }

func Posts(w http.ResponseWriter, r *http.Request) {
    api.Route(w, r, api.Methods{
        "GET": func(w http.ResponseWriter, r *http.Request) {
            page, limit := api.Paginate(r)
            posts := getAllPosts(page, limit)
            api.JSON(w, posts)
        },
    })
}
```

---

## License

MIT

---

<p align="center">
  Built with Go. Inspired by Next.js. Powered by simplicity.
</p>
