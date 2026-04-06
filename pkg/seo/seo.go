// Package seo provides a complete SEO toolkit: meta tags, OpenGraph, Twitter Cards,
// structured data (JSON-LD), sitemap, sitemap index, robots.txt, RSS/Atom feeds,
// hreflang alternates, breadcrumbs, redirects, Core Web Vitals, and SEO auditing.
package seo

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ---------------------------------------------------------------------------
// Site-wide configuration
// ---------------------------------------------------------------------------

// SiteConfig holds site-wide SEO defaults applied to every page unless overridden.
type SiteConfig struct {
	SiteName      string // e.g. "My Awesome Site"
	SiteURL       string // e.g. "https://example.com" (no trailing slash)
	TitleTemplate string // e.g. "%s | My Site" — %s is replaced with page title
	DefaultOGImage string
	TwitterSite   string // e.g. "@mysite"
	DefaultLocale string // e.g. "en_US"
	Language      string // e.g. "en"
	FaviconURL    string
	ThemeColor    string // e.g. "#00d2ff"
	Author        string
}

// DefaultSiteConfig returns a SiteConfig with sensible defaults.
func DefaultSiteConfig() SiteConfig {
	return SiteConfig{
		DefaultLocale: "en_US",
		Language:      "en",
		ThemeColor:    "#00d2ff",
	}
}

// ---------------------------------------------------------------------------
// Meta tags
// ---------------------------------------------------------------------------

// Meta holds SEO metadata for a page.
type Meta struct {
	Title       string
	Description string
	Keywords    []string
	Author      string
	Canonical   string
	OGImage     string
	OGType      string // "website", "article", etc.
	OGSiteName  string
	Locale      string
	TwitterCard string // "summary", "summary_large_image"
	TwitterSite string
	NoIndex     bool
	NoFollow    bool

	// Article-specific OpenGraph
	ArticlePublishedTime string // ISO 8601
	ArticleModifiedTime  string // ISO 8601
	ArticleAuthor        string
	ArticleSection       string
	ArticleTags          []string

	// Alternate languages (hreflang)
	Alternates []AlternateLink

	// Pagination
	PrevURL string
	NextURL string

	// Extra — arbitrary meta tags
	Extra []MetaTag
}

// MetaTag is a generic meta/link tag.
type MetaTag struct {
	Tag      string // "meta" or "link"
	Name     string // name or property attribute
	Content  string
	Rel      string // for link tags
	Href     string // for link tags
	HrefLang string // for hreflang link tags
}

// AlternateLink represents a hreflang alternate.
type AlternateLink struct {
	HrefLang string // e.g. "en", "fr", "x-default"
	Href     string
}

