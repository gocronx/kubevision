---
sidebar_position: 5
title: Global Search
---

# Global Search

Global Search lets you find any Kubernetes resource across all clusters, namespaces, and resource types instantly — without navigating the sidebar.

## Opening the Search Dialog

| Platform | Shortcut |
|----------|---------|
| macOS | `Cmd+K` |
| Windows / Linux | `Ctrl+K` |

Alternatively, click the **Search** icon in the top navigation bar.

## How It Works

Results appear as you type. The search runs client-side against a pre-fetched index of all resource names, kinds, namespaces, and clusters — no extra API call per keystroke.

```
Type "nginx" → matches:
  Deployment  nginx-deployment      default      prod-cluster
  Pod         nginx-deployment-xyz  default      prod-cluster
  Service     nginx-svc             default      prod-cluster
  Ingress     nginx-ingress         kube-system  staging-cluster
```

## Fuzzy Matching

The search uses fuzzy matching, so you do not need to type the exact name:

| Query | Matches |
|-------|---------|
| `ngxdep` | `nginx-deployment` |
| `cfgmap` | `ConfigMap` resources |
| `kube sys` | Resources in `kube-system` namespace |
| `prod nginx` | nginx resources on the `prod-cluster` |

## Keyboard Navigation

Once the dialog is open, use the keyboard exclusively:

| Key | Action |
|-----|--------|
| `Arrow Up / Down` | Move through results |
| `Enter` | Navigate to the selected resource |
| `Escape` | Close the dialog |

## Result Layout

Each result row shows four pieces of information:

```
[Kind icon]  Resource Name          Namespace        Cluster
             nginx-deployment       default          prod-cluster
```

Click or press `Enter` on a result to jump directly to that resource's detail page.

## Scope

Global Search covers all resource types that KubeVision has loaded — both cached types (Pods, Deployments, Services, etc.) and on-demand types discovered via CRD auto-discovery. The index refreshes automatically as the Informer cache receives updates.

:::tip
Global Search is the fastest way to navigate KubeVision. Power users rarely touch the sidebar at all.
:::

## Related

- [Cluster Management](/docs/user-guide/cluster-management) — Add or remove clusters from the search scope
- [Favorites](/docs/user-guide/favorites) — Pin resources you access most often
