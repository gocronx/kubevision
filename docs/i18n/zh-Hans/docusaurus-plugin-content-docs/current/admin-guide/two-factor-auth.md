---
sidebar_position: 2
title: 双因素认证
---

# 双因素认证

KubeVision 支持 TOTP（基于时间的一次性密码）作为第二认证因子，兼容所有符合标准的身份验证器应用。

## 兼容应用

- Google Authenticator（iOS / Android）
- Authy
- 1Password
- Microsoft Authenticator
- 任何符合 RFC 6238 标准的应用

## 设置流程

用户在 **Profile → Security → Enable 2FA** 中启用 2FA：

1. 后端生成随机 TOTP 密钥并返回二维码 URL
2. 用户使用身份验证器应用扫描二维码
3. 用户输入应用中显示的 6 位验证码以确认扫描成功
4. 验证通过后，后端将 2FA 标记为已激活，并返回 **10 个一次性恢复码**

:::warning
恢复码仅显示一次。请告知用户在关闭对话框前将其保存至密码管理器或打印后存放于安全位置。
:::

## 登录流程

```
1. POST /api/v1/auth/login  { username, password }
        ↓ 凭据有效？
2. 服务器检查：该用户是否已启用 2FA？
        ↓ 是
3. 返回 202，携带 { mfa_required: true, mfa_token: "<短期令牌>" }
        ↓
4. POST /api/v1/auth/mfa  { mfa_token, totp_code }
        ↓ 验证码有效？
5. 签发完整 JWT → 登录完成
```

若未启用 2FA，则在第 1 步直接签发完整 JWT。

## 对所有用户强制启用 2FA

管理员可在整个平台范围内强制要求 2FA：

1. 进入 **Settings → Security**
2. 开启 **Require 2FA for all users**
3. 点击 **Save**

强制启用后，尚未设置 2FA 的用户在下次登录时将被重定向至设置页面，完成设置前无法继续操作。

:::tip
也可以按角色强制启用 2FA。例如，对 `admin` 和 `ops` 角色要求 2FA，而对 `readonly` 用户设为可选。
:::

## 恢复码

每位用户在设置时生成 10 个恢复码。在 MFA 验证提示处，恢复码可替代 TOTP 验证码使用：

- 每个恢复码为一次性使用，使用后立即失效
- 剩余恢复码数量显示在 **Profile → Security** 中
- 当剩余恢复码少于 3 个时，界面顶部会显示警告横幅
- 用户可在 **Profile → Security → Regenerate Recovery Codes** 中重新生成全部 10 个恢复码（需要 TOTP 确认）

## 敏感操作重新验证

某些高风险操作即使在活跃会话中也需要用户重新输入 TOTP 验证码：

| 操作 | 是否需要重新验证 |
|--------|--------------------------|
| 删除集群 | 是 |
| 查看 Secret 明文 | 是 |
| 修改 RBAC 分配 | 是 |
| 吊销其他用户的会话 | 是 |

前端将在提交这些请求前内联显示 TOTP 提示框。服务器端会在每个敏感接口上单独验证该验证码。

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — 限制哪些用户可以访问哪些集群
- [审计日志](/docs/admin-guide/audit-logging) — 2FA 启用/禁用事件均被记录