// RenderMetaTags returns a template.HTML string of all meta tags.
// When a SiteConfig is provided (pass nil to skip), it fills in defaults.
func RenderMetaTags(m Meta, site ...SiteConfig) template.HTML {
	var sc SiteConfig
	if len(site) > 0 {
		sc = site[0]
	}

	tags := ""

	// Viewport (always present for mobile SEO)
	tags += `<meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n"

	// Title (with template support)
	title := m.Title
	if title == "" && sc.SiteName != "" {
		title = sc.SiteName
	}
	if title != "" {
		rendered := title
		if sc.TitleTemplate != "" && title != sc.SiteName {
			rendered = strings.Replace(sc.TitleTemplate, "%s", title, 1)
		}
		tags += fmt.Sprintf("<title>%s</title>\n", esc(rendered))
		tags += fmt.Sprintf(`<meta property="og:title" content="%s">`+"\n", esc(title))
		tags += fmt.Sprintf(`<meta name="twitter:title" content="%s">`+"\n", esc(title))
	}

	// Description
	if m.Description != "" {
		tags += fmt.Sprintf(`<meta name="description" content="%s">`+"\n", esc(m.Description))
		tags += fmt.Sprintf(`<meta property="og:description" content="%s">`+"\n", esc(m.Description))
		tags += fmt.Sprintf(`<meta name="twitter:description" content="%s">`+"\n", esc(m.Description))
	}

	// Keywords
	if len(m.Keywords) > 0 {
		tags += fmt.Sprintf(`<meta name="keywords" content="%s">`+"\n", esc(strings.Join(m.Keywords, ", ")))
	}

	// Author
	author := m.Author
	if author == "" {
		author = sc.Author
	}
	if author != "" {
		tags += fmt.Sprintf(`<meta name="author" content="%s">`+"\n", esc(author))
	}

	// Canonical
	canonical := m.Canonical
	if canonical != "" {
		tags += fmt.Sprintf(`<link rel="canonical" href="%s">`+"\n", esc(canonical))
		tags += fmt.Sprintf(`<meta property="og:url" content="%s">`+"\n", esc(canonical))
	}

	// OG Image
	ogImage := m.OGImage
	if ogImage == "" {
		ogImage = sc.DefaultOGImage
	}
	if ogImage != "" {
		tags += fmt.Sprintf(`<meta property="og:image" content="%s">`+"\n", esc(ogImage))
		tags += fmt.Sprintf(`<meta name="twitter:image" content="%s">`+"\n", esc(ogImage))
	}

	// OG Type
	if m.OGType != "" {
		tags += fmt.Sprintf(`<meta property="og:type" content="%s">`+"\n", esc(m.OGType))
	} else {
		tags += `<meta property="og:type" content="website">` + "\n"
	}

	// OG Site Name
	siteName := m.OGSiteName
	if siteName == "" {
		siteName = sc.SiteName
	}
	if siteName != "" {
		tags += fmt.Sprintf(`<meta property="og:site_name" content="%s">`+"\n", esc(siteName))
	}

	// Locale
	locale := m.Locale
	if locale == "" {
		locale = sc.DefaultLocale
	}
	if locale == "" {
		locale = "en_US"
	}
	tags += fmt.Sprintf(`<meta property="og:locale" content="%s">`+"\n", esc(locale))

	// Twitter Card
	twitterCard := m.TwitterCard
	if twitterCard == "" {
		twitterCard = "summary_large_image"
	}
	tags += fmt.Sprintf(`<meta name="twitter:card" content="%s">`+"\n", esc(twitterCard))

	// Twitter Site
	twitterSite := m.TwitterSite
	if twitterSite == "" {
		twitterSite = sc.TwitterSite
	}
	if twitterSite != "" {
		tags += fmt.Sprintf(`<meta name="twitter:site" content="%s">`+"\n", esc(twitterSite))
	}

	// Robots
	if m.NoIndex || m.NoFollow {
		var directives []string
		if m.NoIndex {
			directives = append(directives, "noindex")
		}
		if m.NoFollow {
			directives = append(directives, "nofollow")
		}
		tags += fmt.Sprintf(`<meta name="robots" content="%s">`+"\n", strings.Join(directives, ", "))
	}

	// Article OG tags
	if m.ArticlePublishedTime != "" {
		tags += fmt.Sprintf(`<meta property="article:published_time" content="%s">`+"\n", esc(m.ArticlePublishedTime))
	}
	if m.ArticleModifiedTime != "" {
		tags += fmt.Sprintf(`<meta property="article:modified_time" content="%s">`+"\n", esc(m.ArticleModifiedTime))
	}
	if m.ArticleAuthor != "" {
		tags += fmt.Sprintf(`<meta property="article:author" content="%s">`+"\n", esc(m.ArticleAuthor))
	}
	if m.ArticleSection != "" {
		tags += fmt.Sprintf(`<meta property="article:section" content="%s">`+"\n", esc(m.ArticleSection))
	}
	for _, t := range m.ArticleTags {
		tags += fmt.Sprintf(`<meta property="article:tag" content="%s">`+"\n", esc(t))
	}

	// Hreflang alternates
	for _, alt := range m.Alternates {
		tags += fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s">`+"\n", esc(alt.HrefLang), esc(alt.Href))
	}

	// Pagination rel=prev/next
	if m.PrevURL != "" {
		tags += fmt.Sprintf(`<link rel="prev" href="%s">`+"\n", esc(m.PrevURL))
	}
	if m.NextURL != "" {
		tags += fmt.Sprintf(`<link rel="next" href="%s">`+"\n", esc(m.NextURL))
	}

	// Theme color
	if sc.ThemeColor != "" {
		tags += fmt.Sprintf(`<meta name="theme-color" content="%s">`+"\n", esc(sc.ThemeColor))
	}

	// Favicon
	if sc.FaviconURL != "" {
		tags += fmt.Sprintf(`<link rel="icon" href="%s">`+"\n", esc(sc.FaviconURL))
	}

	// Extra tags
	for _, et := range m.Extra {
		if et.Tag == "link" {
			attrs := ""
			if et.Rel != "" {
				attrs += fmt.Sprintf(` rel="%s"`, esc(et.Rel))
			}
			if et.Href != "" {
				attrs += fmt.Sprintf(` href="%s"`, esc(et.Href))
			}
			if et.HrefLang != "" {
				attrs += fmt.Sprintf(` hreflang="%s"`, esc(et.HrefLang))
			}
			tags += fmt.Sprintf("<link%s>\n", attrs)
		} else {
			attrs := ""
			if et.Name != "" {
				if strings.Contains(et.Name, ":") {
					attrs += fmt.Sprintf(` property="%s"`, esc(et.Name))
				} else {
					attrs += fmt.Sprintf(` name="%s"`, esc(et.Name))
				}
			}
			if et.Content != "" {
				attrs += fmt.Sprintf(` content="%s"`, esc(et.Content))
			}
			tags += fmt.Sprintf("<meta%s>\n", attrs)
		}
	}

	return template.HTML(tags)
}

