---
sidebar_position: 9
title: Favorites
---

# Favorites

Favorites let you bookmark the clusters, namespaces, and individual resources you access most often. They appear in the sidebar for one-click access from anywhere in the dashboard.

## Adding a Favorite

Favorite any item from three places:

| Location | How to favorite |
|----------|----------------|
| **Sidebar cluster / namespace** | Hover over the name and click the star icon |
| **Resource list row** | Click the star icon in the row's action column |
| **Resource detail page** | Click the star icon in the page header |

A filled yellow star means the item is already favorited.

## Accessing Favorites

Favorites appear at the top of the sidebar under a **Starred** section, above the full resource tree. Clicking a favorite takes you directly to:

- A **cluster** → switches the active cluster and lands on the cluster overview
- A **namespace** → switches to that namespace and opens the Pod list
- A **resource** → opens the resource detail page directly

## Reordering Favorites

Drag and drop favorites in the sidebar to set a custom order. The order is saved immediately.

:::tip
Put your most critical Deployments at the top of your favorites list so they are always one click away during an incident.
:::

## Managing All Favorites

1. Click **Starred → Manage** at the bottom of the Starred section, or go to **Settings → Favorites**
2. The management page shows all favorites in a table with cluster, namespace, kind, and name
3. Drag to reorder, or click **Remove** on any row to un-favorite it

## Per-User Storage

Favorites are stored per user in the KubeVision database — not in the browser. This means your favorites follow you across browsers and devices when you log in with the same account.

:::info
Favorites are personal. They are not shared between users and do not appear in other users' dashboards.
:::

## What Happens When a Resource Is Deleted

If a favorited resource is deleted from Kubernetes, the sidebar entry shows a **Not Found** badge. You can remove it from favorites on the management page or by clicking the star again.

## Related

- [Global Search](/docs/user-guide/global-search) — Find resources across all clusters instantly
- [Cluster Management](/docs/user-guide/cluster-management) — Add clusters to your dashboard
