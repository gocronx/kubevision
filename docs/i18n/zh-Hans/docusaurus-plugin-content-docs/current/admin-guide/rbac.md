---
sidebar_position: 1
title: RBAC
---

# RBAC

KubeVision 使用五级角色体系来控制用户对各集群和命名空间中资源的查看与修改权限。权限在用户登录时直接嵌入 JWT，每次请求无需查询数据库。

## 内置角色

| 角色 | 级别 | 适用对象 |
|------|-------|--------------|
| `admin` | 5 | 完整系统访问权限，可管理用户和集群 |
| `ops` | 4 | 跨集群读写，无用户管理权限 |
| `dev` | 3 | 在已分配命名空间内读写 |
| `readonly` | 2 | 在已分配集群/命名空间内只读 |
| `custom` | 1 | 按资源类型逐项定义权限 |

## 权限格式

权限遵循 `resource:action` 约定：

```
clusters:read
clusters:write
clusters:delete
namespaces:read
pods:exec
secrets:reveal
rbac:manage
audit:read
```

`admin` 角色隐式拥有 `*:*`。其他所有内置角色均由上述基本权限单元组合而成。

## JWT Claims

登录后，用户的角色及集群/命名空间分配信息将直接编码进 JWT：

```json
{
  "sub": "user-42",
  "role": "dev",
  "clusters": ["prod-us", "staging"],
  "namespaces": ["team-alpha", "team-beta"],
  "permissions": ["pods:read", "pods:write", "pods:exec", "deployments:read", "deployments:write"],
  "exp": 1760000000
}
```

:::tip
由于权限已包含在令牌中，角色变更将在用户**下次登录**后生效。如需立即生效，请在 **Settings → Users → Sessions** 中吊销该用户的活跃会话。
:::

## 分配角色

1. 进入 **Settings → Users**
2. 点击用户行以打开详情抽屉
3. 在 **Role** 下拉菜单中选择角色
4. 在 **Cluster Access** 下选择一个或多个集群
5. 可选：在 **Namespace Access** 下限制到特定命名空间
6. 点击 **Save**

## 用户-集群-命名空间映射

实际访问权限始终是用户角色权限与其显式集群/命名空间分配的交集：

```
effective_access = role_permissions ∩ assigned_clusters ∩ assigned_namespaces
```

一个被分配到 `prod-us / team-alpha` 的 `dev` 用户，即使其角色原本允许，也无法查看 `team-beta` 或其他集群中的任何资源。

## 自定义角色

`custom` 角色初始没有任何权限，需要手动逐项添加：

1. 进入 **Settings → Roles → New Role**
2. 为角色命名（例如 `ci-runner`）
3. 逐项开启或关闭各项权限
4. 像常规角色一样将其分配给用户

:::warning
自定义角色存储在数据库中，并在登录时解析。拥有自定义角色的用户在角色定义更改后必须重新登录。
:::

## 相关文档

- [双因素认证](/docs/admin-guide/two-factor-auth) — 为特权角色强制启用 2FA
- [审计日志](/docs/admin-guide/audit-logging) — 追踪所有角色分配操作
- [API Keys](/docs/admin-guide/api-keys) — 受用户权限约束的程序化访问
