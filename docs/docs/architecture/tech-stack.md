---
sidebar_position: 4
title: Tech Stack
---

# Tech Stack

## Backend

| Technology | Purpose |
|-----------|---------|
| **Go 1.26.6** | Primary language |
| **Gin** | HTTP framework |
| **GORM** | ORM (SQLite + PostgreSQL) |
| **client-go** | Kubernetes API client & Informer |
| **gorilla/websocket** | WebSocket connections |
| **zap** | Structured logging |
| **pquerna/otp** | TOTP 2FA implementation |

## Frontend

| Technology | Purpose |
|-----------|---------|
| **React 19** | UI framework |
| **TypeScript** | Type safety |
| **Vite** | Build tool & dev server |
| **shadcn/ui** | Component library (Radix + Tailwind) |
| **TanStack Query v5** | Server state management & cache |
| **Monaco Editor** | YAML editing + syntax highlighting |
| **xterm.js** | Web terminal emulator |
| **Recharts** | Overview charts |
| **i18next** | Internationalization (EN/ZH) |

## Database

| Mode | Engine | Use Case |
|------|--------|----------|
| Development | SQLite | Zero-config, file-based |
| Production | PostgreSQL | Scalable, concurrent access |

Both engines are abstracted through GORM — switch with a single config change.

## Why These Choices?

- **Go + Gin**: Fast compilation, low memory, excellent Kubernetes ecosystem (client-go)
- **React + shadcn/ui**: Modern component library with full customization via Tailwind
- **TanStack Query**: Eliminates manual cache management, automatic refetch/invalidation
- **Informer + WebSocket**: Industry-standard pattern for real-time K8s dashboards
- **GORM**: Database-agnostic ORM, easy migration between SQLite and PostgreSQL
