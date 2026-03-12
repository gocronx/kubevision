---
sidebar_position: 3
title: Resource API
---

# Resource API

KubeVision exposes a single generic CRUD pattern for every Kubernetes resource type. The same four endpoints handle Pods, Deployments, ConfigMaps, and any CRD — no per-resource routes are needed.

## Namespace-Scoped Resources

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res` | List resources |
| `POST` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res` | Create a resource |
| `GET` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | Get a resource |
| `PUT` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | Update a resource |
| `DELETE` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | Delete a resource |

## Cluster-Scoped Resources

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/clusters/:c/resources/:res` | List cluster-scoped resources |
| `POST` | `/api/v1/clusters/:c/resources/:res` | Create a cluster-scoped resource |
| `GET` | `/api/v1/clusters/:c/resources/:res/:name` | Get a cluster-scoped resource |
| `PUT` | `/api/v1/clusters/:c/resources/:res/:name` | Update a cluster-scoped resource |
| `DELETE` | `/api/v1/clusters/:c/resources/:res/:name` | Delete a cluster-scoped resource |

## Special Actions

Beyond CRUD, KubeVision exposes resource-specific action endpoints:

```
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/scale
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/restart
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/rollback

POST /api/v1/clusters/:c/resources/nodes/:name/cordon
POST /api/v1/clusters/:c/resources/nodes/:name/uncordon
POST /api/v1/clusters/:c/resources/nodes/:name/drain
```

### Dry-Run

Validate any create or update before applying it against the real API Server:

```http
POST /api/v1/clusters/:c/namespaces/:ns/resources/:res/:name/dry-run
Content-Type: application/json

{
  "manifest": { ... }
}
```

The response `data` contains a diff of the changes that would be applied, or a `422xx` error if the API Server rejects the manifest.

## Other Endpoints

### Global Search

```http
GET /api/v1/search?q=nginx&clusters=prod-us,staging&limit=20
```

Returns a flat list of matching resources across all accessible clusters and namespaces.

### Cluster Overview

```http
GET /api/v1/clusters/:c/overview
```

Returns node count, pod count, namespace count, resource quota summaries, and recent events for a cluster.

### Cross-Cluster Compare

```http
POST /api/v1/compare
Content-Type: application/json

{
  "left":  { "cluster": "prod-us",  "namespace": "default", "resource": "deployments", "name": "api" },
  "right": { "cluster": "staging",  "namespace": "default", "resource": "deployments", "name": "api" }
}
```

### Favorites

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/favorites` | List the current user's favorites |
| `POST` | `/api/v1/favorites` | Add a favorite |
| `DELETE` | `/api/v1/favorites/:id` | Remove a favorite |

### Webhooks

| Method | Path |
|--------|------|
| `GET` | `/api/v1/webhooks` |
| `POST` | `/api/v1/webhooks` |
| `PUT` | `/api/v1/webhooks/:id` |
| `DELETE` | `/api/v1/webhooks/:id` |

:::tip
Use `:res` values that match the lowercase plural Kubernetes resource kind: `pods`, `deployments`, `configmaps`, `customresourcedefinitions`, etc.
:::

## Related

- [WebSocket API](/docs/api/websocket) — Subscribe to live updates for any resource type
- [Error Codes](/docs/api/error-codes) — `404xx` and `422xx` error codes for resource operations
