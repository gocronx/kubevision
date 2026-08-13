---
title: 认证提供商
---

# 认证提供商

KubeVision 可以同时使用本地账户、OAuth/OIDC、目录登录、TOTP 和 WebAuthn
通行密钥/安全密钥。生产环境中的认证接口必须通过 HTTPS 提供。

## OAuth 和 OIDC

在服务端配置中启用提供商：

```yaml
oauth:
  enabled: true
  providers:
    - name: github
      client_id: "client-id"
      client_secret: "client-secret"
      auth_url: "https://github.com/login/oauth/authorize"
      token_url: "https://github.com/login/oauth/access_token"
      userinfo_url: "https://api.github.com/user"
      scopes: ["read:user", "user:email"]
      redirect_url: "https://kubevision.example.com/api/v1/auth/oauth/github/callback"
```

OIDC 提供商可以设置 `issuer` 并使用发现文档提供端点。提供商名称会出现在 URL
路径中，必须唯一。回调地址应准确登记，客户端密钥不得提交到源码仓库。

## 目录登录

在**管理 > 目录**中配置 LDAP 兼容目录。优先使用 LDAPS 或 LDAP + StartTLS。
明文 LDAP 默认关闭，并且在 release 模式下始终拒绝。Bind 密码加密保存，API
不会返回明文。

用户过滤器必须恰好包含一个 `{{username}}` 占位符，搜索前会对输入进行转义。
身份按稳定的目录 ID 关联，不会因为邮箱或用户名相同而自动合并。组映射使用准确
标识符，数值最小的优先级获胜。

启用前使用**测试**检查连接，并通过**预览**检查用户所属组和最终角色。

## 通行密钥和安全密钥

```yaml
auth:
  public_key:
    enabled: true
    rp_id: kubevision.example.com
    rp_display_name: KubeVision
    origins:
      - https://kubevision.example.com
    user_verification: required
    counter_policy: deny
    challenge_ttl: 5m
```

对应变量包括 `KUBEVISION_PUBLIC_KEY_ENABLED`、
`KUBEVISION_PUBLIC_KEY_RP_ID`、`KUBEVISION_PUBLIC_KEY_RP_NAME`、
`KUBEVISION_PUBLIC_KEY_ORIGINS`、`KUBEVISION_PUBLIC_KEY_UV`、
`KUBEVISION_PUBLIC_KEY_COUNTER_POLICY` 和
`KUBEVISION_PUBLIC_KEY_CHALLENGE_TTL`。

来源必须使用 HTTPS，并位于依赖方域名边界内。修改 RP ID 或来源可能导致已有
凭据失效。注册凭据前，用户必须通过当前密码或已启用的 TOTP 验证。
