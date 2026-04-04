// Package deploy provides deployment adapters for Vercel, Cloudflare Workers,
// AWS Lambda, Netlify, Fly.io, Railway, and Docker.
package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform identifies a deployment target.
type Platform string

const (
	PlatformVercel      Platform = "vercel"
	PlatformCloudflare  Platform = "cloudflare"
	PlatformAWSLambda   Platform = "aws-lambda"
	PlatformNetlify     Platform = "netlify"
	PlatformFlyio       Platform = "fly"
	PlatformRailway     Platform = "railway"
	PlatformDocker      Platform = "docker"
	PlatformKubernetes  Platform = "kubernetes"
)

// Config holds deployment configuration.
type Config struct {
	Platform    Platform
	ProjectName string
	Port        int
	Region      string
	MinInstances int
	MaxInstances int
	Memory       int    // MB
	EnvVars      map[string]string
	GoVersion    string
	BinaryName   string
}

// DefaultConfig returns deployment defaults.
func DefaultConfig(name string) Config {
	return Config{
		ProjectName:  name,
		Port:         3000,
		Region:       "auto",
		MinInstances: 1,
		MaxInstances: 10,
		Memory:       256,
		EnvVars:      make(map[string]string),
		GoVersion:    goVersion(),
		BinaryName:   name,
	}
}

// Generate creates all deployment files for a platform.
func Generate(dir string, cfg Config) error {
	switch cfg.Platform {
	case PlatformDocker:
		return generateDocker(dir, cfg)
	case PlatformVercel:
		return generateVercel(dir, cfg)
	case PlatformCloudflare:
		return generateCloudflare(dir, cfg)
	case PlatformAWSLambda:
		return generateLambda(dir, cfg)
	case PlatformNetlify:
		return generateNetlify(dir, cfg)
	case PlatformFlyio:
		return generateFlyio(dir, cfg)
	case PlatformRailway:
		return generateRailway(dir, cfg)
	case PlatformKubernetes:
		return generateKubernetes(dir, cfg)
	default:
		return fmt.Errorf("unsupported platform: %s", cfg.Platform)
	}
}

// --- Docker ---

func generateDocker(dir string, cfg Config) error {
	dockerfile := fmt.Sprintf(`# Build stage
FROM golang:%s-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /%s ./

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /%s .
COPY pages/ ./pages/
COPY layouts/ ./layouts/
COPY components/ ./components/
COPY static/ ./static/
COPY locales/ ./locales/
COPY nexgo.config.json ./
COPY .env* ./

EXPOSE %d
ENV NEXGO_MODE=production
CMD ["./%s", "start", "--port", "%d"]
`, cfg.GoVersion, cfg.BinaryName, cfg.BinaryName, cfg.Port, cfg.BinaryName, cfg.Port)

	dockerignore := `# Dependencies
vendor/
node_modules/

# Build output
.nexgo/
*.exe
*.test
*.out

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Git
.git/
.gitignore

# Docker
Dockerfile
docker-compose.yml
.dockerignore
`

	compose := fmt.Sprintf(`version: '3.8'

services:
  app:
    build: .
    ports:
      - "%d:%d"
    environment:
      - NEXGO_MODE=production
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:%d/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
    deploy:
      resources:
        limits:
          memory: %dM
`, cfg.Port, cfg.Port, cfg.Port, cfg.Memory)

	files := map[string]string{
		"Dockerfile":          dockerfile,
		".dockerignore":       dockerignore,
		"docker-compose.yml":  compose,
	}

	return writeFiles(dir, files)
}

// --- Vercel ---

func generateVercel(dir string, cfg Config) error {
	vercelJSON := fmt.Sprintf(`{
  "version": 2,
  "builds": [
    {
      "src": "main.go",
      "use": "@vercel/go"
    }
  ],
  "routes": [
    { "src": "/static/(.*)", "dest": "/static/$1" },
    { "src": "/(.*)", "dest": "/" }
  ],
  "env": {
    "NEXGO_MODE": "production"
  },
  "regions": ["%s"]
}`, cfg.Region)

	// Vercel serverless adapter
	apiHandler := `package handler

import (
	"net/http"
	"os"
	"sync"

	"` + moduleFromDir(dir) + `/pkg/config"
	"` + moduleFromDir(dir) + `/pkg/server"
)

var (
	srv  *server.Server
	once sync.Once
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		cfg, _ := config.Load(".")
		cfg.DevMode = false
		port := os.Getenv("PORT")
		if port != "" {
			fmt.Sscan(port, &cfg.Port)
		}
		srv, _ = server.New(cfg)
	})
	srv.ServeHTTP(w, r)
}
`

	files := map[string]string{
		"vercel.json":    vercelJSON,
		"api/index.go":   apiHandler,
	}

	return writeFiles(dir, files)
}