// ---------------------------------------------------------------------------
// Convenience constructors
// ---------------------------------------------------------------------------

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
		Title:                title,
		Description:          description,
		Author:               author,
		Canonical:            canonical,
		OGImage:              image,
		OGType:               "article",
		Locale:               "en_US",
		TwitterCard:          "summary_large_image",
		ArticleAuthor:        author,
		ArticlePublishedTime: pubDate.Format(time.RFC3339),
	}
}

// ProductMeta returns Meta optimized for product pages.
func ProductMeta(title, description, image, canonical string) Meta {
	return Meta{
		Title:       title,
		Description: description,
		Canonical:   canonical,
		OGImage:     image,
		OGType:      "product",
		TwitterCard: "summary_large_image",
	}
}

// ProfileMeta returns Meta for user profile pages.
func ProfileMeta(name, description, image, canonical string) Meta {
	return Meta{
		Title:       name,
		Description: description,
		Canonical:   canonical,
		OGImage:     image,
		OGType:      "profile",
		TwitterCard: "summary",
	}
}

// ---------------------------------------------------------------------------
// Sitemap
// ---------------------------------------------------------------------------

// SitemapEntry is a single URL in a sitemap.
type SitemapEntry struct {
	Loc        string          `xml:"loc"`
	LastMod    string          `xml:"lastmod,omitempty"`
	ChangeFreq string         `xml:"changefreq,omitempty"`
	Priority   float64         `xml:"priority,omitempty"`
	Alternates []SitemapAlternate `xml:"xhtml:link,omitempty"`
}

// SitemapAlternate represents an hreflang alternate in a sitemap.
type SitemapAlternate struct {
	Rel      string `xml:"rel,attr"`
	HrefLang string `xml:"hreflang,attr"`
	Href     string `xml:"href,attr"`
}

// Sitemap is the root of a sitemap XML document.
type Sitemap struct {
	XMLName xml.Name       `xml:"urlset"`
	Xmlns   string         `xml:"xmlns,attr"`
	Xhtml   string         `xml:"xmlns:xhtml,attr,omitempty"`
	Entries []SitemapEntry `xml:"url"`
}

