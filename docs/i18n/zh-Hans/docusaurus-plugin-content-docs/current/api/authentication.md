---
sidebar_position: 2
title: 认证
---

# 认证

KubeVision 使用基于 JWT 的认证方式。登录时会颁发一个访问令牌（TTL 15 分钟）和一个刷新令牌（TTL 7 天）。所有受保护的端点都需要在 `Authorization` 请求头中携带访问令牌。

## 登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

### 响应 — 登录成功

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken":  "<jwt>",
    "refreshToken": "<jwt>",
    "expiresIn":    900
  },
  "meta": { "requestId": "req_01HX" }
}
```

### 响应 — 需要双因素认证

当用户启用了双因素认证时，登录响应将包含一个会话令牌，而非最终的 JWT：

```json
{
  "code": 40102,
  "message": "2FA required",
  "data": {
    "requires2fa":    true,
    "sessionToken":   "<short-lived-token>"
  },
  "meta": { "requestId": "req_01HX" }
}
```

## 双因素认证

### 验证 TOTP 码

```http
POST /api/v1/auth/2fa/verify
Content-Type: application/json

{
  "sessionToken": "<short-lived-token>",
  "code": "123456"
}
```

验证成功后，返回与普通登录相同的 `accessToken` / `refreshToken` 对。

### 设置双因素认证

```http
POST /api/v1/auth/2fa/setup
Authorization: Bearer <access_token>
```

返回 `otpauthUrl` 和 `recoveryCodes`。使用任意 TOTP 应用（Google Authenticator、Authy、1Password）扫描二维码即可完成绑定。

```json
{
  "code": 0,
  "data": {
    "otpauthUrl":    "otpauth://totp/KubeVision:admin?secret=BASE32SECRET",
    "recoveryCodes": ["abc12-def34", "ghi56-jkl78"]
  }
}
```

### 启用 / 禁用双因素认证

| 操作 | 端点 |
|--------|----------|
| 启用 | `POST /api/v1/auth/2fa/enable` |
| 禁用 | `POST /api/v1/auth/2fa/disable` — 请求体中需提供当前 TOTP 码 |

:::warning
恢复码**仅在设置时显示一次**。请妥善保存。每个恢复码可代替一次 TOTP 码使用，使用后立即失效。
:::

## 刷新令牌

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "<refresh_jwt>"
}
```

返回一个新的 `accessToken`。刷新令牌本身不会轮换，除非其距离过期时间不足 24 小时，此时也会一并颁发新的刷新令牌。

## 令牌有效期

| 令牌 | 有效期 |
|-------|-----|
| 访问令牌 | 15 分钟 |
| 刷新令牌 | 7 天 |
| 双因素认证会话令牌 | 5 分钟 |

## 令牌吊销

KubeVision 在用户记录上维护一个 `tokenVersion` 字段。通过 **设置 → 用户 → 吊销会话** 递增 `tokenVersion`，可使该时间点之前颁发的所有令牌失效——JWT 验证中间件会拒绝嵌入版本号与当前数据库值不匹配的令牌。

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — JWT 中嵌入的权限信息
- [错误码](/docs/api/error-codes) — `401xx` 认证错误码说明
