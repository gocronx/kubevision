---
sidebar_position: 3
title: 资源 API
---

# 资源 API

KubeVision 为所有 Kubernetes 资源类型提供了一套统一的通用 CRUD 模式。相同的四个端点可处理 Pod、Deployment、ConfigMap 及任意 CRD——无需为每种资源单独设计路由。

## 命名空间级资源

| 方法 | 路径 | 描述 |
|--------|------|-------------|
| `GET` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res` | 列出资源 |
| `POST` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res` | 创建资源 |
| `GET` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | 获取资源 |
| `PUT` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | 更新资源 |
| `DELETE` | `/api/v1/clusters/:c/namespaces/:ns/resources/:res/:name` | 删除资源 |

## 集群级资源

| 方法 | 路径 | 描述 |
|--------|------|-------------|
| `GET` | `/api/v1/clusters/:c/resources/:res` | 列出集群级资源 |
| `POST` | `/api/v1/clusters/:c/resources/:res` | 创建集群级资源 |
| `GET` | `/api/v1/clusters/:c/resources/:res/:name` | 获取集群级资源 |
| `PUT` | `/api/v1/clusters/:c/resources/:res/:name` | 更新集群级资源 |
| `DELETE` | `/api/v1/clusters/:c/resources/:res/:name` | 删除集群级资源 |

## 特殊操作

除 CRUD 之外，KubeVision 还提供资源专属的操作端点：

```
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/scale
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/restart
POST /api/v1/clusters/:c/namespaces/:ns/resources/deployments/:name/rollback

POST /api/v1/clusters/:c/resources/nodes/:name/cordon
POST /api/v1/clusters/:c/resources/nodes/:name/uncordon
POST /api/v1/clusters/:c/resources/nodes/:name/drain
```

### 预演（Dry-Run）

在将变更应用到真实 API Server 之前，先进行验证：

```http
POST /api/v1/clusters/:c/namespaces/:ns/resources/:res/:name/dry-run
Content-Type: application/json

{
  "manifest": { ... }
}
```

响应的 `data` 字段包含将要应用的变更差异，若 API Server 拒绝该 manifest，则返回 `422xx` 错误。

## 其他端点

### 全局搜索

```http
GET /api/v1/search?q=nginx&clusters=prod-us,staging&limit=20
```

返回跨所有可访问集群和命名空间的匹配资源扁平列表。

### 集群概览

```http
GET /api/v1/clusters/:c/overview
```

返回集群的节点数、Pod 数、命名空间数、资源配额摘要以及近期事件。

### 跨集群对比

```http
POST /api/v1/compare
Content-Type: application/json

{
  "left":  { "cluster": "prod-us",  "namespace": "default", "resource": "deployments", "name": "api" },
  "right": { "cluster": "staging",  "namespace": "default", "resource": "deployments", "name": "api" }
}
```

### 收藏夹

| 方法 | 路径 | 描述 |
|--------|------|-------------|
| `GET` | `/api/v1/favorites` | 列出当前用户的收藏 |
| `POST` | `/api/v1/favorites` | 添加收藏 |
| `DELETE` | `/api/v1/favorites/:id` | 删除收藏 |

### Webhook

| 方法 | 路径 |
|--------|------|
| `GET` | `/api/v1/webhooks` |
| `POST` | `/api/v1/webhooks` |
| `PUT` | `/api/v1/webhooks/:id` |
| `DELETE` | `/api/v1/webhooks/:id` |

:::tip
`:res` 的值应与 Kubernetes 资源类型的小写复数形式保持一致，例如：`pods`、`deployments`、`configmaps`、`customresourcedefinitions` 等。
:::

## 相关文档

- [WebSocket API](/docs/api/websocket) — 订阅任意资源类型的实时更新
- [错误码](/docs/api/error-codes) — 资源操作相关的 `404xx` 和 `422xx` 错误码
