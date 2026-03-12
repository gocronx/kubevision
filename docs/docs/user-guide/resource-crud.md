---
sidebar_position: 2
title: Resource Management
---

# Resource Management

KubeVision provides a unified CRUD interface for 26+ Kubernetes resource types. Every resource shares the same list, detail, editor, and delete flows — no per-resource code needed.

## Supported Resource Types

### Cached (sub-millisecond reads)

These 8 types are kept in memory via Informer Watch and return instantly:

`Pods` `Deployments` `Services` `Nodes` `Namespaces` `ConfigMaps` `Secrets` `Events`

### On-Demand (API Server live fetch)

18+ additional types fetched directly from the API Server:

`StatefulSets` `DaemonSets` `ReplicaSets` `Jobs` `CronJobs` `Ingresses` `PersistentVolumes` `PersistentVolumeClaims` `StorageClasses` `ServiceAccounts` `Roles` `RoleBindings` `ClusterRoles` `ClusterRoleBindings` `NetworkPolicies` `HorizontalPodAutoscalers` `ResourceQuotas` `LimitRanges`

### CRD Auto-Discovery

Custom Resource Definitions are discovered automatically at startup. Any CRD installed in the cluster appears in the sidebar under **Custom Resources** without any configuration.

## List Resources

Navigate to a resource type in the sidebar. The table supports:

- **Column sorting** — click any column header
- **Namespace filter** — namespace dropdown in the top bar
- **Live updates** — cached types refresh in real time via WebSocket

## Get / View a Resource

Click any row to open the detail view. The detail page shows:

- **Overview tab** — key fields rendered as structured cards
- **YAML tab** — full resource YAML with syntax highlighting
- **Events tab** — related events scoped to this resource

## Create a Resource

1. Click **Create** in the top-right of any resource list
2. Write or paste YAML in the editor (Monaco with Kubernetes schema autocompletion)
3. Optionally run **Dry-Run** to validate before applying
4. Click **Apply**

```yaml
# Example: create a ConfigMap inline
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key: value
```

## Update a Resource

1. Open the resource detail page
2. Click **Edit** (or switch to the YAML tab and click **Edit YAML**)
3. Make changes in the editor
4. Click **Save** — KubeVision uses a strategic merge patch

## Delete a Resource

Single delete: open the resource and click **Delete** in the action bar.

### Batch Operations

Select multiple rows using the checkbox column, then choose from the action bar:

| Action | Description |
|--------|-------------|
| **Delete** | Delete all selected resources |
| **Restart** | Trigger a rolling restart (Deployments, StatefulSets, DaemonSets) |

:::warning
Batch delete is immediate and irreversible. There is no recycle bin.
:::

## Related

- [Dry-Run Preview](/docs/user-guide/dry-run) — Validate changes before applying
- [Resource Topology](/docs/user-guide/resource-topology) — Visualize ownership relationships
- [kubectl Hints](/docs/user-guide/kubectl-hints) — See the equivalent CLI command for each action
