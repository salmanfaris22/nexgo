# NexGo v2.0.3 — Advanced Features Guide

All new features added in v1.2.0 to make NexGo a scalable, production-ready framework for large applications.

> **Zero external dependencies** — every feature below uses only Go's standard library.

---

## Table of Contents

1. [Environment Variables](#1-environment-variables)
2. [Session Management](#2-session-management)
3. [Authentication (JWT)](#3-authentication-jwt)
4. [CSRF Protection](#4-csrf-protection)
5. [Rate Limiting](#5-rate-limiting)
6. [Internationalization (i18n)](#6-internationalization-i18n)
7. [Image Optimization](#7-image-optimization)
8. [WebSocket Support](#8-websocket-support)
9. [Health Check Endpoints](#9-health-check-endpoints)
10. [Structured Logging](#10-structured-logging)
11. [Metrics & Observability](#11-metrics--observability)
12. [Database Integration](#12-database-integration)
13. [Migration System](#13-migration-system)
14. [Plugin System](#14-plugin-system)
15. [Server Actions](#15-server-actions)
16. [Custom Error Pages & Error Boundaries](#16-custom-error-pages--error-boundaries)
17. [Testing Utilities](#17-testing-utilities)
18. [Per-Route Middleware & CSP](#18-per-route-middleware--csp)
19. [Updated Configuration](#19-updated-configuration)
20. [Deployment Adapters](#20-deployment-adapters)
21. [Redis Cache Adapter](#21-redis-cache-adapter)
22. [Asset Pipeline](#22-asset-pipeline)
23. [ORM Layer](#23-orm-layer)
24. [Cluster Mode](#24-cluster-mode)

---

## 1. Environment Variables

**Package:** `pkg/env`

Load `.env`, `.env.local`, `.env.development`, `.env.production` files with priority ordering.

```go
import "github.com/salmanfaris22/nexgo/pkg/env"

// Load environment (priority: .env < .env.local < .env.production < .env.production.local)
store, err := env.Load(".", "production")

// Read values (OS env always wins)
dbURL := store.Get("DATABASE_URL")
port := store.GetDefault("PORT", "3000")
secret := store.MustGet("SECRET_KEY") // panics if not set

// Check mode
if store.IsProduction() {
    // enable caching
}

// Global convenience
env.LoadGlobal(".", "development")
env.GetEnv("API_KEY")
env.GetEnvDefault("PORT", "3000")
```

### `.env` File Format

```env
# Comments supported
DATABASE_URL="postgres://localhost/mydb"
PORT=3000
SECRET_KEY=my-secret

# Variable expansion
API_URL=https://api.example.com
WEBHOOK_URL=${API_URL}/webhooks

# export prefix supported
export NODE_ENV=production
```

### Priority Order (lowest to highest)

| File | When loaded |
|------|------------|
| `.env` | Always |
| `.env.local` | Always (gitignored) |
| `.env.development` | When mode = "development" |
| `.env.development.local` | When mode = "development" (gitignored) |
| `.env.production` | When mode = "production" |
| `.env.production.local` | When mode = "production" (gitignored) |
| **OS Environment** | **Always wins** |

---

## 2. Session Management

**Package:** `pkg/session`

Cookie-based sessions with pluggable storage backends.

```go
import "github.com/salmanfaris22/nexgo/pkg/session"

// Create session manager
store := session.NewMemoryStore() // or NewFileStore("./sessions")
cfg := session.DefaultConfig()
cfg.Secret = "your-secret-key"
cfg.MaxAge = 24 * time.Hour

mgr := session.NewManager(store, cfg)

// In your handler
func loginHandler(w http.ResponseWriter, r *http.Request) {
    sess, _ := mgr.Start(w, r)
    
    sess.Set("user_id", "123")
    sess.Set("role", "admin")
    
    mgr.Save(sess)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    sess, _ := mgr.Start(w, r)
    
    userID := sess.GetString("user_id")
    if userID == "" {
        http.Redirect(w, r, "/login", 303)
        return
    }
}

// Destroy session (logout)
func logoutHandler(w http.ResponseWriter, r *http.Request) {
    mgr.Destroy(w, r)
    http.Redirect(w, r, "/", 303)
}
```

### Session Middleware

```go
// Auto-inject session into all requests
srv.RegisterRoute("/", session.SessionMiddleware(mgr)(myHandler))

// Access session from context
func handler(w http.ResponseWriter, r *http.Request) {
    sess := session.FromRequest(r)
    sess.Set("views", sess.Get("views").(int) + 1)
}
```

### Flash Messages

```go
sess.Flash("success", "Account created!")

// On next request:
msg := sess.GetFlash("success") // returns "Account created!" then removes it
```

### Storage Backends

| Backend | Use Case |
|---------|----------|
| `MemoryStore` | Development, single instance |
| `FileStore` | Small apps, persistent sessions |
| Custom `Store` | Redis, database, etc. |

---

## 3. Authentication (JWT)

**Package:** `pkg/auth`

JWT token generation, verification, and middleware.

```go
import "github.com/salmanfaris22/nexgo/pkg/auth"

// Create auth manager
cfg := auth.DefaultConfig()
cfg.Secret = "your-jwt-secret"
cfg.TokenExpiry = 1 * time.Hour

a := auth.New(cfg)

// Generate token
token, _ := a.GenerateToken(auth.Claims{
    Sub:   "user-123",
    Email: "user@example.com",
    Name:  "John",
    Role:  "admin",
})

// Verify token
claims, err := a.VerifyToken(token)
// claims.Sub, claims.Email, claims.Role, claims.IsExpired()

// Set as HTTP-only cookie
a.SetTokenCookie(w, token)

// Clear on logout
a.ClearTokenCookie(w)
```

### Auth Middleware

```go
// Require authentication on all API routes
authMiddleware := a.Middleware()

srv.RegisterRoute("/api/profile", authMiddleware(profileHandler))

// Access claims in handler
func profileHandler(w http.ResponseWriter, r *http.Request) {
    claims := auth.GetClaims(r)
    api.JSON(w, map[string]string{
        "user":  claims.Sub,
        "email": claims.Email,
    })
}

// Optional auth (logged in or not)
srv.RegisterRoute("/", a.OptionalMiddleware()(homeHandler))

func homeHandler(w http.ResponseWriter, r *http.Request) {
    if auth.IsAuthenticated(r) {
        claims := auth.GetClaims(r)
        // Show personalized content
    }
}
```

### Role-Based Access

```go
// Only admins and editors
adminOnly := a.RequireRole("admin", "editor")
srv.RegisterRoute("/admin/dashboard", adminOnly(adminHandler))
```

### Password Hashing

```go
salt := "random-salt"
hash := auth.HashPassword("mypassword", salt)
valid := auth.CheckPassword("mypassword", salt, hash) // true
```

---

## 4. CSRF Protection

**Package:** `pkg/csrf`

Token-based CSRF protection for forms.

```go
import "github.com/salmanfaris22/nexgo/pkg/csrf"

cfg := csrf.DefaultConfig()
cfg.Secret = "your-csrf-secret"

protection := csrf.New(cfg)

// Apply as middleware
srv.RegisterRoute("/", protection.Middleware()(handler))
```

### In Templates

```html
<form method="POST" action="/submit">
    {{ csrfField .Request }}
    <input type="text" name="title">
    <button type="submit">Submit</button>
</form>
```

### In JavaScript

```javascript
// Read token from cookie
const token = document.cookie.match(/_nexgo_csrf=([^;]+)/)?.[1];

fetch('/api/submit', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': token,
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({ title: 'Hello' })
});
```

---

## 5. Rate Limiting

**Package:** `pkg/ratelimit`

Token bucket and sliding window rate limiting.

```go
import "github.com/salmanfaris22/nexgo/pkg/ratelimit"

// Token bucket: 60 requests/minute, burst of 120
limiter := ratelimit.PerMinute(60)

// Apply to all API routes
srv.RegisterRoute("/api/", limiter.Middleware()(apiHandler))

// Custom rate limiters
strict := ratelimit.PerSecond(10)    // 10 req/sec
relaxed := ratelimit.PerHour(1000)   // 1000 req/hour

// Custom window
custom := ratelimit.New(100, 5*time.Minute, 200)

// Custom key function (rate limit by API key instead of IP)
srv.RegisterRoute("/api/", limiter.MiddlewareWithKey(func(r *http.Request) string {
    return r.Header.Get("X-API-Key")
})(apiHandler))
```

### Sliding Window

```go
sw := ratelimit.NewSlidingWindow(100, time.Minute)
srv.RegisterRoute("/api/", sw.Middleware()(apiHandler))
```

### Response Headers

Rate-limited responses include:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
Retry-After: 60          (on 429)
```

---

## 6. Internationalization (i18n)

**Package:** `pkg/i18n`

Locale-based routing, translation files, locale detection, and RTL support.

```go
import "github.com/salmanfaris22/nexgo/pkg/i18n"

cfg := i18n.DefaultI18nConfig()
cfg.Locales = []i18n.Locale{
    {Code: "en", Name: "English", Direction: "ltr"},
    {Code: "ar", Name: "Arabic", Direction: "rtl"},
    {Code: "fr", Name: "French", Direction: "ltr"},
}
cfg.URLPrefix = true // /en/about, /ar/about

i := i18n.New(cfg)
i.LoadTranslations(".")

// Apply middleware
srv.RegisterRoute("/", i.Middleware()(handler))
```

### Translation Files

Create `locales/en.json`:

```json
{
    "welcome": "Welcome to NexGo",
    "nav.home": "Home",
    "nav.about": "About",
    "greeting": "Hello, {0}!"
}
```

Create `locales/ar.json`:

```json
{
    "welcome": "NexGo مرحبا بك في",
    "nav.home": "الرئيسية",
    "nav.about": "حول",
    "greeting": "!{0} مرحبا"
}
```

### In Templates

```html
<html lang="{{ .Locale }}" dir="{{ dir .Locale }}">
<head>
    <title>{{ t .Locale "welcome" }}</title>
</head>
<body>
    {{ if isRTL .Locale }}
        <link rel="stylesheet" href="/static/css/rtl.css">
    {{ end }}
    
    <nav>
        {{ range locales }}
            <a href="{{ localePath .Code "/" }}">{{ .Name }}</a>
        {{ end }}
    </nav>
    
    <h1>{{ t .Locale "welcome" }}</h1>
</body>
</html>
```

### Locale Detection Priority

1. URL prefix (`/ar/about`)
2. Cookie (`nexgo_locale=ar`)
3. `Accept-Language` header
4. Default locale

### Built-in RTL Locales

Arabic (ar), Hebrew (he), Persian (fa), Urdu (ur), Pashto (ps), Sindhi (sd), Yiddish (yi), Kurdish (ku)

---

## 7. Image Optimization

**Package:** `pkg/image`

Responsive images, lazy loading, and progressive enhancement.

```go
import "github.com/salmanfaris22/nexgo/pkg/image"

opt := image.New(image.DefaultConfig(), ".")

// Register image handler
srv.RegisterRoute("/_nexgo/image/", opt.Handler())
```

### In Templates

```html
<!-- Basic optimized image -->
{{ img "/static/img/hero.jpg" "Hero image" }}

<!-- With dimensions (enables srcset) -->
{{ imgSize "/static/img/hero.jpg" "Hero image" 1200 600 }}

<!-- Picture element with WebP fallback -->
{{ picture "/static/img/hero.jpg" "Hero image" 1200 600 }}

<!-- Low quality placeholder -->
<img src="{{ placeholder 1200 600 }}" data-src="/static/img/hero.jpg">
```

### Generated HTML

```html
<img src="/static/img/hero.jpg" 
     alt="Hero image" 
     width="1200" height="600"
     loading="lazy" decoding="async"
     srcset="/_nexgo/image/img/hero.jpg?w=320 320w,
             /_nexgo/image/img/hero.jpg?w=640 640w,
             /_nexgo/image/img/hero.jpg?w=768 768w,
             /_nexgo/image/img/hero.jpg?w=1024 1024w,
             /_nexgo/image/img/hero.jpg?w=1280 1280w"
     sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw">
```

### Features

| Feature | Description |
|---------|------------|
| `loading="lazy"` | Native browser lazy loading |
| `decoding="async"` | Non-blocking image decode |
| `srcset` | Responsive breakpoints (320, 640, 768, 1024, 1280, 1920) |
| `sizes` | Viewport-aware sizing |
| Blur placeholder | SVG-based LQIP (Low Quality Image Placeholder) |
| Cache headers | `max-age=31536000, immutable` |

---

## 8. WebSocket Support

**Package:** `pkg/websocket`

Native WebSocket implementation using HTTP hijacking — no external dependencies.

```go
import "github.com/salmanfaris22/nexgo/pkg/websocket"

hub := websocket.NewHub()

srv.RegisterRoute("/ws", func(w http.ResponseWriter, r *http.Request) {
    websocket.Upgrade(w, r, func(conn *websocket.Conn) {
        hub.Add(conn)
        defer hub.Remove(conn)
        
        for {
            msg, err := conn.ReadText()
            if err != nil {
                break
            }
            // Echo to all clients
            hub.Broadcast(msg)
        }
    })
})
```

### Room-Based Messaging

```go
srv.RegisterRoute("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
    room := r.URL.Query().Get("room")
    
    websocket.Upgrade(w, r, func(conn *websocket.Conn) {
        hub.Add(conn)
        hub.Join(conn, room)
        defer func() {
            hub.Leave(conn, room)
            hub.Remove(conn)
        }()
        
        for {
            msg, err := conn.ReadText()
            if err != nil {
                break
            }
            hub.BroadcastTo(room, msg)
        }
    })
})
```

### Client-Side

```javascript
const ws = new WebSocket('ws://localhost:3000/ws/chat?room=general');

ws.onmessage = (event) => {
    console.log('Message:', event.data);
};

ws.send(JSON.stringify({ type: 'chat', text: 'Hello!' }));
```

### Hub Methods

| Method | Description |
|--------|------------|
| `hub.Add(conn)` | Register a connection |
| `hub.Remove(conn)` | Unregister a connection |
| `hub.Join(conn, room)` | Add to room |
| `hub.Leave(conn, room)` | Remove from room |
| `hub.Broadcast(msg)` | Send to all |
| `hub.BroadcastTo(room, msg)` | Send to room |
| `hub.Count()` | Total connections |
| `hub.RoomCount(room)` | Connections in room |

---

## 9. Health Check Endpoints

**Package:** `pkg/health`

Kubernetes-compatible health, readiness, and liveness probes.

```go
import "github.com/salmanfaris22/nexgo/pkg/health"

h := health.New("1.2.0")

// Register dependency checks
h.AddCheck("database", func() error {
    // return db.Ping()
    return nil
})

h.AddCheck("cache", func() error {
    // return cache.Ping()
    return nil
})

// Register endpoints
h.RegisterEndpoints(mux) // adds /health, /ready, /live
```

### Endpoints

**GET /health** — Basic health status

```json
{
    "status": "up",
    "uptime": "2h30m15s",
    "timestamp": "2025-01-15T10:30:00Z",
    "version": "1.2.0",
    "checks": {
        "database": "up",
        "cache": "up"
    }
}
```

**GET /ready** — Readiness probe (includes system info)

```json
{
    "status": "up",
    "uptime": "2h30m15s",
    "checks": { "database": "up" },
    "system": {
        "go_version": "go1.22",
        "goroutines": 12,
        "cpus": 8,
        "mem_alloc_mb": 45,
        "mem_sys_mb": 72
    }
}
```

**GET /live** — Simple liveness probe (always 200)

```json
{
    "status": "ok",
    "uptime": "2h30m15s"
}
```

### Kubernetes Configuration

```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 3000
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 3000
  initialDelaySeconds: 5
  periodSeconds: 10
```

---

## 10. Structured Logging

**Package:** `pkg/logger`

Structured logging with levels, JSON output, colored console, and file rotation.

```go
import "github.com/salmanfaris22/nexgo/pkg/logger"

// Console logger (colored)
log := logger.New(logger.Config{
    Level:    "debug",
    Colorize: true,
})

// JSON logger (production)
log := logger.New(logger.Config{
    Level: "info",
    JSON:  true,
})

// With fields
log.With("user_id", "123").Info("Login successful")
log.WithFields(map[string]interface{}{
    "method": "POST",
    "path":   "/api/users",
    "status": 201,
}).Info("Request completed")

// Log levels
log.Debug("debugging info")
log.Info("server started on :3000")
log.Warn("deprecated endpoint called")
log.Error("failed to connect: %v", err)
log.Fatal("cannot start server") // exits process
```

### JSON Output

```json
{"level":"INFO","msg":"request","time":"2025-01-15T10:30:00Z","fields":{"method":"POST","path":"/api/users","status":201,"duration":"2.5ms"}}
```

### Request Logger Middleware

```go
log.RequestLogger() // logs method, path, status, duration, IP
```

### File Rotation

```go
writer, _ := logger.NewRotatingWriter("./logs", "app", 10*1024*1024) // 10MB per file
log := logger.New(logger.Config{
    Output: writer,
    JSON:   true,
})
```

### Global Logger

```go
logger.SetGlobal(log)
logger.Info("using global logger")
logger.Error("something failed: %v", err)
```

---

## 11. Metrics & Observability

**Package:** `pkg/metrics`

Prometheus-compatible metrics (counters, gauges, histograms).

```go
import "github.com/salmanfaris22/nexgo/pkg/metrics"

// Define metrics
requestCount := metrics.NewCounter("api_requests_total")
activeUsers := metrics.NewGauge("active_users")
latency := metrics.NewHistogram("request_latency_seconds", metrics.DefaultHTTPBuckets)

// Use in handlers
requestCount.Inc()
activeUsers.Set(42)

timer := metrics.NewTimer(latency)
// ... do work ...
timer.Stop()

// Expose endpoints
mux.HandleFunc("/metrics", metrics.Handler())       // Prometheus format
mux.HandleFunc("/metrics/json", metrics.JSONHandler()) // JSON format
```

### Built-in HTTP Metrics Middleware

```go
// Automatically tracks: requests total, duration, active requests, response size
srv.RegisterRoute("/", metrics.HTTPMiddleware()(handler))
```

### Prometheus Output

```
# TYPE http_requests_total counter
http_requests_total 1523
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.005"} 1200
http_request_duration_seconds_bucket{le="0.010"} 1400
http_request_duration_seconds_bucket{le="+Inf"} 1523
http_request_duration_seconds_sum 12.345
http_request_duration_seconds_count 1523
# TYPE http_active_requests gauge
http_active_requests 5
```

### Labeled Metrics

```go
counter := metrics.NewCounter("http_requests_total")
getCounter := counter.WithLabel("method", "GET")
postCounter := counter.WithLabel("method", "POST")

getCounter.Inc()
postCounter.Inc()
```

---

## 12. Database Integration

**Package:** `pkg/db`

Built-in JSON file database (zero deps) + SQL query builder for external drivers.

### JSON Database (Development / Small Apps)

```go
import "github.com/salmanfaris22/nexgo/pkg/db"

database, _ := db.NewJSONDB(".nexgo/data")

// Insert
users := database.Collection("users")
id, _ := users.Insert(map[string]interface{}{
    "name":  "John",
    "email": "john@example.com",
    "role":  "admin",
})

// Find by ID
user, _ := users.FindByID(id)

// Query
admins, _ := users.Find(db.Query{
    Where:   map[string]interface{}{"role": "admin"},
    OrderBy: "name",
    Limit:   10,
})

// Update
users.Update(id, map[string]interface{}{"name": "John Doe"})

// Delete
users.Delete(id)

// Count
count, _ := users.Count(db.Query{
    Where: map[string]interface{}{"role": "admin"},
})
```

### SQL Query Builder

```go
// SELECT
sql, args := db.Table("users").
    Where("role = $1", "admin").
    Where("active = $2", true).
    OrderBy("name", false).
    Limit(10).
    Offset(20).
    SelectSQL("id", "name", "email")
// => "SELECT id, name, email FROM users WHERE role = $1 AND active = $2 ORDER BY name ASC LIMIT 10 OFFSET 20"

// INSERT
sql, args := db.InsertSQL("users", map[string]interface{}{
    "name": "John", "email": "john@example.com",
})

// UPDATE
sql, args := db.UpdateSQL("users", "123", map[string]interface{}{
    "name": "John Doe",
})
```

---

## 13. Migration System

**Package:** `pkg/migrate`

Version-tracked database migrations with up/down support.

```go
import "github.com/salmanfaris22/nexgo/pkg/migrate"

m := migrate.New(".nexgo/migrations.json")

// Register migrations
m.Register("001", "Create users table", 
    func() error {
        // Run CREATE TABLE
        return nil
    },
    func() error {
        // Run DROP TABLE
        return nil
    },
)

m.Register("002", "Add email index",
    func() error { return nil },
    func() error { return nil },
)

// Run pending migrations
applied, err := m.Up()
// applied = ["001", "002"]

// Rollback last migration
rolled, err := m.Down()
// rolled = "002"

// Rollback to specific version
rolled, err := m.DownTo("001")

// Check status
for _, s := range m.Status() {
    fmt.Printf("%s %s applied=%v\n", s.Version, s.Description, s.Applied)
}

// Check pending
pending := m.Pending()
```

### SQL Migrations

```go
m.RegisterSQL(migrate.SQLMigration{
    Version:     "001",
    Description: "Create users table",
    UpSQL:       "CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT, email TEXT UNIQUE);",
    DownSQL:     "DROP TABLE users;",
}, func(sql string) error {
    _, err := db.Exec(sql)
    return err
})
```

### Helper Functions

```go
sql := migrate.CreateTableSQL("users", map[string]string{
    "id":    "SERIAL PRIMARY KEY",
    "name":  "TEXT NOT NULL",
    "email": "TEXT UNIQUE",
})

sql := migrate.DropTableSQL("users")
```

---

## 14. Plugin System

**Package:** `pkg/plugin`

Hook-based plugin architecture for extending NexGo.

```go
import "github.com/salmanfaris22/nexgo/pkg/plugin"

// Create plugin manager
pm := plugin.NewManager()

// Register a plugin
pm.Register(&MyPlugin{})

// Emit lifecycle hooks
pm.EmitHook(plugin.HookBeforeStart)
pm.EmitHook(plugin.HookAfterStart)
```

### Writing a Plugin

```go
type AnalyticsPlugin struct{}

func (p *AnalyticsPlugin) Name() string    { return "analytics" }
func (p *AnalyticsPlugin) Version() string { return "1.0.0" }

func (p *AnalyticsPlugin) Init(ctx *plugin.Context) error {
    // Add middleware
    ctx.AddMiddleware(func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // Track page view
            next(w, r)
        }
    })
    
    // Add routes
    ctx.AddRoute("/analytics", analyticsHandler)
    
    // Add template functions
    ctx.AddTemplateFunc("trackEvent", func(name string) string {
        return fmt.Sprintf(`<script>track("%s")</script>`, name)
    })
    
    // Register hooks
    ctx.On(plugin.HookBeforeRequest, "analytics", func() {
        // Before every request
    })
    
    ctx.On(plugin.HookOnError, "analytics", func() error {
        // On error
        return nil
    })
    
    return nil
}
```

### Available Hooks

| Hook | When |
|------|------|
| `HookBeforeStart` | Before server starts |
| `HookAfterStart` | After server starts |
| `HookBeforeStop` | Before server stops |
| `HookAfterStop` | After server stops |
| `HookBeforeRequest` | Before each request |
| `HookAfterRequest` | After each request |
| `HookBeforeRender` | Before page render |
| `HookAfterRender` | After page render |
| `HookBeforeBuild` | Before SSG build |
| `HookAfterBuild` | After SSG build |
| `HookOnRouteMatch` | When a route matches |
| `HookOnError` | On any error |

---

## 15. Server Actions

**Package:** `pkg/actions`

Form-based server mutations without manual API routes (like Next.js Server Actions).

```go
import "github.com/salmanfaris22/nexgo/pkg/actions"

registry := actions.NewRegistry()

// Register action
registry.Register("createPost", func(ctx context.Context, form actions.FormData) (*actions.Result, error) {
    // Validate
    v := actions.NewValidator()
    v.Required(form, "title", "Title is required")
    v.MinLength(form, "title", 3, "Title must be at least 3 characters")
    v.Required(form, "body", "Body is required")
    
    if !v.IsValid() {
        return v.Result(), nil
    }
    
    // Create post
    title := form.Get("title")
    body := form.Get("body")
    
    // ... save to database ...
    
    return actions.RedirectTo("/posts/" + id), nil
})

// Register handler
srv.RegisterRoute("/_nexgo/action/", registry.Handler())

// Or use as middleware (auto-intercepts forms with _action field)
srv.RegisterRoute("/", registry.Middleware()(pageHandler))
```

### In Templates

```html
<form method="POST" action="/_nexgo/action/createPost">
    {{ actionField "createPost" }}
    
    <input type="text" name="title" placeholder="Post title">
    <textarea name="body" placeholder="Write your post..."></textarea>
    <button type="submit">Create Post</button>
</form>
```

### Result Types

```go
actions.OK(data)              // Success with data
actions.Fail(errors)          // Validation errors
actions.RedirectTo("/path")   // Redirect
actions.WithMessage("Done!")  // Success message
```

### Validation Helpers

| Method | Description |
|--------|------------|
| `v.Required(form, field, msg)` | Field must not be empty |
| `v.MinLength(form, field, n, msg)` | Minimum string length |
| `v.MaxLength(form, field, n, msg)` | Maximum string length |
| `v.Email(form, field, msg)` | Basic email validation |
| `v.Custom(field, bool, msg)` | Custom validation rule |

---

## 16. Custom Error Pages & Error Boundaries

**Package:** `pkg/errorpage`

Convention-based error pages and per-directory error boundaries.

### File Conventions

```
pages/
  404.html          # Global 404 page
  500.html          # Global 500 page
  403.html          # Global 403 page
  blog/
    error.html      # Error boundary for /blog/*
    not-found.html  # Not-found page for /blog/*
  admin/
    error.html      # Error boundary for /admin/*
    not-found.html  # Not-found page for /admin/*
```

### Usage

```go
import "github.com/salmanfaris22/nexgo/pkg/errorpage"

ep := errorpage.New(errorpage.Config{
    PagesDir: "pages",
    DevMode:  true,
})
ep.LoadAll(templateFuncMap)

// Render errors
ep.RenderNotFound(w, r)                          // 404
ep.RenderError(w, r, 500, fmt.Errorf("oops"))    // 500

// Use as middleware (catches panics)
srv.RegisterRoute("/", ep.Middleware()(handler))
```

### Error Page Template Data

```html
<!-- pages/404.html -->
<div class="error-page">
    <h1>{{ .Status }} {{ .StatusText }}</h1>
    <p>{{ .Message }}</p>
    <p>Path: {{ .Path }}</p>
    {{ if .DevMode }}
        <pre>{{ .Message }}</pre>
    {{ end }}
    <a href="/">Go Home</a>
</div>
```

### Resolution Priority

1. Directory-specific `error.html` (walks up directory tree)
2. Status-specific page (`404.html`, `500.html`)
3. Built-in default error page

---

## 17. Testing Utilities

**Package:** `pkg/testing`

Test client, assertion helpers, and benchmarking for NexGo apps.

```go
import nextest "github.com/salmanfaris22/nexgo/pkg/testing"

// Create test client
client := nextest.NewClient(myHandler)

// Make requests
resp := client.GET("/api/users")
resp := client.POST("/api/users", map[string]interface{}{
    "name": "John", "email": "john@example.com",
})
resp := client.PUT("/api/users/1", updateData)
resp := client.DELETE("/api/users/1")
resp := client.PostForm("/login", url.Values{
    "email": {"john@example.com"}, "password": {"secret"},
})
```

### Response Helpers

```go
resp.StatusCode()       // 200
resp.BodyString()       // raw body
resp.BodyJSON(&result)  // decode JSON
resp.BodyMap()          // decode as map
resp.Header("X-Custom") // get header
resp.IsOK()            // status == 200
resp.IsRedirect()      // status 3xx
resp.IsJSON()          // Content-Type is JSON
resp.IsHTML()          // Content-Type is HTML
resp.ContainsString("Hello") // body contains
```

### Auth Testing

```go
client.SetAuth("jwt-token-here")
client.SetHeader("X-API-Key", "my-key")
client.SetCookie(&http.Cookie{Name: "session", Value: "abc"})
```

### Assertions

```go
func TestAPI(t *testing.T) {
    client := nextest.NewClient(handler)
    resp := client.GET("/api/health")
    
    if err := nextest.AssertStatus(resp, 200); err != nil {
        t.Fatal(err)
    }
    if err := nextest.AssertBodyContains(resp, "ok"); err != nil {
        t.Fatal(err)
    }
    if err := nextest.AssertJSON(resp, "status", "up"); err != nil {
        t.Fatal(err)
    }
}
```

### Route Testing

```go
rt := nextest.NewRouteTest(myHandlerFunc)
resp := rt.GET("/hello")
resp := rt.POST("/hello", body)
```

### Benchmarking

```go
result := nextest.BenchmarkHandler(handler, "GET", "/api/users", 10000)
fmt.Printf("Requests: %d, Avg: %dns, RPS: %.0f\n",
    result.Requests, result.AvgNs, result.ReqPerSec)
```

---

## 18. Per-Route Middleware & CSP

**Package:** `pkg/middleware` (updated)

### Per-Route Middleware

```go
import "github.com/salmanfaris22/nexgo/pkg/middleware"

// Apply middleware only to matching routes
authOnAPI := middleware.RouteMiddleware("/api/*", authMiddleware)

// Apply middleware group to a prefix
adminMW := middleware.RouteGroup("/admin/",
    authMiddleware,
    middleware.SecurityHeaders,
)

// Request timeout
srv.RegisterRoute("/api/slow", middleware.Timeout(5*time.Second)(handler))

// Request ID tracking
srv.RegisterRoute("/", middleware.RequestID(handler))
```

### Content Security Policy

```go
// Static CSP
csp := middleware.CSP("default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

// Nonce-based CSP (more secure)
csp := middleware.CSPWithNonce("default-src 'self'; script-src 'nonce-{nonce}'")
```

### Full Middleware Chain Example

```go
handler := middleware.Chain(
    middleware.Recover,
    middleware.RequestID,
    middleware.Logger,
    middleware.SecurityHeaders,
    middleware.CSP("default-src 'self'"),
    middleware.RouteGroup("/api/", rateLimiter.Middleware()),
    middleware.RouteGroup("/admin/", auth.Middleware()),
)(mainHandler)
```

---

## 19. Updated Configuration

All new features are configurable in `nexgo.config.json`:

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
    "defaultRenderMode": "ssr",

    "sessionSecret": "your-session-secret",
    "sessionMaxAge": 86400,
    "authSecret": "your-jwt-secret",
    "csrfSecret": "your-csrf-secret",

    "rateLimitPerMinute": 60,

    "defaultLocale": "en",
    "locales": ["en", "ar", "fr"],
    "translationDir": "locales",

    "imageOptimization": true,

    "databaseDriver": "json",
    "databaseDir": ".nexgo/data",

    "logLevel": "info",
    "logFormat": "text",
    "logDir": ".nexgo/logs",

    "metricsEnabled": true,
    "healthEnabled": true,
    "webSocketEnabled": true
}
```

---

## 20. Deployment Adapters

**Package:** `pkg/deploy`

One-command deployment config generation for 8 platforms.

```go
import "github.com/salmanfaris22/nexgo/pkg/deploy"

cfg := deploy.DefaultConfig("my-app")
cfg.Platform = deploy.PlatformDocker
cfg.Port = 3000
cfg.Memory = 256

deploy.Generate(".", cfg) // generates Dockerfile, docker-compose.yml, .dockerignore
```

### Supported Platforms

| Platform | Command | Files Generated |
|----------|---------|----------------|
| **Docker** | `deploy.PlatformDocker` | `Dockerfile`, `docker-compose.yml`, `.dockerignore` |
| **Vercel** | `deploy.PlatformVercel` | `vercel.json`, `api/index.go` |
| **Cloudflare Workers** | `deploy.PlatformCloudflare` | `wrangler.toml`, `build/worker.mjs` |
| **AWS Lambda** | `deploy.PlatformAWSLambda` | `template.yaml` (SAM), `Makefile`, `lambda_adapter.go` |
| **Netlify** | `deploy.PlatformNetlify` | `netlify.toml` |
| **Fly.io** | `deploy.PlatformFlyio` | `fly.toml`, `Procfile` |
| **Railway** | `deploy.PlatformRailway` | `railway.toml`, `nixpacks.toml` |
| **Kubernetes** | `deploy.PlatformKubernetes` | `k8s/deployment.yaml` (Deployment + Service + Ingress + HPA) |

### Docker (Multi-stage Build)

```bash
# Generate
deploy.Generate(".", deploy.Config{Platform: deploy.PlatformDocker, ProjectName: "my-app", Port: 3000})

# Run
docker build -t my-app .
docker run -p 3000:3000 my-app
```

Generated Dockerfile uses multi-stage build: Go build in Alpine, runtime in minimal Alpine with `ca-certificates`. Final image ~15MB.

### Kubernetes (Full Stack)

Generates a complete K8s manifest with:
- **Deployment** with liveness/readiness probes hitting `/live` and `/ready`
- **Service** (ClusterIP)
- **Ingress** with nginx rewrite
- **HorizontalPodAutoscaler** scaling 1-10 pods at 70% CPU

```bash
deploy.Generate(".", deploy.Config{
    Platform:     deploy.PlatformKubernetes,
    ProjectName:  "my-app",
    MinInstances: 2,
    MaxInstances: 20,
    Memory:       256,
})

kubectl apply -f k8s/deployment.yaml
```

### Build Scripts

```go
script := deploy.BuildScript(cfg)
// Docker:    "docker build -t my-app ."
// Lambda:    "GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-s -w' -tags lambda.norpc -o bootstrap ."
// Default:   "CGO_ENABLED=0 go build -ldflags='-s -w' -o my-app ."
```

---

## 21. Redis Cache Adapter

**Package:** `pkg/cache` (new file: `redis.go`)

Distributed caching via Redis using raw RESP protocol — zero external dependencies.

```go
import "github.com/salmanfaris22/nexgo/pkg/cache"

// Connect to Redis
rc, err := cache.NewRedis(cache.RedisConfig{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
    MaxIdle:  10,
    KeyPrefix: "myapp:",
})

// Basic operations
rc.Set("user:123", []byte(`{"name":"John"}`), 5*time.Minute)
data, _ := rc.Get("user:123")
rc.Delete("user:123")

// Atomic counter
count, _ := rc.Incr("page_views")

// Check existence & TTL
exists, _ := rc.Exists("user:123")
ttl, _ := rc.TTL("user:123")

// Bulk delete by prefix
rc.DeletePrefix("session:")

// Health check
err = rc.Ping()

// Close connections
rc.Close()
```

### Features

| Feature | Description |
|---------|------------|
| Raw RESP protocol | No CGO, no external Redis library |
| Connection pooling | Configurable idle connections |
| AUTH support | Password authentication |
| DB selection | SELECT 0-15 |
| Key prefixing | Auto-prefix all keys (e.g. `myapp:`) |
| SCAN-based delete | Delete keys by prefix without KEYS command |
| Atomic increment | INCR for counters |
| TTL management | SETEX, TTL queries |

### Cache Adapter (drop-in replacement for in-memory)

```go
// Wrap Redis as a response cache (replaces the default in-memory cache)
adapter := cache.NewRedisCacheAdapter(rc, 5*time.Minute)

adapter.SetCached("page:/about", 200, htmlBytes)
status, body, ok := adapter.GetCached("page:/about")
adapter.DeleteCached("page:/about")
adapter.ClearCached()
```

### Multi-Instance Setup

```
                    ┌──────────┐
                    │  Redis   │
                    │ (shared) │
                    └────┬─────┘
                         │
           ┌─────────────┼─────────────┐
           │             │             │
      ┌────┴────┐  ┌────┴────┐  ┌────┴────┐
      │ NexGo 1 │  │ NexGo 2 │  │ NexGo 3 │
      │ :3001   │  │ :3002   │  │ :3003   │
      └─────────┘  └─────────┘  └─────────┘
```

All instances share the same cache — invalidation on one propagates to all.

---

## 22. Asset Pipeline

**Package:** `pkg/asset`

CSS/JS minification, bundling, fingerprinting, and critical CSS extraction.

```go
import "github.com/salmanfaris22/nexgo/pkg/asset"

pipeline := asset.New(asset.DefaultConfig(), ".")
result, err := pipeline.Build()

// result.Manifest:
// "/static/css/global.css" -> "/assets/bundle.a1b2c3d4.css"
// "/static/js/app.js"     -> "/assets/bundle.e5f6g7h8.js"

fmt.Printf("CSS: %d files bundled, JS: %d files bundled, Duration: %s\n",
    result.CSSBundled, result.JSBundled, result.Duration)
```

### CSS Minification

```go
minified := asset.MinifyCSS(`
  /* Header styles */
  .header {
    background-color: #fff;
    padding: 1rem 2rem;
  }
  
  .header .nav {
    display: flex;
    gap: 1rem;
  }
`)
// Result: ".header{background-color:#fff;padding:1rem 2rem}.header .nav{display:flex;gap:1rem}"
```

### JS Minification

```go
minified := asset.MinifyJS(`
  // Initialize app
  function init() {
    const app = document.getElementById('app');
    /* Setup event listeners */
    app.addEventListener('click', handleClick);
  }
`)
// Comments removed, whitespace collapsed
```

### HTML Minification

```go
minified := asset.MinifyHTML(`
  <div>
    <p>  Hello   World  </p>
  </div>
`)
// Result: "<div> <p> Hello World </p> </div>"
```

### Content-Hash Fingerprinting

```
static/css/global.css  ->  .nexgo/assets/css/global.a1b2c3d4.css
static/js/app.js       ->  .nexgo/assets/js/app.e5f6g7h8.js
```

Immutable caching: browsers cache forever, new deploys get new hashes.

### Critical CSS

```go
critical := asset.CriticalCSS(fullCSS, []string{
    "body", ".header", ".hero", "h1",
})
// Extracts only rules matching above-the-fold selectors
```

### Inline Assets

```go
asset.InlineCSS("static/css/critical.css")  // <style>minified CSS</style>
asset.InlineJS("static/js/inline.js")       // <script>minified JS</script>
```

### Pipeline Config

```go
cfg := asset.Config{
    SourceDir:   "static",
    OutputDir:   ".nexgo/assets",
    Minify:      true,
    Fingerprint: true,
    BundleCSS:   true,  // combine all CSS into bundle.css
    BundleJS:    true,   // combine all JS into bundle.js
}
```

---

## 23. ORM Layer

**Package:** `pkg/orm`

Model definitions, relations, auto-migrations, query DSL, and JSON-based storage.

### Define Models

```go
import "github.com/salmanfaris22/nexgo/pkg/orm"

schema := orm.NewSchema()

// Define User model
schema.Define("User", []orm.Field{
    {Name: "name", Type: orm.TypeString, Required: true},
    {Name: "email", Type: orm.TypeString, Required: true, Unique: true},
    {Name: "role", Type: orm.TypeString, Default: "user"},
    {Name: "age", Type: orm.TypeInt},
    {Name: "active", Type: orm.TypeBool, Default: true},
})

// Define Post model with foreign key
schema.Define("Post", []orm.Field{
    {Name: "title", Type: orm.TypeString, Required: true},
    {Name: "body", Type: orm.TypeString},
    {Name: "user_id", Type: orm.TypeInt, ForeignKey: "users.id", Index: true},
    {Name: "published", Type: orm.TypeBool, Default: false},
})
```

### Auto-generated features

- `id` primary key (auto-increment) added automatically
- `created_at`, `updated_at` timestamps when `Timestamps: true`
- Unique constraint validation
- Required field validation
- Default values

### CRUD Operations (JSON backend)

```go
db, _ := orm.NewJSONORM(".nexgo/data", schema)
defer db.Close()

// Create
id, _ := db.Create("User", map[string]interface{}{
    "name":  "John",
    "email": "john@example.com",
    "role":  "admin",
})

// Read
user, _ := db.FindByID("User", id)

// Query
admins, _ := db.FindAll("User",
    map[string]interface{}{"role": "admin"}, // where
    "name",  // order by
    false,   // ascending
    10,      // limit
    0,       // offset
)

// Update
db.Update("User", id, map[string]interface{}{"name": "John Doe"})

// Delete
db.Delete("User", id)

// Count
count, _ := db.Count("User", map[string]interface{}{"role": "admin"})
```

### SQL Query DSL

```go
user := schema.Get("User")

// Fluent query builder
sql, args := user.Query().
    Select("id", "name", "email").
    Eq("role", "admin").
    Gt("age", 18).
    Like("name", "John%").
    OrderByDesc("created_at").
    Limit(10).
    Offset(20).
    ToSQL()
// => SELECT id, name, email FROM users WHERE role = $1 AND age > $2 AND name LIKE $3 ORDER BY created_at DESC LIMIT 10 OFFSET 20

// Joins
sql, args := user.Query().
    Join("posts", "users.id = posts.user_id").
    Eq("posts.published", true).
    ToSQL()

// Count query
sql, args := user.Query().Eq("active", true).CountSQL()

// Insert SQL
sql, args := user.InsertSQL(map[string]interface{}{"name": "John", "email": "john@example.com"})

// Update SQL
sql, args := user.UpdateSQL(1, map[string]interface{}{"name": "John Doe"})

// Delete SQL
sql, args := user.DeleteSQL(1)
```

### Auto-Migration SQL

```go
user := schema.Get("User")
fmt.Println(user.CreateTableSQL())
```

Output:

```sql
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  role TEXT DEFAULT user,
  age INTEGER,
  active BOOLEAN DEFAULT true,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
```

### Struct to Map helper

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

data := orm.StructToMap(User{Name: "John", Email: "john@example.com", Age: 25})
// map[string]interface{}{"name": "John", "email": "john@example.com", "age": 25}
```

---

## 24. Cluster Mode

**Package:** `pkg/cluster`

Multi-process server with graceful shutdown, zero-downtime restart, and load balancing.

### Basic Usage

```go
import "github.com/salmanfaris22/nexgo/pkg/cluster"

cfg := cluster.DefaultConfig()
cfg.Workers = 4 // 0 = auto-detect (NumCPU)

c := cluster.New(cfg, myHandler)
c.ListenAndServe(":3000")
// Starts 4 worker goroutines sharing one TCP listener
// Auto-handles SIGINT/SIGTERM for graceful shutdown
```

### Graceful Shutdown

```go
cfg := cluster.Config{
    Workers:         8,
    GracefulTimeout: 30 * time.Second, // wait up to 30s for in-flight requests
    ReadTimeout:     15 * time.Second,
    WriteTimeout:    30 * time.Second,
    IdleTimeout:     60 * time.Second,
}

c := cluster.New(cfg, handler)
go c.ListenAndServe(":3000")

<-c.Ready() // wait until all workers are up

// Later: graceful shutdown
c.Shutdown() // drains all active requests, then stops
```

### Zero-Downtime Restart

```go
// Hot-swap the handler without dropping connections
c.GracefulRestart(newHandler)
// 1. Starts new workers with newHandler
// 2. Drains old workers (waits for in-flight requests)
// 3. All new requests go to new workers
```

### Cluster Statistics

```go
stats := c.GetStats()
fmt.Printf("Workers: %d, Active: %d, Total: %d, Errors: %d, Uptime: %s\n",
    stats.WorkerCount,
    stats.ActiveRequests,
    stats.TotalRequests,
    stats.TotalErrors,
    stats.Uptime,
)
```

### Load Balancer

Built-in round-robin load balancer with health checking for multi-instance setups:

```go
lb := cluster.NewLoadBalancer([]string{
    "10.0.0.1:3000",
    "10.0.0.2:3000",
    "10.0.0.3:3000",
})

// Start periodic health checks
lb.StartHealthCheck("/health", 30*time.Second, 5*time.Second)

// Get next healthy backend
backend, err := lb.Next()

// Manual health management
lb.MarkDown("10.0.0.2:3000")
lb.MarkUp("10.0.0.2:3000")

// List healthy backends
healthy := lb.Healthy()
```

### Architecture

```
                    ┌─────────────────┐
                    │   TCP Listener   │
                    │    :3000         │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
         │Worker 0 │   │Worker 1 │   │Worker 2 │
         │(goroutine)  │(goroutine)  │(goroutine)
         └─────────┘   └─────────┘   └─────────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                    ┌────────┴────────┐
                    │  Shared Stats   │
                    │  (atomic.Int64) │
                    └─────────────────┘
```

All workers share the same TCP listener — the OS kernel distributes connections via `SO_REUSEPORT`-like behavior.

---

## Complete Feature List

| # | Feature | Package | Status |
|---|---------|---------|--------|
| 1 | Environment Variables (.env) | `pkg/env` | **NEW** |
| 2 | Session Management | `pkg/session` | **NEW** |
| 3 | JWT Authentication | `pkg/auth` | **NEW** |
| 4 | CSRF Protection | `pkg/csrf` | **NEW** |
| 5 | Rate Limiting (Token Bucket + Sliding Window) | `pkg/ratelimit` | **NEW** |
| 6 | i18n (Locale Routing, RTL, Translations) | `pkg/i18n` | **NEW** |
| 7 | Image Optimization (Lazy, Responsive, Srcset) | `pkg/image` | **NEW** |
| 8 | WebSocket Support (Rooms, Hub) | `pkg/websocket` | **NEW** |
| 9 | Health Check Endpoints | `pkg/health` | **NEW** |
| 10 | Structured Logging (JSON, Rotation) | `pkg/logger` | **NEW** |
| 11 | Prometheus Metrics | `pkg/metrics` | **NEW** |
| 12 | Database (JSON DB + SQL Builder) | `pkg/db` | **NEW** |
| 13 | Migration System | `pkg/migrate` | **NEW** |
| 14 | Plugin System (Hooks) | `pkg/plugin` | **NEW** |
| 15 | Server Actions (Form Mutations) | `pkg/actions` | **NEW** |
| 16 | Custom Error Pages & Boundaries | `pkg/errorpage` | **NEW** |
| 17 | Testing Utilities | `pkg/testing` | **NEW** |
| 18 | Per-Route Middleware | `pkg/middleware` | **UPDATED** |
| 19 | Content Security Policy (CSP + Nonce) | `pkg/middleware` | **UPDATED** |
| 20 | Request Timeout Middleware | `pkg/middleware` | **UPDATED** |
| 21 | Request ID Tracking | `pkg/middleware` | **UPDATED** |
| 22 | Extended Configuration | `pkg/config` | **UPDATED** |
| 23 | Deployment Adapters (8 platforms) | `pkg/deploy` | **NEW** |
| 24 | Redis Cache (raw RESP, zero deps) | `pkg/cache` | **NEW** |
| 25 | Asset Pipeline (Minify, Bundle, Fingerprint) | `pkg/asset` | **NEW** |
| 26 | ORM (Models, Relations, Query DSL) | `pkg/orm` | **NEW** |
| 27 | Cluster Mode (Multi-worker, Graceful) | `pkg/cluster` | **NEW** |

---

## Architecture

```
pkg/
├── actions/       Server actions (form mutations)
├── api/           API route helpers
├── asset/         CSS/JS minification, bundling, fingerprinting
├── auth/          JWT authentication
├── builder/       Static site generation
├── cache/         Response caching + Redis adapter
├── cluster/       Multi-process, graceful restart, load balancer
├── config/        Framework configuration
├── csrf/          CSRF protection
├── db/            Database (JSON + SQL query builder)
├── deploy/        Deployment adapters (Docker, Vercel, K8s, Lambda...)
├── devtools/      Development tools UI
├── env/           Environment variables (.env loading)
├── errorpage/     Custom error pages & boundaries
├── health/        Health check endpoints (/health, /ready, /live)
├── i18n/          Internationalization, RTL, translations
├── image/         Image optimization, responsive images
├── isr/           Incremental Static Regeneration
├── logger/        Structured logging (JSON, rotation)
├── metrics/       Prometheus metrics & observability
├── middleware/     HTTP middleware (CORS, CSP, rate limit, per-route)
├── migrate/       Database migrations (up/down)
├── orm/           ORM (models, relations, query DSL)
├── plugin/        Hook-based plugin system
├── ratelimit/     Rate limiting (token bucket + sliding window)
├── renderer/      Template rendering engine
├── router/        File-based routing
├── seo/           SEO helpers (meta, sitemap, robots.txt)
├── server/        HTTP server & HMR
├── session/       Session management (memory + file stores)
├── stream/        Streaming SSR & SSE
├── testing/       Test utilities & benchmarking
├── watcher/       File watcher (polling-based)
├── websocket/     WebSocket (native, no deps)
└── worker/        Worker pool for concurrency
```

> **35 packages. 50 Go files. 14,000+ lines. Zero external dependencies. One binary.**
