---
sidebar_position: 7
title: 资源配额
---

# 资源配额

KubeVision 将命名空间级别的资源配额以实时进度条的形式可视化展示，让用户一目了然地了解命名空间距其限额还有多少余量。

## 显示内容

对于每个拥有 `ResourceQuota` 对象的命名空间，KubeVision 展示以下指标：

| 指标 | 来源对象 | 单位 |
|--------|---------------|------|
| CPU requests | `ResourceQuota` | 核 |
| CPU limits | `ResourceQuota` | 核 |
| Memory requests | `ResourceQuota` | GiB |
| Memory limits | `ResourceQuota` | GiB |
| Pod 数量 | `ResourceQuota` | 个 |
| 容器默认 CPU/内存 | `LimitRange` | 单独显示 |

若命名空间未配置 `ResourceQuota`，配额面板将显示"未配置配额"，而非空进度条。

## 进度条阈值

每个进度条的颜色根据使用率百分比变化：

| 使用率 | 进度条颜色 | 含义 |
|-------|-----------|---------|
| 0 – 79% | 绿色 | 余量充足 |
| 80 – 94% | 橙色 | 即将达到限额 |
| 95 – 100% | 红色 | 接近或已达限额 |

:::warning
当命名空间的 Pod 数量或 CPU 配额达到 100% 时，Kubernetes API Server 将以 `403 Forbidden` 配额超限错误拒绝新的工作负载。请主动监控橙色状态的命名空间。
:::

## 访问配额视图

1. 从顶部导航栏选择集群
2. 进入 **Namespaces**
3. 点击任意命名空间行以打开详情抽屉
4. 选择 **Resource Quota** 标签页

此外，命名空间列表表格中也会为每个命名空间内嵌显示 CPU 和内存的简洁配额条，便于在不深入查看的情况下快速发现受限命名空间。

## API

```
GET /api/v1/clusters/:cluster/namespaces/:namespace/quota
```

**响应示例：**

```json
{
  "namespace": "team-alpha",
  "cluster": "prod-us",
  "quotas": [
    {
      "name": "default",
      "hard": { "cpu": "8", "memory": "16Gi", "pods": "20" },
      "used": { "cpu": "5200m", "memory": "11Gi", "pods": "14" },
      "percentages": { "cpu": 65, "memory": 68, "pods": 70 }
    }
  ],
  "limit_ranges": [
    {
      "name": "default-limits",
      "default_cpu": "500m",
      "default_memory": "256Mi"
    }
  ]
}
```

:::tip
`percentages` 字段由后端预先计算，前端只需直接渲染该值，无需在组件中进行数学运算。
:::

## 告警

当命名空间超过橙色阈值（80%）时，侧边栏和命名空间列表中该命名空间名称旁会出现警告徽章。点击徽章可直接跳转至配额标签页。

如需在配额接近上限时接收外部通知，请配置一个过滤条件为 `resource: ResourceQuota` 且 `event: update` 的 Webhook。

## 相关文档

- [Webhooks](/docs/admin-guide/webhooks) — 配额使用率变化时接收通知
- [RBAC](/docs/admin-guide/rbac) — 命名空间级别的访问控制决定谁可以查看配额数据
- [集群管理](/docs/user-guide/cluster-management) — 选择要查看的集群
