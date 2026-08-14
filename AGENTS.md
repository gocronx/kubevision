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

- Keep clear ownership boundaries: pages coordinate routing and feature state, hooks own data access and reusable stateful behavior, feature components own domain UI, and `components/ui` remains domain-agnostic.
- Do not combine data fetching, complex mutations, dialogs, table definitions, and large rendering trees in one component. Extract a boundary when it makes ownership or lifecycle easier to verify.
- Treat 400-500 lines as a review signal, not an automatic split rule. Split by cohesive responsibility and keep tightly coupled logic together.
- Add new top-level pages through route-level lazy imports. Keep loading fallbacks compact and preserve existing route and authorization behavior.
- Declare API response types at the request boundary, for example `api.get<Resource[]>()`. Do not use `any` or `as unknown as` in feature code; unavoidable browser or transport boundary conversions must stay localized and documented by their surrounding types.
- Keep React Query keys stable and include every input that changes the response. Mutations must invalidate or update all affected caches without changing unrelated entries.
- Every `useEffect` that creates a subscription, timer, WebSocket, event listener, terminal, observer, or async request must return cleanup.
- Prefer `AbortController` or React Query cancellation-aware patterns for fetches that can outlive the component.
- Store timer, socket, and xterm handles in refs, not state.
- Clear existing timers and detach stale handlers before opening replacement WebSockets or terminal sessions. Explicit disconnects and unmounts must never schedule reconnects.
- Bound long-lived client data such as logs, events, terminal output, and chat history. Use pagination, a ring buffer, or an explicit retention limit instead of unbounded arrays.
- Keep React state immutable. Do not mutate arrays or objects in place.
- Do not add an abstraction only to reduce line count. Shared hooks or components must remove real duplication, centralize a lifecycle, or establish a clear feature boundary.
- Preserve behavior during component extraction: routes, query keys, request payloads, permissions, confirmation flows, and visible interaction semantics must remain unchanged unless the task explicitly extends them.
- Add focused tests for lifecycle-sensitive behavior, including reconnect, explicit close, cancellation, timer cleanup, unmount, and bounded stream retention where applicable.

## Documentation Sync

- For every new or changed user-facing feature, explicitly assess whether the documentation must change.
- When documentation is affected, update both `README.md` and `README_CN.md` where the feature belongs in the project overview, quick start, examples, configuration, or security guidance.
- Keep the English docs under `docs/docs/` and the Chinese docs under `docs/i18n/zh-Hans/docusaurus-plugin-content-docs/current/` aligned. Do not update only one language.
- Update documentation navigation, homepage content, screenshots, configuration examples, and API references when the feature changes those surfaces.
- Documentation must describe implemented behavior and its security or operational limits. Do not claim planned behavior as available.

## Verification Before Commit

Run the narrowest relevant checks, then expand based on risk:

- Commit messages must be a single concise English line and follow Conventional Commits, for example `feat: add AI action safeguards`.

- Backend only: `go test ./cmd/kubevision` or the touched package, then `go test -race ./...` for concurrency or shared-service changes.
- Frontend only: `make frontend-lint`, `make frontend-typecheck`, targeted `pnpm --dir web test`, and `pnpm --dir web build` for routing, dependency, or production-bundle changes.
- Deployment/Helm: `make helm-validate`.
- Documentation or user-facing feature changes: confirm the English and Chinese README/docs are synchronized, then run `make docs`.
- Release or CI changes: inspect rendered workflow paths and run `make verify` locally when feasible.

For leak-prone changes, include at least one test or manual verification note that proves cleanup happens on cancellation, unmount, close, or shutdown.
