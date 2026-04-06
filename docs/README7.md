# NexGo v2.2.1 — SEO Killer Mode

```
  ███╗   ██╗███████╗██╗  ██╗ ██████╗  ██████╗
  ████╗  ██║██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
  ██╔██╗ ██║█████╗   ╚███╔╝ ██║  ███╗██║   ██║
  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║██║   ██║
  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝
```

**The most SEO-complete Go web framework. Period.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Version](https://img.shields.io/badge/Version-2.2.1-7b2ff7?style=flat-square)](#)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](#)
[![Zero Dependencies](https://img.shields.io/badge/Zero-Dependencies-00d2ff?style=flat-square)](#)

---

## What's New in v2.2.1

NexGo v2.2.1 introduces **SEO Killer Mode** — a comprehensive, built-in SEO toolkit that makes NexGo the most SEO-ready Go framework available. Every feature works out-of-the-box with zero configuration.

### SEO Feature Matrix: NexGo vs Others

| Feature | NexGo v2.2.1 | Next.js | Hugo | Astro |
|---------|:---:|:---:|:---:|:---:|
| Auto sitemap.xml | Built-in | Plugin | Built-in | Plugin |
| Auto robots.txt | Built-in | Manual | Built-in | Plugin |
| Auto canonical URLs | Built-in | Manual | Manual | Plugin |
| JSON-LD structured data | 10 types | Manual | Manual | Plugin |
| OpenGraph + Twitter Cards | Built-in | Manual | Manual | Plugin |
| Article OG tags | Built-in | Manual | Manual | Manual |
| Hreflang alternates | Built-in | Manual | Built-in | Plugin |
| RSS feed generation | Built-in | Manual | Built-in | Plugin |
| BreadcrumbList schema | Auto from URL | Manual | Manual | Manual |
| FAQ schema | Built-in | Manual | Manual | Manual |
| Product schema | Built-in | Manual | Manual | Manual |
| Core Web Vitals tracking | Built-in | Plugin | N/A | Plugin |
| SEO audit/validation | Built-in | Plugin | N/A | Plugin |
| Title templates | Built-in | Manual | Built-in | Manual |
| 301/302 redirects | Config-based | Config | Config | Config |
| Trailing slash normalization | Built-in | Config | Config | Config |
| Preload/prefetch hints | Template func | Manual | Manual | Manual |
| URL slug helpers | Built-in | Manual | Built-in | Manual |
| SEO HTTP headers | Auto | Manual | N/A | Manual |
| HSTS header | Built-in | Manual | N/A | Manual |
| Content-Language header | Auto | Manual | Manual | Manual |
| Default SEO scaffold | `nexgo create` | N/A | N/A | N/A |

---

## Table of Contents

1. [Quick Start](#1-quick-start)
2. [SEO Configuration](#2-seo-configuration)
3. [Auto Sitemap](#3-auto-sitemap)
4. [Auto Robots.txt](#4-auto-robotstxt)
5. [Meta Tags & Title Templates](#5-meta-tags--title-templates)
6. [OpenGraph & Twitter Cards](#6-opengraph--twitter-cards)
7. [Article OG Tags](#7-article-og-tags)
8. [JSON-LD Structured Data](#8-json-ld-structured-data)
9. [Breadcrumbs](#9-breadcrumbs)
10. [FAQ Schema](#10-faq-schema)
11. [Product & Review Schema](#11-product--review-schema)
12. [HowTo Schema](#12-howto-schema)
13. [RSS Feed Generation](#13-rss-feed-generation)
14. [Hreflang Alternates](#14-hreflang-alternates)
15. [Pagination SEO](#15-pagination-seo)
16. [Redirects](#16-redirects)
17. [Trailing Slash Normalization](#17-trailing-slash-normalization)
18. [Preload & Prefetch Hints](#18-preload--prefetch-hints)
19. [Core Web Vitals](#19-core-web-vitals)
20. [SEO HTTP Headers](#20-seo-http-headers)
21. [SEO Audit & Validation](#21-seo-audit--validation)
22. [URL Slug Helpers](#22-url-slug-helpers)
23. [Default SEO Scaffold](#23-default-seo-scaffold)
24. [Template Functions Reference](#24-template-functions-reference)
25. [Go API Reference](#25-go-api-reference)
26. [Best Practices Checklist](#26-best-practices-checklist)

---

## 1. Quick Start

```bash
go install github.com/salmanfaris22/nexgo/cmd/nexgo@latest
nexgo create my-app
cd my-app
go mod tidy
nexgo dev
```

Your app ships with:
- Auto-generated `/sitemap.xml` from your routes
- Auto-generated `/robots.txt` with sane defaults
- SEO meta tags in every layout (via template functions)
- Structured data (JSON-LD) for WebSite + BreadcrumbList
- RSS feed link in `<head>`
- Core Web Vitals tracking (opt-in)
- A `seo/` folder with customizable defaults and demo data

---

## 2. SEO Configuration

All SEO settings live in `nexgo.config.json` under the `"seo"` key:

```json
{
  "projectName": "my-app",
  "port": 3000,
  "seo": {
    "siteName": "My Awesome Site",
    "siteURL": "https://example.com",
    "titleTemplate": "%s | My Awesome Site",
    "defaultOGImage": "https://example.com/static/images/og-default.png",
    "twitterSite": "@mysite",
    "language": "en",
    "themeColor": "#00d2ff",
    "faviconURL": "/static/favicon.ico",
    "author": "John Doe",
    "autoSitemap": true,
    "autoRobotsTxt": true,
    "autoCanonical": true,
    "coreWebVitals": false,
    "vitalsEndpoint": "/api/vitals",
    "robotsAllow": ["/"],
    "robotsDisallow": ["/api/", "/_nexgo/", "/admin/"],
    "redirects": [
      {"from": "/old-page", "to": "/new-page", "status": 301},
      {"from": "/temp", "to": "/temporary-redirect", "status": 302}
    ]
  }
}
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `siteName` | string | `""` | Site name for og:site_name and structured data |
| `siteURL` | string | `""` | Full site URL (no trailing slash) |
| `titleTemplate` | string | `""` | Title template — `%s` is replaced with page title |
| `defaultOGImage` | string | `""` | Fallback OG image for pages without one |
| `twitterSite` | string | `""` | Twitter @handle for twitter:site |
| `language` | string | `"en"` | Content-Language header value |
| `themeColor` | string | `""` | Browser theme-color meta tag |
| `faviconURL` | string | `""` | Favicon path |
| `author` | string | `""` | Default author for meta tags |
| `autoSitemap` | bool | `true` | Auto-serve /sitemap.xml |
| `autoRobotsTxt` | bool | `true` | Auto-serve /robots.txt |
| `autoCanonical` | bool | `true` | Auto-set canonical URL from route |
| `coreWebVitals` | bool | `false` | Inject CWV tracking script |
| `vitalsEndpoint` | string | `"/api/vitals"` | Endpoint for CWV beacon reports |
| `robotsAllow` | []string | `["/"]` | Robots.txt Allow directives |
| `robotsDisallow` | []string | `["/api/","/_nexgo/"]` | Robots.txt Disallow directives |
| `redirects` | []object | `[]` | 301/302 redirect rules |

---

## 3. Auto Sitemap

NexGo automatically generates `/sitemap.xml` from your route table. No configuration needed.

**What's included:**
- All static page routes (e.g. `/`, `/about`, `/blog`)
- Proper `<lastmod>`, `<changefreq>`, `<priority>` tags
- Homepage gets `priority: 1.0`, other pages get `0.5`
- API routes and dynamic routes (with `[params]`) are excluded

**Example output:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <lastmod>2026-04-06</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1</priority>
  </url>
  <url>
    <loc>https://example.com/about</loc>
    <lastmod>2026-04-06</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://example.com/blog</loc>
    <lastmod>2026-04-06</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>
```

### Custom Sitemap (Advanced)

For dynamic routes, build the sitemap manually:

```go
srv.RegisterRoute("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
    entries := seo.AutoSitemap("https://example.com", staticRoutes, 0.5)

    // Add dynamic blog posts
    posts := getBlogPosts()
    for _, post := range posts {
        entries = append(entries, seo.SitemapEntry{
            Loc:        "https://example.com/blog/" + post.Slug,
            LastMod:    post.UpdatedAt.Format("2006-01-02"),
            ChangeFreq: "monthly",
            Priority:   0.7,
        })
    }

    data, _ := seo.RenderSitemap(entries)
    w.Header().Set("Content-Type", "application/xml")
    w.Write(data)
})
```

### Sitemap Index (50k+ URLs)

For large sites with more than 50,000 URLs:

```go
index, _ := seo.RenderSitemapIndex([]seo.SitemapIndexEntry{
    {Loc: "https://example.com/sitemap-pages.xml", LastMod: "2026-04-06"},
    {Loc: "https://example.com/sitemap-blog.xml", LastMod: "2026-04-06"},
    {Loc: "https://example.com/sitemap-products.xml", LastMod: "2026-04-06"},
})
```

### Hreflang in Sitemap

```go
entries := []seo.SitemapEntry{
    {
        Loc: "https://example.com/about",
        Alternates: []seo.SitemapAlternate{
            {Rel: "alternate", HrefLang: "en", Href: "https://example.com/about"},
            {Rel: "alternate", HrefLang: "fr", Href: "https://example.com/fr/about"},
            {Rel: "alternate", HrefLang: "x-default", Href: "https://example.com/about"},
        },
    },
}
```

---

## 4. Auto Robots.txt

NexGo auto-serves `/robots.txt` based on your config:

```
User-agent: *
Allow: /
Disallow: /api/
Disallow: /_nexgo/

Sitemap: https://example.com/sitemap.xml
```

### Advanced Robots.txt

For multiple user-agent blocks:

```go
content := seo.RobotsTxtAdvanced([]seo.RobotsRule{
    {
        UserAgent: "*",
        Allow:     []string{"/"},
        Disallow:  []string{"/api/", "/admin/"},
    },
    {
        UserAgent:  "Googlebot",
        Allow:      []string{"/"},
        CrawlDelay: 1,
    },
    {
        UserAgent: "GPTBot",
        Disallow:  []string{"/"},
    },
}, []string{"https://example.com/sitemap.xml"})
```

---

## 5. Meta Tags & Title Templates

### In Templates

The simplest way — use `seoTags` in your layout:

```html
<!-- layouts/default.html -->
<head>
  <meta charset="UTF-8">
  {{ seoTags (seoMeta .Title .Description .Canonical) }}
</head>
```

This generates:
- `<meta name="viewport">` (always)
- `<title>` with title template applied (e.g. "About | My Site")
- `<meta name="description">`
- `<meta property="og:title">`, `og:description`, `og:url`, `og:image`, `og:type`, `og:site_name`, `og:locale`
- `<meta name="twitter:card">`, `twitter:title`, `twitter:description`, `twitter:image`, `twitter:site`
- `<link rel="canonical">`
- `<meta name="theme-color">`
- `<link rel="icon">`

### Title Template

Set in config:

```json
"titleTemplate": "%s | My Site"
```

- Page title "About" renders as `<title>About | My Site</title>`
- The site name alone is never templated (avoids "My Site | My Site")

### In Go Code

```go
meta := seo.Meta{
    Title:       "About Us",
    Description: "Learn about our company and mission.",
    Canonical:   "https://example.com/about",
    OGImage:     "https://example.com/images/about-og.png",
    OGType:      "website",
    Keywords:    []string{"about", "company", "mission"},
}

html := seo.RenderMetaTags(meta, seo.SiteConfig{
    SiteName:      "My Site",
    TitleTemplate: "%s | My Site",
    TwitterSite:   "@mysite",
    ThemeColor:    "#00d2ff",
})
```

---

## 6. OpenGraph & Twitter Cards

Every field is supported:

```go
meta := seo.Meta{
    Title:       "My Article",
    Description: "A great article about Go.",
    Canonical:   "https://example.com/blog/my-article",
    OGImage:     "https://example.com/images/article.png",
    OGType:      "article",
    OGSiteName:  "My Site",
    Locale:      "en_US",
    TwitterCard: "summary_large_image",
    TwitterSite: "@mysite",
}
```

**Supported OG types:** `website`, `article`, `product`, `profile`, `book`, `music.song`, `video.movie`

**Supported Twitter cards:** `summary`, `summary_large_image`, `app`, `player`

---

## 7. Article OG Tags

For blog posts and articles, use the full article meta:

```go
meta := seo.ArticleMeta(
    "How to Build Fast Web Apps",        // title
    "A guide to building performant...",  // description
    "John Doe",                           // author
    "https://example.com/img/article.png", // image
    "https://example.com/blog/fast-apps", // canonical
    time.Now(),                           // publish date
)

// Add extra article tags
meta.ArticleModifiedTime = time.Now().Format(time.RFC3339)
meta.ArticleSection = "Technology"
meta.ArticleTags = []string{"Go", "Web", "Performance"}
```

Generated tags:

```html
<meta property="og:type" content="article">
<meta property="article:published_time" content="2026-04-06T12:00:00Z">
<meta property="article:modified_time" content="2026-04-06T14:30:00Z">
<meta property="article:author" content="John Doe">
<meta property="article:section" content="Technology">
<meta property="article:tag" content="Go">
<meta property="article:tag" content="Web">
<meta property="article:tag" content="Performance">
```

---

## 8. JSON-LD Structured Data

NexGo provides **10 pre-built structured data types** plus a generic builder.

### WebSite Schema (with Sitelinks Searchbox)

```html
<!-- In template -->
{{ websiteSchema "My Site" "https://example.com" "https://example.com/search?q=" }}
```

```go
// In Go
seo.WebSiteSchema("My Site", "https://example.com", "https://example.com/search?q=")
```

### Organization Schema

```html
{{ orgSchema "My Company" "https://example.com" "https://example.com/logo.png" }}
```

```go
seo.OrganizationSchema("My Company", "https://example.com", "https://example.com/logo.png",
    []string{"https://twitter.com/myco", "https://github.com/myco"},
)
```

### LocalBusiness Schema

```go
seo.LocalBusinessSchema(
    "Joe's Coffee", "https://joescoffee.com", "+1-555-0123",
    "123 Main St", "Springfield", "IL", "62701", "US",
)
```

### Article Schema

```go
seo.ArticleSchema(seo.ArticleSchemaInput{
    Title:         "How to Build Fast Web Apps",
    Description:   "A comprehensive guide...",
    Author:        "John Doe",
    Image:         "https://example.com/img/article.png",
    URL:           "https://example.com/blog/fast-apps",
    Published:     time.Now(),
    Modified:      time.Now(),
    Section:       "Technology",
    Tags:          []string{"Go", "Performance"},
    Publisher:     "My Site",
    PublisherLogo: "https://example.com/logo.png",
})
```

### Generic JSON-LD

```html
{{ jsonld (dict "@context" "https://schema.org" "@type" "Event" "name" "Go Conference" "startDate" "2026-06-15") }}
```

---

## 9. Breadcrumbs

Auto-generated from the URL path:

```html
<!-- In template — generates BreadcrumbList JSON-LD -->
{{ breadcrumbs .Path }}
```

For `/blog/seo-best-practices`, this generates:

```json
{
  "@context": "https://schema.org",
  "@type": "BreadcrumbList",
  "itemListElement": [
    {"@type": "ListItem", "position": 1, "name": "Home", "item": "https://example.com/"},
    {"@type": "ListItem", "position": 2, "name": "Blog", "item": "https://example.com/blog"},
    {"@type": "ListItem", "position": 3, "name": "Seo Best Practices", "item": "https://example.com/blog/seo-best-practices"}
  ]
}
```

### Custom Breadcrumbs

```go
items := []seo.BreadcrumbItem{
    {Name: "Home", URL: "https://example.com/"},
    {Name: "Products", URL: "https://example.com/products"},
    {Name: "Widget Pro", URL: "https://example.com/products/widget-pro"},
}
html := seo.BreadcrumbSchema(items)
```

---

## 10. FAQ Schema

```html
<!-- In template -->
{{ faqSchema "What is NexGo?" "A Go web framework." "Is it fast?" "200k+ req/sec." }}
```

```go
// In Go
seo.FAQSchema([]seo.FAQItem{
    {Question: "What is NexGo?", Answer: "A Go web framework."},
    {Question: "Is it fast?", Answer: "200k+ req/sec."},
    {Question: "Do I need Node.js?", Answer: "No."},
})
```

---

## 11. Product & Review Schema

### Product

```go
seo.ProductSchema(seo.ProductSchemaInput{
    Name:         "Widget Pro",
    Description:  "The best widget money can buy.",
    Image:        "https://example.com/img/widget.png",
    Brand:        "WidgetCo",
    SKU:          "WP-001",
    Price:        "29.99",
    Currency:     "USD",
    Availability: "https://schema.org/InStock",
    URL:          "https://example.com/products/widget-pro",
    RatingValue:  4.8,
    ReviewCount:  142,
})
```

### Review

```go
seo.ReviewSchema(seo.ReviewSchemaInput{
    ItemName:      "Widget Pro",
    Author:        "Jane Smith",
    RatingValue:   5.0,
    Body:          "Best widget I've ever used!",
    DatePublished: time.Now(),
})
```

---

## 12. HowTo Schema

```go
seo.HowToSchema("How to Install NexGo", "Install NexGo in 3 steps", []seo.HowToStep{
    {Name: "Install Go", Text: "Download and install Go 1.22+"},
    {Name: "Install NexGo", Text: "Run: go install github.com/salmanfaris22/nexgo/cmd/nexgo@latest"},
    {Name: "Create Project", Text: "Run: nexgo create my-app"},
})
```

---

## 13. RSS Feed Generation

### Auto RSS Setup

In your `main.go`, register an RSS endpoint:

```go
srv.RegisterRoute("/rss.xml", func(w http.ResponseWriter, r *http.Request) {
    posts := getPublishedPosts() // your data source

    items := make([]seo.FeedItem, len(posts))
    for i, p := range posts {
        items[i] = seo.FeedItem{
            Title:       p.Title,
            Link:        "https://example.com/blog/" + p.Slug,
            Description: p.Excerpt,
            Author:      p.Author,
            PubDate:     p.PublishedAt,
            Categories:  p.Tags,
        }
    }

    data, err := seo.RenderRSS(seo.Feed{
        Title:       "My Site Blog",
        Link:        "https://example.com",
        Description: "Latest articles",
        Language:    "en",
        Items:       items,
    })
    if err != nil {
        http.Error(w, "RSS error", 500)
        return
    }

    w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
    w.Write(data)
})
```

### Link in Layout

```html
<link rel="alternate" type="application/rss+xml" title="RSS Feed" href="/rss.xml">
```

---

## 14. Hreflang Alternates

For multilingual sites:

```go
meta := seo.Meta{
    Title:       "About",
    Description: "About us",
    Alternates: []seo.AlternateLink{
        {HrefLang: "en", Href: "https://example.com/about"},
        {HrefLang: "fr", Href: "https://example.com/fr/about"},
        {HrefLang: "de", Href: "https://example.com/de/about"},
        {HrefLang: "x-default", Href: "https://example.com/about"},
    },
}
```

Generates:

```html
<link rel="alternate" hreflang="en" href="https://example.com/about">
<link rel="alternate" hreflang="fr" href="https://example.com/fr/about">
<link rel="alternate" hreflang="de" href="https://example.com/de/about">
<link rel="alternate" hreflang="x-default" href="https://example.com/about">
```

---

## 15. Pagination SEO

For paginated content (blog listing, search results):

```go
meta := seo.Meta{
    Title:       "Blog - Page 2",
    Description: "Blog posts, page 2 of 10.",
    Canonical:   "https://example.com/blog?page=2",
    PrevURL:     "https://example.com/blog?page=1",
    NextURL:     "https://example.com/blog?page=3",
}
```

Generates:

```html
<link rel="prev" href="https://example.com/blog?page=1">
<link rel="next" href="https://example.com/blog?page=3">
```

---

## 16. Redirects

### Config-Based Redirects

```json
{
  "seo": {
    "redirects": [
      {"from": "/old-blog", "to": "/blog", "status": 301},
      {"from": "/sale", "to": "/products/summer-sale", "status": 302}
    ]
  }
}
```

### Programmatic Redirects

```go
redirects := []seo.Redirect{
    {From: "/old-page", To: "/new-page", StatusCode: 301},
    {From: "/temp",     To: "/temporary", StatusCode: 302},
}
handler := seo.RedirectMiddleware(redirects, yourMux)
```

---

## 17. Trailing Slash Normalization

Prevent duplicate URLs (kills SEO rankings):

```go
// Enforce NO trailing slash: /about/ -> 301 -> /about
handler := seo.TrailingSlashMiddleware(false, yourHandler)

// Enforce WITH trailing slash: /about -> 301 -> /about/
handler := seo.TrailingSlashMiddleware(true, yourHandler)
```

---

## 18. Preload & Prefetch Hints

### In Templates

```html
<!-- Preload critical CSS -->
{{ preload "/static/css/global.css" "style" }}

<!-- Preload a font -->
{{ preload "/static/fonts/outfit.woff2" "font" }}
```

### In Go

```go
hints := []seo.PreloadHint{
    {Href: "/static/css/global.css", As: "style"},
    {Href: "/static/fonts/outfit.woff2", As: "font", Type: "font/woff2", CrossOrigin: true},
    {Href: "/static/js/app.js", As: "script"},
}
html := seo.RenderPreloadTags(hints)
```

### Prefetch & Prerender

```go
// Prefetch likely next page
seo.PrefetchTag("/about")

// Prerender high-probability next page (uses more resources)
seo.PrerenderTag("/pricing")
```

---

## 19. Core Web Vitals

Track LCP, CLS, and INP automatically:

1. Enable in config:

```json
{
  "seo": {
    "coreWebVitals": true,
    "vitalsEndpoint": "/api/vitals"
  }
}
```

2. Use `{{ vitals }}` in your layout (already included in scaffold)

3. Create an API endpoint to receive reports:

```go
// pages/api/vitals.go
func Vitals(w http.ResponseWriter, r *http.Request) {
    // Read CWV beacon data
    var data map[string]interface{}
    json.NewDecoder(r.Body).Decode(&data)
    log.Printf("[CWV] LCP=%.0fms CLS=%.4f INP=%.0fms URL=%s",
        data["lcp"], data["cls"], data["inp"], data["url"])
    w.WriteHeader(204)
}
```

**Measured metrics:**
- **LCP** (Largest Contentful Paint) — loading performance
- **CLS** (Cumulative Layout Shift) — visual stability
- **INP** (Interaction to Next Paint) — responsiveness

---

## 20. SEO HTTP Headers

NexGo automatically adds these headers to every response:

| Header | Value | Purpose |
|--------|-------|---------|
| `Content-Language` | From config `language` | Tells crawlers the page language |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME sniffing |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer info |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Restricts browser APIs |

### HSTS (Optional)

```go
handler := seo.HSTSMiddleware(31536000, yourHandler) // 1 year
```

Adds: `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`

---

## 21. SEO Audit & Validation

### Audit Meta Tags

```go
results := seo.AuditMeta(meta)
for _, r := range results {
    fmt.Printf("[%s] %s: %s\n", r.Level, r.Code, r.Message)
}
```

**Checks performed:**
- Missing title (`MISSING_TITLE`)
- Title too long > 60 chars (`TITLE_TOO_LONG`)
- Title too short < 10 chars (`TITLE_TOO_SHORT`)
- Missing description (`MISSING_DESCRIPTION`)
- Description too long > 160 chars (`DESC_TOO_LONG`)
- Description too short < 50 chars (`DESC_TOO_SHORT`)
- Missing canonical URL (`MISSING_CANONICAL`)
- Missing OG image (`MISSING_OG_IMAGE`)
- No Twitter card type (`NO_TWITTER_CARD`)

### Audit HTML

```go
results := seo.AuditHTML(htmlString)
```

**Checks performed:**
- Missing `<h1>` tag (`MISSING_H1`)
- Multiple `<h1>` tags (`MULTIPLE_H1`)
- Images without alt attributes (`IMG_MISSING_ALT`)
- Missing viewport meta tag (`MISSING_VIEWPORT`)

---

## 22. URL Slug Helpers

Convert any string to an SEO-friendly URL slug:

```go
seo.Slugify("Hello World!")           // "hello-world"
seo.Slugify("  Café & Résumé  ")     // "cafe-resume"
seo.Slugify("100% Pure Go Framework") // "100-pure-go-framework"
seo.Slugify("NexGo v2.2.1 Release")  // "nexgo-v221-release"
```

In templates:

```html
<a href="/blog/{{ slugify .Props.post.Title }}">{{ .Props.post.Title }}</a>
```

---

## 23. Default SEO Scaffold

When you run `nexgo create my-app`, a `seo/` folder is created with:

```
my-app/
├── seo/
│   └── seo.go              # Site-wide SEO config + demo data
├── layouts/
│   └── default.html         # Layout with SEO template functions
├── nexgo.config.json        # SEO section pre-configured
└── main.go                  # Imports seo/ package
```

### seo/seo.go

Contains:
- `SiteConfig` — your site-wide defaults
- `HomeMeta()` — SEO meta for homepage
- `AboutMeta()` — SEO meta for about page
- `BlogMeta()` — SEO meta for blog
- `WebsiteSchemaTag()` — WebSite structured data
- `OrganizationSchemaTag()` — Organization structured data
- `DemoFAQ()` — Example FAQ structured data
- `DemoRSSFeed()` — Example RSS feed builder

### layouts/default.html

Pre-configured with:

```html
<head>
  <meta charset="UTF-8">
  {{ seoTags (seoMeta .Title .Description .Canonical) }}
  {{ preload "/static/css/global.css" "style" }}
  <link rel="stylesheet" href="/static/css/global.css">
  <link rel="alternate" type="application/rss+xml" title="RSS Feed" href="/rss.xml">
  {{ breadcrumbs .Path }}
  {{ vitals }}
</head>
```

---

## 24. Template Functions Reference

### SEO Functions

| Function | Usage | Description |
|----------|-------|-------------|
| `seoTags` | `{{ seoTags (seoMeta .Title .Description .Canonical) }}` | Render all SEO meta tags |
| `seoMeta` | `{{ seoMeta "Title" "Desc" "/url" }}` | Create a Meta struct |
| `breadcrumbs` | `{{ breadcrumbs .Path }}` | Auto BreadcrumbList JSON-LD |
| `jsonld` | `{{ jsonld (dict "@context" "..." ...) }}` | Generic JSON-LD tag |
| `websiteSchema` | `{{ websiteSchema "Name" "URL" "SearchURL" }}` | WebSite schema |
| `orgSchema` | `{{ orgSchema "Name" "URL" "Logo" }}` | Organization schema |
| `faqSchema` | `{{ faqSchema "Q?" "A." "Q?" "A." }}` | FAQ schema |
| `preload` | `{{ preload "/path" "style" }}` | Resource preload hint |
| `vitals` | `{{ vitals }}` | Core Web Vitals script |
| `slugify` | `{{ slugify "Hello World" }}` | SEO-friendly slug |

### Existing Functions (still available)

| Function | Usage |
|----------|-------|
| `json`, `safeHTML`, `dict`, `slice` | Data helpers |
| `asset`, `link`, `times`, `default` | Utility helpers |
| `upper`, `lower`, `title`, `replace`, `trim`, `split`, `join` | String functions |
| `add`, `sub`, `mul`, `div` | Math functions |
| `island`, `props`, `islandRuntime` | Islands architecture |
| `renderState` | State hydration |

---

## 25. Go API Reference

### Package: `github.com/salmanfaris22/nexgo/v2/pkg/seo`

#### Types

```go
// Site-wide config
type SiteConfig struct { ... }

// Page meta
type Meta struct { ... }

// Sitemap
type SitemapEntry struct { ... }
type SitemapAlternate struct { ... }
type SitemapIndexEntry struct { ... }

// Robots
type RobotsRule struct { ... }

// Structured Data
type BreadcrumbItem struct { ... }
type ArticleSchemaInput struct { ... }
type FAQItem struct { ... }
type HowToStep struct { ... }
type ProductSchemaInput struct { ... }
type ReviewSchemaInput struct { ... }

// RSS Feed
type Feed struct { ... }
type FeedItem struct { ... }

// Resource hints
type PreloadHint struct { ... }

// Redirects
type Redirect struct { ... }

// Audit
type AuditResult struct { ... }
```

#### Functions

```go
// Meta tags
func RenderMetaTags(m Meta, site ...SiteConfig) template.HTML
func DefaultMeta(title, description, canonical string) Meta
func ArticleMeta(title, description, author, image, canonical string, pubDate time.Time) Meta
func ProductMeta(title, description, image, canonical string) Meta
func ProfileMeta(name, description, image, canonical string) Meta

// Sitemap
func RenderSitemap(entries []SitemapEntry) ([]byte, error)
func RenderSitemapIndex(sitemaps []SitemapIndexEntry) ([]byte, error)
func AutoSitemap(baseURL string, routes []string, defaultPriority float64) []SitemapEntry

// Robots.txt
func RobotsTxt(allow, disallow []string, sitemapURL string) string
func RobotsTxtAdvanced(rules []RobotsRule, sitemapURLs []string) string

// JSON-LD
func JSONLD(data map[string]interface{}) template.HTML
func JSONLDTyped(v interface{}) template.HTML
func WebSiteSchema(name, url, searchURL string) template.HTML
func OrganizationSchema(name, url, logo string, sameAs []string) template.HTML
func LocalBusinessSchema(name, url, phone, street, city, state, zip, country string) template.HTML
func BreadcrumbSchema(items []BreadcrumbItem) template.HTML
func AutoBreadcrumbs(baseURL, path string) []BreadcrumbItem
func ArticleSchema(a ArticleSchemaInput) template.HTML
func FAQSchema(items []FAQItem) template.HTML
func HowToSchema(name, description string, steps []HowToStep) template.HTML
func ProductSchema(p ProductSchemaInput) template.HTML
func ReviewSchema(r ReviewSchemaInput) template.HTML

// RSS
func RenderRSS(feed Feed) ([]byte, error)

// Preload
func RenderPreloadTags(hints []PreloadHint) template.HTML
func PrefetchTag(href string) template.HTML
func PrerenderTag(href string) template.HTML

// Middleware
func RedirectMiddleware(redirects []Redirect, next http.Handler) http.Handler
func TrailingSlashMiddleware(add bool, next http.Handler) http.Handler
func SEOHeaders(language string, next http.HandlerFunc) http.HandlerFunc
func HSTSMiddleware(maxAge int, next http.HandlerFunc) http.HandlerFunc

// Core Web Vitals
func CoreWebVitalsScript(reportEndpoint string) template.HTML

// Audit
func AuditMeta(m Meta) []AuditResult
func AuditHTML(html string) []AuditResult

// URL
func Slugify(s string) string
```

---

## 26. Best Practices Checklist

Use this checklist when deploying your NexGo app:

### Essential (Do First)

- [ ] Set `siteURL` in config (required for canonical URLs and sitemap)
- [ ] Set `siteName` for og:site_name and structured data
- [ ] Set `titleTemplate` (e.g. `"%s | My Site"`)
- [ ] Set `defaultOGImage` (1200x630px recommended)
- [ ] Set `twitterSite` handle
- [ ] Verify `/sitemap.xml` is accessible
- [ ] Verify `/robots.txt` is correct
- [ ] Submit sitemap to Google Search Console
- [ ] Submit sitemap to Bing Webmaster Tools

### Per Page

- [ ] Every page has a unique `<title>` (10-60 chars)
- [ ] Every page has a unique `<meta description>` (50-160 chars)
- [ ] Every page has exactly one `<h1>`
- [ ] Every image has an `alt` attribute
- [ ] Use canonical URLs to prevent duplicate content
- [ ] Add structured data where applicable (Article, Product, FAQ, etc.)

### Performance (SEO Impact)

- [ ] Enable compression: `"compression": true`
- [ ] Use `{{ preload }}` for critical CSS and fonts
- [ ] Enable Core Web Vitals tracking to monitor LCP, CLS, INP
- [ ] Use Islands Architecture for interactive components (zero JS by default)
- [ ] Enable response caching in production: `"responseCache": true`
- [ ] Enable cluster mode for multi-core: `"clusterMode": true`

### Advanced

- [ ] Add RSS feed for blog content (`/rss.xml`)
- [ ] Add hreflang alternates for multilingual content
- [ ] Set up 301 redirects for changed URLs
- [ ] Enable HSTS for HTTPS ranking signal
- [ ] Block AI crawlers (GPTBot, CCBot) in robots.txt if desired
- [ ] Use `seo.AuditMeta()` in tests to catch SEO regressions

---

## Migration from v2.2.0

v2.2.1 is fully backward compatible. To upgrade:

1. Update your dependency:

```bash
go get github.com/salmanfaris22/nexgo/v2@v2.2.1
```

2. Add SEO config to `nexgo.config.json`:

```json
{
  "seo": {
    "siteName": "My Site",
    "siteURL": "https://example.com",
    "titleTemplate": "%s | My Site",
    "autoSitemap": true,
    "autoRobotsTxt": true,
    "autoCanonical": true
  }
}
```

3. Update your layout to use SEO template functions:

```html
<head>
  {{ seoTags (seoMeta .Title .Description .Canonical) }}
  {{ breadcrumbs .Path }}
</head>
```

4. (Optional) Create a `seo/` folder with your site defaults.

Your existing code continues to work. The new SEO features are additive.

---

## Source Files

| File | Lines | Purpose |
|------|-------|---------|
| `pkg/seo/seo.go` | ~700 | Full SEO toolkit |
| `pkg/config/config.go` | ~220 | SEO configuration |
| `pkg/server/server.go` | ~750 | Auto sitemap/robots endpoints |
| `pkg/renderer/renderer.go` | ~870 | SEO template functions |
| `cmd/nexgo/main.go` | ~520 | Scaffold with SEO defaults |

---

## License

MIT

---

Built with Go. Optimized for search engines. Powered by NexGo.
