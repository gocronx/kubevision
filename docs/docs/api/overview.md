---
sidebar_position: 1
title: API Overview
---

# API Overview

KubeVision exposes a RESTful HTTP API consumed by the frontend and available for programmatic use.

## Base URL

All API endpoints are mounted under:

```
/api/v1
```

When running locally this resolves to `http://localhost:8080/api/v1`.

## Unified Response Format

Every response — success or error — uses the same JSON envelope:

```json
{
  "code":    0,
  "message": "success",
  "data":    {},
  "meta":    {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| `code` | `int` | Business status code. `0` = success. Non-zero values indicate an error. See [Error Codes](/docs/api/error-codes). |
| `message` | `string` | Human-readable description of the result. |
| `data` | `object \| array \| null` | The response payload. `null` for operations that produce no output (e.g., delete). |
| `meta` | `object` | Request metadata. Present on all responses. |

:::tip
All HTTP responses return **status 200**. The business outcome is determined entirely by the `code` field. Never branch on the HTTP status code alone.
:::

## Meta Object

The `meta` field carries context about how the response was produced:

```json
{
  "meta": {
    "source":    "cache",
    "stale":     false,
    "total":     142,
    "requestId": "req_01HXYZ4B9K0VGMZS"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `source` | `"cache" \| "apiserver"` | Whether the data came from the Informer cache or a live API Server call. |
| `stale` | `bool` | `true` if the cache entry is older than the configured TTL. |
| `total` | `int` | Total number of items for list endpoints. Used for pagination display. |
| `requestId` | `string` | Unique identifier for this request. Include this in bug reports. |

## Content Type

All requests must include:

```
Content-Type: application/json
```

All responses are returned as `application/json; charset=utf-8`.

## Authentication

Requests must include a valid JWT in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

See [Authentication](/docs/api/authentication) for how to obtain tokens.

## Related

- [Authentication](/docs/api/authentication) — Login, 2FA, token refresh
- [Resource API](/docs/api/resources) — CRUD endpoints for Kubernetes resources
- [WebSocket API](/docs/api/websocket) — Real-time watch, terminal, and log streaming
- [Error Codes](/docs/api/error-codes) — Full table of business status codes
