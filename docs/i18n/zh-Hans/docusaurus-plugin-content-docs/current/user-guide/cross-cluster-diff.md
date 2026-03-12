---
sidebar_position: 7
title: 跨集群对比
---

# 跨集群对比

跨集群对比功能让您可以并排比较同一 Kubernetes 资源在两个不同集群或环境中的配置差异。在配置偏差演变成故障前及时发现问题。

## 打开对比视图

1. 打开任意资源的详情页
2. 点击操作栏中的 **Compare** 按钮
3. 在弹出对话框中选择：
   - **Source** — 当前集群 / Namespace / 资源（已预填）
   - **Target** — 要进行对比的目标集群、Namespace 和资源名称
4. 点击 **Compare**

Monaco Diff Editor 随即打开，左侧显示来源 YAML，右侧显示目标 YAML。

## 读懂差异对比

| 高亮颜色 | 含义 |
|-----------|---------|
| 绿色行 | 存在于目标，不存在于来源 |
| 红色行 | 存在于来源，不存在于目标 |
| 黄色行 | 键名相同，但值不同 |
| 无高亮 | 完全相同 |

默认情况下，对比会忽略 `managedFields`、`resourceVersion`、`uid` 和 `creationTimestamp`，以减少由元数据天然差异带来的噪音。

## API 参考

该功能由单个端点驱动：

```http
POST /api/v1/compare
Content-Type: application/json

{
  "source": {
    "cluster":   "prod-cluster",
    "namespace": "default",
    "resource":  "deployments",
    "name":      "api-server"
  },
  "target": {
    "cluster":   "staging-cluster",
    "namespace": "default",
    "resource":  "deployments",
    "name":      "api-server"
  }
}
```

响应：

```json
{
  "source": { /* 完整资源 YAML 对象 */ },
  "target": { /* 完整资源 YAML 对象 */ },
  "hasDiff": true
}
```

## 常见使用场景

**发现环境配置偏差**
```
prod-cluster/default/deployments/api-server
       vs.
staging-cluster/default/deployments/api-server
```

**验证变更晋级**
将变更从 staging 晋级到 prod 后，确认 Deployment 配置完全一致。

**审查副本配置**
对比不同区域的资源请求与限制，确保配置一致性。

:::tip
您不仅可以跨集群对比资源，还可以在同一集群的不同 Namespace 之间进行对比，这对多租户场景非常有用。
:::

## 相关文档

- [Dry-Run 预览](/docs/user-guide/dry-run) — 应用变更前先进行验证
- [集群管理](/docs/user-guide/cluster-management) — 添加要进行对比的集群
- [资源管理](/docs/user-guide/resource-crud) — 查看差异后对资源进行编辑
