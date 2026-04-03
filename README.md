# NexGo

```
  ███╗   ██╗███████╗██╗  ██╗ ██████╗  ██████╗
  ████╗  ██║██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║  ███╗██║   ██║
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║██║   ██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝
```

**The Go-Powered Web Framework Inspired by Next.js**

File-based routing | Server-side rendering | Hot reload | API routes | Single binary deploy

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Version](https://img.shields.io/badge/Version-1.0.5-7b2ff7?style=flat-square)](#)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](#)
[![Zero Dependencies](https://img.shields.io/badge/Zero-Dependencies-00d2ff?style=flat-square)](#)

---

## What is NexGo?

NexGo brings the **Next.js developer experience** to the **Go ecosystem**. Build full-stack web applications with file-based routing, server-side rendering, API routes, and hot reload -- all compiled into a **single binary** with **zero runtime dependencies**.

```
50,000+ requests/sec  |  ~10MB binary  |  No Node.js  |  No npm  |  No runtime
```

---

## Quick Start

### Install

```bash
go install github.com/salmanfaris22/nexgo/cmd/nexgo@v1.0.5
```

Add to PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
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
nexgo create blog-app
nexgo dev --port 8080
nexgo build
nexgo start --port 4000
```

---

## Project Structure

```
my-app/
├── main.go                  # Application entry point
├── nexgo.config.json        # Framework configuration
├── go.mod                   # Go module definition
│
├── pages/                   # File-based routes (auto-discovered)
│   ├── index.html           #   -> /
│   ├── about.html           #   -> /about
│   ├── blog/
│   │   ├── index.html       #   -> /blog
│   │   └── [slug].html      #   -> /blog/:slug (dynamic)
│   └── api/
│       └── hello.go         #   -> /api/hello (API endpoint)
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
│   ├── [id].html          -> /users/123
│   └── [id]/
│       └── posts.html     -> /users/123/posts
└── [...catchall].html     -> /any/path/here
```

Access parameters in templates:

```html
<h1>User {{ .Params.id }}</h1>
```

#### Route Priority

Routes are matched by specificity (higher priority first):

1. Exact static routes (`/about`) -- Priority 100
2. Dynamic segments (`/blog/[slug]`) -- Priority 50
3. Catch-all routes (`/[...rest]`) -- Priority 10

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
  -> looks for layouts/blog.html
  -> falls back to layouts/default.html
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
    Path         string                 // Current URL path
    Params       map[string]string      // Route parameters
    Query        map[string][]string    // Query string values
    Props        map[string]interface{} // Data from DataLoader
    State        map[string]interface{} // Global state
    NexGoVersion string                 // Framework version
    DevMode      bool                   // Development mode flag
    BuildID      string                 // Cache-busting ID
}
```

---

### API Routes

Create Go files in `pages/api/` to build REST endpoints:

```go
// pages/api/hello.go
package api

import (
    "net/http"
    "github.com/salmanfaris22/nexgo/pkg/api"
    "github.com/salmanfaris22/nexgo/pkg/router"
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
<article>
  <h1>{{ .Props.post.Title }}</h1>
  <p>{{ .Props.post.Body }}</p>
</article>
```

---

### State Management

NexGo provides server-side state injection and client-side hydration:

```go
// Server-side: register global state
srv.RegisterGlobalState("user", map[string]interface{}{"name": "John"})
```

In templates, render state for client hydration:

```html
{{ renderState .State }}
```

Client-side access:

```javascript
// Read state
const user = NexGo.getState('user');

// Update state
NexGo.setState('counter', 1);

// Subscribe to changes
NexGo.subscribe((state) => {
  console.log('State changed:', state);
});
```

---

### Middleware

NexGo includes built-in middleware applied in this order:

```
Request -> Recover -> Logger -> SecurityHeaders -> [Gzip] -> Handler -> Response
```

| Middleware | Description |
|-----------|-------------|
| `Recover` | Catches panics and returns 500 |
| `Logger` | Logs method, path, status, and duration |
| `SecurityHeaders` | Adds `X-Content-Type-Options`, `X-Frame-Options`, etc. |
| `Gzip` | Compresses responses (optional, enabled via config) |
| `CORS(origins...)` | Configurable CORS headers (pass origins to enable) |
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
├── css/global.css    -> /static/css/global.css
├── js/app.js         -> /static/js/app.js
└── images/logo.png   -> /static/images/logo.png
```

Use the `asset` template function for cache-busted URLs:

```html
<link rel="stylesheet" href="{{ asset "/static/css/global.css" }}">
```

---

## Development Mode

### Hot Reload (HMR)

Run `nexgo dev` to start the dev server with automatic hot reload.

```bash
nexgo dev
# [NexGo] Dev server -> http://localhost:3000
# [NexGo] Hot reload enabled. Press Ctrl+C to stop.
```

**How it works:**

1. A file watcher polls `pages/`, `layouts/`, and `components/` every 500ms
2. When a file changes, the server recompiles all templates and re-scans routes
3. A reload message is broadcast to all connected browsers via SSE
4. The browser fetches updated content and replaces it without full page reload

### DevTools Panel

In dev mode, visit `http://localhost:3000/_nexgo/devtools` for a built-in debug panel:

| Tab | What it shows |
|-----|---------------|
| **Routes** | All discovered routes with type badges (PAGE / API) |
| **Requests** | Live request log with method, status, path, duration |
| **Logs** | Server logs, HMR events, errors |
| **Colors** | Theme customization with CSS variable output |
| **SEO** | Meta tag generator with Google SERP preview |
| **Performance** | Total requests, avg response time, error rate |
| **Config** | Current server and build configuration |

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

### Production Server

```bash
nexgo start
nexgo start --port 4000
```

Production mode disables:
- Hot reload / HMR
- DevTools panel
- Debug endpoints (returns 404)

Production mode enables:
- Gzip compression (if configured)
- Cache-Control headers
- Security headers

### Deploying

NexGo compiles to a **single binary** -- deploy anywhere Go runs:

```bash
go build -o myapp ./main.go
scp myapp server:/opt/myapp/
scp -r pages/ layouts/ static/ nexgo.config.json server:/opt/myapp/
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
  "minify": true
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

---

## Template Functions Reference

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
| `renderState` | `{{ renderState .State }}` | Inject state hydration script |

---

## API Helpers Reference

Import `github.com/salmanfaris22/nexgo/pkg/api` in your API route handlers:

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
    return
}
```

- Enforces 4MB body size limit
- Disallows unknown JSON fields
- Returns false and writes 400 on failure

### Method Guard

```go
if !api.MethodGuard(w, r, "GET", "POST") {
    return
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

## Source Files

| File | Purpose |
|------|---------|
| `cmd/nexgo/main.go` | CLI entry point, project scaffolding |
| `pkg/config/config.go` | Config loading with defaults |
| `pkg/router/router.go` | File-based route matching |
| `pkg/router/context.go` | Request context helpers |
| `pkg/renderer/renderer.go` | Template engine with layouts |
| `pkg/server/server.go` | HTTP server, HMR, runtime JS |
| `pkg/middleware/middleware.go` | Logger, CORS, Gzip, Security |
| `pkg/watcher/watcher.go` | Polling-based file watcher |
| `pkg/builder/builder.go` | Static site generation |
| `pkg/devtools/panel.go` | DevTools panel UI |
| `pkg/api/helpers.go` | JSON response, routing helpers |

---

## License

MIT

---

Built with Go. Inspired by Next.js. Powered by simplicity.
