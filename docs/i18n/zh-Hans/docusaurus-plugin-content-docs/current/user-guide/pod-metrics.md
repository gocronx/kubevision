---
sidebar_position: 8
title: Pod 指标
---

# Pod 指标

KubeVision 在 **工作负载 → Pod** 中展示当前 CPU 和内存使用量。列表显示每个
Pod 的总计，并在页面打开期间每 15 秒刷新一次。

打开 Pod 后，可以在**概览**标签中查看：

- CPU 和内存总使用量
- 配置的请求值与限制值
- 相对于限制值的使用百分比
- 每个常规容器的 CPU 和内存使用量

## 前提条件

集群必须通过 Metrics Server 提供 `metrics.k8s.io/v1beta1` Kubernetes Metrics
API。连接集群所使用的凭据需要拥有该 API Group 下 `pods` 的 `get` 和 `list`
权限。KubeVision Helm Chart 只为集群内 ServiceAccount 授予这两项只读权限。

Metrics Server 启动期间，新 Pod 可能短暂显示指标等待中。如果 API 不可用或
访问被拒绝，Pod 页面仍然可以正常使用，并显示指标不可用，而不会让资源请求
整体失败。

Metrics Server 不提供容器文件系统或 PVC 实际使用量。这些数据需要 kubelet、
cAdvisor 或 Prometheus 等监控数据源，KubeVision 不会根据容器内 `df` 显示的
节点文件系统数值进行估算。
