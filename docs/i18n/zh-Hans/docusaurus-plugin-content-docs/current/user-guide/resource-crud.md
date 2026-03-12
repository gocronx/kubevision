---
sidebar_position: 2
title: 资源管理
---

# 资源管理

KubeVision 为 26 种以上的 Kubernetes 资源类型提供统一的 CRUD 界面。每种资源共享相同的列表、详情、编辑器和删除流程，无需为每种资源单独编写代码。

## 支持的资源类型

### 缓存类型（亚毫秒级读取）

以下 8 种类型通过 Informer Watch 保存在内存中，可即时返回：

`Pods` `Deployments` `Services` `Nodes` `Namespaces` `ConfigMaps` `Secrets` `Events`

### 按需获取类型（从 API Server 实时拉取）

18 种以上的额外类型直接从 API Server 拉取：

`StatefulSets` `DaemonSets` `ReplicaSets` `Jobs` `CronJobs` `Ingresses` `PersistentVolumes` `PersistentVolumeClaims` `StorageClasses` `ServiceAccounts` `Roles` `RoleBindings` `ClusterRoles` `ClusterRoleBindings` `NetworkPolicies` `HorizontalPodAutoscalers` `ResourceQuotas` `LimitRanges`

### CRD 自动发现

自定义资源定义（CRD）在启动时自动发现。集群中已安装的任何 CRD 都会自动出现在侧边栏的 **Custom Resources** 下，无需任何额外配置。

## 资源列表

在侧边栏中导航到某个资源类型。表格支持以下操作：

- **列排序** — 点击任意列标题进行排序
- **Namespace 过滤** — 使用顶部栏中的 Namespace 下拉菜单
- **实时更新** — 缓存类型通过 WebSocket 实时刷新

## 查看资源详情

点击任意行以打开详情视图。详情页面展示以下内容：

- **Overview 标签** — 以结构化卡片呈现关键字段
- **YAML 标签** — 带语法高亮的完整资源 YAML
- **Events 标签** — 与该资源相关的事件

## 创建资源

1. 在任意资源列表的右上角点击 **Create**
2. 在编辑器中编写或粘贴 YAML（Monaco 编辑器，支持 Kubernetes Schema 自动补全）
3. 可选择先执行 **Dry-Run** 进行验证，再提交
4. 点击 **Apply**

```yaml
# 示例：内联创建一个 ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key: value
```

## 更新资源

1. 打开资源详情页
2. 点击 **Edit**（或切换到 YAML 标签后点击 **Edit YAML**）
3. 在编辑器中修改内容
4. 点击 **Save** — KubeVision 使用 strategic merge patch 进行更新

## 删除资源

单个删除：打开资源详情页，点击操作栏中的 **Delete**。

### 批量操作

使用复选框列选中多行，然后在操作栏中选择：

| 操作 | 说明 |
|--------|-------------|
| **Delete** | 删除所有选中的资源 |
| **Restart** | 触发滚动重启（适用于 Deployments、StatefulSets、DaemonSets） |

:::warning
批量删除操作立即执行且不可撤销，没有回收站功能。
:::

## 相关文档

- [Dry-Run 预览](/docs/user-guide/dry-run) — 在应用变更前进行验证
- [资源拓扑](/docs/user-guide/resource-topology) — 可视化资源所有权关系
- [kubectl 提示](/docs/user-guide/kubectl-hints) — 查看每个操作对应的等效 CLI 命令