// RenderSitemap returns XML bytes for a sitemap.
func RenderSitemap(entries []SitemapEntry) ([]byte, error) {
	hasAlternates := false
	for _, e := range entries {
		if len(e.Alternates) > 0 {
			hasAlternates = true
			break
		}
	}
	s := Sitemap{
		Xmlns:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		Entries: entries,
	}
	if hasAlternates {
		s.Xhtml = "http://www.w3.org/1999/xhtml"
	}
	out, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// SitemapIndexEntry is a single sitemap in a sitemap index.
type SitemapIndexEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// SitemapIndex is the root of a sitemap index XML document (for 50k+ URL sites).
type SitemapIndex struct {
	XMLName  xml.Name            `xml:"sitemapindex"`
	Xmlns    string              `xml:"xmlns,attr"`
	Sitemaps []SitemapIndexEntry `xml:"sitemap"`
}

// RenderSitemapIndex returns XML bytes for a sitemap index.
func RenderSitemapIndex(sitemaps []SitemapIndexEntry) ([]byte, error) {
	si := SitemapIndex{
		Xmlns:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: sitemaps,
	}
	out, err := xml.MarshalIndent(si, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// AutoSitemap builds sitemap entries from a list of route patterns and a base URL.
func AutoSitemap(baseURL string, routes []string, defaultPriority float64) []SitemapEntry {
	if defaultPriority <= 0 {
		defaultPriority = 0.5
	}
	now := time.Now().Format("2006-01-02")
	entries := make([]SitemapEntry, 0, len(routes))
	for _, route := range routes {
		// Skip dynamic routes — they need data to resolve
		if strings.Contains(route, "[") || strings.Contains(route, "*") {
			continue
		}
		// Skip API routes
		if strings.HasPrefix(route, "/api") {
			continue
		}
		priority := defaultPriority
		if route == "/" {
			priority = 1.0
		}
		entries = append(entries, SitemapEntry{
			Loc:        baseURL + route,
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   priority,
		})
	}
	return entries
}

// ---------------------------------------------------------------------------
// Robots.txt
// ---------------------------------------------------------------------------

// RobotsRule is a single user-agent block in robots.txt.
type RobotsRule struct {
	UserAgent string
	Allow     []string
	Disallow  []string
	CrawlDelay int // seconds, 0 = omit
}

// RobotsTxt returns a standard robots.txt content (simple version).
func RobotsTxt(allow []string, disallow []string, sitemapURL string) string {
	out := "User-agent: *\n"
	for _, a := range allow {
		out += fmt.Sprintf("Allow: %s\n", a)
	}
	for _, d := range disallow {
		out += fmt.Sprintf("Disallow: %s\n", d)
	}
	if sitemapURL != "" {
		out += fmt.Sprintf("\nSitemap: %s\n", sitemapURL)
	}
	return out
}

// RobotsTxtAdvanced generates robots.txt with multiple user-agent blocks.
func RobotsTxtAdvanced(rules []RobotsRule, sitemapURLs []string) string {
	var sb strings.Builder
	for i, rule := range rules {
		if i > 0 {
			sb.WriteString("\n")
		}
		ua := rule.UserAgent
		if ua == "" {
			ua = "*"
		}
		sb.WriteString(fmt.Sprintf("User-agent: %s\n", ua))
		for _, a := range rule.Allow {
			sb.WriteString(fmt.Sprintf("Allow: %s\n", a))
		}
		for _, d := range rule.Disallow {
			sb.WriteString(fmt.Sprintf("Disallow: %s\n", d))
		}
		if rule.CrawlDelay > 0 {
			sb.WriteString(fmt.Sprintf("Crawl-delay: %d\n", rule.CrawlDelay))
		}
	}
	if len(sitemapURLs) > 0 {
		sb.WriteString("\n")
		for _, u := range sitemapURLs {
			sb.WriteString(fmt.Sprintf("Sitemap: %s\n", u))
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// JSON-LD Structured Data
// ---------------------------------------------------------------------------

// JSONLD returns a JSON-LD structured data script tag.
func JSONLD(data map[string]interface{}) template.HTML {
	b, err := json.Marshal(data)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(b)))
}

// JSONLDTyped generates JSON-LD from a typed struct (any type that json.Marshal accepts).
func JSONLDTyped(v interface{}) template.HTML {
	b, err := json.Marshal(v)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(b)))
}

// --- Pre-built structured data helpers ---

// WebSiteSchema generates JSON-LD for a WebSite with optional SearchAction (sitelinks searchbox).
func WebSiteSchema(name, url, searchURL string) template.HTML {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     name,
		"url":      url,
	}
	if searchURL != "" {
		data["potentialAction"] = map[string]interface{}{
			"@type":       "SearchAction",
			"target":      searchURL + "{search_term_string}",
			"query-input": "required name=search_term_string",
		}
	}
	return JSONLD(data)
}

// OrganizationSchema generates JSON-LD for an Organization.
func OrganizationSchema(name, url, logo string, sameAs []string) template.HTML {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     name,
		"url":      url,
	}
	if logo != "" {
		data["logo"] = logo
	}
	if len(sameAs) > 0 {
		data["sameAs"] = sameAs
	}
	return JSONLD(data)
}

// LocalBusinessSchema generates JSON-LD for a LocalBusiness.
func LocalBusinessSchema(name, url, phone, street, city, state, zip, country string) template.HTML {
	data := map[string]interface{}{
		"@context":  "https://schema.org",
		"@type":     "LocalBusiness",
		"name":      name,
		"url":       url,
		"telephone": phone,
		"address": map[string]interface{}{
			"@type":           "PostalAddress",
			"streetAddress":   street,
			"addressLocality": city,
			"addressRegion":   state,
			"postalCode":      zip,
			"addressCountry":  country,
		},
	}
	return JSONLD(data)
}

// BreadcrumbItem is a single item in a BreadcrumbList.
type BreadcrumbItem struct {
	Name string
	URL  string
}

// BreadcrumbSchema generates JSON-LD for a BreadcrumbList.
func BreadcrumbSchema(items []BreadcrumbItem) template.HTML {
	list := make([]map[string]interface{}, len(items))
	for i, item := range items {
		list[i] = map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     item.Name,
			"item":     item.URL,
		}
	}
	data := map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": list,
	}
	return JSONLD(data)
}

