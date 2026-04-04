// Package asset provides CSS/JS minification, bundling, fingerprinting, and code splitting.
// Zero external dependencies — implements minification using Go stdlib.
package asset

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Config holds asset pipeline configuration.
type Config struct {
	SourceDir  string // "static" by default
	OutputDir  string // ".nexgo/assets"
	Minify     bool
	Fingerprint bool   // append content hash to filenames
	BundleCSS  bool   // combine CSS files
	BundleJS   bool   // combine JS files
}

// DefaultConfig returns sensible asset pipeline defaults.
func DefaultConfig() Config {
	return Config{
		SourceDir:   "static",
		OutputDir:   ".nexgo/assets",
		Minify:      true,
		Fingerprint: true,
		BundleCSS:   true,
		BundleJS:    true,
	}
}

// Pipeline processes static assets.
type Pipeline struct {
	config   Config
	rootDir  string
	mu       sync.RWMutex
	manifest map[string]string // original path -> fingerprinted path
}

// New creates an asset pipeline.
func New(cfg Config, rootDir string) *Pipeline {
	return &Pipeline{
		config:   cfg,
		rootDir:  rootDir,
		manifest: make(map[string]string),
	}
}

// Build processes all assets and returns a manifest.
func (p *Pipeline) Build() (*BuildResult, error) {
	start := time.Now()
	result := &BuildResult{
		Manifest: make(map[string]string),
	}

	srcDir := filepath.Join(p.rootDir, p.config.SourceDir)
	outDir := filepath.Join(p.rootDir, p.config.OutputDir)

	os.MkdirAll(outDir, 0755)

	// Collect files by type
	var cssFiles, jsFiles, otherFiles []string

	filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		ext := filepath.Ext(path)
		switch ext {
		case ".css":
			cssFiles = append(cssFiles, rel)
		case ".js":
			jsFiles = append(jsFiles, rel)
		default:
			otherFiles = append(otherFiles, rel)
		}
		return nil
	})

	// Process CSS
	if p.config.BundleCSS && len(cssFiles) > 0 {
		bundled, err := p.bundleFiles(srcDir, cssFiles, ".css")
		if err != nil {
			return nil, fmt.Errorf("bundling CSS: %w", err)
		}
		if p.config.Minify {
			bundled = MinifyCSS(bundled)
		}
		name := p.writeAsset(outDir, "bundle.css", []byte(bundled))
		for _, f := range cssFiles {
			result.Manifest["/static/"+filepath.ToSlash(f)] = "/assets/" + name
		}
		result.Manifest["/static/css/bundle.css"] = "/assets/" + name
		result.CSSBundled = len(cssFiles)
	} else {
		for _, f := range cssFiles {
			data, _ := os.ReadFile(filepath.Join(srcDir, f))
			content := string(data)
			if p.config.Minify {
				content = MinifyCSS(content)
			}
			name := p.writeAsset(outDir, f, []byte(content))
			result.Manifest["/static/"+filepath.ToSlash(f)] = "/assets/" + name
		}
	}

	// Process JS
	if p.config.BundleJS && len(jsFiles) > 0 {
		bundled, err := p.bundleFiles(srcDir, jsFiles, ".js")
		if err != nil {
			return nil, fmt.Errorf("bundling JS: %w", err)
		}
		if p.config.Minify {
			bundled = MinifyJS(bundled)
		}
		name := p.writeAsset(outDir, "bundle.js", []byte(bundled))
		for _, f := range jsFiles {
			result.Manifest["/static/"+filepath.ToSlash(f)] = "/assets/" + name
		}
		result.Manifest["/static/js/bundle.js"] = "/assets/" + name
		result.JSBundled = len(jsFiles)
	} else {
		for _, f := range jsFiles {
			data, _ := os.ReadFile(filepath.Join(srcDir, f))
			content := string(data)
			if p.config.Minify {
				content = MinifyJS(content)
			}
			name := p.writeAsset(outDir, f, []byte(content))
			result.Manifest["/static/"+filepath.ToSlash(f)] = "/assets/" + name
		}
	}

	// Copy other files
	for _, f := range otherFiles {
		data, _ := os.ReadFile(filepath.Join(srcDir, f))
		name := p.writeAsset(outDir, f, data)
		result.Manifest["/static/"+filepath.ToSlash(f)] = "/assets/" + name
		result.OtherCopied++
	}

	p.mu.Lock()
	p.manifest = result.Manifest
	p.mu.Unlock()

	result.Duration = time.Since(start)
	return result, nil
}

