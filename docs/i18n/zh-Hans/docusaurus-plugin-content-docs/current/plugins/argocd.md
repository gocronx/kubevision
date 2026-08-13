---
sidebar_position: 3
title: Argo CD 集成
---

# Argo CD 集成

Argo CD 插件连接已有 Argo CD API 并列出应用。打开**管理 > 插件**，配置 `url`
和只读 API `token`，启用插件并执行健康检查。

当前数据接口为：

```http
GET /api/v1/plugins/argocd/applications
```

响应会包含 Argo CD 返回的应用信息，以及可用时的同步和健康状态。KubeVision
后端必须能够访问配置的地址。

当前前端只管理插件配置和健康状态，不在 Kubernetes 资源上显示应用标记，也不
提供同步操作。
