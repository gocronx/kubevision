---
sidebar_position: 6
title: 资源拓扑
---

# 资源拓扑

拓扑视图为任意资源渲染一个可交互的所有权关系图，让您一眼就能理解 Kubernetes 对象之间的关联关系。

## 打开拓扑视图

1. 打开任意资源的详情页（Deployment、Service、Pod 等）
2. 点击 **Topology** 标签

图形使用 Informer 缓存中已有的数据立即渲染，无需额外的 API 请求。

## 所有权关系

KubeVision 通过 `metadata.ownerReferences` 和常见的标签选择器来追踪所有权关系：

```
Deployment ──owns──▶ ReplicaSet ──owns──▶ Pod
                                           │
CronJob ──owns──▶ Job ──owns──▶ Pod        │
                                           │
Service ──selects──▶ Endpoints ──▶ Pod ◀──┘
                                           │
PersistentVolumeClaim ──bound──▶ PersistentVolume
```

### 支持的关系类型

| 来源 | 关系 | 目标 |
|--------|-------------|--------|
| Deployment | 拥有 | ReplicaSet |
| ReplicaSet | 拥有 | Pod |
| StatefulSet | 拥有 | Pod |
| DaemonSet | 拥有 | Pod |
| Job | 拥有 | Pod |
| CronJob | 拥有 | Job |
| Service | 选择 | Pod |
| Ingress | 路由到 | Service |
| PVC | 绑定到 | PV |

## 图形交互

| 交互方式 | 操作效果 |
|------------|--------|
| **点击节点** | 在右侧面板中打开资源详情 |
| **滚轮** | 放大 / 缩小 |
| **拖拽背景** | 平移视口 |
| **拖拽节点** | 重新定位节点（布局不会保存） |
| **双击节点** | 跳转到完整的资源详情页 |

## 节点颜色说明

| 颜色 | 状态含义 |
|-------|--------|
| 绿色 | Running / Available / Bound |
| 黄色 | Pending / Progressing |
| 红色 | Failed / Error / Terminating |
| 灰色 | Unknown 或无状态条件 |

## 使用场景

- **排查故障 Deployment** — 立即查看哪些 Pod 处于错误状态，无需逐个导航
- **追踪流量路径** — 沿 Ingress → Service → Pod 的链路验证路由配置
- **审查存储** — 确认 Namespace 内 PVC/PV 的绑定状态

:::tip
对于副本数较多的大规模 Deployment，图形会将 Pod 分组并显示数量徽标，以保持视图的可读性。
:::

## 相关文档

- [资源管理](/docs/user-guide/resource-crud) — 列表查看与编辑资源
- [跨集群对比](/docs/user-guide/cross-cluster-diff) — 对比不同环境的拓扑结构
