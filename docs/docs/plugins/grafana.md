---
sidebar_position: 2
title: Grafana Integration
---

# Grafana Integration

The Grafana plugin lets you embed existing Grafana dashboards inside KubeVision and link individual Kubernetes resources to specific Grafana panels. No Grafana data is duplicated — KubeVision renders Grafana panels in iframes using Grafana's native embed URL format.

## Setup

### 1. Configure the Plugin

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    # Optional: service account token for server-side panel link resolution
    serviceAccountToken: ""
```

### 2. Allow Embedding in Grafana

In your Grafana `grafana.ini`, set:

```ini
[security]
allow_embedding = true

[auth.proxy]
enabled = false
```

:::warning
`allow_embedding = true` is required for iframe embedding to work. Without it, Grafana returns a `X-Frame-Options: deny` header and the panel will not render.
:::

## Linking Resources to Panels

Open any resource detail page, click the **Grafana** tab, and click **Link Panel**. Paste a Grafana panel share URL and save. The link is stored per `(cluster, namespace, resource-type, resource-name)` and persists across sessions.

### Panel URL Format

KubeVision accepts both the full share URL and the short embed URL:

```
# Full share URL (copied from Grafana Share → Link)
https://grafana.example.com/d/abc123/dashboard?orgId=1&viewPanel=7

# Embed URL (used in iframes)
https://grafana.example.com/d-solo/abc123/dashboard?orgId=1&panelId=7
```

## Embedded Dashboard View

Navigate to **Clusters → Overview → Grafana** to see a full-width Grafana dashboard embedded for the selected cluster. Configure which dashboard to show per cluster:

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    clusterDashboards:
      prod-us: "d/cluster-overview/cluster-overview"
      staging: "d/cluster-dev/dev-cluster"
```

## Single Sign-On Passthrough

If Grafana and KubeVision share the same identity provider (e.g., OIDC), KubeVision can pass the user's bearer token to Grafana via the embed URL, enabling seamless SSO:

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    ssoPassthrough: true
```

:::tip
SSO passthrough requires Grafana to be configured with `auth.jwt` or `auth.proxy`. Consult the [Grafana authentication docs](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/) for details.
:::

## Related

- [Prometheus Integration](/docs/plugins/prometheus) — Native metrics panels without Grafana
- [ArgoCD Integration](/docs/plugins/argocd) — GitOps status alongside resource metrics
