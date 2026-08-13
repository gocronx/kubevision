---
sidebar_position: 1
title: API Overview
---

# API Overview

KubeVision exposes the HTTP API used by its web application under `/api/v1`.
When running locally with the default configuration, the base URL is
`http://localhost:8080/api/v1`.

## Response Format

Most REST handlers return this envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "meta": {
    "total": 10,
    "requestId": "request-id"
  }
}
```

`code` is `0` on success. A non-zero value is a business error; see
[Error Codes](/docs/api/error-codes). `data` and `meta` are omitted when they
are not needed. List handlers may include `meta.total`, and resource handlers
may include cache source and staleness metadata.

Most enveloped business errors currently use HTTP status `200`, so clients must
inspect `code`. Health probes and WebSocket upgrade failures use conventional
HTTP status codes and do not necessarily use the envelope.

## Authentication and Authorization

Protected requests use an access token:

```http
Authorization: Bearer <access-token>
```

Authentication establishes the user identity. RBAC then checks the operation,
cluster, and namespace. The AI assistant and Kubernetes HTTP access perform
additional resource-aware authorization in their handlers.

## Request Data

Send JSON bodies with `Content-Type: application/json`. Query parameters are
used for list filtering, namespace selection, pagination, and search. Path
parameters shown as `:id` or `:name` must be URL-encoded by the client.

## Health Probes

| Path | Purpose |
|------|---------|
| `GET /healthz` | Process liveness |
| `GET /readyz` | Readiness, including database connectivity |

These probes are outside `/api/v1` and do not require authentication.

## References

- [Authentication](/docs/api/authentication)
- [Resource API](/docs/api/resources)
- [Endpoint Index](/docs/api/endpoints)
- [WebSocket API](/docs/api/websocket)
- [Error Codes](/docs/api/error-codes)
