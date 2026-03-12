---
sidebar_position: 3
title: ArgoCD 集成
---

# ArgoCD 集成

ArgoCD 插件将 GitOps 应用状态直接呈现在 KubeVision 中。你可以在不离开控制台的情况下，查看每个 ArgoCD 应用的同步状态、健康状态和最近应用的版本。

:::info
ArgoCD 集成**计划在未来版本中发布**。本页文档描述了预期的设计方案。配置键已保留，但在当前版本中不起任何作用。
:::

## 计划中的配置

```yaml
plugins:
  argocd:
    enabled: true
    serverUrl: "https://argocd.example.com"
    # ArgoCD API 令牌（建议使用只读服务账号）
    token: ""
    # 验证 ArgoCD 的 TLS 证书
    tlsVerify: true
```

## 计划中的功能

### 按应用显示同步状态

集群概览页面将新增一个专属的 **GitOps** 标签页，列出所有 ArgoCD 应用及其当前的同步和健康状态。

| 列名 | 描述 |
|--------|-------------|
| 应用 | ArgoCD 应用名称 |
| 同步状态 | `Synced` / `OutOfSync` / `Unknown` |
| 健康状态 | `Healthy` / `Degraded` / `Progressing` / `Missing` |
| 版本 | 最近应用版本的 Git commit SHA |
| 上次同步时间 | 最近一次同步操作的时间戳 |

### 资源级 GitOps 徽标

由 ArgoCD 管理的资源（即带有 `app.kubernetes.io/managed-by: argocd` 标签的资源）将在资源列表和详情视图中显示一个小型 GitOps 徽标。

### 跳转到 ArgoCD 界面

每个 ArgoCD 应用行都将包含一个直达链接，可在 ArgoCD Web 界面中打开对应的应用页面。

```
https://argocd.example.com/applications/<app-name>
```

### 触发同步（计划中）

未来版本可能允许直接从 KubeVision 触发手动同步。该功能将需要具有写权限的服务账号令牌。

:::warning
从 KubeVision 触发同步将受 RBAC 系统中 `argocd:sync` 权限原子的管控。只有拥有至少 `ops` 角色的用户才能发起同步。
:::

## 发布前的替代方案

在插件正式发布之前，你可以手动添加 ArgoCD 链接：

1. 前往 **设置 → 外部链接**
2. 添加一条模式为 `https://argocd.example.com/applications/{{app}}` 的链接
3. 该链接将出现在带有对应标签的资源操作栏中

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — ArgoCD 插件使用的权限原子
- [Grafana 集成](/docs/plugins/grafana) — 在 GitOps 状态旁嵌入 Grafana 仪表板
