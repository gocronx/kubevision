---
sidebar_position: 1
title: Installation
---

# Installation

KubeVision can be deployed via Helm, Docker, or from source.

## Prerequisites

- Kubernetes cluster (v1.24+)
- `kubectl` configured with cluster access
- For development: Go 1.23+, Node.js 18+, pnpm

## Helm (Recommended)

```bash
helm repo add kubevision https://kubevision.github.io/charts
helm repo update

helm install kubevision kubevision/kubevision \
  --namespace kubevision \
  --create-namespace
```

### Custom Values

```yaml
# values.yaml
replicaCount: 1
image:
  repository: kubevision/kubevision
  tag: latest

service:
  type: ClusterIP
  port: 8080

database:
  driver: sqlite     # sqlite | postgres
  dsn: kubevision.db

auth:
  jwtSecret: ""      # auto-generated if empty
```

```bash
helm install kubevision kubevision/kubevision -f values.yaml
```

## Docker

```bash
# Build
docker build -f deploy/Dockerfile -t kubevision:latest .

# Run with local kubeconfig
docker run -p 8080:8080 \
  -v ~/.kube/config:/root/.kube/config:ro \
  kubevision:latest
```

## From Source

```bash
git clone https://github.com/kubevision/kubevision.git
cd kubevision

# Backend
go mod tidy
make dev    # starts on :8080

# Frontend (new terminal)
cd web
pnpm install
pnpm dev    # starts on :5173, proxies /api → :8080
```

## Verify Installation

Open `http://localhost:8080` in your browser.

Default credentials:
- **Username:** `admin`
- **Password:** `admin123`

:::warning
Change the default password immediately after first login.
:::

## Next Steps

- [Quick Start](/docs/getting-started/quick-start) — Add your first cluster
- [Configuration](/docs/getting-started/configuration) — Customize settings
