---
sidebar_position: 5
title: Streaming APIs
---

# Streaming APIs

KubeVision uses WebSockets for resource watches, Pod terminals, and Pod logs,
and Server-Sent Events (SSE) for AI chat.

## Resource Watch

Connect to:

```text
wss://example.com/api/v1/ws/watch
```

This route uses the normal authentication middleware. The browser client sends
the authenticated upgrade request and then manages subscriptions for cluster,
namespace, and resource scopes. RBAC is applied before the connection is
accepted.

## Pod Terminal

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec?token=<access-token>&container=<name>
```

The handler authenticates the `token` query parameter because browser
WebSocket APIs cannot add an `Authorization` header. Terminal data uses binary
frames; resize/control messages use JSON. Sessions can be recorded when
terminal recording is enabled.

## Pod Logs

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs?token=<access-token>&container=<name>&tail=100
```

The handler streams historical and follow-up container logs after checking
cluster, namespace, Pod, and container access.

:::warning
Use HTTPS/WSS in production. Query-string tokens may be visible to proxies or
access logs, so infrastructure should redact query strings and avoid retaining
them unnecessarily.
:::

## AI Chat (SSE)

`POST /api/v1/ai/chat` returns an SSE stream. Each tool invocation is checked
against the current user's permissions. Mutating actions require explicit
confirmation and continue through `POST /api/v1/ai/continue-action`.

## Reconnection

Clients should reconnect transiently failed streams with bounded exponential
backoff and stop retrying after authentication or authorization failures.
