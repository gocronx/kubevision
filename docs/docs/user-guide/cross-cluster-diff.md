---
sidebar_position: 7
title: Cross-Cluster Diff
---

# Cross-Cluster Diff

Cross-Cluster Diff lets you compare the same Kubernetes resource across two different clusters or environments side by side. Spot configuration drift before it becomes an incident.

## Opening the Diff View

1. Open any resource detail page
2. Click the **Compare** button in the action bar
3. In the dialog, choose:
   - **Source** — current cluster / namespace / resource (pre-filled)
   - **Target** — the cluster, namespace, and resource name to compare against
4. Click **Compare**

The Monaco Diff Editor opens with the source YAML on the left and the target YAML on the right.

## Reading the Diff

| Highlight | Meaning |
|-----------|---------|
| Green lines | Present in target, not in source |
| Red lines | Present in source, not in target |
| Yellow lines | Same key, different value |
| No highlight | Identical |

The diff ignores `managedFields`, `resourceVersion`, `uid`, and `creationTimestamp` by default to reduce noise from metadata that differs by nature.

## API Reference

The diff is powered by a single endpoint:

```http
POST /api/v1/compare
Content-Type: application/json

{
  "source": {
    "cluster":   "prod-cluster",
    "namespace": "default",
    "resource":  "deployments",
    "name":      "api-server"
  },
  "target": {
    "cluster":   "staging-cluster",
    "namespace": "default",
    "resource":  "deployments",
    "name":      "api-server"
  }
}
```

Response:

```json
{
  "source": { /* full resource YAML as object */ },
  "target": { /* full resource YAML as object */ },
  "hasDiff": true
}
```

## Common Use Cases

**Catch environment drift**
```
prod-cluster/default/deployments/api-server
       vs.
staging-cluster/default/deployments/api-server
```

**Verify a promotion**
After promoting a change from staging to prod, confirm the Deployment spec matches exactly.

**Audit replica counts**
Compare resource requests/limits across regions to ensure consistency.

:::tip
You can compare resources across namespaces within the same cluster, not just across clusters. This is useful for multi-tenant setups.
:::

## Related

- [Dry-Run Preview](/docs/user-guide/dry-run) — Validate a change before applying it
- [Cluster Management](/docs/user-guide/cluster-management) — Add clusters to compare against
- [Resource Management](/docs/user-guide/resource-crud) — Edit resources after reviewing the diff
