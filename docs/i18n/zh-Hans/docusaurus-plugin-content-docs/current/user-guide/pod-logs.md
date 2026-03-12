---
sidebar_position: 4
title: Pod 日志
---

# Pod 日志

KubeVision 通过 WebSocket 连接 Kubernetes API Server 的日志接口，将容器日志实时流式传输到浏览器。无需任何日志聚合管道。

## 打开日志

1. 进入 **Workloads → Pods**，点击一个 Pod
2. 选择 **Logs** 标签
3. 所选容器的日志将立即开始流式输出

## 容器选择

对于包含多个容器的 Pod，使用日志标签顶部的 **Container** 下拉菜单切换容器。切换后日志流将自动重新连接。

## 功能特性

| 功能 | 使用方式 |
|---------|-----------|
| **实时流式传输** | 日志通过 WebSocket 持续追踪输出，无需手动刷新 |
| **搜索 / 过滤** | 在搜索框中输入关键词，匹配行将高亮显示（不区分大小写） |
| **时间戳显示** | 切换 **Timestamps** 开关以显示或隐藏 RFC3339 格式时间戳 |
| **上一个容器** | 启用 **Previous** 以查看上一个已终止容器实例的日志 |
| **下载** | 点击 **Download** 将当前日志缓冲区保存为 `.log` 文件 |
| **暂停 / 继续** | 点击 **Pause** 暂停流式输出并自由滚动，点击 **Resume** 继续追踪最新日志 |

## 日志搜索

搜索栏对缓冲区中的日志行执行实时客户端过滤：

```
# 搜索模式示例
Error
OOMKilled
"connection refused"
```

匹配行以黄色高亮显示。不匹配的行会变暗但不会隐藏，以保留上下文信息。

:::tip
若要按日志级别过滤，可搜索 `ERROR`、`WARN` 或 `INFO`。大多数结构化日志框架会一致地输出这些标识符。
:::

## 控制日志量

| 选项 | 默认值 | 说明 |
|--------|---------|-------------|
| **Tail lines** | 100 | 打开日志时加载的历史行数 |
| **Max buffer** | 5,000 行 | 内存中保留的最大行数；超出后最旧的行将被丢弃 |

在 Container 选择器旁边的下拉菜单中调整 **Tail lines**。

## 上一个容器的日志

如果容器已重启（例如因 OOMKill 或崩溃），启用 **Previous** 开关即可查看已终止实例的日志。等效命令为：

```bash
kubectl logs <pod> -c <container> --previous
```

## 相关文档

- [Pod 终端](/docs/user-guide/pod-terminal) — 在运行中的容器内执行命令
- [kubectl 提示](/docs/user-guide/kubectl-hints) — 复制等效的 `kubectl logs` 命令
