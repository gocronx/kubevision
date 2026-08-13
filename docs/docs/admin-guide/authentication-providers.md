---
title: Authentication Providers
---

# Authentication Providers

KubeVision can combine local accounts with OAuth/OIDC, directory login, TOTP,
and WebAuthn passkeys or security keys. Always serve authentication endpoints
over HTTPS in production.

## OAuth and OIDC

Enable providers in the server configuration:

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

For an OIDC provider, set `issuer`; discovery can supply the provider
endpoints. Provider names are URL path identifiers and must be unique. Register
the exact callback URL and keep the client secret outside source control.

## Directory Login

Configure LDAP-compatible directories under **Administration > Directory**.
Prefer LDAPS or LDAP with StartTLS. Plain LDAP is disabled by default and is
rejected in release mode. The bind password is encrypted at rest and is never
returned by the API.

The user filter must contain exactly one `{{username}}` placeholder. Input is
escaped before search. Identities are linked by their stable directory ID;
matching email addresses or usernames are not merged automatically. Group
mappings use exact identifiers, and the lowest numeric priority wins.

Use **Test** to verify connectivity and **Preview** to inspect a user's groups
and resulting role before enabling login.

## Passkeys and Security Keys

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

Equivalent variables are `KUBEVISION_PUBLIC_KEY_ENABLED`,
`KUBEVISION_PUBLIC_KEY_RP_ID`, `KUBEVISION_PUBLIC_KEY_RP_NAME`,
`KUBEVISION_PUBLIC_KEY_ORIGINS`, `KUBEVISION_PUBLIC_KEY_UV`,
`KUBEVISION_PUBLIC_KEY_COUNTER_POLICY`, and
`KUBEVISION_PUBLIC_KEY_CHALLENGE_TTL`.

Origins must use HTTPS and remain within the relying-party domain boundary.
Changing the RP ID or origins can make existing credentials unusable. A user
must authenticate with the current password or an enabled TOTP factor before
registering a credential.
