# KubeVision Engineering Guide

This repository is a Go backend with a React/Vite frontend, Docker image, and Helm chart. Keep changes small, explicit, and aligned with the existing package boundaries.

## Development Focus

- Backend changes need Go lifecycle review: handlers, services, repositories, Kubernetes clients, WebSockets, goroutines, channels, and context cancellation.
- Frontend changes need React lifecycle review: hooks, timers, WebSockets, React Query, xterm, and long-lived UI state.
- Security-sensitive changes need explicit review: auth, RBAC, token handling, webhook delivery, registry access, public-key login, OAuth, and any user-controlled URLs or YAML.
- Deployment changes need release review: Docker, Helm, GitHub Actions, health checks, rollback behavior, and artifact versioning.
- Risky changes need a verification pass: run the smallest useful test first, then broaden checks when shared behavior changes.
- Suspected leaks or flakes need diagnosis before refactoring: reproduce, measure goroutines or memory, isolate the owner, then fix the lifecycle.

## Backend Rules

- Pass `context.Context` as the first parameter on request-scoped and I/O functions. Do not store contexts in structs.
- Every goroutine must have an explicit stop condition: parent context, closed channel, or bounded worker lifetime.
- Every `time.NewTicker`, `time.NewTimer`, `setInterval`, and retry loop must have cleanup. In Go, always `defer ticker.Stop()` after successful creation.
- Avoid `context.Background()` inside request paths. It is acceptable only for process-level startup, shutdown, best-effort background persistence, or tests.
- Always close HTTP response bodies, WebSocket connections, Kubernetes streams, LDAP connections, files, pipes, and terminal sessions.
- Use buffered channels or `select` with cancellation when a goroutine sends to a channel owned by another component.
- Do not start background workers from constructors. Prefer an explicit `Start(ctx)` / `Stop()` lifecycle unless the existing type has another established pattern.
- Keep repository methods context-aware and avoid hidden global state.
- Wrap errors with operation context, but return domain errors through the existing `internal/pkg/errors` API where handlers depend on it.
- Do not log secrets: kubeconfigs, tokens, passwords, OTP codes, API keys, OAuth state, private keys, registry auth headers, or webhook payload secrets.

## Frontend Rules

- Every `useEffect` that creates a subscription, timer, WebSocket, event listener, terminal, observer, or async request must return cleanup.
- Prefer `AbortController` or React Query cancellation-aware patterns for fetches that can outlive the component.
- Store timer, socket, and xterm handles in refs, not state.
- Clear reconnect timers before opening a replacement WebSocket.
- Keep React state immutable. Do not mutate arrays or objects in place.
- Avoid `any`. If an API shape is uncertain, model the narrow shape the component actually reads.
- Keep components focused: data fetching in hooks or page-level components, rendering in leaf components.

## Verification Before Commit

Run the narrowest relevant checks, then expand based on risk:

- Backend only: `go test ./cmd/kubevision` or the touched package, then `go test -race ./...` for concurrency or shared-service changes.
- Frontend only: `make frontend-lint`, `make frontend-typecheck`, and targeted `pnpm --dir web test`.
- Deployment/Helm: `make helm-validate`.
- Release or CI changes: inspect rendered workflow paths and run `make verify` locally when feasible.

For leak-prone changes, include at least one test or manual verification note that proves cleanup happens on cancellation, unmount, close, or shutdown.
