---
sidebar_position: 1
title: 集群管理
---

# 集群管理

KubeVision 可以在单个仪表板中管理多个 Kubernetes 集群。所有集群共享相同的 UI、RBAC 配置和审计日志。

## 从 kubeconfig 自动检测

启动时，KubeVision 会读取本地 kubeconfig（默认路径为 `~/.kube/config`），并将其中发现的每个上下文注册为受管集群。

```bash
# 通过环境变量指定 kubeconfig 路径
KUBECONFIG=/path/to/kubeconfig ./kubevision
```

:::tip
如果您的 kubeconfig 包含多个上下文，所有上下文均会自动导入，无需手动添加。
:::

## 添加集群

1. 进入 **Settings → Clusters**
2. 点击 **Add Cluster**
3. 填写表单：

| 字段 | 说明 |
|-------|-------------|
| **Name** | 显示在侧边栏中的名称 |
| **kubeconfig** | 粘贴 kubeconfig YAML 内容或上传文件 |

4. 点击 **Save** — KubeVision 会在保存前验证连通性

## 删除集群

1. 进入 **Settings → Clusters**
2. 找到对应集群行，点击 **Delete** 图标
3. 在弹出对话框中确认操作

:::warning
删除集群会清除该集群的所有缓存资源数据。与该集群相关的审计日志和收藏记录会被保留。
:::

## 切换集群

使用顶部导航栏中的**集群选择器**来切换当前活跃集群。除非您明确选择"All Clusters"，否则所有资源列表、搜索结果和拓扑视图均以所选集群为范围。

## 集群健康状态

侧边栏会在每个集群名称旁显示实时健康徽标：

| 徽标 | 含义 |
|-------|---------|
| 绿色圆点 | API Server 可达，Informer 已连接 |
| 黄色圆点 | 降级 — 部分连通或缓存已过期 |
| 红色圆点 | 不可达 — 所有读操作回退到缓存数据 |

健康状态每 30 秒通过对 API Server 发送轻量级 `GET /healthz` 探测来检查一次。

## 相关文档

- [全局搜索](/docs/user-guide/global-search) — 搜索当前选中的集群
- [配置](/docs/getting-started/configuration) — Kubeconfig 路径与 TLS 设置
