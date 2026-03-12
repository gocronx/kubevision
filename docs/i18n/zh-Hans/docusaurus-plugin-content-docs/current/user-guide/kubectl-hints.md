---
sidebar_position: 10
title: kubectl 提示
---

# kubectl 提示

在 KubeVision 中执行的每个操作都会展示对应的等效 `kubectl` 命令。这让您在使用仪表板的同时自然地学习 Kubernetes CLI——并为任何操作提供可自动化或脚本化的备用方案。

## 在哪里查看提示

每个操作的页脚区域都会出现一个 **kubectl** 提示标签：

- 资源列表页页脚 — 显示当前视图对应的 `get` 命令
- 资源详情页标题区域 — 显示 `get -o yaml` 命令
- 任何创建 / 更新 / 删除操作后 — 弹出通知中显示等效命令
- YAML 编辑器工具栏 — 显示带有临时文件名的 `apply -f` 或 `delete -f` 命令

点击任意提示旁边的**复制**图标，即可将命令立即复制到剪贴板。

## 工作原理

提示完全在前端根据当前 UI 状态生成，无需额外的 API 请求。前端已知集群、Namespace、资源类型和资源名称，并通过 `kubectl` 动词映射表组装命令。

## 示例提示

### 列出资源

```bash
# 查看 prod-cluster 上 "default" Namespace 中的 Deployments
kubectl get deployments -n default --context prod-cluster

# 跨 Namespace 查看所有 Pods
kubectl get pods -A --context prod-cluster
```

### 获取资源

```bash
kubectl get deployment nginx-deployment -n default -o yaml --context prod-cluster
```

### 创建 / 应用

```bash
kubectl apply -f nginx-deployment.yaml --context prod-cluster
```

### 删除

```bash
kubectl delete deployment nginx-deployment -n default --context prod-cluster
```

### Dry Run

```bash
kubectl apply -f nginx-deployment.yaml --dry-run=server --context prod-cluster
```

### Pod 操作

```bash
# 打开终端
kubectl exec -it nginx-pod -n default -c app --context prod-cluster -- bash

# 流式输出日志
kubectl logs -f nginx-pod -n default -c app --context prod-cluster

# 上一个容器的日志
kubectl logs nginx-pod -n default -c app --previous --context prod-cluster
```

### 扩缩容

```bash
kubectl scale deployment nginx-deployment --replicas=3 -n default --context prod-cluster
```

## --context 标志

`--context` 标志始终包含在提示中，其值与 KubeVision 中注册的集群名称匹配。如果集群名称与本地 kubeconfig 中的上下文名称不一致，请在集群设置中对齐两者。

:::tip
kubectl 提示是帮助 Kubernetes 新手工程师学习的绝佳工具。将其与[资源拓扑](/docs/user-guide/resource-topology)视图结合使用，可以在完成工作的同时建立对集群的直观认知。
:::

## 相关文档

- [资源管理](/docs/user-guide/resource-crud) — 完整的 CRUD 操作
- [Pod 终端](/docs/user-guide/pod-terminal) — 浏览器内 Shell 访问
- [Pod 日志](/docs/user-guide/pod-logs) — 实时日志流式传输
