---
sidebar_position: 2
title: Building
---

# Building

This page covers everything needed to build KubeVision locally — from individual layer builds to producing a release binary.

## Prerequisites

| Requirement | Minimum version | Install |
|-------------|-----------------|---------|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/) |
| pnpm | 8+ | `npm install -g pnpm` |
| Docker | 24+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Make | any | OS package manager |

Verify your environment:

```bash
go version    # go version go1.23.x ...
node -v       # v18.x.x or higher
pnpm -v       # 8.x.x or higher
docker --version
```

## Backend Build

```bash
# Download dependencies
go mod tidy

# Compile to ./bin/kubevision
make build

# Run directly (with hot-reload via Air)
make dev
```

The `make build` target produces a statically linked binary at `./bin/kubevision`. The server starts on `:8080` by default.

## Frontend Build

```bash
cd web

# Install dependencies
pnpm install

# Production build → outputs to web/dist/
pnpm build

# Development server with HMR (proxies /api to :8080)
pnpm dev
```

The production build emits static assets into `web/dist/`. These assets are embedded into the Go binary in embedded mode (see below).

## Embedded Mode (Single Binary)

KubeVision uses `go:embed` to bundle the compiled frontend assets directly into the backend binary, producing a single self-contained executable.

```go
//go:embed web/dist
var staticFS embed.FS
```

To build the full embedded binary:

```bash
# 1. Build frontend first
cd web && pnpm install && pnpm build && cd ..

# 2. Build the embedded Go binary
make build-embed
```

The resulting `./bin/kubevision` serves both the API and the static UI with no external file dependency.

## Docker Build

```bash
# Build image using the multi-stage Dockerfile
docker build -f deploy/Dockerfile -t kubevision:latest .

# Run with a local kubeconfig
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  kubevision:latest
```

The `deploy/Dockerfile` uses a two-stage build: a Go builder stage compiles the binary (with embedded frontend), and a minimal `distroless` image is used as the final runtime layer.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile backend binary to `./bin/kubevision` |
| `make build-embed` | Build frontend then compile embedded binary |
| `make dev` | Start backend with live reload (Air) |
| `make lint` | Run `golangci-lint` across all packages |
| `make test` | Run all Go tests |
| `make test-cover` | Run tests with HTML coverage report |
| `make clean` | Remove `./bin` and build artifacts |
| `make docker` | Build Docker image tagged `kubevision:latest` |

## Cross-Compilation

Go's cross-compilation support makes it straightforward to target any platform:

```bash
# Linux amd64 (common server target)
GOOS=linux GOARCH=amd64 make build

# Linux arm64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 make build

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 make build

# Windows amd64
GOOS=windows GOARCH=amd64 make build
```

:::tip
When cross-compiling for release, always build the embedded binary (`make build-embed`) so the frontend assets are included in the cross-compiled output.
:::

## Related

- [Contributing](/docs/development/contributing) — Code style and PR workflow
- [Testing](/docs/development/testing) — Run the test suite after building
