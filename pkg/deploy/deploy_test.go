package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("myapp")
	if cfg.ProjectName != "myapp" {
		t.Errorf("expected myapp, got %s", cfg.ProjectName)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
	if cfg.Region != "auto" {
		t.Errorf("expected auto, got %s", cfg.Region)
	}
	if cfg.Memory != 256 {
		t.Errorf("expected 256MB, got %d", cfg.Memory)
	}
}

func TestGenerate_Docker(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformDocker

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "Dockerfile", ".dockerignore", "docker-compose.yml")

	dockerfile, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(string(dockerfile), "FROM golang:") {
		t.Error("expected golang base image")
	}
}

func TestGenerate_Vercel(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformVercel

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "vercel.json", "api/index.go")

	vercelJSON, _ := os.ReadFile(filepath.Join(dir, "vercel.json"))
	if !strings.Contains(string(vercelJSON), "@vercel/go") {
		t.Error("expected vercel go builder")
	}
}

func TestGenerate_Cloudflare(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformCloudflare

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "wrangler.toml", "build/worker.mjs")
}

func TestGenerate_AWSLambda(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformAWSLambda

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "template.yaml", "Makefile", "lambda_adapter.go")
}

func TestGenerate_Netlify(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformNetlify

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "netlify.toml")
}

func TestGenerate_Flyio(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformFlyio

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "fly.toml", "Procfile")
}

func TestGenerate_Railway(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformRailway

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "railway.toml", "nixpacks.toml")
}

func TestGenerate_Kubernetes(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = PlatformKubernetes

	err := Generate(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checkFiles(t, dir, "k8s/deployment.yaml")

	deploy, _ := os.ReadFile(filepath.Join(dir, "k8s/deployment.yaml"))
	content := string(deploy)
	if !strings.Contains(content, "kind: Deployment") {
		t.Error("expected Deployment kind")
	}
	if !strings.Contains(content, "kind: Service") {
		t.Error("expected Service kind")
	}
	if !strings.Contains(content, "kind: Ingress") {
		t.Error("expected Ingress kind")
	}
	if !strings.Contains(content, "kind: HorizontalPodAutoscaler") {
		t.Error("expected HPA kind")
	}
}

func TestGenerate_Unsupported(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig("myapp")
	cfg.Platform = "unsupported"

	err := Generate(dir, cfg)
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func TestBuildScript(t *testing.T) {
	cfg := DefaultConfig("myapp")

	cfg.Platform = PlatformDocker
	script := BuildScript(cfg)
	if !strings.Contains(script, "docker build") {
		t.Error("expected docker build command")
	}

	cfg.Platform = PlatformAWSLambda
	script = BuildScript(cfg)
	if !strings.Contains(script, "GOOS=linux") {
		t.Error("expected GOOS=linux")
	}

	cfg.Platform = PlatformCloudflare
	script = BuildScript(cfg)
	if !strings.Contains(script, "GOOS=js") {
		t.Error("expected GOOS=js")
	}

	cfg.Platform = PlatformVercel
	script = BuildScript(cfg)
	if !strings.Contains(script, "go build") {
		t.Error("expected go build")
	}
}

func TestGoVersion(t *testing.T) {
	v := goVersion()
	if v == "" {
		t.Error("expected non-empty go version")
	}
	if !strings.Contains(v, ".") {
		t.Errorf("expected version with dot, got %s", v)
	}
}

func TestModuleFromDir(t *testing.T) {
	dir := t.TempDir()
	goMod := "module github.com/test/app\n\ngo 1.22\n"
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)

	mod := moduleFromDir(dir)
	if mod != "github.com/test/app" {
		t.Errorf("expected github.com/test/app, got %s", mod)
	}
}

func TestModuleFromDir_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	mod := moduleFromDir(dir)
	if mod != "myapp" {
		t.Errorf("expected myapp fallback, got %s", mod)
	}
}

func checkFiles(t *testing.T, dir string, files ...string) {
	t.Helper()
	for _, f := range files {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s: %v", f, err)
		}
	}
}
