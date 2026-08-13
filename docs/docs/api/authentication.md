---
sidebar_position: 2
title: Authentication
---

# Authentication

KubeVision supports local passwords, TOTP, passkeys/security keys, directory
accounts, and configured OAuth/OIDC providers. Successful authentication issues
an access token and refresh token using the TTLs configured by the operator.

## Password or Directory Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{"username":"admin","password":"your-password"}
```

Directory users use the same endpoint. When directory login is enabled, the
backend resolves the identity and group-to-role mapping according to the saved
directory policy.

If TOTP is enabled, login returns business code `40102` and a short-lived
temporary token. Complete it with one of these public endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/auth/2fa/verify` | Verify a TOTP code |
| `POST` | `/auth/2fa/recovery` | Consume a recovery code |

## Refresh

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{"refreshToken":"<refresh-token>"}
```

Use the returned access token in protected requests:

```http
Authorization: Bearer <access-token>
```

## OAuth and OIDC

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/auth/oauth/providers` | List configured providers |
| `GET` | `/auth/oauth/:provider/authorize` | Begin authorization |
| `GET` | `/auth/oauth/:provider/callback` | Provider callback |

The provider name must match an entry in `oauth.providers`. KubeVision supports
OIDC discovery through `issuer`, or explicit authorization, token, and user-info
URLs for standard OAuth providers. The callback URL must exactly match the URL
registered with the provider.

## Passkeys and Security Keys

First check whether public-key authentication is enabled:

```http
GET /api/v1/auth/public-key/config
```

Authentication is a two-step WebAuthn exchange:

| Method | Path |
|--------|------|
| `POST` | `/auth/public-key/login/begin` |
| `POST` | `/auth/public-key/login/finish` |

Registration and credential management require an existing authenticated
session and are listed in the [Endpoint Index](/docs/api/endpoints). Browser
origin, relying-party ID, and HTTPS requirements are described in
[Authentication Providers](/docs/admin-guide/authentication-providers).

## Token Security

- Store access tokens only where the KubeVision client expects them; do not put
  bearer tokens in arbitrary URLs.
- Changing a user's password or security state can invalidate existing
  sessions through the user's token version.
- Store TOTP recovery codes securely. They are intended for one-time recovery.
- Use TLS for login, OAuth callbacks, WebAuthn, and all authenticated APIs.

## Related

- [Two-Factor Authentication](/docs/admin-guide/two-factor-auth)
- [Authentication Providers](/docs/admin-guide/authentication-providers)
- [RBAC](/docs/admin-guide/rbac)
