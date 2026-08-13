---
title: Extended Access
---

# Extended Access

This page describes outbound and proxy features that require a deliberately
configured trust boundary.

## Registry Tag Discovery

Image tag lookup is anonymous and read-only. Docker Hub is allowed by default.
Add exact registry and token-service hosts through `registry.allowed_registries`
and `registry.allowed_auth_hosts`, or through `KUBEVISION_REGISTRY_ALLOWED` and
`KUBEVISION_REGISTRY_AUTH_HOSTS`.

Private destinations and plain HTTP are disabled unless an administrator sets
`registry.allow_private` or `registry.allow_http`. Those switches broaden
server-side network access and should only be used on trusted networks. This
feature does not store registry credentials.

## Kubernetes HTTP Access

The backend can proxy bounded `GET` and `HEAD` requests to a selected Pod or
Service through the Kubernetes proxy subresource. The route accepts structured
cluster, namespace, kind, name, port, and path data rather than an arbitrary
destination URL.

The handler checks `pods:get` or `services:get`, removes request credentials and
hop-by-hop headers, does not follow redirects, and limits response bodies. It is
not an open HTTP proxy. Browser navigation cannot safely attach a bearer token,
so there is no unauthenticated new-tab handoff.

## Related Controls

- [Authentication Providers](/docs/admin-guide/authentication-providers)
- [Helm Package Releases](/docs/user-guide/package-releases)
- [RBAC](/docs/admin-guide/rbac)
