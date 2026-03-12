---
sidebar_position: 1
title: Prometheus Integration
---

# Prometheus Integration

KubeVision includes an optional Prometheus plugin that adds CPU, memory, and network metrics directly into the resource views. It requires an existing Prometheus instance reachable from the KubeVision backend.

## Setup

### 1. Enable the Plugin

Add the following block to your `values.yaml` (Helm) or `config.yaml` (source):

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus.monitoring.svc.cluster.local:9090"
    timeout: 10s
```

### 2. Restart the Backend

```bash
# Helm
helm upgrade kubevision kubevision/kubevision -f values.yaml

# From source
make dev
```

:::tip
KubeVision connects to Prometheus from the backend, not the browser. The endpoint must be reachable from inside the cluster, not from your laptop.
:::

## Metrics Displayed

| View | Metrics |
|------|---------|
| Node detail | CPU usage %, Memory usage %, Network in/out bytes/s |
| Pod detail | Container CPU/memory usage vs limits |
| Deployment list | Aggregate CPU and memory for all pods |
| Namespace overview | Top-N resource consumers |

## Custom Queries

You can define additional PromQL queries that appear as extra panels on resource detail pages.

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    customQueries:
      - name: "Request Rate"
        query: 'rate(http_requests_total{pod=~"{{pod}}"}[5m])'
        unit: "req/s"
      - name: "Error Rate"
        query: 'rate(http_errors_total{pod=~"{{pod}}"}[5m])'
        unit: "err/s"
```

The `{{pod}}`, `{{namespace}}`, and `{{cluster}}` template variables are replaced at query time with the currently viewed resource.

## Alerts Visualization

When Prometheus Alertmanager is configured, active alerts are surfaced inline on resource cards. Firing alerts appear as a red badge; pending alerts as yellow.

```yaml
plugins:
  prometheus:
    enabled: true
    endpoint: "http://prometheus:9090"
    alertmanager:
      endpoint: "http://alertmanager.monitoring.svc.cluster.local:9093"
```

:::warning
Alert data is fetched on every page load. In large clusters with many active alerts, set `alertmanager.maxAlerts` to cap the number returned.
:::

## Related

- [Grafana Integration](/docs/plugins/grafana) — Embed full Grafana dashboards alongside metrics
- [Architecture Overview](/docs/architecture/overview) — How plugins attach to the backend plugin registry
