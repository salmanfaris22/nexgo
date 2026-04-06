package seo

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMetaTags(t *testing.T) {
	m := Meta{
		Title:       "My Page",
		Description: "A great page",
		Keywords:    []string{"go", "web"},
		Author:      "John",
		Canonical:   "https://example.com/page",
		OGImage:     "https://example.com/og.png",
		OGType:      "article",
		Locale:      "en_US",
		TwitterCard: "summary_large_image",
		TwitterSite: "@example",
	}

	html := string(RenderMetaTags(m))

	checks := []string{
		"<title>My Page</title>",
		`property="og:title" content="My Page"`,
		`name="description" content="A great page"`,
		`name="keywords" content="go, web"`,
		`name="author" content="John"`,
		`rel="canonical" href="https://example.com/page"`,
		`property="og:image" content="https://example.com/og.png"`,
		`property="og:type" content="article"`,
		`name="twitter:card" content="summary_large_image"`,
		`name="twitter:site" content="@example"`,
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected %q in output", check)
		}
	}
}

func TestRenderMetaTags_NoIndex(t *testing.T) {
	m := Meta{NoIndex: true, NoFollow: true}
	html := string(RenderMetaTags(m))
	if !strings.Contains(html, "noindex, nofollow") {
		t.Error("expected noindex, nofollow meta tag")
	}

	// Test NoIndex alone
	m2 := Meta{NoIndex: true}
	html2 := string(RenderMetaTags(m2))
	if !strings.Contains(html2, "noindex") {
		t.Error("expected noindex meta tag")
	}
}

func TestRenderMetaTags_Defaults(t *testing.T) {
	m := Meta{Title: "Test"}
	html := string(RenderMetaTags(m))
	if !strings.Contains(html, `property="og:type" content="website"`) {
		t.Error("expected default og:type")
	}
	if !strings.Contains(html, `property="og:locale" content="en_US"`) {
		t.Error("expected default og:locale")
	}
}

func TestRenderMetaTags_Escaping(t *testing.T) {
	m := Meta{Title: "<script>alert('xss')</script>"}
	html := string(RenderMetaTags(m))
	if strings.Contains(html, "<script>") {
		t.Error("expected HTML escaping")
	}
}

func TestRenderSitemap(t *testing.T) {
	entries := []SitemapEntry{
		{Loc: "https://example.com/", LastMod: "2024-01-01", ChangeFreq: "daily", Priority: 1.0},
		{Loc: "https://example.com/about", LastMod: "2024-01-02", ChangeFreq: "monthly", Priority: 0.5},
	}

	xml, err := RenderSitemap(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(xml)
	if !strings.Contains(body, "https://example.com/") {
		t.Error("expected first URL in sitemap")
	}
	if !strings.Contains(body, "xmlns=") {
		t.Error("expected xmlns attribute")
	}
}

func TestRobotsTxt(t *testing.T) {
	txt := RobotsTxt(
		[]string{"/public"},
		[]string{"/admin", "/private"},
		"https://example.com/sitemap.xml",
	)

	checks := []string{
		"User-agent: *",
		"Allow: /public",
		"Disallow: /admin",
		"Disallow: /private",
		"Sitemap: https://example.com/sitemap.xml",
	}

	for _, check := range checks {
		if !strings.Contains(txt, check) {
			t.Errorf("expected %q in robots.txt", check)
		}
	}
}

func TestRobotsTxt_Empty(t *testing.T) {
	txt := RobotsTxt(nil, nil, "")
	if txt != "User-agent: *\n" {
		t.Errorf("unexpected robots.txt: %q", txt)
	}
}

func TestJSONLD(t *testing.T) {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     "My Company",
	}

	html := string(JSONLD(data))
	if !strings.Contains(html, `<script type="application/ld+json">`) {
		t.Error("expected JSON-LD script tag")
	}
	if !strings.Contains(html, "My Company") {
		t.Error("expected data in JSON-LD")
	}
}

func TestDefaultMeta(t *testing.T) {
	m := DefaultMeta("Title", "Desc", "https://example.com")
	if m.Title != "Title" {
		t.Errorf("expected Title=Title, got %s", m.Title)
	}
	if m.OGType != "website" {
		t.Errorf("expected OGType=website, got %s", m.OGType)
	}
	if m.TwitterCard != "summary_large_image" {
		t.Errorf("expected TwitterCard=summary_large_image, got %s", m.TwitterCard)
	}
}

func TestArticleMeta(t *testing.T) {
	pubDate := time.Now()
	m := ArticleMeta("Article", "Desc", "Author", "img.jpg", "https://example.com", pubDate)
	if m.Title != "Article" {
		t.Errorf("expected Title=Article, got %s", m.Title)
	}
	if m.OGType != "article" {
		t.Errorf("expected OGType=article, got %s", m.OGType)
	}
	if m.Author != "Author" {
		t.Errorf("expected Author=Author, got %s", m.Author)
	}
}
