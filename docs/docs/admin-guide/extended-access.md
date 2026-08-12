---
sidebar_position: 8
title: Extended Access and Authentication
---

# Extended Access and Authentication

This page describes the security boundaries for registry tag discovery, Helm
release operations, directory login, public-key credentials, and Kubernetes
HTTP access.

## Registry tag discovery

Image tag lookup is anonymous and read-only. Docker Hub is allowed by default.
Add exact registry and token-service hosts through `registry.allowed_registries`
and `registry.allowed_auth_hosts`, or the comma-separated
`KUBEVISION_REGISTRY_ALLOWED` and `KUBEVISION_REGISTRY_AUTH_HOSTS` variables.
Private destinations and plain HTTP remain disabled unless an administrator
explicitly enables them. Registry credentials are not stored by this feature.

## Package releases

The Packages workspace lists Helm releases, revision history, and redacted
values. Editors can roll back or remove a release; viewers receive read-only
access. Removal requires typing the exact release name. Existing cluster and
namespace role assignments are enforced when present, and concurrent changes
to the same release are rejected.

This delivery does not install or upgrade charts and does not register chart
repositories. Those operations require a separate preview and credential
management design.

## Directory login

Directory settings are managed under **Administration > Directory**. Use LDAPS
or LDAP with StartTLS. Plain LDAP is off by default and is always rejected when
Gin runs in release mode. The bind password is encrypted at rest and is never
returned by the API.

The user filter must contain exactly one `{{username}}` placeholder. KubeVision
escapes the supplied identifier before searching. A directory identity is
linked by its stable directory identifier; matching email addresses or user
names are not merged automatically. Group mappings use exact identifiers and
the lowest numeric priority wins. Saving directory policy revokes current
directory sessions so changed privileges take effect on the next login.

## Passkeys and security keys

Enable public-key authentication with a stable relying-party configuration:

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

The equivalent environment variables are
`KUBEVISION_PUBLIC_KEY_ENABLED`, `KUBEVISION_PUBLIC_KEY_RP_ID`,
`KUBEVISION_PUBLIC_KEY_RP_NAME`, `KUBEVISION_PUBLIC_KEY_ORIGINS`,
`KUBEVISION_PUBLIC_KEY_UV`, `KUBEVISION_PUBLIC_KEY_COUNTER_POLICY`, and
`KUBEVISION_PUBLIC_KEY_CHALLENGE_TTL`. Origins must use HTTPS and match the RP
domain boundary. Changing the RP ID or allowed origins can make registered
credentials unusable, so treat these values as persistent security state.

Registration requires the current password or an enabled TOTP factor. Users
can rename or revoke their own credentials, but cannot remove their last usable
authentication or recovery method. Administrators may revoke credentials but
cannot register one for another user.

## Kubernetes HTTP access

The backend can proxy GET and HEAD requests to a selected Pod or Service through
the Kubernetes proxy subresource. It accepts structured cluster, namespace,
resource, port, and path fields rather than an arbitrary URL. Request credentials
and hop-by-hop headers are removed, redirects are not followed, and response
bodies are bounded.

There is currently no new-tab UI for this endpoint. Browser navigation cannot
attach the application's bearer token safely, and tokens are never placed in a
URL. Use the authenticated API until a same-origin browser handoff is designed.
