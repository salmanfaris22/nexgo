package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/salmanfaris22/nexgo/v2/pkg/builder"
	"github.com/salmanfaris22/nexgo/v2/pkg/config"
	"github.com/salmanfaris22/nexgo/v2/pkg/server"
)

const version = "2.2.0"

func main() {
	if len(os.Args) < 2 {

		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "dev":
		runDev(args)
	case "build":
		runBuild(args)
	case "start":
		runStart(args)
	case "create":
		runCreate(args)
	case "version", "-v", "--version":
		fmt.Printf("NexGo v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runDev(args []string) {
	rootDir := getRootDir(args)
	cfg, err := config.Load(rootDir)
	if err != nil {
		fatal("Loading config: %v", err)
	}
	cfg.DevMode = true
	if port := getFlag(args, "--port", "-p"); port != "" {
		if _, err := fmt.Sscan(port, &cfg.Port); err != nil {
			fatal("Invalid port number: %s", port)
		}
	}
	fmt.Printf("[NexGo] Dev server -> http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Println("[NexGo] Hot reload enabled. Press Ctrl+C to stop.")

	srv, err := server.New(cfg)
	if err != nil {
		fatal("Server init failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		fatal("Server failed: %v", err)
	}
}

func runBuild(args []string) {
	rootDir := getRootDir(args)
	cfg, err := config.Load(rootDir)
	if err != nil {
		fatal("Loading config: %v", err)
	}
	b := builder.New(cfg)
	result, err := b.Build()
	if err != nil {
		fatal("Build failed: %v", err)
	}
	if len(result.Errors) > 0 {
		fmt.Printf("[NexGo] Build completed with %d warning(s)\n", len(result.Errors))
	}
}

func runStart(args []string) {
	rootDir := getRootDir(args)

	cfg, err := config.Load(rootDir)
	if err != nil {
		fatal("Loading config: %v", err)
	}

	cfg.DevMode = false

	if port := getFlag(args, "--port", "-p"); port != "" {
		if _, err := fmt.Sscan(port, &cfg.Port); err != nil {
			fatal("Invalid port number: %s", port)
		}
	}

	fmt.Printf("[NexGo] Production server -> http://%s:%d\n", cfg.Host, cfg.Port)

	srv, err := server.New(cfg)
	if err != nil {
		fatal("Server init failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		fatal("Server failed: %v", err)
	}
}
func runCreate(args []string) {
	name := "my-nexgo-app"
	if len(args) > 0 {
		name = args[0]
	}
	dirs := []string{
		name,
		filepath.Join(name, "pages"),
		filepath.Join(name, "pages", "api"),
		filepath.Join(name, "pages", "blog"),
		filepath.Join(name, "components"),
		filepath.Join(name, "layouts"),
		filepath.Join(name, "islands"),
		filepath.Join(name, "static", "css"),
		filepath.Join(name, "static", "js"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fatal("Creating directory %s: %v", dir, err)
		}
	}
	files := scaffoldFiles(name)
	for path, content := range files {
		fullPath := filepath.Join(name, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			fatal("Creating dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			fatal("Writing %s: %v", path, err)
		}
	}
	fmt.Printf("\n  ✨ Created NexGo app: \033[36m%s\033[0m\n\n", name)
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    \033[90mcd %s\033[0m\n", name)
	fmt.Printf("    \033[90mgo mod tidy\033[0m\n")
	fmt.Printf("    \033[90mnexgo dev\033[0m\n\n")
}

func getRootDir(args []string) string {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			abs, err := filepath.Abs(arg)
			if err != nil {
				fatal("Invalid path: %s", arg)
			}
			return abs
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		fatal("Cannot determine working directory: %v", err)
	}
	return cwd
}

func getFlag(args []string, flags ...string) string {
	for i, arg := range args {
		for _, flag := range flags {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\033[31m[NexGo Error]\033[0m "+format+"\n", args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`
  ███╗   ██╗███████╗██╗  ██╗ ██████╗  ██████╗ 
  ████╗  ██║██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║  ███╗██║   ██║
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║██║   ██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝

  NexGo v%s — The Go-powered web framework

  USAGE:
    nexgo <command> [options]

  COMMANDS:
    create    Create a new NexGo project
    dev       Start development server with hot reload
    build     Build for production (static site generation)
    start     Start production server
    version   Show version number

  OPTIONS:
    --port, -p <port>    Override port  (default: 3000)
    --help, -h           Show this help

  EXAMPLES:
    nexgo create my-app
    nexgo dev
    nexgo dev --port 8080
    nexgo build
    nexgo start

`, version)
}

func scaffoldFiles(name string) map[string]string {
	return map[string]string{
		"nexgo.config.json": fmt.Sprintf(`{
  "projectName": "%s",
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
}`, name),
		"go.mod": fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire github.com/salmanfaris22/nexgo v1.0.0\n", name),
		"main.go": `package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/salmanfaris22/nexgo/v2/pkg/builder"
	"github.com/salmanfaris22/nexgo/v2/pkg/config"
	"github.com/salmanfaris22/nexgo/v2/pkg/renderer"
	"github.com/salmanfaris22/nexgo/v2/pkg/server"
)

// Define your data loaders once, use them for both dev server and static build
var loaders = map[string]renderer.DataLoader{
	// Example: load blog posts for /blog
	// "/blog": func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
	//     return map[string]interface{}{
	//         "posts": []map[string]interface{}{
	//             {"slug": "hello-world", "title": "Hello World", "excerpt": "My first post"},
	//         },
	//     }, nil
	// },
}

func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatal(err)
	}

	// Check if we're building or serving
	if len(os.Args) > 1 && os.Args[1] == "build" {
		runBuild(cfg)
		return
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Register data loaders for server
	for route, loader := range loaders {
		srv.RegisterDataLoader(route, loader)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func runBuild(cfg *config.NexGoConfig) {
	b := builder.New(cfg)

	// Register the same data loaders for static build
	for route, loader := range loaders {
		b.RegisterDataLoader(route, loader)
	}

	if _, err := b.Build(); err != nil {
		log.Fatal(err)
	}
}

// Ensure loaders variable is used (remove when you add real loaders)
var _ = loaders
`,
		"layouts/default.html": `<!DOCTYPE html>
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
  <nav class="nav">
    <a href="/" class="nav-logo">⚡ NexGo</a>
    <div class="nav-links">
      <a href="/">Home</a>
      <a href="/about">About</a>
      <a href="/blog">Blog</a>
    </div>
  </nav>
  <main id="nexgo-root">{{ .Content }}</main>
  <footer class="footer">
    <p>Built with <strong>NexGo</strong> ⚡</p>
  </footer>
</body>
</html>`,
		"pages/index.html": `<section class="hero">
  <h1>Welcome to <span class="gradient">NexGo</span></h1>
  <p class="hero-sub">The fastest Go-powered full-stack web framework.</p>
  <div class="cta">
    <a href="/about" class="btn btn-primary">Learn More →</a>
    <a href="/blog" class="btn btn-ghost">Read Blog</a>
  </div>
</section>
<section class="features">
  <div class="feature-card"><span class="icon">⚡</span><h3>Blazing Fast</h3><p>200k+ req/sec with cluster mode.</p></div>
  <div class="feature-card"><span class="icon">📁</span><h3>File-Based Routing</h3><p>pages/about.html → /about. Dynamic: pages/blog/[slug].html</p></div>
  <div class="feature-card"><span class="icon">🔥</span><h3>Hot Reload</h3><p>Instant browser refresh on template save.</p></div>
  <div class="feature-card"><span class="icon">🔌</span><h3>API Routes</h3><p>pages/api/*.go → REST API endpoints.</p></div>
  <div class="feature-card"><span class="icon">🌐</span><h3>SSR + SSG</h3><p>Server render or static generate — your choice.</p></div>
  <div class="feature-card"><span class="icon">📦</span><h3>Single Binary</h3><p>Deploy one file. No Node.js, no npm, no runtime.</p></div>
</section>
<section class="page-content" style="text-align:center;padding-top:0;">
  <h2>Islands Architecture Demo</h2>
  <p style="color:var(--muted);margin-bottom:2rem;">This counter is an <strong>island</strong> — only its JS is shipped. The rest of the page is static HTML with zero JavaScript.</p>
  {{ island "counter" (props "count" 0) "client:load" }}
</section>`,
		"pages/about.html": `<div class="page-content">
  <h1>About NexGo</h1>
  <p>NexGo is a Next.js-inspired framework built with Go for maximum performance.</p>
  <h2>Features</h2>
  <ul>
    <li>🚀 File-based routing (dynamic routes supported)</li>
    <li>🔄 Hot reload during development</li>
    <li>🌐 SSR and SSG support</li>
    <li>🔌 API routes as Go handlers</li>
    <li>🎨 Layouts and reusable components</li>
    <li>⚡ 200,000+ requests/second (cluster mode)</li>
    <li>📦 Single binary — zero runtime deps</li>
  </ul>
</div>`,
		"pages/blog/index.html": `<div class="page-content">
  <h1>Blog</h1>
  {{ if .Props.posts }}
    {{ range .Props.posts }}
    <article class="blog-card">
      <h2><a href="/blog/{{ .slug }}">{{ .title }}</a></h2>
      <p>{{ .excerpt }}</p>
    </article>
    {{ end }}
  {{ else }}
    <p class="muted">Register a data loader for /blog in main.go to load posts.</p>
  {{ end }}
</div>`,
		"pages/api/hello.go": `package api

import (
	"encoding/json"
	"net/http"
	"github.com/salmanfaris22/nexgo/v2/pkg/router"
)

func init() { router.RegisterAPI("/api/hello", Hello) }

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Hello from NexGo! 👋",
		"method":  r.Method,
	})
}`,
		// --- Example Islands ---
		"islands/counter.html": `<div class="island-counter">
  <span class="counter-value">{{ .count }}</span>
  <button class="btn btn-ghost counter-btn" data-action="decrement">-</button>
  <button class="btn btn-primary counter-btn" data-action="increment">+</button>
</div>`,
		"islands/counter.js": `// Island: counter
// This JS only loads when the island hydrates — not on every page.
export default function init(el, props) {
  let count = props.count || 0;
  const display = el.querySelector('.counter-value');
  const buttons = el.querySelectorAll('.counter-btn');

  function render() {
    display.textContent = count;
  }

  buttons.forEach(function(btn) {
    btn.addEventListener('click', function() {
      if (btn.dataset.action === 'increment') count++;
      else count--;
      render();
    });
  });
}
`,
		"static/css/global.css": `@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;700;900&family=JetBrains+Mono:wght@400;600&display=swap');
:root{--bg:#050505;--surface:#0f0f0f;--border:#1c1c1c;--text:#ebebeb;--muted:#666;--accent:#00d2ff;--accent2:#7b2ff7;--radius:12px;--font:'Outfit',system-ui,sans-serif;--mono:'JetBrains Mono',monospace}
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{font-family:var(--font);background:var(--bg);color:var(--text);line-height:1.65;min-height:100vh}
.nav{display:flex;align-items:center;justify-content:space-between;padding:1rem 2rem;border-bottom:1px solid var(--border);backdrop-filter:blur(12px);position:sticky;top:0;z-index:100;background:rgba(5,5,5,.85)}
.nav-logo{font-weight:800;color:var(--text);text-decoration:none;font-size:1.1rem}
.nav-links{display:flex;gap:2rem}.nav-links a{color:var(--muted);text-decoration:none;font-size:.9rem;transition:color .2s}.nav-links a:hover{color:var(--text)}
.hero{text-align:center;padding:8rem 2rem 5rem;max-width:820px;margin:0 auto}
.hero h1{font-size:clamp(3rem,9vw,6.5rem);font-weight:900;letter-spacing:-.04em;line-height:1;margin-bottom:1.5rem}
.gradient{background:linear-gradient(135deg,var(--accent) 0%,var(--accent2) 100%);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.hero-sub{font-size:1.2rem;color:var(--muted);margin-bottom:2.5rem;font-weight:300}
.cta{display:flex;gap:1rem;justify-content:center;flex-wrap:wrap}
.btn{display:inline-flex;align-items:center;padding:.8rem 2rem;border-radius:100px;font-weight:600;font-size:.95rem;text-decoration:none;transition:all .2s;cursor:pointer;border:none;font-family:var(--font)}
.btn-primary{background:linear-gradient(135deg,var(--accent),var(--accent2));color:#fff}
.btn-primary:hover{opacity:.85;transform:translateY(-2px);box-shadow:0 8px 30px rgba(0,210,255,.2)}
.btn-ghost{background:transparent;color:var(--text);border:1px solid var(--border)}.btn-ghost:hover{border-color:var(--muted);background:var(--surface)}
.features{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:1.25rem;max-width:1000px;margin:0 auto;padding:3rem 2rem 6rem}
.feature-card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:2rem;transition:border-color .2s,transform .2s}
.feature-card:hover{border-color:var(--accent);transform:translateY(-3px)}
.feature-card .icon{font-size:1.75rem;display:block;margin-bottom:1rem}
.feature-card h3{font-size:1.05rem;font-weight:700;margin-bottom:.5rem}
.feature-card p{color:var(--muted);font-size:.875rem;line-height:1.6}
.page-content{max-width:760px;margin:0 auto;padding:4rem 2rem}
.page-content h1{font-size:2.75rem;font-weight:900;letter-spacing:-.03em;margin-bottom:1.5rem}
.page-content h2{font-size:1.4rem;font-weight:700;margin:2.5rem 0 .75rem}
.page-content p,.page-content li{color:var(--muted)}
.page-content ul{padding-left:1.5rem}.page-content li{margin-bottom:.5rem}
.page-content code{font-family:var(--mono);background:var(--surface);border:1px solid var(--border);padding:.15em .4em;border-radius:4px;font-size:.85em}
.blog-card{border:1px solid var(--border);border-radius:var(--radius);padding:1.75rem;margin-bottom:1rem;transition:border-color .2s}
.blog-card:hover{border-color:var(--accent)}.blog-card h2{font-size:1.2rem;margin-bottom:.5rem}
.blog-card h2 a{color:var(--text);text-decoration:none}.blog-card p{color:var(--muted);font-size:.875rem}
.footer{text-align:center;padding:2.5rem 2rem;color:var(--muted);font-size:.85rem;border-top:1px solid var(--border)}
main{animation:fadeUp .25s ease}@keyframes fadeUp{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}
.island-counter{display:inline-flex;align-items:center;gap:1rem;background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:1.5rem 2rem}
.counter-value{font-size:2.5rem;font-weight:900;min-width:3ch;text-align:center;font-family:var(--mono);background:linear-gradient(135deg,var(--accent),var(--accent2));-webkit-background-clip:text;-webkit-text-fill-color:transparent}
.counter-btn{width:3rem;height:3rem;display:flex;align-items:center;justify-content:center;font-size:1.5rem;border-radius:50%;padding:0}`,
	}
}
