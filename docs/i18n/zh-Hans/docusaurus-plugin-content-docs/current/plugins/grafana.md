---
sidebar_position: 2
title: Grafana 集成
---

# Grafana 集成

Grafana 插件允许你在 KubeVision 中嵌入现有的 Grafana 仪表板，并将单个 Kubernetes 资源关联到特定的 Grafana 面板。Grafana 的数据不会被复制——KubeVision 使用 Grafana 原生的嵌入 URL 格式，通过 iframe 渲染 Grafana 面板。

## 配置

### 1. 配置插件

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    # 可选：用于服务端面板链接解析的服务账号令牌
    serviceAccountToken: ""
```

### 2. 在 Grafana 中允许嵌入

在 Grafana 的 `grafana.ini` 中设置：

```ini
[security]
allow_embedding = true

[auth.proxy]
enabled = false
```

:::warning
iframe 嵌入需要设置 `allow_embedding = true`。若未设置，Grafana 将返回 `X-Frame-Options: deny` 响应头，面板将无法渲染。
:::

## 将资源关联到面板

打开任意资源详情页，点击 **Grafana** 标签页，然后点击 **Link Panel**。粘贴 Grafana 面板的分享 URL 并保存。关联关系按 `(集群, 命名空间, 资源类型, 资源名称)` 存储，并在会话之间持久保留。

### 面板 URL 格式

KubeVision 同时接受完整分享 URL 和简短嵌入 URL：

```
# 完整分享 URL（从 Grafana Share → Link 复制）
https://grafana.example.com/d/abc123/dashboard?orgId=1&viewPanel=7

# 嵌入 URL（用于 iframe）
https://grafana.example.com/d-solo/abc123/dashboard?orgId=1&panelId=7
```

## 嵌入仪表板视图

前往 **集群 → 概览 → Grafana**，可查看为所选集群嵌入的全宽 Grafana 仪表板。按集群配置要显示的仪表板：

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    clusterDashboards:
      prod-us: "d/cluster-overview/cluster-overview"
      staging: "d/cluster-dev/dev-cluster"
```

## 单点登录透传

如果 Grafana 和 KubeVision 共享同一身份提供商（例如 OIDC），KubeVision 可以通过嵌入 URL 将用户的 bearer token 传递给 Grafana，从而实现无缝 SSO：

```yaml
plugins:
  grafana:
    enabled: true
    baseUrl: "https://grafana.example.com"
    ssoPassthrough: true
```

:::tip
SSO 透传要求 Grafana 配置了 `auth.jwt` 或 `auth.proxy`。详情请参阅 [Grafana 认证文档](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/)。
:::

## 相关文档

- [Prometheus 集成](/docs/plugins/prometheus) — 无需 Grafana 的原生指标面板
- [ArgoCD 集成](/docs/plugins/argocd) — 在资源指标旁显示 GitOps 状态
