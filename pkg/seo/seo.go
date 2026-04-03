// Package seo provides meta tags, OpenGraph, Twitter Cards, sitemap, and robots.txt helpers.
package seo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Meta holds SEO metadata for a page.
type Meta struct {
	Title       string
	Description string
	Keywords    []string
	Author      string
	Canonical   string
	OGImage     string
	OGType      string // "website", "article", etc.
	Locale      string
	TwitterCard string // "summary", "summary_large_image"
	TwitterSite string
	NoIndex     bool
}

// RenderMetaTags returns a template.HTML string of all meta tags.
func RenderMetaTags(m Meta) template.HTML {
	tags := ""
	if m.Title != "" {
		tags += fmt.Sprintf("<title>%s</title>\n", escape(m.Title))
		tags += fmt.Sprintf(`<meta property="og:title" content="%s">`+"\n", escape(m.Title))
	}
	if m.Description != "" {
		tags += fmt.Sprintf(`<meta name="description" content="%s">`+"\n", escape(m.Description))
		tags += fmt.Sprintf(`<meta property="og:description" content="%s">`+"\n", escape(m.Description))
	}
	if m.Keywords != nil && len(m.Keywords) > 0 {
		tags += fmt.Sprintf(`<meta name="keywords" content="%s">`+"\n", escape(strings.Join(m.Keywords, ", ")))
	}
	if m.Author != "" {
		tags += fmt.Sprintf(`<meta name="author" content="%s">`+"\n", escape(m.Author))
	}
	if m.Canonical != "" {
		tags += fmt.Sprintf(`<link rel="canonical" href="%s">`+"\n", escape(m.Canonical))
		tags += fmt.Sprintf(`<meta property="og:url" content="%s">`+"\n", escape(m.Canonical))
	}
	if m.OGImage != "" {
		tags += fmt.Sprintf(`<meta property="og:image" content="%s">`+"\n", escape(m.OGImage))
	}
	if m.OGType != "" {
		tags += fmt.Sprintf(`<meta property="og:type" content="%s">`+"\n", escape(m.OGType))
	} else {
		tags += `<meta property="og:type" content="website">` + "\n"
	}
	if m.Locale != "" {
		tags += fmt.Sprintf(`<meta property="og:locale" content="%s">`+"\n", escape(m.Locale))
	} else {
		tags += `<meta property="og:locale" content="en_US">` + "\n"
	}
	if m.TwitterCard != "" {
		tags += fmt.Sprintf(`<meta name="twitter:card" content="%s">`+"\n", escape(m.TwitterCard))
	}
	if m.TwitterSite != "" {
		tags += fmt.Sprintf(`<meta name="twitter:site" content="%s">`+"\n", escape(m.TwitterSite))
	}
	if m.NoIndex {
		tags += `<meta name="robots" content="noindex, nofollow">` + "\n"
	}
	return template.HTML(tags)
}

// SitemapEntry is a single URL in a sitemap.
type SitemapEntry struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// Sitemap is the root of a sitemap XML document.
type Sitemap struct {
	XMLName xml.Name       `xml:"urlset"`
	Xmlns   string         `xml:"xmlns,attr"`
	Entries []SitemapEntry `xml:"url"`
}

// RenderSitemap returns XML bytes for a sitemap.
func RenderSitemap(entries []SitemapEntry) ([]byte, error) {
	s := Sitemap{
		Xmlns:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		Entries: entries,
	}
	return xml.MarshalIndent(s, "", "  ")
}

// RobotsTxt returns a standard robots.txt content.
func RobotsTxt(allow []string, disallow []string, sitemapURL string) string {
	out := "User-agent: *\n"
	for _, a := range allow {
		out += fmt.Sprintf("Allow: %s\n", a)
	}
	for _, d := range disallow {
		out += fmt.Sprintf("Disallow: %s\n", d)
	}
	if sitemapURL != "" {
		out += fmt.Sprintf("Sitemap: %s\n", sitemapURL)
	}
	return out
}

// JSONLD returns a JSON-LD structured data script tag.
func JSONLD(data map[string]interface{}) template.HTML {
	b, err := json.Marshal(data)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(b)))
}

func escape(s string) string {
	return template.HTMLEscapeString(s)
}

// DefaultMeta returns a Meta with sensible defaults.
func DefaultMeta(title, description, canonical string) Meta {
	return Meta{
		Title:       title,
		Description: description,
		Canonical:   canonical,
		Locale:      "en_US",
		OGType:      "website",
		TwitterCard: "summary_large_image",
	}
}

// ArticleMeta returns Meta optimized for blog articles.
func ArticleMeta(title, description, author, image, canonical string, pubDate time.Time) Meta {
	return Meta{
		Title:       title,
		Description: description,
		Author:      author,
		Canonical:   canonical,
		OGImage:     image,
		OGType:      "article",
		Locale:      "en_US",
		TwitterCard: "summary_large_image",
	}
}
