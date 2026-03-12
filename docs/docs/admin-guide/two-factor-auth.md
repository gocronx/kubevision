---
sidebar_position: 2
title: Two-Factor Authentication
---

# Two-Factor Authentication

KubeVision supports TOTP (Time-based One-Time Password) as a second authentication factor. It is compatible with any standard authenticator app.

## Compatible Apps

- Google Authenticator (iOS / Android)
- Authy
- 1Password
- Microsoft Authenticator
- Any RFC 6238-compliant app

## Setup Flow

Users enable 2FA from **Profile → Security → Enable 2FA**:

1. The backend generates a random TOTP secret and returns a QR code URL
2. The user scans the QR code with their authenticator app
3. The user enters the 6-digit code shown in the app to confirm the scan
4. On successful verification, the backend marks 2FA as active and returns **10 single-use recovery codes**

:::warning
Recovery codes are shown only once. Instruct users to save them in a password manager or printed in a secure location before closing the dialog.
:::

## Login Flow

```
1. POST /api/v1/auth/login  { username, password }
        ↓ credentials valid?
2. Server checks: is 2FA enabled for this user?
        ↓ yes
3. Return 202 with { mfa_required: true, mfa_token: "<short-lived token>" }
        ↓
4. POST /api/v1/auth/mfa  { mfa_token, totp_code }
        ↓ code valid?
5. Issue full JWT → login complete
```

If 2FA is not enabled the full JWT is issued at step 1.

## Enforcing 2FA for All Users

Admins can make 2FA mandatory across the entire platform:

1. Go to **Settings → Security**
2. Toggle **Require 2FA for all users**
3. Click **Save**

Once enforced, users who have not yet set up 2FA are redirected to the setup page on their next login and cannot proceed until setup is complete.

:::tip
You can also enforce 2FA per role. For example, require it for `admin` and `ops` while leaving it optional for `readonly` users.
:::

## Recovery Codes

Each user has 10 recovery codes generated at setup time. A recovery code can be used in place of a TOTP code at the MFA prompt:

- Each code is single-use and is invalidated immediately after use
- The remaining code count is shown in **Profile → Security**
- When fewer than 3 codes remain, a warning banner appears in the UI
- Users can regenerate all 10 codes from **Profile → Security → Regenerate Recovery Codes** (requires TOTP confirmation)

## Sensitive Operation Re-verification

Certain high-impact actions require the user to re-enter their TOTP code even within an active session:

| Action | Re-verification Required |
|--------|--------------------------|
| Delete a cluster | Yes |
| View Secret plaintext | Yes |
| Modify RBAC assignments | Yes |
| Revoke another user's sessions | Yes |

The frontend will show a TOTP prompt inline before submitting these requests. The code is verified server-side on each sensitive endpoint.

## Related

- [RBAC](/docs/admin-guide/rbac) — Restrict which users can access which clusters
- [Audit Logging](/docs/admin-guide/audit-logging) — 2FA enable/disable events are recorded
