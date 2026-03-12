---
sidebar_position: 4
title: WebSocket API
---

# WebSocket API

KubeVision 提供三个 WebSocket 端点，分别用于实时资源监听、Pod 终端访问和实时日志流。三个端点均需通过查询参数传递令牌进行认证。

## 认证

在初始升级请求中，以查询参数的形式传递访问令牌：

```
ws://localhost:8080/api/v1/ws/watch?token=<access_token>
```

:::warning
通过查询参数传递令牌时，令牌会在服务器访问日志中可见。在生产环境中，请使用 TLS（`wss://`）保护传输中的令牌。生产构建中会强制进行来源验证——来自不在配置的 `allowedOrigins` 列表中的来源的连接将被拒绝。
:::

## 1. 资源监听 — `/api/v1/ws/watch`

订阅任意 Kubernetes 资源类型的实时创建、更新和删除事件。

### 订阅

连接后发送一条 JSON 消息：

```json
{
  "action":    "subscribe",
  "cluster":   "prod-us",
  "namespace": "default",
  "resource":  "pods"
}
```

将 `namespace` 设为 `""` 可监听所有命名空间。将 `resource` 设为 `"*"` 可监听所有已缓存的资源类型。

### 事件消息

当被监听的资源发生变化时，服务端会推送一条事件：

```json
{
  "type":     "MODIFIED",
  "cluster":  "prod-us",
  "resource": "pods",
  "object":   { ... }
}
```

| `type` 值 | 含义 |
|-------------|---------|
| `ADDED` | 新资源被创建 |
| `MODIFIED` | 已有资源被更新 |
| `DELETED` | 资源被删除 |

### 取消订阅

```json
{
  "action":    "unsubscribe",
  "cluster":   "prod-us",
  "namespace": "default",
  "resource":  "pods"
}
```

## 2. Pod 终端 — `/api/v1/ws/terminal/:cluster/:ns/:pod`

使用底层的 `kubectl exec` 在运行中的 Pod 容器内开启一个交互式 Shell 会话。

```
wss://localhost:8080/api/v1/ws/terminal/prod-us/default/my-pod?token=<access_token>&container=app
```

终端协议与 [xterm.js](https://xtermjs.org/) 兼容。二进制帧传输 PTY 数据，JSON 帧传输调整窗口大小的事件：

```json
{ "type": "resize", "cols": 220, "rows": 50 }
```

:::tip
启用会话录制后，终端会话将以 asciinema 格式录制。录制文件可从 **设置 → 审计 → 终端会话** 中访问。
:::

## 3. 日志流 — `/api/v1/ws/logs/:cluster/:ns/:pod`

实时流式传输容器日志。

```
wss://localhost:8080/api/v1/ws/logs/prod-us/default/my-pod?token=<access_token>&container=app&tail=100
```

| 查询参数 | 默认值 | 描述 |
|-----------------|---------|-------------|
| `container` | 第一个容器 | 要流式传输日志的容器名称 |
| `tail` | `100` | 在流式传输新日志之前，先发送的历史日志行数 |
| `timestamps` | `false` | 是否在每行日志前添加 RFC3339 时间戳 |

服务端的每个文本帧包含一行日志。

## 自动重连

KubeVision 前端会以指数退避策略自动重连（基础间隔 1 秒，最大间隔 30 秒）。如果你构建自己的客户端，请实现相同的策略：

```typescript
function connect(attempt = 0) {
  const ws = new WebSocket(url);
  ws.onclose = () => {
    const delay = Math.min(1000 * 2 ** attempt, 30_000);
    setTimeout(() => connect(attempt + 1), delay);
  };
}
```

## 相关文档

- [资源 API](/docs/api/resources) — 对相同资源执行 CRUD 操作的 REST 端点
- [认证](/docs/api/authentication) — 如何获取 `token` 查询参数所需的访问令牌
