---
sidebar_position: 2
title: Authentication
---

# Authentication

KubeVision uses JWT-based authentication. An access token (15-minute TTL) and a refresh token (7-day TTL) are issued at login. All protected endpoints require the access token in the `Authorization` header.

## Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

### Response — Login Success

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

### Response — 2FA Required

When the user has 2FA enabled, the login response contains a session token instead of the final JWTs:

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

## Two-Factor Authentication

### Verify a TOTP Code

```http
POST /api/v1/auth/2fa/verify
Content-Type: application/json

{
  "sessionToken": "<short-lived-token>",
  "code": "123456"
}
```

Returns the same `accessToken` / `refreshToken` pair as a normal login on success.

### Set Up 2FA

```http
POST /api/v1/auth/2fa/setup
Authorization: Bearer <access_token>
```

Returns a `otpauthUrl` and `recoveryCodes`. Scan the QR code in any TOTP app (Google Authenticator, Authy, 1Password).

```json
{
  "code": 0,
  "data": {
    "otpauthUrl":    "otpauth://totp/KubeVision:admin?secret=BASE32SECRET",
    "recoveryCodes": ["abc12-def34", "ghi56-jkl78"]
  }
}
```

### Enable / Disable 2FA

| Action | Endpoint |
|--------|----------|
| Enable | `POST /api/v1/auth/2fa/enable` |
| Disable | `POST /api/v1/auth/2fa/disable` — requires current TOTP code in body |

:::warning
Recovery codes are shown **only once** at setup time. Store them securely. Each code can be used in place of a TOTP code and is invalidated after one use.
:::

## Refresh Tokens

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "<refresh_jwt>"
}
```

Returns a new `accessToken`. The refresh token itself is not rotated unless it is within 24 hours of expiry, in which case a new refresh token is also issued.

## Token Expiry

| Token | TTL |
|-------|-----|
| Access token | 15 minutes |
| Refresh token | 7 days |
| 2FA session token | 5 minutes |

## Token Revocation

KubeVision uses a `tokenVersion` field on the user record. Incrementing `tokenVersion` (via **Settings → Users → Revoke Sessions**) invalidates all tokens issued before that point — the JWT validation middleware rejects tokens whose embedded version does not match the current database value.

## Related

- [RBAC](/docs/admin-guide/rbac) — What permissions are embedded in the JWT
- [Error Codes](/docs/api/error-codes) — `401xx` auth error codes explained
