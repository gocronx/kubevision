<div align="center">

<img src="docs/assets/logo.png" width="120" alt="KubeVision Logo">
<!-- TODO: Replace with actual logo -->

# KubeVision

A modern, real-time Kubernetes dashboard

**Kube + Vision — See your clusters clearly, act on them instantly.**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev) [![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev) [![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://typescriptlang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)

**English** &nbsp;&nbsp;|&nbsp;&nbsp; [中文](./README_CN.md)

<br>

<!-- TODO: Replace with actual animated GIF showing a 15-20s workflow -->
<!-- <img src="docs/assets/demo.gif" width="800" alt="KubeVision Demo"> -->
<img src="docs/assets/screenshot-overview.png" width="800" alt="KubeVision Dashboard">
<!-- TODO: Replace with actual screenshot -->

</div>

<br>

## Why KubeVision?

<table>
<tr>
<td width="33%" align="center">

**Real-time Architecture**

Informer Watch → WebSocket Push.

Sub-second updates, zero polling.

</td>
<td width="33%" align="center">

**Fully Extensible**

1 line of Go + 1 config block per resource.

CRDs auto-discovered at runtime.

</td>
<td width="33%" align="center">

**Production Ready**

Multi-cluster, RBAC, 2FA, audit logging.

Built for teams, not just demos.

</td>
</tr>
</table>

<br>

## Screenshots

<!-- TODO: Replace with actual screenshots -->

<table>
<tr>
<td><img src="docs/assets/screenshot-overview.png" alt="Cluster Overview"></td>
<td><img src="docs/assets/screenshot-terminal.png" alt="Pod Terminal"></td>
</tr>
<tr>
<td><img src="docs/assets/screenshot-topology.png" alt="Resource Topology"></td>
<td><img src="docs/assets/screenshot-diff.png" alt="Dry-Run Diff"></td>
</tr>
<tr>
<td><img src="docs/assets/screenshot-search.png" alt="Global Search"></td>
<td><img src="docs/assets/screenshot-dark.png" alt="Dark Mode"></td>
</tr>
</table>

<br>

## Features

- :zap: **Real-time Sync** — Informer → WebSocket, sub-second updates
- :globe_with_meridians: **Multi-cluster** — Single dashboard for all clusters
- :computer: **Pod Terminal & Logs** — Session recording & playback
- :rocket: **Deployment Ops** — Scale, restart, rollback
- :eyes: **Dry-Run Diff** — Preview changes before applying
- :twisted_rightwards_arrows: **Cross-cluster Diff** — Spot configuration drift
- :world_map: **Resource Topology** — Visual ownership graph
- :mag: **Global Search** — `Cmd+K` fuzzy search
- :keyboard: **kubectl Hints** — Auto-generated commands
- :jigsaw: **Resource Templates** — One-click deploy
- :shield: **RBAC** — 5 built-in roles + custom
- :key: **2FA (TOTP)** — QR setup + recovery codes
- :memo: **Audit Logging** — Async writes + retention
- :bell: **Webhooks** — Slack, Discord, custom endpoints
- :package: **26+ Resources** — 8 cached + 18 on-demand + CRD
- :zap: **Batch Ops** — Multi-select delete & restart
- :earth_africa: **i18n** — English & Chinese
- :crescent_moon: **Dark Mode** — System-aware themes

<br>

## Quick Start

### Helm (recommended)

```bash
helm install kubevision deploy/helm/kubevision
```

### Docker

```bash
docker build -f deploy/Dockerfile -t kubevision:latest .
docker run -p 8080:8080 kubevision:latest
```

> Open http://localhost:8080
>
> Default login: `admin` / `admin123`

### Development

```bash
git clone https://github.com/kubevision/kubevision.git
cd kubevision

# Backend — :8080
go mod tidy && make dev

# Frontend — :5173, proxies /api → :8080
cd web && pnpm install && pnpm dev
```

<br>

## Architecture

```
                    Browser
            ┌────────┴────────┐
            REST          WebSocket
            │                 │
┌───────────▼─────────────────▼──────────────┐
│  Middleware: RequestID → Logger → Auth       │
│                                             │
│  Handler ──→ Service ──→ K8sRepo            │
│                            │                │
│                 ┌──────────┴──────────┐     │
│                 │                     │     │
│          Informer Cache         API Server  │
│                 │                            │
│     Informer Watch ──→ EventListener        │
│                            │                │
│                        WS Hub ──→ Browser   │
│                                             │
│  DB: SQLite (dev) / PostgreSQL (prod)       │
└─────────────────────────────────────────────┘
```

> **Read**: Informer cache first, fallback to API Server
>
> **Write**: API Server → Informer → WS Hub → browser

<br>

## Built With

1. [Gin](https://github.com/gin-gonic/gin) — HTTP framework
2. [client-go](https://github.com/kubernetes/client-go) — Kubernetes API client & Informer
3. [GORM](https://gorm.io) — ORM (SQLite + PostgreSQL)
4. [shadcn/ui](https://ui.shadcn.com) — Component library built on [Radix UI](https://radix-ui.com)
5. [TanStack Query](https://tanstack.com/query) — Server state management & cache
6. [xterm.js](https://xtermjs.org) — Pod terminal emulator

<br>

## Configuration

```yaml
server:
  port: 8080

database:
  driver: sqlite           # sqlite | postgres
  dsn: kubevision.db

auth:
  jwt_secret: ""           # auto-generated if empty
  access_token_ttl: 15m
  refresh_token_ttl: 168h

kubernetes:
  kubeconfig: ""           # empty = in-cluster
  informer_resync: 30m
```

| Variable | Description |
|----------|-------------|
| `KUBEVISION_SERVER_PORT` | HTTP port (default: 8080) |
| `KUBEVISION_DB_DRIVER` | `sqlite` or `postgres` |
| `KUBEVISION_DB_DSN` | Database connection string |
| `KUBEVISION_JWT_SECRET` | JWT signing secret |
| `KUBECONFIG` | Path to kubeconfig file |

<br>

## Documentation

| | |
|---|---|
| [Architecture](docs/ARCHITECTURE.md) | System design, data flow, component interactions |
| [Design](docs/DESIGN.md) | Technical deep-dive and implementation details |
| [Comparison](docs/COMPARISON.md) | How KubeVision compares to alternative approaches |

<br>

## Contributing

Contributions are welcome! Please open an issue first to discuss what you would like to change.

<!-- ## Contributors -->
<!-- <a href="https://github.com/kubevision/kubevision/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=kubevision/kubevision" />
</a> -->
