---
sidebar_position: 1
slug: /intro
title: Introduction
---

# KubeVision

**AI-native Kubernetes operations with human control.**

KubeVision is an AI-native Kubernetes dashboard built with Go and React. Its
assistant works with live cluster context and Kubernetes tools so operators can
investigate failures and prepare changes through natural language without
leaving the dashboard.

It is intentionally human-in-the-loop: AI tools inherit the current user's
RBAC permissions, mutating operations require explicit approval, and approved
changes are recorded for audit. The real-time dashboard remains available for
direct inspection and manual operation at every step.

## Why KubeVision?

- **AI Operations Workspace** — Inspect resources and logs, explain failures, query metrics, and prepare changes using live context
- **Guarded Execution** — Tool-level RBAC, explicit mutation confirmation, execution-time re-authorization, and audit records
- **Model Choice** — Connect any supported OpenAI-compatible endpoint instead of coupling cluster operations to one vendor
- **Real-time Architecture** — Informer Watch → WebSocket Push delivers sub-second updates with zero polling
- **Multi-cluster Management** — Single dashboard for all your clusters
- **Enterprise Security** — 2FA (TOTP), 5-level RBAC, audit logging, Secrets masking
- **Developer Friendly** — kubectl hints, global search (Cmd+K), favorites, resource templates
- **DevOps Tooling** — Dry-run previews, terminal recording, and resource topology

## AI Operating Model

| Stage | Behavior |
|-------|----------|
| **Context** | The assistant receives the selected cluster, namespace, page, and resource |
| **Investigate** | It can read Kubernetes resources, Pod logs, cluster health, and configured Prometheus data |
| **Authorize** | Every tool call is checked against the current user's role |
| **Approve** | Create, update, patch, and delete operations pause for explicit confirmation |
| **Execute** | Permission is checked again immediately before an approved mutation runs |
| **Audit** | AI mutations record the actor, target, tool, correlation ID, and outcome |

KubeVision does not run autonomous remediation in the background and does not
bypass Kubernetes or application RBAC. It is an operator copilot, not an
unattended cluster controller.

## Operational Foundation

| Feature | Description |
|---------|-------------|
| **2FA (TOTP)** | Two-factor authentication with QR setup and recovery codes |
| **Dry-Run Diff** | Preview changes before applying, validated by API Server |
| **Terminal Recording** | asciinema-format session recording and playback |
| **kubectl Hints** | Auto-generated CLI commands for every UI action |
| **Secrets Masking** | Secrets hidden by default in all views |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.23, Gin, GORM, client-go |
| Frontend | React 19, TypeScript, Vite, shadcn/ui |
| State | TanStack Query v5 |
| Database | SQLite (dev) / PostgreSQL (prod) |
| Realtime | gorilla/websocket, Informer cache |

## What's Next?

- [Installation](/docs/getting-started/installation) — Get KubeVision running
- [Quick Start](/docs/getting-started/quick-start) — First steps after installation
- [Architecture](/docs/architecture/overview) — Understand the system design
