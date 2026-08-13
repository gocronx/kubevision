---
sidebar_position: 1
title: Prometheus Integration
---

# Prometheus Integration

The Prometheus plugin provides backend health checks and bounded instant
PromQL queries against an existing Prometheus server.

Open **Administration > Plugins**, configure Prometheus with its backend-
reachable `url`, enable it, and run the health check. The URL normally points to
an internal service such as
`http://prometheus.monitoring.svc.cluster.local:9090`.

The data endpoint is:

```http
GET /api/v1/plugins/prometheus/query?query=<url-encoded-promql>
```

Queries require authentication and plugin permissions. Query length is bounded,
and unrestricted full-metric-name scans are rejected. The current frontend
does not provide custom dashboard panels or Alertmanager visualization; clients
can consume the API endpoint directly.
