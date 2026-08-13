---
sidebar_position: 1
title: Architecture Overview
---

# Architecture Overview

KubeVision follows a clean layered architecture with real-time data flow.

The AI assistant is an orchestration layer over the same repositories,
authorization model, and audit system used by the dashboard. It does not have a
separate privileged Kubernetes client.

## System Diagram

```text
                    Browser (React + shadcn/ui)
            ┌────────┴────────┐
            REST          WebSocket
            │                 │
┌───────────▼─────────────────▼──────────────┐
│         KubeVision Backend (Go + Gin)       │
│                                             │
│  Middleware: RequestID → Logger → Auth       │
│             → RBAC → Audit                  │
│                                             │
│  Handler ──→ Service ──→ K8sRepo            │
│  (params)    (logic)     (cache+fallback)   │
│                            │                │
│                 ┌──────────┴──────────┐     │
│                 │                     │     │
│          Informer Cache         API Server  │
│                 │                            │
│     Informer Watch ──→ EventListener        │
│                            │                │
│                        WS Hub ──→ Browser   │
│                                             │
│  Plugins: Prometheus │ Grafana │ ArgoCD     │
│  DB: SQLite (dev) / PostgreSQL (prod)       │
└─────────────────────────────────────────────┘
```

## Design Principles

| # | Principle | Description |
|---|-----------|-------------|
| 1 | **Simple First** | Use stdlib when possible, minimize third-party deps |
| 2 | **Progressive Enhancement** | MVP core first, advanced features plug in |
| 3 | **Generic Resources** | One Handler + registry covers all K8s resources |
| 4 | **Real-time Feedback** | Informer cache + WebSocket, sub-second updates |
| 5 | **Interface Driven** | All dependencies injected via interfaces |
| 6 | **Secure by Default** | Secrets masked, audit on, least privilege |
| 7 | **Guarded AI** | Bounded tools, inherited RBAC, explicit approval, auditable mutations |

## Layers

### Handler Layer
HTTP request entry point. Parses parameters, validates input, delegates to Service layer. A single generic `ResourceHandler` handles CRUD for all Kubernetes resource types.

### Service Layer
Business logic: input validation, error mapping, caching strategy. `ResourceService` provides unified CRUD with cache-first reads and API Server fallback.

### Repository Layer
Data access abstraction. `K8sRepo` wraps Informer cache and dynamic client. `UserRepo`, `ClusterRepo`, etc. wrap GORM for database operations.

### Middleware Stack
Requests flow through: `RequestID → Logger → Auth (JWT) → RBAC → Audit`

Each middleware is composable and can be applied per-route.

## Key Interfaces

```go
type K8sResourceRepo interface {
    List(ctx, cluster, ns, resource) (items, stale, error)
    Get(ctx, cluster, ns, resource, name) (*Unstructured, error)
    Create(ctx, cluster, ns, resource, obj) (*Unstructured, error)
    Update(ctx, cluster, ns, resource, name, obj) (*Unstructured, error)
    Delete(ctx, cluster, ns, resource, name) error
    DryRun(ctx, cluster, ns, resource, name, obj) (*Unstructured, error)
}

type ClusterManager interface {
    DynamicClient(cluster string) (dynamic.Interface, error)
    RESTConfig(cluster string) (*rest.Config, error)
}

type ResourceRegistry interface {
    Get(name string) (*ResourceInfo, error)
    All() []ResourceInfo
}
```