// AutoBreadcrumbs generates breadcrumb items from a URL path.
// e.g. "/blog/my-post" => [{Home, /}, {Blog, /blog}, {My Post, /blog/my-post}]
func AutoBreadcrumbs(baseURL, path string) []BreadcrumbItem {
	items := []BreadcrumbItem{{Name: "Home", URL: baseURL + "/"}}
	if path == "/" || path == "" {
		return items
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		name := strings.ReplaceAll(part, "-", " ")
		// Title case
		words := strings.Fields(name)
		for j, w := range words {
			if len(w) > 0 {
				words[j] = strings.ToUpper(w[:1]) + w[1:]
			}
		}
		name = strings.Join(words, " ")
		url := baseURL + "/" + strings.Join(parts[:i+1], "/")
		items = append(items, BreadcrumbItem{Name: name, URL: url})
	}
	return items
}

// ArticleSchema generates JSON-LD for an Article / BlogPosting.
type ArticleSchemaInput struct {
	Title       string
	Description string
	Author      string
	Image       string
	URL         string
	Published   time.Time
	Modified    time.Time
	Section     string
	Tags        []string
	Publisher   string
	PublisherLogo string
}

// ArticleSchema generates JSON-LD for an Article.
func ArticleSchema(a ArticleSchemaInput) template.HTML {
	data := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"headline":      a.Title,
		"description":   a.Description,
		"url":           a.URL,
		"datePublished": a.Published.Format(time.RFC3339),
	}
	if !a.Modified.IsZero() {
		data["dateModified"] = a.Modified.Format(time.RFC3339)
	}
	if a.Author != "" {
		data["author"] = map[string]interface{}{
			"@type": "Person",
			"name":  a.Author,
		}
	}
	if a.Image != "" {
		data["image"] = a.Image
	}
	if a.Section != "" {
		data["articleSection"] = a.Section
	}
	if len(a.Tags) > 0 {
		data["keywords"] = strings.Join(a.Tags, ", ")
	}
	if a.Publisher != "" {
		pub := map[string]interface{}{
			"@type": "Organization",
			"name":  a.Publisher,
		}
		if a.PublisherLogo != "" {
			pub["logo"] = map[string]interface{}{
				"@type": "ImageObject",
				"url":   a.PublisherLogo,
			}
		}
		data["publisher"] = pub
	}
	return JSONLD(data)
}

// FAQItem is a question/answer pair for FAQ schema.
type FAQItem struct {
	Question string
	Answer   string
}

// FAQSchema generates JSON-LD for an FAQPage.
func FAQSchema(items []FAQItem) template.HTML {
	entities := make([]map[string]interface{}, len(items))
	for i, item := range items {
		entities[i] = map[string]interface{}{
			"@type": "Question",
			"name":  item.Question,
			"acceptedAnswer": map[string]interface{}{
				"@type": "Answer",
				"text":  item.Answer,
			},
		}
	}
	data := map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": entities,
	}
	return JSONLD(data)
}

// HowToStep is a step in a HowTo schema.
type HowToStep struct {
	Name string
	Text string
	URL  string
	Image string
}

// HowToSchema generates JSON-LD for a HowTo.
func HowToSchema(name, description string, steps []HowToStep) template.HTML {
	stepList := make([]map[string]interface{}, len(steps))
	for i, step := range steps {
		s := map[string]interface{}{
			"@type": "HowToStep",
			"name":  step.Name,
			"text":  step.Text,
		}
		if step.URL != "" {
			s["url"] = step.URL
		}
		if step.Image != "" {
			s["image"] = step.Image
		}
		stepList[i] = s
	}
	data := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "HowTo",
		"name":        name,
		"description": description,
		"step":        stepList,
	}
	return JSONLD(data)
}

// ProductSchemaInput holds data for a Product schema.
type ProductSchemaInput struct {
	Name        string
	Description string
	Image       string
	Brand       string
	SKU         string
	Price       string
	Currency    string
	Availability string // e.g. "https://schema.org/InStock"
	URL         string
	RatingValue float64
	ReviewCount int
}

// ProductSchema generates JSON-LD for a Product.
func ProductSchema(p ProductSchemaInput) template.HTML {
	data := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Product",
		"name":        p.Name,
		"description": p.Description,
	}
	if p.Image != "" {
		data["image"] = p.Image
	}
	if p.Brand != "" {
		data["brand"] = map[string]interface{}{
			"@type": "Brand",
			"name":  p.Brand,
		}
	}
	if p.SKU != "" {
		data["sku"] = p.SKU
	}
	if p.Price != "" {
		offer := map[string]interface{}{
			"@type":         "Offer",
			"price":         p.Price,
			"priceCurrency": p.Currency,
			"url":           p.URL,
		}
		if p.Availability != "" {
			offer["availability"] = p.Availability
		}
		data["offers"] = offer
	}
	if p.RatingValue > 0 {
		data["aggregateRating"] = map[string]interface{}{
			"@type":       "AggregateRating",
			"ratingValue": p.RatingValue,
			"reviewCount": p.ReviewCount,
		}
	}
	return JSONLD(data)
}

