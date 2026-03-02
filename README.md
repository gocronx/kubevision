<div align="center">

# KubeVision

**A modern, real-time Kubernetes dashboard**

*Kube + Vision — See your clusters clearly, act on them instantly.*

Go + React + shadcn/ui &nbsp;|&nbsp; Informer Cache + WebSocket Push &nbsp;|&nbsp; 26 Resource Types

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev) [![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev) [![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://typescriptlang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)

<br>

**English** &nbsp;&nbsp;|&nbsp;&nbsp; [中文](./README_CN.md)

<br>

<!-- TODO: Replace with actual screenshot -->
<!-- <img src="docs/screenshot.png" width="800" alt="KubeVision Dashboard"> -->

</div>

<br>

## 🚀 Why KubeVision?

Most K8s dashboards either require **hundreds of lines of boilerplate per resource type**, or skip real-time updates entirely. KubeVision takes a different approach:

### 🧩 Zero-boilerplate Resources

Add a new K8s resource type with **1 line** of Go code and **1 config block** in the frontend. No new files, no duplicated handlers.

```go
// That's it. One line.
r.resources["cronjobs"] = Meta{
  Name: "cronjobs",
  GVR:  schema.GroupVersionResource{...},
  ...
}
```

### ⚡ True Real-time

Informer Watch detects changes → WebSocket Hub pushes to browser → TanStack Query cache auto-invalidates. **Sub-second updates, zero polling.**

```
K8s API ──→ Informer ──→ WS Hub ──→ Browser
  write         watch        push     auto-refresh
```

<br>

## ✨ Features

- 🔄 **Real-time Updates** — Informer → WebSocket → auto-refresh
- 📦 **26 Resource Types** — 8 cached + 18 on-demand, one unified handler
- 🌐 **Multi-cluster** — Manage all clusters from one dashboard
- 🔐 **JWT + Token Revocation** — Access/Refresh tokens, instant revocation via version
- 🌙 **Dark Mode** — System-aware, one-click toggle
- 🌍 **i18n Ready** — English & Chinese built-in
- ⚙️ **Generic CRUD** — List, Get, Create, Update, Delete, Patch
- 🏷️ **Namespace Filtering** — Quick namespace switch with dropdown
- 🛡️ **RBAC (5 Roles)** — admin / ops / dev / readonly / custom

<br>

## 🏗️ Architecture

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

> **Read path**: Informer cache first, fallback to API Server on miss
>
> **Write path**: API Server → Informer Watch → WS Hub → browser auto-refresh

<br>

## ⚡ Quick Start

### Prerequisites

| Dependency | Version |
|-----------|---------|
| Go | 1.23+ |
| Node.js | 20+ |
| pnpm | 9+ |
| Kubernetes | 1.28+ (kubeconfig) |

### Development

```bash
git clone https://github.com/kubevision/kubevision.git
cd kubevision

# Backend — starts on :8080
go mod tidy && make dev

# Frontend — starts on :5173, proxies /api → :8080
cd web && pnpm install && pnpm dev
```

> Default login: **admin** / **admin123**

### Docker

```bash
docker build -f deploy/Dockerfile -t kubevision:latest .
docker run -p 8080:8080 kubevision:latest
```

<br>

## 🛠️ Tech Stack

| | Technology | Purpose |
|---|-----------|---------|
| **Backend** | Go 1.23 &middot; Gin &middot; GORM &middot; client-go &middot; zap | API server, K8s integration |
| **Frontend** | React 19 &middot; TypeScript &middot; Vite 7 &middot; shadcn/ui &middot; TanStack Query v5 | Modern UI with real-time data |
| **Database** | SQLite / PostgreSQL | User, cluster, audit persistence |
| **Auth** | JWT (15min access + 7d refresh) &middot; bcrypt | Token revocation via version check |
| **Real-time** | Informer &middot; gorilla/websocket | Sub-second push to browser |

<br>

## 📋 Supported Resources

<table>
<tr>
<td><b>Cached</b> (Informer, real-time push)</td>
<td>

`Pods` `Deployments` `StatefulSets` `DaemonSets` `ReplicaSets` `Services` `Nodes` `Namespaces`

</td>
</tr>
<tr>
<td><b>On-demand</b> (API Server query)</td>
<td>

`Jobs` `CronJobs` `Ingresses` `ConfigMaps` `Secrets` `Events` `PVs` `PVCs` `StorageClasses` `ServiceAccounts` `Roles` `ClusterRoles` `RoleBindings` `ClusterRoleBindings` `HPAs` `Endpoints` `NetworkPolicies`

</td>
</tr>
</table>

<br>

## 📡 API

```
Auth
  POST   /api/v1/auth/login                          Login
  POST   /api/v1/auth/refresh                        Refresh token
  GET    /api/v1/users/me                             Current user

Clusters
  GET    /api/v1/clusters                             List clusters
  POST   /api/v1/clusters                             Add cluster
  DELETE /api/v1/clusters/:id                         Remove cluster

Resources (generic CRUD)
  GET    /api/v1/clusters/:id/resources/:res          List
  GET    /api/v1/clusters/:id/resources/:res/:name    Get
  POST   /api/v1/clusters/:id/resources/:res          Create
  PUT    /api/v1/clusters/:id/resources/:res/:name    Update
  DELETE /api/v1/clusters/:id/resources/:res/:name    Delete
  PATCH  /api/v1/clusters/:id/resources/:res/:name    Patch

Real-time
  GET    /api/v1/ws/watch                             WebSocket events

Health
  GET    /healthz                                     Health check
```

<br>

## 📁 Project Structure

```
kubevision/
├── cmd/kubevision/main.go         Entry point, DI wiring
├── internal/
│   ├── config/                    YAML + env config
│   ├── server/                    HTTP server + router
│   ├── middleware/                 Auth, RequestID, Logger, Audit
│   ├── handler/                   HTTP handlers (auth, cluster, resource, ws)
│   ├── service/                   Business logic layer
│   ├── repository/                Data access (GORM + K8s dynamic client)
│   ├── model/                     Database models (11 tables)
│   ├── kubernetes/
│   │   ├── cluster/               Multi-cluster connection manager
│   │   ├── informer/              Informer lifecycle + EventListener
│   │   └── resource/              Resource registry (26 types)
│   ├── auth/                      JWT + bcrypt
│   └── pkg/                       Response helpers, error codes
├── web/src/
│   ├── components/ui/             shadcn/ui (17 components)
│   ├── components/shared/         DataTable, StatusBadge, NamespaceSelector
│   ├── hooks/                     useResource, useCluster, useWebSocket
│   ├── pages/                     Login, Overview, ResourceList, ResourceDetail
│   ├── lib/                       API client, WebSocket client, K8s utils
│   └── config/                    Resource UI config (icons, columns)
├── deploy/Dockerfile              3-stage: Node → Go → Alpine
└── Makefile                       dev, build, test, lint, tidy
```

<br>

## ⚙️ Configuration

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

All settings can be overridden via environment variables:

| Variable | Description |
|----------|-------------|
| `KUBEVISION_SERVER_PORT` | HTTP port (default: 8080) |
| `KUBEVISION_DB_DRIVER` | `sqlite` or `postgres` |
| `KUBEVISION_DB_DSN` | Database connection string |
| `KUBEVISION_JWT_SECRET` | JWT signing secret |
| `KUBECONFIG` | Path to kubeconfig file |

<br>

## 🗺️ Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| **P1** | Single cluster, generic CRUD, Informer + WebSocket, JWT auth, base UI | Done |
| **P2** | Pod terminal/logs, Deployment ops, multi-cluster, global search, dry-run diff | Planned |
| **P3** | RBAC, 2FA (TOTP), audit, topology, cross-cluster diff, webhook notifications | Planned |
| **P4** | Plugin system (Prometheus/Grafana/ArgoCD), OAuth/OIDC, CRD discovery, Helm chart | Planned |

<br>

## 🤝 Contributing

Contributions are welcome! Please open an issue first to discuss what you would like to change.

