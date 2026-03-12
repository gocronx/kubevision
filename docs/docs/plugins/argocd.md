---
sidebar_position: 3
title: ArgoCD Integration
---

# ArgoCD Integration

The ArgoCD plugin surfaces GitOps application status inside KubeVision. You can see sync state, health, and last-applied revision for every ArgoCD application without leaving the dashboard.

:::info
The ArgoCD integration is **planned for a future release**. This page documents the intended design. The configuration keys are reserved but have no effect in the current version.
:::

## Planned Setup

```yaml
plugins:
  argocd:
    enabled: true
    serverUrl: "https://argocd.example.com"
    # ArgoCD API token (read-only service account recommended)
    token: ""
    # Verify ArgoCD's TLS certificate
    tlsVerify: true
```

## Planned Features

### Sync Status per Application

A dedicated **GitOps** tab will appear on the cluster overview page, listing all ArgoCD applications with their current sync and health status.

| Column | Description |
|--------|-------------|
| Application | ArgoCD application name |
| Sync Status | `Synced` / `OutOfSync` / `Unknown` |
| Health | `Healthy` / `Degraded` / `Progressing` / `Missing` |
| Revision | Git commit SHA of the last applied revision |
| Last Synced | Timestamp of the most recent sync operation |

### Resource-level GitOps Badge

Resources managed by ArgoCD (i.e., those carrying the `app.kubernetes.io/managed-by: argocd` label) will display a small GitOps badge in the resource list and detail views.

### Link to ArgoCD UI

Every ArgoCD application row will include a direct link that opens the corresponding application page in the ArgoCD web UI.

```
https://argocd.example.com/applications/<app-name>
```

### Sync Trigger (Planned)

Future versions may allow triggering a manual sync directly from KubeVision. This will require a service account token with write permissions.

:::warning
Triggering syncs from KubeVision will be gated behind the `argocd:sync` permission atom in the RBAC system. Only users with at least the `ops` role will be able to initiate syncs.
:::

## Workaround Until Release

Until the plugin ships, you can link to ArgoCD manually:

1. Go to **Settings → External Links**
2. Add a link with pattern `https://argocd.example.com/applications/{{app}}`
3. The link will appear in the resource action bar for labeled resources

## Related

- [RBAC](/docs/admin-guide/rbac) — Permission atoms used by the ArgoCD plugin
- [Grafana Integration](/docs/plugins/grafana) — Embed Grafana dashboards alongside GitOps status
