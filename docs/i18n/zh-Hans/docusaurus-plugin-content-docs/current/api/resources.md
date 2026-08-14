---
sidebar_position: 3
title: 资源 API
---

# 资源 API

KubeVision 使用同一组集群级 CRUD 路由管理 Kubernetes 内置资源和已发现的
自定义资源：

| 方法 | 路径 | 操作 |
|------|------|------|
| `GET` | `/clusters/:id/resources/:resource` | 列表 |
| `GET` | `/clusters/:id/resources/:resource/:name` | 详情 |
| `POST` | `/clusters/:id/resources/:resource` | 创建 |
| `PUT` | `/clusters/:id/resources/:resource/:name` | 替换/更新 |
| `PATCH` | `/clusters/:id/resources/:resource/:name` | 补丁更新 |
| `DELETE` | `/clusters/:id/resources/:resource/:name` | 删除 |

本页路径均相对于 `/api/v1`。命名空间资源通过 Web 客户端采用的请求格式传递
命名空间（列表接口使用查询参数，资源清单使用 `metadata.namespace`）。资源名
使用小写复数，例如 `pods`、`deployments` 或 CRD 的复数名。

Pod 列表和详情请求可以增加 `includeMetrics=true`，返回当前 CPU、内存、容器级
使用量、请求值和限制值。KubeVision 读取 `metrics.k8s.io/v1beta1`；如果 Metrics
Server 不可用，资源请求仍会成功，并返回 `metricsStatus: "unavailable"`，但不
包含 `metrics` 对象。

## Dry Run

| 方法 | 路径 | 操作 |
|------|------|------|
| `POST` | `/clusters/:id/resources/:resource/dry-run` | 预览创建 |
| `PUT` | `/clusters/:id/resources/:resource/:name/dry-run` | 预览更新 |

请求体与对应写入操作相同。后端让 Kubernetes API Server 验证操作但不持久化，
并返回预览或差异。

## 工作负载操作

| 方法 | 路径 | 支持资源 |
|------|------|----------|
| `PUT` | `/clusters/:id/namespaces/:namespace/:kind/:name/scale` | Deployment、StatefulSet、ReplicaSet |
| `POST` | `/clusters/:id/namespaces/:namespace/:kind/:name/restart` | Deployment、StatefulSet、DaemonSet |
| `GET` | `/clusters/:id/namespaces/:namespace/deployments/:name/history` | Deployment |
| `POST` | `/clusters/:id/namespaces/:namespace/deployments/:name/rollback` | Deployment |

## 批量操作

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/clusters/:id/resources/batch-delete` | 删除选中的资源 |
| `POST` | `/clusters/:id/batch-restart` | 重启选中的工作负载 |

每个条目单独授权，因此一次批量请求可能同时包含成功和失败结果。

## 发现与视图

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/clusters/:id/search` | 在单个集群内搜索资源 |
| `GET` | `/clusters/:id/overview` | 集群概览数据 |
| `GET` | `/clusters/:id/quota-summary` | 资源配额汇总 |
| `GET` | `/clusters/:id/crds` | CRD 列表 |
| `POST` | `/clusters/:id/crds/refresh` | 刷新 CRD 发现 |
| `GET` | `/clusters/:id/namespaces/:namespace/topology` | 命名空间拓扑 |

搜索接口是集群级接口，不是 `/api/v1/search`。

## 相关文档

- [资源增删改查](/docs/user-guide/resource-crud)
- [Pod 指标](/docs/user-guide/pod-metrics)
- [Dry Run](/docs/user-guide/dry-run)
- [批量操作](/docs/user-guide/batch-actions)
- [自定义资源](/docs/user-guide/custom-resources)
