---
sidebar_position: 7
title: Resource Quotas
---

# Resource Quotas

KubeVision visualizes namespace-level resource quotas as live progress bars, making it easy to see how close a namespace is to its limits at a glance.

## What Is Displayed

For each namespace that has a `ResourceQuota` object, KubeVision shows:

| Metric | Source Object | Unit |
|--------|---------------|------|
| CPU requests | `ResourceQuota` | cores |
| CPU limits | `ResourceQuota` | cores |
| Memory requests | `ResourceQuota` | GiB |
| Memory limits | `ResourceQuota` | GiB |
| Pod count | `ResourceQuota` | count |
| Default container CPU/memory | `LimitRange` | displayed separately |

If a namespace has no `ResourceQuota`, the quota panel shows "No quota configured" rather than empty bars.

## Progress Bar Thresholds

The color of each progress bar changes based on utilization percentage:

| Usage | Bar Color | Meaning |
|-------|-----------|---------|
| 0 – 79 % | Green | Healthy headroom |
| 80 – 94 % | Amber | Approaching limit |
| 95 – 100 % | Red | Near or at limit |

:::warning
When a namespace reaches 100% of its pod or CPU quota, new workloads will be rejected by the Kubernetes API Server with a `403 Forbidden` quota exceeded error. Monitor amber namespaces proactively.
:::

## Accessing the Quota View

1. Select a cluster from the top navigation bar
2. Go to **Namespaces**
3. Click any namespace row to open the detail drawer
4. Select the **Resource Quota** tab

Alternatively, the namespace list table shows a compact inline quota bar for CPU and memory so you can spot constrained namespaces without drilling in.

## API

```
GET /api/v1/clusters/:cluster/namespaces/:namespace/quota
```

**Response example:**

```json
{
  "namespace": "team-alpha",
  "cluster": "prod-us",
  "quotas": [
    {
      "name": "default",
      "hard": { "cpu": "8", "memory": "16Gi", "pods": "20" },
      "used": { "cpu": "5200m", "memory": "11Gi", "pods": "14" },
      "percentages": { "cpu": 65, "memory": 68, "pods": 70 }
    }
  ],
  "limit_ranges": [
    {
      "name": "default-limits",
      "default_cpu": "500m",
      "default_memory": "256Mi"
    }
  ]
}
```

:::tip
The `percentages` field is pre-computed by the backend, so the frontend only needs to render the value — no math in the component.
:::

## Alerts

If a namespace exceeds the amber threshold (80%), a warning badge appears next to the namespace name in the sidebar and in the namespace list. Clicking the badge jumps directly to the quota tab.

To receive external notifications when quotas are near capacity, configure a webhook filtered to `resource: ResourceQuota` and `event: update`.

## Related

- [Webhooks](/docs/admin-guide/webhooks) — Get notified when quota utilization changes
- [RBAC](/docs/admin-guide/rbac) — Namespace-level access controls who can view quota data
- [Cluster Management](/docs/user-guide/cluster-management) — Select the cluster to inspect
