# NexGo v1.2.0 — Advanced Features Guide
done
Complete guide to NexGo's built-in features: State Management, SEO, Template Caching, Parallel Data Loading, Streaming SSR, Response Caching, Worker Pools, and Incremental Static Regeneration (ISR).

---

## Table of Contents

1. [State Management](#state-management)
2. [SEO (Meta, Sitemap, OpenGraph)](#seo)
3. [Template Caching](#template-caching)
4. [Parallel Data Loading](#parallel-data-loading)
5. [Streaming SSR](#streaming-ssr)
6. [Response Caching](#response-caching)
7. [Worker Pools](#worker-pools)
8. [Incremental Static Regeneration (ISR)](#isr)
9. [Full Todo App Example](#full-todo-app-example)

---

## State Management

Thread-safe global state shared across all handlers. No mutex boilerplate.

```go
import "github.com/salmanfaris22/nexgo/pkg/api"

// Store
api.SetState("counter", 0)
api.SetState("users", []User{...})

// Retrieve
count := api.GetState("counter").(int)
users := api.GetState("users").([]User)

// Delete
api.DeleteState("counter")

// Isolated state (optional)
session := api.NewState()
session.Set("user", "alice")
```

---

## SEO

Meta tags, OpenGraph, Twitter Cards, sitemap.xml, robots.txt, and JSON-LD structured data.

### Meta Tags

```go
import "github.com/salmanfaris22/nexgo/pkg/seo"

// In your page template or handler:
meta := seo.DefaultMeta("My Site", "A great website", "https://mysite.com")
meta.Keywords = []string{"go", "web", "framework"}
meta.OGImage = "https://mysite.com/og.png"
meta.TwitterCard = "summary_large_image"
meta.TwitterSite = "@mysite"

// Use in template:
// {{ .MetaTags }}  ← pass as template.HTML
```

### Article Meta

```go
meta := seo.ArticleMeta(
    "How to Build APIs in Go",
    "A guide to building fast APIs",
    "John Doe",
    "https://mysite.com/images/cover.jpg",
    "https://mysite.com/blog/go-apis",
    time.Now(),
)
```

### Sitemap

```go
entries := []seo.SitemapEntry{
    {Loc: "https://mysite.com/", ChangeFreq: "daily", Priority: 1.0},
    {Loc: "https://mysite.com/about", ChangeFreq: "monthly", Priority: 0.8},
    {Loc: "https://mysite.com/blog", ChangeFreq: "weekly", Priority: 0.9},
}
xml, _ := seo.RenderSitemap(entries)
// Write xml to /sitemap.xml response
```

### Robots.txt

```go
robots := seo.RobotsTxt(
    []string{"/"},                    // allow
    []string{"/admin", "/api"},       // disallow
    "https://mysite.com/sitemap.xml", // sitemap URL
)
// Write robots to /robots.txt response
```

### JSON-LD Structured Data

```go
data := map[string]interface{}{
    "@context": "https://schema.org",
    "@type": "Organization",
    "name": "My Company",
    "url": "https://mysite.com",
}
// In template: {{ .JSONLD }}
jsonld := seo.JSONLD(data)
```

---

## Template Caching

Templates are compiled once and cached for fast subsequent renders.

```go
import "github.com/salmanfaris22/nexgo/pkg/renderer"

r := renderer.New(cfg)
r.LoadAll()  // Compiles and caches all templates

// Check cache stats
info := r.CacheInfo()
// map[templates:5 layouts:1 components:3 loaded_at:2026-04-03...]
```

### Parallel Template Loading

For large projects, compile templates in parallel:

```go
r.LoadParallel()  // Uses worker pool to compile templates concurrently
```

---

## Parallel Data Loading

Load multiple data sources concurrently and merge results — like `Promise.all()` in JS.

```go
import (
    "github.com/salmanfaris22/nexgo/pkg/renderer"
    "net/http"
)

// Create parallel loader
loader := renderer.NewParallelLoader().
    Add("users", fetchUsers).
    Add("posts", fetchPosts).
    Add("stats", fetchStats)

// Execute all concurrently
props, err := loader.Execute(req, params)
// props = map[string]interface{}{
//     "users": [...], "posts": [...], "stats": {...}
// }

func fetchUsers(req *http.Request, params map[string]string) (map[string]interface{}, error) {
    users := db.Query("SELECT * FROM users")
    return map[string]interface{}{"users": users}, nil
}

func fetchPosts(req *http.Request, params map[string]string) (map[string]interface{}, error) {
    posts := db.Query("SELECT * FROM posts LIMIT 10")
    return map[string]interface{}{"posts": posts}, nil
}
```

---

## Streaming SSR

Send HTML progressively as it's ready — show the shell immediately, stream content later.

```go
import "github.com/salmanfaris22/nexgo/pkg/stream"

func handlePage(w http.ResponseWriter, r *http.Request) {
    sw := stream.StreamHTML(w)

    // Send head + layout immediately
    sw.WriteString(`<!DOCTYPE html><html><head><title>My Page</title></head><body>`)
    sw.WriteString(`<header>Navigation</header><main>`)
    sw.Flush()

    // Do slow work (DB query, API call)
    data := slowDatabaseQuery()

    // Stream content
    sw.WriteString(fmt.Sprintf("<h1>%s</h1>", data.Title))
    sw.WriteString(fmt.Sprintf("<p>%s</p>", data.Body))

    // Close HTML
    sw.WriteString(`</main></body></html>`)
}
```

### Server-Sent Events (SSE)

```go
func handleSSE(w http.ResponseWriter, r *http.Request) {
    sse := stream.NewSSE(w)
    defer sse.Flush()

    for {
        sse.Send("update", `{"count": 42}`)
        sse.Ping()
        time.Sleep(1 * time.Second)
    }
}
```

---

## Response Caching

Cache GET responses in memory with TTL. Automatic cache headers.

### Middleware

```go
import "github.com/salmanfaris22/nexgo/pkg/api"

// Cache a handler for 5 minutes
mux.HandleFunc("/api/posts", api.Cache(5*time.Minute)(handlePosts))

// Cache with custom TTL
mux.HandleFunc("/api/slow", api.Cache(30*time.Second)(handleSlow))
```

### Manual Cache

```go
// Set cache
api.CacheSet("/api/data", 200, headers, body)

// Get cache
code, headers, body, ok := api.CacheGet("/api/data")
if ok {
    w.WriteHeader(code)
    w.Write(body)
    return
}

// Delete specific key
api.CacheDelete("/api/data")

// Clear all
api.CacheClear()
```

### Cache Headers

Responses include `X-Cache: HIT` or `X-Cache: MISS` headers automatically.

---

## Worker Pools

Run tasks concurrently with controlled parallelism.

### Basic Pool

```go
import "github.com/salmanfaris22/nexgo/pkg/worker"

pool := worker.New(4)  // 4 workers
pool.Start()

pool.Submit(func() error {
    return processFile("data1.csv")
})
pool.Submit(func() error {
    return processFile("data2.csv")
})

errs := pool.Wait()  // blocks until all done
```

### Map (concurrent transform)

```go
results := worker.Map(4, items, func(item Item) Result {
    return process(item)
})
// results are in original order
```

### Map with Errors

```go
results, err := worker.MapErr(4, urls, func(url string) (Page, error) {
    return fetchPage(url)
})
```

### ForEach

```go
worker.ForEach(8, emails, func(email string) {
    sendEmail(email)
})
```

### Global Pool

```go
worker.Start()  // starts global pool with 4 workers
worker.Submit(func() error { return doWork() })
errs := worker.Wait()
```

---

## ISR (Incremental Static Regeneration)

Cache pages and rebuild them in the background after a configurable interval — like Next.js ISR.

### Basic ISR

```go
import "github.com/salmanfaris22/nexgo/pkg/isr"

// Create ISR with 60-second revalidation
cache := isr.New(60 * time.Second)

func handlePage(w http.ResponseWriter, r *http.Request) {
    cache.Serve(w, r, func() (int, http.Header, []byte, error) {
        // This runs only on first request or after 60s
        html := renderExpensivePage()
        return 200, nil, []byte(html), nil
    })
}
```

### How ISR Works

1. **First request** → generates page, caches it, returns with `X-ISR: MISS`
2. **Subsequent requests (within TTL)** → serves cached page with `X-ISR: STALE`
3. **After TTL expires** → serves stale content, regenerates in background
4. **Next request** → gets fresh cached content

### Manual Revalidation

```go
// Trigger background rebuild of a specific path
cache.Revalidate("/blog/post-1", func() (int, http.Header, []byte, error) {
    return 200, nil, []byte(renderPost()), nil
})

// Purge from cache
cache.Purge("/blog/post-1")
cache.PurgeAll()
```

### Global ISR

```go
isr.SetRevalidate(120 * time.Second)

isr.Serve(w, r, func() (int, http.Header, []byte, error) {
    return 200, nil, []byte(html), nil
})

isr.Revalidate("/page", generateFunc)
isr.Purge("/page")
```

---

## Full Todo App Example

Demonstrating all features together:

### `pages/api/todos.go`

```go
package api

import (
    "fmt"
    "net/http"
    "strconv"

    "github.com/salmanfaris22/nexgo/pkg/api"
    "github.com/salmanfaris22/nexgo/pkg/isr"
    "github.com/salmanfaris22/nexgo/pkg/router"
)

type Todo struct {
    ID   int
    Text string
}

func init() {
    router.RegisterAPI("/api/todos", handle)
    api.SetState("todos", []Todo{})
    api.SetState("nextID", 1)
}

func handle(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        todos := api.GetState("todos").([]Todo)
        if api.IsHTMX(r) {
            api.HTMXHTML(w, renderList(todos))
            return
        }
        api.JSON(w, map[string]interface{}{"todos": todos})

    case "POST":
        text := r.FormValue("text")
        if text == "" {
            api.BadRequest(w, "text is required")
            return
        }
        todos := api.GetState("todos").([]Todo)
        id := api.GetState("nextID").(int)
        todos = append(todos, Todo{ID: id, Text: text})
        api.SetState("todos", todos)
        api.SetState("nextID", id+1)

        if api.IsHTMX(r) {
            api.HTMXTrigger(w, "todo-added")
            api.HTMXHTML(w, renderList(todos))
            return
        }
        api.JSON(w, todos[len(todos)-1])

    case "DELETE":
        id, _ := strconv.Atoi(r.URL.Query().Get("id"))
        todos := api.GetState("todos").([]Todo)
        for i, t := range todos {
            if t.ID == id {
                todos = append(todos[:i], todos[i+1:]...)
                break
            }
        }
        api.SetState("todos", todos)

        if api.IsHTMX(r) {
            api.HTMXTrigger(w, "todo-deleted")
            api.HTMXHTML(w, renderList(todos))
            return
        }
        api.JSON(w, map[string]string{"status": "ok"})
    }
}

func renderList(todos []Todo) string {
    if len(todos) == 0 {
        return `<p class="empty-msg">No tasks yet!</p>`
    }
    html := ""
    for _, t := range todos {
        html += fmt.Sprintf(`<div class="todo-item">
            <span>%s</span>
            <button hx-delete="/api/todos?id=%d" hx-target="#todo-list" hx-swap="innerHTML">Delete</button>
        </div>`, api.Escape(t.Text), t.ID)
    }
    return html
}
```

### `pages/index.html`

```html
<div class="todo-app">
  <h1>Todo App</h1>

  <form hx-post="/api/todos" hx-target="#todo-list" hx-swap="innerHTML">
    <input type="text" name="text" placeholder="Add a task..." required />
    <button type="submit">Add</button>
  </form>

  <div id="todo-list" hx-get="/api/todos" hx-trigger="load, todo-added from:body, todo-deleted from:body">
    <p class="empty-msg">No tasks yet!</p>
  </div>
</div>

<script src="https://unpkg.com/htmx.org@1.9.12" defer></script>
```

---

## Quick Reference

| Package | Feature | Key Functions |
|---|---|---|
| `pkg/api` | State, HTMX, Cache, JSON | `SetState()`, `IsHTMX()`, `HTMXHTML()`, `Cache()`, `JSON()` |
| `pkg/seo` | Meta, Sitemap, Robots | `RenderMetaTags()`, `RenderSitemap()`, `RobotsTxt()`, `JSONLD()` |
| `pkg/cache` | Response Caching | `New()`, `Middleware()`, `Set()`, `Get()`, `Delete()` |
| `pkg/stream` | Streaming SSR, SSE | `StreamHTML()`, `NewSSE()`, `Send()`, `WriteString()` |
| `pkg/worker` | Worker Pools | `New()`, `Run()`, `Map()`, `MapErr()`, `ForEach()` |
| `pkg/isr` | Incremental Regeneration | `New()`, `Serve()`, `Revalidate()`, `Purge()` |
| `pkg/renderer` | Template Caching | `LoadAll()`, `LoadParallel()`, `CacheInfo()`, `NewParallelLoader()` |
