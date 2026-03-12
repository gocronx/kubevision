---
sidebar_position: 4
title: API Keys
---

# API Keys

API Keys 为程序化访问 KubeVision API 提供了 JWT Bearer 令牌之外的替代方案，专为 CI/CD 流水线、脚本及其他非交互式客户端设计。

## 创建 API Key

1. 进入 **Profile → API Keys**
2. 点击 **New API Key**
3. 输入一个描述性名称（例如 `github-actions-deploy`）
4. 可选：设置过期日期
5. 点击 **Generate**

密钥在创建后**仅显示一次**，请立即将其复制到安全位置，之后无法再次查看。

:::warning
若丢失 API Key，请吊销后重新生成。创建对话框关闭后，密钥值将无法再次查看。
:::

## 使用 API Key

在 `Authorization` 请求头中使用 `ApiKey` 方案传递密钥：

```bash
curl https://kubevision.example.com/api/v1/clusters \
  -H "Authorization: ApiKey kv_live_abc123xyz..."
```

服务器查找哈希后的密钥，解析出其所有者用户，并以与 JWT 认证请求完全相同的方式处理后续流程。

## 安全模型

| 属性 | 行为 |
|----------|----------|
| 存储方式 | 数据库中仅存储 SHA-256 哈希值，明文永不持久化 |
| 权限范围 | 与所有者用户的 RBAC 角色和集群分配完全一致 |
| 有效期 | 可选——未设置过期时间的密钥在显式吊销前始终有效 |
| 速率限制 | 与 JWT 会话使用相同的每用户速率限制 |
| 审计 | 所有 API Key 请求均记录在所有者用户名下，以密钥名称作为代理标识 |

## 吊销 API Key

1. 进入 **Profile → API Keys**
2. 找到对应密钥行，点击 **Revoke**
3. 在对话框中确认操作

吊销立即生效——使用已吊销密钥的进行中请求从此刻起将全部失败。

:::tip
定期轮换 API Keys。使用 **Expiry** 字段强制自动轮换。即将过期的密钥在密钥列表中会显示橙色徽章。
:::

## 管理员视图

管理员可在 **Settings → Users → (用户) → API Keys** 下查看并吊销任意用户的 API Keys，适用于团队成员离职或密钥疑似泄露的场景。

## CI/CD 示例

```yaml
# GitHub Actions 示例
- name: Scale deployment
  env:
    KUBEVISION_API_KEY: ${{ secrets.KUBEVISION_API_KEY }}
  run: |
    curl -X PATCH https://kubevision.example.com/api/v1/clusters/prod/namespaces/default/deployments/api-server \
      -H "Authorization: ApiKey $KUBEVISION_API_KEY" \
      -H "Content-Type: application/json" \
      -d '{"spec": {"replicas": 5}}'
```

## 相关文档

- [RBAC](/docs/admin-guide/rbac) — 适用于 API Key 请求的权限体系
- [审计日志](/docs/admin-guide/audit-logging) — API Key 使用情况记录在审计日志中
- [双因素认证](/docs/admin-guide/two-factor-auth) — API Keys 绕过 MFA（专为自动化场景设计）
