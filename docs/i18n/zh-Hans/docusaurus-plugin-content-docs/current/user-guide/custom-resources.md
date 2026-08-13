---
title: 自定义资源
---

# 自定义资源

KubeVision 会发现每个已连接集群的 CustomResourceDefinition（CRD），并通过
与内置资源相同的列表、详情、YAML、创建、更新和删除流程管理自定义资源。

打开 **CRD** 可查看已发现的定义。KubeVision 连接后新安装的 CRD 尚未出现时，
可手动刷新；定期发现周期由 `kubernetes.crd_discovery_interval` 控制。

RBAC 按自定义资源的复数资源名和作用域执行。CRD 发现不会绕过 Kubernetes
授权。删除 CRD 可能同时删除该类型的全部自定义资源，破坏性操作前必须检查
Finalizer 和转换 Webhook。
