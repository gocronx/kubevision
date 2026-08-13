---
sidebar_position: 1
slug: /intro
title: Introduction
---

# KubeVision

**See your clusters clearly, act on them instantly.**

KubeVision is a modern, real-time Kubernetes multi-cluster management dashboard built with Go and React. It combines enterprise-grade security with an intuitive developer experience.

## Why KubeVision?

- **Real-time Architecture** — Informer Watch → WebSocket Push delivers sub-second updates with zero polling
- **Multi-cluster Management** — Single dashboard for all your clusters
- **Enterprise Security** — 2FA (TOTP), 5-level RBAC, audit logging, Secrets masking
- **Developer Friendly** — kubectl hints, global search (Cmd+K), favorites, resource templates
- **DevOps Tooling** — Dry-run previews, terminal recording, and resource topology

## Key Differentiators

These features are **unique to KubeVision** — no other Kubernetes dashboard offers them:

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
