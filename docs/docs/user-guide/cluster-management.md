---
sidebar_position: 1
title: Cluster Management
---

# Cluster Management

KubeVision can manage multiple Kubernetes clusters from a single dashboard. All clusters share the same UI, RBAC, and audit log.

## Auto-Detection from kubeconfig

On startup, KubeVision reads the local kubeconfig (`~/.kube/config` by default) and registers every context it finds as a managed cluster.

```bash
# Override the kubeconfig path via environment variable
KUBECONFIG=/path/to/kubeconfig ./kubevision
```

:::tip
If your kubeconfig has multiple contexts, all of them are imported automatically. You do not need to add them manually.
:::

## Adding a Cluster

1. Go to **Settings → Clusters**
2. Click **Add Cluster**
3. Fill in the form:

| Field | Description |
|-------|-------------|
| **Name** | Display name shown in the sidebar |
| **Authentication Type** | Use **Kubeconfig** for local or external clusters; use **In-Cluster** only when KubeVision runs inside the target cluster |
| **kubeconfig** | Paste the kubeconfig YAML or upload the file |

4. Click **Add Cluster** — KubeVision validates connectivity before saving

For a local k3d cluster, export a standalone kubeconfig with `k3d kubeconfig get <cluster-name>` and paste or upload that output. Kubeconfig contents contain client credentials and should not be committed to source control.

## Removing a Cluster

1. Go to **Settings → Clusters**
2. Find the cluster row and click the **Delete** icon
3. Confirm the dialog

:::warning
Removing a cluster deletes all cached resource data for that cluster. Audit logs and favorites referencing it are retained.
:::

## Switching Between Clusters

Use the **cluster selector** in the top navigation bar to switch the active
cluster. Resource lists, search results, and topology views are scoped to the
selected cluster.

## Cluster Health Status

The sidebar shows a live health badge next to each cluster name:

| Badge | Meaning |
|-------|---------|
| Green dot | API Server reachable, informer connected |
| Yellow dot | Degraded — partial connectivity or stale cache |
| Red dot | Unreachable — all reads fall back to cached data |

Health is checked every 30 seconds via a lightweight `GET /healthz` probe against the API Server.

## Related

- [Global Search](/docs/user-guide/global-search) — Search the selected cluster
- [Configuration](/docs/getting-started/configuration) — Kubeconfig path and TLS settings
