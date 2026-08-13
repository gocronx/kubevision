---
sidebar_position: 1
title: API 概览
---

# API 概览

KubeVision Web 应用使用的 HTTP API 位于 `/api/v1`。使用默认配置在本地
运行时，基础地址为 `http://localhost:8080/api/v1`。

## 响应格式

大多数 REST 处理器返回统一结构：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "meta": {"total": 10, "requestId": "request-id"}
}
```

`code` 为 `0` 表示成功，非零值表示业务错误，详见
[错误码](/docs/api/error-codes)。不需要时会省略 `data` 或 `meta`。列表接口
可能提供 `meta.total`，资源接口还可能提供缓存来源和过期状态。

当前多数带统一结构的业务错误使用 HTTP `200`，客户端必须检查 `code`。
健康检查和 WebSocket 握手错误使用常规 HTTP 状态码，也不一定采用统一结构。

## 认证与授权

受保护的请求需要访问令牌：

```http
Authorization: Bearer <access-token>
```

认证用于确认用户身份，RBAC 随后检查操作、集群和命名空间权限。AI 助手和
Kubernetes HTTP 访问还会在处理器中执行资源级授权。

## 请求数据

JSON 请求体应设置 `Content-Type: application/json`。列表过滤、命名空间、
分页和搜索使用查询参数。路径中的 `:id`、`:name` 等参数需要由客户端正确
进行 URL 编码。

## 健康检查

| 路径 | 用途 |
|------|------|
| `GET /healthz` | 进程存活检查 |
| `GET /readyz` | 就绪检查，包括数据库连接 |

这两个接口不在 `/api/v1` 下，也不需要认证。

## 参考

- [认证](/docs/api/authentication)
- [资源 API](/docs/api/resources)
- [接口索引](/docs/api/endpoints)
- [流式 API](/docs/api/websocket)
- [错误码](/docs/api/error-codes)
