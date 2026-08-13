---
sidebar_position: 3
title: Resource API
---

# Resource API

KubeVision uses one cluster-scoped CRUD route family for built-in Kubernetes
resources and discovered custom resources:

| Method | Path | Operation |
|--------|------|-----------|
| `GET` | `/clusters/:id/resources/:resource` | List |
| `GET` | `/clusters/:id/resources/:resource/:name` | Get |
| `POST` | `/clusters/:id/resources/:resource` | Create |
| `PUT` | `/clusters/:id/resources/:resource/:name` | Replace/update |
| `PATCH` | `/clusters/:id/resources/:resource/:name` | Patch |
| `DELETE` | `/clusters/:id/resources/:resource/:name` | Delete |

All paths on this page are relative to `/api/v1`. For namespaced resources,
pass the namespace using the request format accepted by the web client (the
list endpoint uses query parameters and manifests carry `metadata.namespace`).
Use lowercase plural resource names such as `pods`, `deployments`, or the
plural name of a CRD.

## Dry Run

| Method | Path | Operation |
|--------|------|-----------|
| `POST` | `/clusters/:id/resources/:resource/dry-run` | Preview create |
| `PUT` | `/clusters/:id/resources/:resource/:name/dry-run` | Preview update |

The body is the same manifest used by the corresponding write operation. A dry
run asks the Kubernetes API server to validate the operation without persisting
it and returns the resulting preview/diff.

## Workload Actions

| Method | Path | Supported resources |
|--------|------|---------------------|
| `PUT` | `/clusters/:id/namespaces/:namespace/:kind/:name/scale` | Deployment, StatefulSet, ReplicaSet |
| `POST` | `/clusters/:id/namespaces/:namespace/:kind/:name/restart` | Deployment, StatefulSet, DaemonSet |
| `GET` | `/clusters/:id/namespaces/:namespace/deployments/:name/history` | Deployment |
| `POST` | `/clusters/:id/namespaces/:namespace/deployments/:name/rollback` | Deployment |

## Batch Operations

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/clusters/:id/resources/batch-delete` | Delete selected resources |
| `POST` | `/clusters/:id/batch-restart` | Restart selected workloads |

Each item is authorized independently. A mixed batch can therefore contain
both successful and failed results.

## Discovery and Views

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/clusters/:id/search` | Search resources in one cluster |
| `GET` | `/clusters/:id/overview` | Cluster overview data |
| `GET` | `/clusters/:id/quota-summary` | Resource quota summary |
| `GET` | `/clusters/:id/crds` | List discovered CRDs |
| `POST` | `/clusters/:id/crds/refresh` | Refresh CRD discovery |
| `GET` | `/clusters/:id/namespaces/:namespace/topology` | Namespace topology |

The search endpoint is cluster-scoped; it is not `/api/v1/search`.

## Related

- [Resource CRUD](/docs/user-guide/resource-crud)
- [Dry Run](/docs/user-guide/dry-run)
- [Batch Actions](/docs/user-guide/batch-actions)
- [Custom Resources](/docs/user-guide/custom-resources)
