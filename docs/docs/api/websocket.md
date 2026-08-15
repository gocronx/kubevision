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

## WebSocket Tickets

Pod terminal and log connections do not put the reusable access token in the
WebSocket URL. First request a 30-second, WebSocket-only ticket through the
normal authenticated API:

```http
POST /api/v1/ws/ticket
Authorization: Bearer <access-token>
```

The response `data.ticket` value cannot authenticate ordinary API requests.
Request a fresh ticket for each connection or reconnection attempt. Tickets are
short-lived bearer credentials, not one-time credentials: a captured ticket can
be replayed until its 30-second expiry because the server does not keep a
cross-replica consumption record.

## Pod Terminal

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec?ticket=<ws-ticket>&container=<name>
```

Terminal data uses text frames; resize and input messages use JSON. Sessions
can be recorded when terminal recording is enabled.

## Pod Logs

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs?ticket=<ws-ticket>&container=<name>&tailLines=100
```

The handler streams historical and follow-up container logs after checking
cluster, namespace, Pod, and container access.

:::warning
Use HTTPS/WSS in production. Although tickets expire after 30 seconds and are
not API access tokens, infrastructure should still redact query strings from
access logs.
:::

## AI Chat (SSE)

`POST /api/v1/ai/chat` returns an SSE stream. Each tool invocation is checked
against the current user's permissions. Mutating actions require explicit
confirmation and continue through `POST /api/v1/ai/continue-action`.

## Reconnection

Clients should reconnect transiently failed streams with bounded exponential
backoff and stop retrying after authentication or authorization failures.
