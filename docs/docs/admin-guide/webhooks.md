---
sidebar_position: 5
title: Webhooks
---

# Webhooks

Webhooks push resource change events from KubeVision to external services in real time. Use them to send alerts to chat platforms, trigger CI/CD pipelines, or feed data into observability tools.

## Supported Platforms

| Platform | Type | Notes |
|----------|------|-------|
| Slack | Incoming Webhook | Block Kit message format |
| Discord | Incoming Webhook | Embed format |
| DingTalk | Robot Webhook | Markdown card format |
| Feishu (Lark) | Bot Webhook | Interactive card format |
| WeCom (WeChat Work) | Bot Webhook | Markdown message format |
| Custom HTTP | POST endpoint | Raw JSON payload, HMAC-signed |

## Creating a Webhook

1. Go to **Settings → Webhooks**
2. Click **New Webhook**
3. Choose a platform and paste the webhook URL
4. Configure event filters (see below)
5. Click **Save**, then use **Test** to verify delivery

## Event Filters

Narrow which events trigger delivery to avoid noise:

| Filter | Options |
|--------|---------|
| **Event type** | `create`, `update`, `delete`, `scale` |
| **Cluster** | One or more managed clusters, or `*` for all |
| **Resource type** | `Deployment`, `Pod`, `Service`, `Node`, etc. |
| **Namespace** | Specific namespace or `*` |

Events must match **all** active filters to be delivered.

## Payload Format (Custom HTTP)

```json
{
  "event": "update",
  "cluster": "prod-us",
  "namespace": "team-alpha",
  "resource": "Deployment",
  "name": "api-server",
  "actor": "alice",
  "timestamp": "2026-03-05T10:23:00Z",
  "detail": {
    "field": "spec.replicas",
    "old": 3,
    "new": 5
  }
}
```

## HMAC-SHA256 Signature

Every delivery to a custom HTTP endpoint includes a signature header so you can verify the request originated from KubeVision:

```
X-KubeVision-Signature: sha256=<hex digest>
```

Verification example in Go:

```go
mac := hmac.New(sha256.New, []byte(webhookSecret))
mac.Write(body)
expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
if !hmac.Equal([]byte(expected), []byte(r.Header.Get("X-KubeVision-Signature"))) {
    http.Error(w, "invalid signature", http.StatusUnauthorized)
}
```

:::tip
Store the webhook secret in an environment variable or secret manager on the receiver side — never hardcode it in source code.
:::

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/webhooks` | List all webhooks |
| `POST` | `/api/v1/webhooks` | Create a webhook |
| `PUT` | `/api/v1/webhooks/:id` | Update a webhook |
| `DELETE` | `/api/v1/webhooks/:id` | Delete a webhook |
| `POST` | `/api/v1/webhooks/:id/test` | Send a test event |

## Delivery Guarantees

KubeVision attempts delivery with an exponential backoff retry policy (3 attempts, max 30-second delay). Failed deliveries are recorded in the webhook's **Delivery History** tab in the UI with the HTTP status code and response body.

:::warning
Webhooks are best-effort. If your receiver is down for an extended period, events that exhaust the retry budget are permanently lost. For guaranteed delivery, consume events via the audit log export instead.
:::

## Related

- [Audit Logging](/docs/admin-guide/audit-logging) — Full event history when webhooks are insufficient
- [RBAC](/docs/admin-guide/rbac) — Only `admin` and `ops` roles can manage webhooks