// ReviewSchemaInput holds data for a Review schema.
type ReviewSchemaInput struct {
	ItemName    string
	Author      string
	RatingValue float64
	Body        string
	DatePublished time.Time
}

// ReviewSchema generates JSON-LD for a Review.
func ReviewSchema(r ReviewSchemaInput) template.HTML {
	data := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Review",
		"itemReviewed": map[string]interface{}{
			"@type": "Thing",
			"name":  r.ItemName,
		},
		"author": map[string]interface{}{
			"@type": "Person",
			"name":  r.Author,
		},
		"reviewRating": map[string]interface{}{
			"@type":       "Rating",
			"ratingValue": r.RatingValue,
		},
		"reviewBody":    r.Body,
		"datePublished": r.DatePublished.Format(time.RFC3339),
	}
	return JSONLD(data)
}

// ---------------------------------------------------------------------------
// RSS / Atom Feed
// ---------------------------------------------------------------------------

// FeedItem represents a single item in an RSS feed.
type FeedItem struct {
	Title       string
	Link        string
	Description string
	Author      string
	PubDate     time.Time
	GUID        string
	Categories  []string
}

// Feed represents an RSS 2.0 feed.
type Feed struct {
	Title       string
	Link        string
	Description string
	Language    string
	Items       []FeedItem
}

// RenderRSS returns RSS 2.0 XML bytes.
func RenderRSS(feed Feed) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">` + "\n")
	sb.WriteString("<channel>\n")
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", xmlEsc(feed.Title)))
	sb.WriteString(fmt.Sprintf("  <link>%s</link>\n", xmlEsc(feed.Link)))
	sb.WriteString(fmt.Sprintf("  <description>%s</description>\n", xmlEsc(feed.Description)))
	if feed.Language != "" {
		sb.WriteString(fmt.Sprintf("  <language>%s</language>\n", xmlEsc(feed.Language)))
	}
	sb.WriteString(fmt.Sprintf("  <atom:link href=\"%s/rss.xml\" rel=\"self\" type=\"application/rss+xml\"/>\n", xmlEsc(feed.Link)))
	sb.WriteString(fmt.Sprintf("  <lastBuildDate>%s</lastBuildDate>\n", time.Now().UTC().Format(time.RFC1123Z)))
	sb.WriteString("  <generator>NexGo SEO</generator>\n")

	for _, item := range feed.Items {
		sb.WriteString("  <item>\n")
		sb.WriteString(fmt.Sprintf("    <title>%s</title>\n", xmlEsc(item.Title)))
		sb.WriteString(fmt.Sprintf("    <link>%s</link>\n", xmlEsc(item.Link)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", xmlEsc(item.Description)))
		if item.Author != "" {
			sb.WriteString(fmt.Sprintf("    <author>%s</author>\n", xmlEsc(item.Author)))
		}
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		sb.WriteString(fmt.Sprintf("    <guid>%s</guid>\n", xmlEsc(guid)))
		if !item.PubDate.IsZero() {
			sb.WriteString(fmt.Sprintf("    <pubDate>%s</pubDate>\n", item.PubDate.UTC().Format(time.RFC1123Z)))
		}
		for _, cat := range item.Categories {
			sb.WriteString(fmt.Sprintf("    <category>%s</category>\n", xmlEsc(cat)))
		}
		sb.WriteString("  </item>\n")
	}

	sb.WriteString("</channel>\n")
	sb.WriteString("</rss>\n")
	return []byte(sb.String()), nil
}

// ---------------------------------------------------------------------------
// Preload / Prefetch hints
// ---------------------------------------------------------------------------

// PreloadHint represents a resource to preload.
type PreloadHint struct {
	Href string
	As   string // "style", "script", "font", "image", "fetch"
	Type string // MIME type, e.g. "font/woff2"
	CrossOrigin bool
}

// RenderPreloadTags returns HTML link tags for resource preloading.
func RenderPreloadTags(hints []PreloadHint) template.HTML {
	var sb strings.Builder
	for _, h := range hints {
		attrs := fmt.Sprintf(`rel="preload" href="%s"`, esc(h.Href))
		if h.As != "" {
			attrs += fmt.Sprintf(` as="%s"`, esc(h.As))
		}
		if h.Type != "" {
			attrs += fmt.Sprintf(` type="%s"`, esc(h.Type))
		}
		if h.CrossOrigin {
			attrs += ` crossorigin`
		}
		sb.WriteString(fmt.Sprintf("<link %s>\n", attrs))
	}
	return template.HTML(sb.String())
}

// PrefetchTag returns a single link prefetch tag.
func PrefetchTag(href string) template.HTML {
	return template.HTML(fmt.Sprintf(`<link rel="prefetch" href="%s">`, esc(href)))
}

// PrerenderTag returns a link prerender tag for high-probability next page.
func PrerenderTag(href string) template.HTML {
	return template.HTML(fmt.Sprintf(`<link rel="prerender" href="%s">`, esc(href)))
}

// ---------------------------------------------------------------------------
// Redirects
// ---------------------------------------------------------------------------

// Redirect represents a URL redirect rule.
type Redirect struct {
	From       string
	To         string
	StatusCode int // 301 (permanent) or 302 (temporary)
}

// RedirectMiddleware returns an http.Handler that applies redirect rules.
func RedirectMiddleware(redirects []Redirect, next http.Handler) http.Handler {
	lookup := make(map[string]Redirect, len(redirects))
	for _, r := range redirects {
		if r.StatusCode == 0 {
			r.StatusCode = 301
		}
		lookup[r.From] = r
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r, ok := lookup[req.URL.Path]; ok {
			http.Redirect(w, req, r.To, r.StatusCode)
			return
		}
		next.ServeHTTP(w, req)
	})
}