// Resolve maps an original asset path to its fingerprinted output path.
func (p *Pipeline) Resolve(originalPath string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if resolved, ok := p.manifest[originalPath]; ok {
		return resolved
	}
	return originalPath
}

// Manifest returns a copy of the full asset manifest.
func (p *Pipeline) Manifest() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := make(map[string]string, len(p.manifest))
	for k, v := range p.manifest {
		m[k] = v
	}
	return m
}

func (p *Pipeline) bundleFiles(srcDir string, files []string, ext string) (string, error) {
	var sb strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(srcDir, f))
		if err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("/* %s */\n", f))
		sb.Write(data)
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

func (p *Pipeline) writeAsset(outDir, name string, data []byte) string {
	if p.config.Fingerprint {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		hash := contentHash(data)
		name = base + "." + hash[:8] + ext
	}

	outPath := filepath.Join(outDir, name)
	os.MkdirAll(filepath.Dir(outPath), 0755)
	os.WriteFile(outPath, data, 0644)
	return name
}

// BuildResult holds asset build statistics.
type BuildResult struct {
	Manifest    map[string]string
	CSSBundled  int
	JSBundled   int
	OtherCopied int
	Duration    time.Duration
}

// --- CSS Minification ---

// MinifyCSS removes comments, whitespace, and optimizes CSS.
func MinifyCSS(css string) string {
	// Remove multi-line comments
	re := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	css = re.ReplaceAllString(css, "")

	// Remove newlines and extra whitespace
	css = regexp.MustCompile(`\s+`).ReplaceAllString(css, " ")

	// Remove spaces around special characters
	for _, ch := range []string{"{", "}", ":", ";", ",", ">", "~", "+"} {
		css = strings.ReplaceAll(css, " "+ch, ch)
		css = strings.ReplaceAll(css, ch+" ", ch)
	}

	// Remove trailing semicolons before closing braces
	css = strings.ReplaceAll(css, ";}", "}")

	// Remove empty rules
	css = regexp.MustCompile(`[^{}]+\{\s*\}`).ReplaceAllString(css, "")

	return strings.TrimSpace(css)
}

// --- JS Minification ---

// MinifyJS removes comments and unnecessary whitespace from JavaScript.
func MinifyJS(js string) string {
	// Remove single-line comments (but not URLs with //)
	js = regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(js, "")
	js = regexp.MustCompile(`\s+//[^'"]*$`).ReplaceAllString(js, "")

	// Remove multi-line comments
	js = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(js, "")

	// Collapse whitespace (preserve strings)
	lines := strings.Split(js, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	js = strings.Join(result, "\n")

	// Remove blank lines
	js = regexp.MustCompile(`\n{2,}`).ReplaceAllString(js, "\n")

	return strings.TrimSpace(js)
}

// --- HTML Minification ---

// MinifyHTML removes comments and extra whitespace from HTML.
func MinifyHTML(html string) string {
	// Remove HTML comments
	html = regexp.MustCompile(`<!--[\s\S]*?-->`).ReplaceAllString(html, "")

	// Collapse whitespace between tags
	html = regexp.MustCompile(`>\s+<`).ReplaceAllString(html, "> <")

	// Collapse internal whitespace
	html = regexp.MustCompile(`\s{2,}`).ReplaceAllString(html, " ")

	return strings.TrimSpace(html)
}

// --- Helpers ---

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// InlineCSS reads a CSS file and returns it as a <style> tag.
func InlineCSS(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	minified := MinifyCSS(string(data))
	return "<style>" + minified + "</style>"
}

// InlineJS reads a JS file and returns it as a <script> tag.
func InlineJS(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	minified := MinifyJS(string(data))
	return "<script>" + minified + "</script>"
}

// CriticalCSS extracts CSS rules matching a set of selectors (above-the-fold).
func CriticalCSS(css string, selectors []string) string {
	var result strings.Builder
	selectorSet := make(map[string]bool, len(selectors))
	for _, s := range selectors {
		selectorSet[s] = true
	}

	// Simple rule extraction
	re := regexp.MustCompile(`([^{}]+)\{([^}]*)\}`)
	matches := re.FindAllStringSubmatch(css, -1)
	for _, m := range matches {
		selector := strings.TrimSpace(m[1])
		for s := range selectorSet {
			if strings.Contains(selector, s) {
				result.WriteString(selector)
				result.WriteString("{")
				result.WriteString(strings.TrimSpace(m[2]))
				result.WriteString("}")
			}
		}
	}

	return result.String()
}
