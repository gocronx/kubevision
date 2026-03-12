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
| **kubeconfig** | Paste the kubeconfig YAML or upload the file |

4. Click **Save** — KubeVision validates connectivity before saving

## Removing a Cluster

1. Go to **Settings → Clusters**
2. Find the cluster row and click the **Delete** icon
3. Confirm the dialog

:::warning
Removing a cluster deletes all cached resource data for that cluster. Audit logs and favorites referencing it are retained.
:::

## Switching Between Clusters

Use the **cluster selector** in the top navigation bar to switch the active cluster. Every resource list, search result, and topology view scopes to the selected cluster unless you explicitly choose "All Clusters".

## Cluster Health Status

The sidebar shows a live health badge next to each cluster name:

| Badge | Meaning |
|-------|---------|
| Green dot | API Server reachable, informer connected |
| Yellow dot | Degraded — partial connectivity or stale cache |
| Red dot | Unreachable — all reads fall back to cached data |

Health is checked every 30 seconds via a lightweight `GET /healthz` probe against the API Server.

## Related

- [Global Search](/docs/user-guide/global-search) — Search across all clusters at once
- [Cross-Cluster Diff](/docs/user-guide/cross-cluster-diff) — Compare resources between clusters
- [Configuration](/docs/getting-started/configuration) — Kubeconfig path and TLS settings
