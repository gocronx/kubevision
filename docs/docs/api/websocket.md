---
sidebar_position: 4
title: WebSocket API
---

# WebSocket API

KubeVision provides three WebSocket endpoints for real-time resource watching, Pod terminal access, and live log streaming. All three require authentication via a query parameter token.

## Authentication

Pass the access token as a query parameter on the initial upgrade request:

```
ws://localhost:8080/api/v1/ws/watch?token=<access_token>
```

:::warning
The token is visible in server access logs when passed as a query parameter. In production, use TLS (`wss://`) to protect the token in transit. Origin validation is enforced in production builds — connections from origins not matching the configured `allowedOrigins` list are rejected.
:::

## 1. Resource Watch — `/api/v1/ws/watch`

Subscribe to live create, update, and delete events for any Kubernetes resource type.

### Subscribe

Send a JSON message after connecting:

```json
{
  "action":    "subscribe",
  "cluster":   "prod-us",
  "namespace": "default",
  "resource":  "pods"
}
```

Set `namespace` to `""` to watch all namespaces. Set `resource` to `"*"` to watch all cached resource types.

### Event Message

The server pushes an event whenever a watched resource changes:

```json
{
  "type":     "MODIFIED",
  "cluster":  "prod-us",
  "resource": "pods",
  "object":   { ... }
}
```

| `type` value | Meaning |
|-------------|---------|
| `ADDED` | A new resource was created |
| `MODIFIED` | An existing resource was updated |
| `DELETED` | A resource was deleted |

### Unsubscribe

```json
{
  "action":    "unsubscribe",
  "cluster":   "prod-us",
  "namespace": "default",
  "resource":  "pods"
}
```

## 2. Pod Terminal — `/api/v1/ws/terminal/:cluster/:ns/:pod`

Opens an interactive shell session inside a running Pod container using `kubectl exec` under the hood.

```
wss://localhost:8080/api/v1/ws/terminal/prod-us/default/my-pod?token=<access_token>&container=app
```

The terminal protocol is compatible with [xterm.js](https://xtermjs.org/). Binary frames carry PTY data; JSON frames carry resize events:

```json
{ "type": "resize", "cols": 220, "rows": 50 }
```

:::tip
Terminal sessions are recorded in asciinema format when session recording is enabled. Recordings are accessible from **Settings → Audit → Terminal Sessions**.
:::

## 3. Log Streaming — `/api/v1/ws/logs/:cluster/:ns/:pod`

Streams container logs in real time.

```
wss://localhost:8080/api/v1/ws/logs/prod-us/default/my-pod?token=<access_token>&container=app&tail=100
```

| Query Parameter | Default | Description |
|-----------------|---------|-------------|
| `container` | first container | Container name to stream logs from |
| `tail` | `100` | Number of historical lines to send before streaming new lines |
| `timestamps` | `false` | Prepend RFC3339 timestamps to each log line |

Each text frame from the server contains one log line.

## Auto-Reconnect

The KubeVision frontend reconnects automatically with exponential backoff (base 1 s, max 30 s). If you build your own client, implement the same pattern:

```typescript
function connect(attempt = 0) {
  const ws = new WebSocket(url);
  ws.onclose = () => {
    const delay = Math.min(1000 * 2 ** attempt, 30_000);
    setTimeout(() => connect(attempt + 1), delay);
  };
}
```

## Related

- [Resource API](/docs/api/resources) — REST endpoints for CRUD operations on the same resources
- [Authentication](/docs/api/authentication) — How to obtain the access token used in the `token` query parameter