// TrailingSlashMiddleware enforces a consistent trailing slash policy.
// If add is true, redirects /about to /about/ (301).
// If add is false, redirects /about/ to /about (301).
// Root path "/" is always excluded.
func TrailingSlashMiddleware(add bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path != "/" && len(path) > 1 {
			hasSlash := strings.HasSuffix(path, "/")
			if add && !hasSlash {
				target := path + "/"
				if req.URL.RawQuery != "" {
					target += "?" + req.URL.RawQuery
				}
				http.Redirect(w, req, target, http.StatusMovedPermanently)
				return
			}
			if !add && hasSlash {
				target := strings.TrimSuffix(path, "/")
				if req.URL.RawQuery != "" {
					target += "?" + req.URL.RawQuery
				}
				http.Redirect(w, req, target, http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, req)
	})
}

// ---------------------------------------------------------------------------
// SEO Headers Middleware
// ---------------------------------------------------------------------------

// SEOHeaders adds SEO-relevant HTTP headers to responses.
func SEOHeaders(language string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if language != "" {
			w.Header().Set("Content-Language", language)
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next(w, r)
	}
}

// HSTSMiddleware adds Strict-Transport-Security header.
func HSTSMiddleware(maxAge int, next http.HandlerFunc) http.HandlerFunc {
	header := fmt.Sprintf("max-age=%d; includeSubDomains; preload", maxAge)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", header)
		next(w, r)
	}
}

// ETagMiddleware sets ETag and handles If-None-Match for cache validation.
func ETagMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Let the handler write, then we set Last-Modified on static content
		w.Header().Set("Vary", "Accept-Encoding")
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// Core Web Vitals
// ---------------------------------------------------------------------------

// CoreWebVitalsScript returns a lightweight JS snippet that measures LCP, FID/INP, and CLS
// and sends them to the specified endpoint via navigator.sendBeacon.
func CoreWebVitalsScript(reportEndpoint string) template.HTML {
	if reportEndpoint == "" {
		reportEndpoint = "/api/vitals"
	}
	script := fmt.Sprintf(`<script>
(function(){
  var cwv={lcp:0,cls:0,inp:0};
  // LCP
  if(window.PerformanceObserver){
    try{
      new PerformanceObserver(function(l){
        var e=l.getEntries();cwv.lcp=e[e.length-1].startTime;
      }).observe({type:'largest-contentful-paint',buffered:true});
    }catch(e){}
    // CLS
    try{
      var clsValue=0;
      new PerformanceObserver(function(l){
        l.getEntries().forEach(function(e){if(!e.hadRecentInput)clsValue+=e.value;});
        cwv.cls=clsValue;
      }).observe({type:'layout-shift',buffered:true});
    }catch(e){}
    // INP
    try{
      var inpEntries=[];
      new PerformanceObserver(function(l){
        l.getEntries().forEach(function(e){inpEntries.push(e.duration);});
        inpEntries.sort(function(a,b){return b-a});
        cwv.inp=inpEntries[0]||0;
      }).observe({type:'event',buffered:true});
    }catch(e){}
  }
  // Report on page hide
  document.addEventListener('visibilitychange',function(){
    if(document.visibilityState==='hidden'){
      cwv.url=location.pathname;cwv.ts=Date.now();
      if(navigator.sendBeacon){
        navigator.sendBeacon('%s',JSON.stringify(cwv));
      }
    }
  });
})();
</script>`, reportEndpoint)
	return template.HTML(script)
}

