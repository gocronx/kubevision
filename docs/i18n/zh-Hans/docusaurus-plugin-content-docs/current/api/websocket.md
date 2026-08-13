---
sidebar_position: 5
title: 流式 API
---

# 流式 API

KubeVision 使用 WebSocket 提供资源监听、Pod 终端和 Pod 日志，并使用 SSE
提供 AI 对话。

## 资源监听

```text
wss://example.com/api/v1/ws/watch
```

该路由使用常规认证中间件。浏览器客户端发起已认证的升级请求，然后管理集群、
命名空间和资源范围订阅。连接建立前会执行 RBAC 检查。

## Pod 终端

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec?token=<access-token>&container=<name>
```

由于浏览器 WebSocket API 无法添加 `Authorization` 请求头，处理器会认证查询
参数中的 `token`。终端数据使用二进制帧，调整大小等控制消息使用 JSON。启用
终端录制后可记录会话。

## Pod 日志

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs?token=<access-token>&container=<name>&tail=100
```

处理器检查集群、命名空间、Pod 和容器权限后，发送历史日志并持续跟踪新日志。

:::warning
生产环境必须使用 HTTPS/WSS。查询参数令牌可能被代理或访问日志记录，基础设施
应隐藏查询字符串并避免不必要的保留。
:::

## AI 对话（SSE）

`POST /api/v1/ai/chat` 返回 SSE 流。每次工具调用都会检查当前用户权限；修改
操作需要明确确认，并通过 `POST /api/v1/ai/continue-action` 继续。

## 重连

客户端应使用有上限的指数退避重连临时中断的流；遇到认证或授权失败应停止重试。
