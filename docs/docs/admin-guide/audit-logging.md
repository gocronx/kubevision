---
sidebar_position: 3
title: Audit Logging
---

# Audit Logging

Every write operation in KubeVision is recorded asynchronously to a structured audit log. Logs are stored in the database and auto-purged after 90 days.

## What Gets Logged

All state-changing operations are captured, including:

- Creating, updating, or deleting Kubernetes resources
- Adding or removing clusters
- User login, logout, and failed authentication attempts
- RBAC assignment changes
- 2FA enable/disable events
- API key creation and revocation
- Webhook configuration changes
- Terminal session start and end

Read-only operations (list, get, watch) are not logged by default to keep volume manageable.

## Log Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique log entry identifier |
| `user` | string | Username who performed the action |
| `action` | string | Verb: `create`, `update`, `delete`, `login`, etc. |
| `resource` | string | Resource kind: `Deployment`, `Secret`, `User`, etc. |
| `resource_name` | string | Name of the affected resource |
| `cluster` | string | Target cluster (empty for system actions) |
| `namespace` | string | Target namespace (empty for cluster-scoped resources) |
| `status_code` | int | HTTP status code of the response |
| `detail` | JSON | Before/after diff or request payload summary |
| `ip` | string | Client IP address |
| `timestamp` | RFC3339 | When the action occurred |

## Async Batch Writes

Audit entries are written in batches to avoid adding latency to user requests:

```
Request → Handler → (response returned) → channel ← batch worker → DB (every 500ms or 100 entries)
```

If the application crashes before a batch is flushed, entries in that batch may be lost. This is an accepted trade-off for write performance. For compliance environments requiring guaranteed delivery, configure PostgreSQL and enable synchronous writes via `audit.sync: true` in the configuration file.

:::warning
Setting `audit.sync: true` adds a synchronous DB write to every mutating request. Benchmark your database before enabling this in production.
:::

## Retention Policy

Audit logs are automatically deleted after **90 days**. A background job runs nightly at 02:00 UTC and removes entries older than the retention window.

To change the retention period:

```yaml
# config.yaml
audit:
  retention_days: 90   # set to 0 to disable auto-cleanup
```

## Viewing Audit Logs

Go to **Settings → Audit Logs** in the admin panel.

### Filters

| Filter | Description |
|--------|-------------|
| **User** | Filter by a specific username |
| **Action** | Filter by verb (create, update, delete, login, ...) |
| **Resource** | Filter by resource kind |
| **Cluster** | Filter to a specific cluster |
| **Time Range** | From / to date-time picker |
| **Status** | Success (2xx) or failure (4xx/5xx) |

Results are paginated (50 per page) and can be exported as CSV from the toolbar.

:::tip
Combine the **User** and **Resource: Secret** filters to audit all access to sensitive resources for a specific team member.
:::

## Related

- [RBAC](/docs/admin-guide/rbac) — Control who can perform which actions
- [Two-Factor Authentication](/docs/admin-guide/two-factor-auth) — Auth events in the audit log
- [API Keys](/docs/admin-guide/api-keys) — API key usage is attributed to the owning user
