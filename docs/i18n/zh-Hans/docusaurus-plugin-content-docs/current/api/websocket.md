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

## WebSocket Ticket

Pod 终端和日志连接不会把可复用的 access token 放入 WebSocket URL。客户端先通过
常规认证 API 获取有效期 30 秒、仅限 WebSocket 使用的 ticket：

```http
POST /api/v1/ws/ticket
Authorization: Bearer <access-token>
```

响应中的 `data.ticket` 不能用于普通 API 认证。每次连接或重连都应获取新 ticket。
ticket 是短时 bearer 凭据而非一次性凭据：服务端不保存跨副本消费记录，因此被截获
的 ticket 在 30 秒有效期内仍可重放。

## Pod 终端

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec?ticket=<ws-ticket>&container=<name>
```

终端数据使用文本帧，调整大小和输入消息使用 JSON。启用终端录制后可记录会话。

## Pod 日志

```text
wss://example.com/api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs?ticket=<ws-ticket>&container=<name>&tailLines=100
```

处理器检查集群、命名空间、Pod 和容器权限后，发送历史日志并持续跟踪新日志。

:::warning
生产环境必须使用 HTTPS/WSS。尽管 ticket 在 30 秒后过期且不能用于普通 API，
基础设施仍应从访问日志中隐藏查询字符串。
:::

## AI 对话（SSE）

`POST /api/v1/ai/chat` 返回 SSE 流。每次工具调用都会检查当前用户权限；修改
操作需要明确确认，并通过 `POST /api/v1/ai/continue-action` 继续。

## 重连

客户端应使用有上限的指数退避重连临时中断的流；遇到认证或授权失败应停止重试。
