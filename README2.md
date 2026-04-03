# NexGo v1.1.0 — Built-in API Helpers & State Management

## New Features

NexGo now includes built-in helpers for writing shorter, cleaner API handlers with HTMX support and easy state management.

---

## HTMX Helpers

No more page reloads. Send HTML fragments from Go and let HTMX update the DOM.

### `api.IsHTMX(r)` — Check if request is from HTMX

```go
if api.IsHTMX(r) {
    // Return HTML fragment
    api.HTMXHTML(w, "<li>New item</li>")
} else {
    // Return JSON
    api.JSON(w, data)
}
```

### `api.HTMXHTML(w, html)` — Send HTML fragment response

```go
api.HTMXHTML(w, "<div>Updated content</div>")
```

### `api.HTMXTrigger(w, event)` — Fire client-side event after response

```go
api.HTMXTrigger(w, "todo-added")
```

### `api.HTMXHeader(w, key, value)` — Set any HTMX header

```go
api.HTMXHeader(w, "HX-Redirect", "/dashboard")
```

---

## State Management

Thread-safe global state shared across all handlers. No mutex boilerplate.

### `api.SetState(key, value)` — Store a value

```go
api.SetState("todos", []Todo{...})
api.SetState("counter", 42)
```

### `api.GetState(key)` — Retrieve a value

```go
todos := api.GetState("todos").([]Todo)
```

### `api.DeleteState(key)` — Remove a key

```go
api.DeleteState("todos")
```

### `api.NewState()` — Create isolated state (optional)

```go
sessionState := api.NewState()
sessionState.Set("user", "alice")
```

---

## HTML Helpers

### `api.Escape(s)` — Sanitize HTML output

```go
safe := api.Escape("<script>alert('xss')</script>")
// "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
```

### `api.HTML(w, html)` — Send raw HTML response

```go
api.HTML(w, "<h1>Hello</h1>")
```

---

## Complete Example: Todo App (Short Code)

### `pages/api/todos.go`

```go
package api

import (
	"github.com/salmanfaris22/nexgo/pkg/api"
	"github.com/salmanfaris22/nexgo/pkg/router"
	"net/http"
)

type Todo struct {
	ID   int
	Text string
}

func init() {
	router.RegisterAPI("/api/todos", handle)
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
		todos = append(todos, Todo{ID: len(todos) + 1, Text: text})
		api.SetState("todos", todos)

		if api.IsHTMX(r) {
			api.HTMXTrigger(w, "todo-added")
			api.HTMXHTML(w, renderList(todos))
			return
		}
		api.JSON(w, todos[len(todos)-1])

	case "DELETE":
		id := queryInt(r.URL.Query().Get("id"))
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
		html += `<div class="todo-item">
			<span>` + api.Escape(t.Text) + `</span>
			<button class="btn-delete" hx-delete="/api/todos?id=` + itoa(t.ID) + `"
				hx-target="#todo-list" hx-swap="innerHTML">Delete</button>
		</div>`
	}
	return html
}
```

### `pages/index.html`

```html
<div class="todo-app">
  <h1>Todo App</h1>

  <form class="todo-form" hx-post="/api/todos" hx-target="#todo-list" hx-swap="innerHTML">
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

| Function | Purpose |
|---|---|
| `api.IsHTMX(r)` | Check if HTMX request |
| `api.HTMXHTML(w, html)` | Send HTML fragment |
| `api.HTMXTrigger(w, event)` | Fire client event |
| `api.SetState(k, v)` | Store global state |
| `api.GetState(k)` | Get global state |
| `api.DeleteState(k)` | Remove from state |
| `api.Escape(s)` | Sanitize HTML |
| `api.HTML(w, html)` | Send raw HTML |
| `api.JSON(w, data)` | Send JSON response |
| `api.BadRequest(w, msg)` | Send 400 error |
| `api.NotFound(w, msg)` | Send 404 error |
