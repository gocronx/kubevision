---
sidebar_position: 1
title: RBAC
---

# RBAC

KubeVision uses a 5-level role system to control who can see and modify resources across clusters and namespaces. Permissions are embedded in the JWT at login time — no database lookup is required per request.

## Built-in Roles

| Role | Level | Intended For |
|------|-------|--------------|
| `admin` | 5 | Full system access, user and cluster management |
| `ops` | 4 | Cross-cluster read/write, no user management |
| `dev` | 3 | Read/write within assigned namespaces |
| `readonly` | 2 | Read-only across assigned clusters/namespaces |
| `custom` | 1 | Permissions defined explicitly per resource type |

## Permission Format

Permissions follow a `resource:action` convention:

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

The `admin` role implicitly holds `*:*`. All other built-in roles are composed from a fixed set of these atoms.

## JWT Claims

After login, the user's role and cluster/namespace assignments are encoded directly into the JWT:

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
Because permissions are in the token, role changes take effect on the user's **next login**. To force immediate effect, revoke the user's active sessions from **Settings → Users → Sessions**.
:::

## Assigning Roles

1. Go to **Settings → Users**
2. Click the user row to open the detail drawer
3. Select a role from the **Role** dropdown
4. Under **Cluster Access**, choose one or more clusters
5. Optionally restrict to specific namespaces under **Namespace Access**
6. Click **Save**

## User-Cluster-Namespace Mapping

Access is always the intersection of the user's role permissions and their explicit cluster/namespace assignments:

```
effective_access = role_permissions ∩ assigned_clusters ∩ assigned_namespaces
```

A `dev` user assigned to `prod-us / team-alpha` cannot see any resource in `team-beta` or in a different cluster, even if their role would otherwise allow it.

## Custom Roles

The `custom` role starts with no permissions. You build it up manually:

1. Go to **Settings → Roles → New Role**
2. Name the role (e.g., `ci-runner`)
3. Toggle individual permissions on or off
4. Assign the role to users as usual

:::warning
Custom roles are stored in the database and resolved at login. A user with a custom role must re-login after the role definition changes.
:::

## Related

- [Two-Factor Authentication](/docs/admin-guide/two-factor-auth) — Enforce 2FA for privileged roles
- [Audit Logging](/docs/admin-guide/audit-logging) — Track all role assignments
- [API Keys](/docs/admin-guide/api-keys) — Programmatic access scoped to user permissions
