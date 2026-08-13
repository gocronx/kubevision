---
sidebar_position: 2
title: Grafana Integration
---

# Grafana Integration

The Grafana plugin connects to an existing Grafana API and lists dashboards.
Open **Administration > Plugins**, configure `url` and an API/service-account
`token`, enable the plugin, and run the health check.

The current data endpoint is:

```http
GET /api/v1/plugins/grafana/dashboards
```

KubeVision makes this request from the backend, so the Grafana URL must be
reachable from the KubeVision server. Use a read-only token with only the
permissions required to list/search dashboards.

The current frontend manages plugin configuration and health only. It does not
embed Grafana iframes, persist resource-to-panel links, or pass user bearer
tokens to Grafana.