// --- Cloudflare Workers ---

func generateCloudflare(dir string, cfg Config) error {
	wranglerToml := fmt.Sprintf(`name = "%s"
main = "build/worker.mjs"
compatibility_date = "2024-01-01"

[build]
command = "GOOS=js GOARCH=wasm go build -o build/app.wasm . && cp $(go env GOROOT)/misc/wasm/wasm_exec.js build/"

[vars]
NEXGO_MODE = "production"

[[routes]]
pattern = "%s.workers.dev/*"
`, cfg.ProjectName, cfg.ProjectName)

	workerJS := `// Cloudflare Workers entry point for NexGo
// Requires Go WASM build
import "./wasm_exec.js";

export default {
  async fetch(request, env, ctx) {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(
      fetch("./app.wasm"),
      go.importObject
    );
    go.run(result.instance);

    // Forward request to Go handler
    const url = new URL(request.url);
    const resp = await globalThis.__nexgo_handle(
      request.method,
      url.pathname + url.search,
      Object.fromEntries(request.headers),
      await request.text()
    );

    return new Response(resp.body, {
      status: resp.status,
      headers: resp.headers,
    });
  },
};
`

	files := map[string]string{
		"wrangler.toml":     wranglerToml,
		"build/worker.mjs":  workerJS,
	}

	return writeFiles(dir, files)
}

// --- AWS Lambda ---

func generateLambda(dir string, cfg Config) error {
	samTemplate := fmt.Sprintf(`AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: %s NexGo Application

Globals:
  Function:
    Timeout: 30
    MemorySize: %d
    Runtime: provided.al2
    Architectures:
      - arm64
    Environment:
      Variables:
        NEXGO_MODE: production

Resources:
  NexGoFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      CodeUri: .
      Events:
        CatchAll:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: ANY
        Root:
          Type: Api
          Properties:
            Path: /
            Method: ANY
    Metadata:
      BuildMethod: makefile

Outputs:
  ApiEndpoint:
    Description: API Gateway endpoint URL
    Value: !Sub "https://${ServerlessRestApi}.execute-api.${AWS::Region}.amazonaws.com/Prod/"
`, cfg.ProjectName, cfg.Memory)

	makefile := fmt.Sprintf(`build-%s:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap .
`, "NexGoFunction")

	// Lambda adapter
	lambdaMain := `//go:build lambda
// +build lambda

package main

import (
	"context"
	"net/http"

	"github.com/salmanfaris22/nexgo/pkg/config"
	"github.com/salmanfaris22/nexgo/pkg/server"
)

// LambdaAdapter wraps NexGo server for AWS Lambda
type LambdaAdapter struct {
	srv *server.Server
}

func NewLambdaAdapter() (*LambdaAdapter, error) {
	cfg, err := config.Load(".")
	if err != nil {
		return nil, err
	}
	cfg.DevMode = false
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}
	return &LambdaAdapter{srv: srv}, nil
}
`

	files := map[string]string{
		"template.yaml":    samTemplate,
		"Makefile":          makefile,
		"lambda_adapter.go": lambdaMain,
	}

	return writeFiles(dir, files)
}

// --- Netlify ---

func generateNetlify(dir string, cfg Config) error {
	netlifyToml := fmt.Sprintf(`[build]
  command = "go build -ldflags='-s -w' -o functions/nexgo ."
  functions = "functions"
  publish = "static"

[build.environment]
  GO_VERSION = "%s"
  NEXGO_MODE = "production"

[[redirects]]
  from = "/api/*"
  to = "/.netlify/functions/nexgo/:splat"
  status = 200

[[redirects]]
  from = "/*"
  to = "/.netlify/functions/nexgo/:splat"
  status = 200

[functions]
  node_bundler = "esbuild"
  included_files = ["pages/**", "layouts/**", "components/**", "static/**", "locales/**", "nexgo.config.json"]
`, cfg.GoVersion)

	files := map[string]string{
		"netlify.toml": netlifyToml,
	}

	return writeFiles(dir, files)
}

