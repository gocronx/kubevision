---
sidebar_position: 2
title: 认证
---

# 认证

KubeVision 支持本地密码、TOTP、通行密钥/安全密钥、目录账户以及已配置的
OAuth/OIDC 提供商。认证成功后会按照管理员配置的有效期签发访问令牌和刷新令牌。

## 密码或目录登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{"username":"admin","password":"your-password"}
```

目录用户使用同一接口。启用目录登录后，后端会根据目录策略解析身份以及组到
角色的映射。

启用 TOTP 时，登录返回业务码 `40102` 和短期临时令牌，再调用以下公开接口：

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/auth/2fa/verify` | 验证 TOTP |
| `POST` | `/auth/2fa/recovery` | 使用恢复码 |

## 刷新令牌

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{"refreshToken":"<refresh-token>"}
```

受保护请求使用返回的访问令牌：

```http
Authorization: Bearer <access-token>
```

## OAuth 和 OIDC

| 方法 | 路径 | 用途 |
|------|------|------|
| `GET` | `/auth/oauth/providers` | 列出已配置的提供商 |
| `GET` | `/auth/oauth/:provider/authorize` | 开始授权 |
| `GET` | `/auth/oauth/:provider/callback` | 提供商回调 |

提供商名称必须对应 `oauth.providers` 中的条目。OIDC 可通过 `issuer` 进行发现，
标准 OAuth 提供商也可以显式配置授权、令牌和用户信息地址。回调地址必须与提供商
中登记的地址完全一致。

## 通行密钥和安全密钥

先检查公钥认证是否启用：

```http
GET /api/v1/auth/public-key/config
```

认证使用两步 WebAuthn 交换：

| 方法 | 路径 |
|------|------|
| `POST` | `/auth/public-key/login/begin` |
| `POST` | `/auth/public-key/login/finish` |

注册和凭据管理需要已有登录会话，接口见[接口索引](/docs/api/endpoints)。浏览器
来源、依赖方 ID 和 HTTPS 要求见[认证提供商](/docs/admin-guide/authentication-providers)。

## 令牌安全

- 不要将 Bearer Token 放入任意 URL。
- 修改密码或认证安全状态可能通过用户令牌版本使已有会话失效。
- TOTP 恢复码应安全保存，并仅用于一次性恢复。
- 登录、OAuth 回调、WebAuthn 和所有认证接口都应使用 TLS。
