---
sidebar_position: 1
title: API 概览
---

# API 概览

KubeVision 提供一套 RESTful HTTP API，供前端调用，同时也可用于编程访问。

## 基础 URL

所有 API 端点均挂载于：

```
/api/v1
```

本地运行时解析为 `http://localhost:8080/api/v1`。

## 统一响应格式

所有响应——无论成功还是失败——均使用相同的 JSON 包装结构：

```json
{
  "code":    0,
  "message": "success",
  "data":    {},
  "meta":    {}
}
```

| 字段 | 类型 | 描述 |
|-------|------|-------------|
| `code` | `int` | 业务状态码。`0` 表示成功，非零值表示错误。详见[错误码](/docs/api/error-codes)。 |
| `message` | `string` | 结果的人类可读描述。 |
| `data` | `object \| array \| null` | 响应载荷。对于无输出的操作（如删除），值为 `null`。 |
| `meta` | `object` | 请求元数据，所有响应中均存在。 |

:::tip
所有 HTTP 响应均返回**状态码 200**。业务结果完全由 `code` 字段决定。不要单独依赖 HTTP 状态码进行分支判断。
:::

## Meta 对象

`meta` 字段携带了关于本次响应如何产生的上下文信息：

```json
{
  "meta": {
    "source":    "cache",
    "stale":     false,
    "total":     142,
    "requestId": "req_01HXYZ4B9K0VGMZS"
  }
}
```

| 字段 | 类型 | 描述 |
|-------|------|-------------|
| `source` | `"cache" \| "apiserver"` | 数据来源于 Informer 缓存还是实时的 API Server 调用。 |
| `stale` | `bool` | `true` 表示缓存条目已超过配置的 TTL。 |
| `total` | `int` | 列表端点的数据总条数，用于分页显示。 |
| `requestId` | `string` | 本次请求的唯一标识符，提交 bug 报告时请附上此值。 |

## 内容类型

所有请求必须包含：

```
Content-Type: application/json
```

所有响应均以 `application/json; charset=utf-8` 格式返回。

## 认证

请求必须在 `Authorization` 请求头中携带有效的 JWT：

```
Authorization: Bearer <access_token>
```

关于如何获取令牌，请参阅[认证](/docs/api/authentication)。

## 相关文档

- [认证](/docs/api/authentication) — 登录、双因素认证、令牌刷新
- [资源 API](/docs/api/resources) — Kubernetes 资源的 CRUD 端点
- [WebSocket API](/docs/api/websocket) — 实时监听、终端及日志流
- [错误码](/docs/api/error-codes) — 完整业务状态码表