// --- Fly.io ---

func generateFlyio(dir string, cfg Config) error {
	flyToml := fmt.Sprintf(`app = "%s"
primary_region = "%s"
kill_signal = "SIGINT"
kill_timeout = "5s"

[build]
  builder = "paketobuildpacks/builder:base"

[env]
  PORT = "%d"
  NEXGO_MODE = "production"

[http_service]
  internal_port = %d
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = %d
  processes = ["app"]

[[http_service.checks]]
  grace_period = "5s"
  interval = "30s"
  method = "GET"
  path = "/health"
  timeout = "5s"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = %d
`, cfg.ProjectName, cfg.Region, cfg.Port, cfg.Port, cfg.MinInstances, cfg.Memory)

	procfile := fmt.Sprintf("web: ./%s start --port $PORT\n", cfg.BinaryName)

	files := map[string]string{
		"fly.toml": flyToml,
		"Procfile": procfile,
	}

	return writeFiles(dir, files)
}

// --- Railway ---

func generateRailway(dir string, cfg Config) error {
	railwayToml := fmt.Sprintf(`[build]
builder = "nixpacks"
buildCommand = "go build -ldflags='-s -w' -o %s ."

[deploy]
startCommand = "./%s start --port $PORT"
healthcheckPath = "/health"
healthcheckTimeout = 5
restartPolicyType = "on_failure"
restartPolicyMaxRetries = 3
`, cfg.BinaryName, cfg.BinaryName)

	nixpacksToml := fmt.Sprintf(`[phases.setup]
nixPkgs = ["go_%s"]

[phases.build]
cmds = ["go build -ldflags='-s -w' -o %s ."]

[start]
cmd = "./%s start --port $PORT"
`, strings.ReplaceAll(cfg.GoVersion, ".", "_"), cfg.BinaryName, cfg.BinaryName)

	files := map[string]string{
		"railway.toml":  railwayToml,
		"nixpacks.toml": nixpacksToml,
	}

	return writeFiles(dir, files)
}

// --- Kubernetes ---

func generateKubernetes(dir string, cfg Config) error {
	deployment := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  labels:
    app: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
        - name: %s
          image: %s:latest
          ports:
            - containerPort: %d
          env:
            - name: NEXGO_MODE
              value: "production"
            - name: PORT
              value: "%d"
          resources:
            requests:
              memory: "%dMi"
              cpu: "100m"
            limits:
              memory: "%dMi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /live
              port: %d
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: %d
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: %s
  ports:
    - port: 80
      targetPort: %d
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: %s.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: %s
                port:
                  number: 80
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: %d
  maxReplicas: %d
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
`,
		cfg.ProjectName, cfg.ProjectName, cfg.MinInstances,
		cfg.ProjectName, cfg.ProjectName, cfg.ProjectName, cfg.ProjectName,
		cfg.Port, cfg.Port, cfg.Memory, cfg.Memory*2, cfg.Port, cfg.Port,
		cfg.ProjectName, cfg.ProjectName, cfg.Port,
		cfg.ProjectName, cfg.ProjectName, cfg.ProjectName,
		cfg.ProjectName, cfg.ProjectName, cfg.MinInstances, cfg.MaxInstances,
	)

	files := map[string]string{
		"k8s/deployment.yaml": deployment,
	}

	return writeFiles(dir, files)
}

// --- Helpers ---

func writeFiles(dir string, files map[string]string) error {
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
	return nil
}

func goVersion() string {
	v := runtime.Version()
	v = strings.TrimPrefix(v, "go")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return "1.22"
}

func moduleFromDir(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "myapp"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "myapp"
}

// --- Build helpers for serverless ---

// BuildScript generates a build script for the target platform.
func BuildScript(cfg Config) string {
	switch cfg.Platform {
	case PlatformDocker:
		return "docker build -t " + cfg.ProjectName + " ."
	case PlatformAWSLambda:
		return "GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags='-s -w' -tags lambda.norpc -o bootstrap ."
	case PlatformCloudflare:
		return "GOOS=js GOARCH=wasm go build -o build/app.wasm ."
	default:
		return fmt.Sprintf("CGO_ENABLED=0 go build -ldflags='-s -w' -o %s .", cfg.BinaryName)
	}
}
