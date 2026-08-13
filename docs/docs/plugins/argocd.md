---
sidebar_position: 3
title: Argo CD Integration
---

# Argo CD Integration

The Argo CD plugin connects to an existing Argo CD API and lists applications.
Open **Administration > Plugins**, configure `url` and a read-only API `token`,
enable the plugin, and run the health check.

The current data endpoint is:

```http
GET /api/v1/plugins/argocd/applications
```

The response exposes application information supplied by Argo CD, including
sync and health state where available. The backend must be able to reach the
configured URL.

The current frontend manages plugin configuration and health only. It does not
show application badges on Kubernetes resources or trigger sync operations.