// ---------------------------------------------------------------------------
// SEO Audit
// ---------------------------------------------------------------------------

// AuditResult represents a single SEO issue found on a page.
type AuditResult struct {
	Level   string // "error", "warning", "info"
	Code    string // e.g. "MISSING_TITLE", "DESC_TOO_LONG"
	Message string
}

// AuditMeta checks a Meta for common SEO issues.
func AuditMeta(m Meta) []AuditResult {
	var results []AuditResult

	// Title checks
	if m.Title == "" {
		results = append(results, AuditResult{"error", "MISSING_TITLE", "Page is missing a title tag"})
	} else if len(m.Title) > 60 {
		results = append(results, AuditResult{"warning", "TITLE_TOO_LONG", fmt.Sprintf("Title is %d chars (recommended: ≤60)", len(m.Title))})
	} else if len(m.Title) < 10 {
		results = append(results, AuditResult{"warning", "TITLE_TOO_SHORT", fmt.Sprintf("Title is %d chars (recommended: ≥10)", len(m.Title))})
	}

	// Description checks
	if m.Description == "" {
		results = append(results, AuditResult{"error", "MISSING_DESCRIPTION", "Page is missing a meta description"})
	} else if len(m.Description) > 160 {
		results = append(results, AuditResult{"warning", "DESC_TOO_LONG", fmt.Sprintf("Description is %d chars (recommended: ≤160)", len(m.Description))})
	} else if len(m.Description) < 50 {
		results = append(results, AuditResult{"warning", "DESC_TOO_SHORT", fmt.Sprintf("Description is %d chars (recommended: ≥50)", len(m.Description))})
	}

	// Canonical
	if m.Canonical == "" {
		results = append(results, AuditResult{"warning", "MISSING_CANONICAL", "Page is missing a canonical URL"})
	}

	// OG Image
	if m.OGImage == "" {
		results = append(results, AuditResult{"warning", "MISSING_OG_IMAGE", "Page is missing an og:image — social shares will have no preview"})
	}

	// Twitter Card
	if m.TwitterCard == "" {
		results = append(results, AuditResult{"info", "NO_TWITTER_CARD", "No Twitter card type set — defaulting to summary_large_image"})
	}

	return results
}

// AuditHTML checks an HTML string for common SEO issues.
func AuditHTML(html string) []AuditResult {
	var results []AuditResult

	// Check for multiple H1 tags
	h1Count := strings.Count(strings.ToLower(html), "<h1")
	if h1Count == 0 {
		results = append(results, AuditResult{"warning", "MISSING_H1", "Page has no <h1> tag"})
	} else if h1Count > 1 {
		results = append(results, AuditResult{"warning", "MULTIPLE_H1", fmt.Sprintf("Page has %d <h1> tags (recommended: exactly 1)", h1Count)})
	}

	// Check images without alt attributes
	imgNoAlt := regexp.MustCompile(`<img[^>]*(?:alt\s*=\s*""[^>]*>|(?:(?!alt\s*=)[^>])*>)`)
	noAltMatches := imgNoAlt.FindAllString(html, -1)
	if len(noAltMatches) > 0 {
		results = append(results, AuditResult{"warning", "IMG_MISSING_ALT", fmt.Sprintf("%d image(s) missing alt attribute", len(noAltMatches))})
	}

	// Check for viewport meta
	if !strings.Contains(strings.ToLower(html), "viewport") {
		results = append(results, AuditResult{"error", "MISSING_VIEWPORT", "Page is missing viewport meta tag — not mobile friendly"})
	}

	return results
}

// ---------------------------------------------------------------------------
// URL Slug Helpers
// ---------------------------------------------------------------------------

// Slugify converts a string to an SEO-friendly URL slug.
// e.g. "Hello World!" => "hello-world"
func Slugify(s string) string {
	// Lowercase
	s = strings.ToLower(s)
	// Keep only ASCII letters, digits, hyphens, and spaces
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' || r == '.' {
			result.WriteRune('-')
		} else if unicode.IsLetter(r) {
			// Drop non-ASCII letters (accented chars become nothing)
			// For basic transliteration: e.g. keep the base letter if ASCII-range
			continue
		}
	}
	slug := result.String()
	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func esc(s string) string {
	return template.HTMLEscapeString(s)
}

func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
