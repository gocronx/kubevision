---
sidebar_position: 4
title: Endpoint Index
---

# Endpoint Index

This index covers non-resource routes registered by the current backend. Paths
are relative to `/api/v1`; protected routes require a bearer token and RBAC.

## Account and Identity

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/users/me` | Current user |
| `PUT` | `/users/me/password` | Change own password |
| `POST` | `/auth/2fa/setup` | Start TOTP setup |
| `POST` | `/auth/2fa/enable` | Enable TOTP |
| `POST` | `/auth/2fa/disable` | Disable TOTP |
| `POST` | `/auth/public-key/register/begin` | Begin passkey registration |
| `POST` | `/auth/public-key/register/finish` | Finish passkey registration |
| `GET` | `/auth/public-key/credentials` | List own credentials |
| `PUT` | `/auth/public-key/credentials/:id` | Rename a credential |
| `DELETE` | `/auth/public-key/credentials/:id` | Revoke a credential |

Public login, OAuth, refresh, and public-key login routes are documented in
[Authentication](/docs/api/authentication).

## Administration

| Method | Path | Description |
|--------|------|-------------|
| `GET`, `POST` | `/users` | List or create users |
| `GET`, `PUT`, `DELETE` | `/users/:id` | Read, update, or delete a user |
| `PUT` | `/users/:id/reset-password` | Reset a password |
| `DELETE` | `/users/:id/public-key-credentials/:credentialId` | Revoke a user's credential |
| `GET`, `PUT` | `/directory/config` | Read or update directory settings |
| `POST` | `/directory/test` | Test directory connectivity/settings |
| `POST` | `/directory/preview` | Preview group-to-role mapping |
| `GET` | `/audit-logs` | Query audit events |
| `GET` | `/terminal-sessions` | List recorded terminal sessions |
| `GET` | `/terminal-sessions/:id` | Session metadata |
| `GET` | `/terminal-sessions/:id/play` | Replay data |

## Clusters and Packages

| Method | Path | Description |
|--------|------|-------------|
| `GET`, `POST` | `/clusters` | List or add clusters |
| `GET`, `DELETE` | `/clusters/:id` | Read or remove a cluster |
| `GET` | `/clusters/:id/package-releases` | List Helm releases |
| `GET` | `/clusters/:id/package-releases/:namespace/:name` | Release details |
| `GET` | `/clusters/:id/package-releases/:namespace/:name/history` | Revision history |
| `POST` | `/clusters/:id/package-releases/:namespace/:name/rollback` | Roll back |
| `DELETE` | `/clusters/:id/package-releases/:namespace/:name` | Remove release |
| `GET` | `/registry-tags` | Discover image tags |

## AI, Plugins, and Automation

| Method | Path | Description |
|--------|------|-------------|
| `GET`, `PUT` | `/ai/config` | Read or update AI configuration |
| `POST` | `/ai/models` | Query models from a configured provider |
| `POST` | `/ai/chat` | Stream an assistant response using SSE |
| `POST` | `/ai/continue-action` | Continue a confirmed tool action |
| `GET` | `/plugins` | List built-in plugins |
| `GET`, `PUT` | `/plugins/:name` | Read or update plugin configuration |
| `GET` | `/plugins/:name/health` | Check plugin health |
| `GET` | `/plugins/prometheus/query` | Run a bounded instant PromQL query |
| `GET` | `/plugins/grafana/dashboards` | List Grafana dashboards |
| `GET` | `/plugins/argocd/applications` | List Argo CD applications |
| `GET`, `POST` | `/webhooks` | List or create webhooks |
| `PUT`, `DELETE` | `/webhooks/:id` | Update or delete a webhook |
| `POST` | `/webhooks/:id/test` | Send a test event |

Favorites (`/favorites`) and saved templates (`/templates`) also expose their
standard list/create/delete routes. See the corresponding user guides for UI
workflows and RBAC behavior.

## Kubernetes HTTP Access

`GET` and `HEAD` requests to
`/clusters/:id/namespaces/:namespace/http/:kind/:name/*path` proxy a bounded,
resource-authorized request to a Pod or Service. It is not a general-purpose
URL proxy; see [Extended Access](/docs/admin-guide/extended-access).
