---
sidebar_position: 4
title: API Keys
---

# API Keys

API keys provide an alternative to JWT bearer tokens for programmatic access to the KubeVision API. They are designed for CI/CD pipelines, scripts, and other non-interactive clients.

## Creating an API Key

1. Go to **Profile → API Keys**
2. Click **New API Key**
3. Enter a descriptive name (e.g., `github-actions-deploy`)
4. Optionally set an expiry date
5. Click **Generate**

The key is shown **once** immediately after creation. Copy it to a secure location — it cannot be retrieved again.

:::warning
If you lose an API key, revoke it and generate a new one. There is no way to view the key value after the creation dialog is closed.
:::

## Using an API Key

Pass the key in the `Authorization` header using the `ApiKey` scheme:

```bash
curl https://kubevision.example.com/api/v1/clusters \
  -H "Authorization: ApiKey kv_live_abc123xyz..."
```

The server looks up the hashed key, resolves the owning user, and proceeds identically to a JWT-authenticated request.

## Security Model

| Property | Behavior |
|----------|----------|
| Storage | SHA-256 hash stored in database; plaintext never persisted |
| Permissions | Identical to the owning user's RBAC role and cluster assignments |
| Expiry | Optional — keys without an expiry are valid until explicitly revoked |
| Rate limiting | Subject to the same per-user rate limits as JWT sessions |
| Audit | All API key requests are logged under the owning user's name with the key name as the agent |

## Revoking an API Key

1. Go to **Profile → API Keys**
2. Find the key row and click **Revoke**
3. Confirm the dialog

Revocation takes effect immediately — in-flight requests using the revoked key will fail from that point on.

:::tip
Rotate API keys regularly. Use the **Expiry** field to enforce automatic rotation. A key nearing expiry shows an amber badge in the key list.
:::

## Admin View

Admins can see and revoke API keys for any user under **Settings → Users → (user) → API Keys**. This is useful when a team member leaves or a key is suspected to be compromised.

## CI/CD Example

```yaml
# GitHub Actions example
- name: Scale deployment
  env:
    KUBEVISION_API_KEY: ${{ secrets.KUBEVISION_API_KEY }}
  run: |
    curl -X PATCH https://kubevision.example.com/api/v1/clusters/prod/namespaces/default/deployments/api-server \
      -H "Authorization: ApiKey $KUBEVISION_API_KEY" \
      -H "Content-Type: application/json" \
      -d '{"spec": {"replicas": 5}}'
```

## Related

- [RBAC](/docs/admin-guide/rbac) — Permissions that apply to API key requests
- [Audit Logging](/docs/admin-guide/audit-logging) — API key usage is recorded in the audit log
- [Two-Factor Authentication](/docs/admin-guide/two-factor-auth) — API keys bypass MFA (intended for automation)
