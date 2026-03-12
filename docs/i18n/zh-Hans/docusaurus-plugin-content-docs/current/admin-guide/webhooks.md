---
sidebar_position: 5
title: Webhooks
---

# Webhooks

Webhooks 将 KubeVision 中的资源变更事件实时推送至外部服务。可用于向聊天平台发送告警、触发 CI/CD 流水线，或将数据接入可观测性工具。

## 支持的平台

| 平台 | 类型 | 备注 |
|----------|------|-------|
| Slack | Incoming Webhook | Block Kit 消息格式 |
| Discord | Incoming Webhook | Embed 格式 |
| DingTalk | 机器人 Webhook | Markdown 卡片格式 |
| Feishu (Lark) | 机器人 Webhook | 互动卡片格式 |
| WeCom (WeChat Work) | 机器人 Webhook | Markdown 消息格式 |
| 自定义 HTTP | POST 端点 | 原始 JSON 载荷，HMAC 签名 |

## 创建 Webhook

1. 进入 **Settings → Webhooks**
2. 点击 **New Webhook**
3. 选择平台并粘贴 Webhook URL
4. 配置事件筛选条件（见下文）
5. 点击 **Save**，然后使用 **Test** 验证是否能成功送达

## 事件筛选

通过缩小触发条件来减少无效通知：

| 筛选项 | 可选值 |
|--------|---------|
| **事件类型** | `create`、`update`、`delete`、`scale` |
| **集群** | 一个或多个托管集群，或 `*` 表示全部 |
| **资源类型** | `Deployment`、`Pod`、`Service`、`Node` 等 |
| **命名空间** | 特定命名空间或 `*` |

事件须**同时满足**所有已启用的筛选条件才会被送达。

## 载荷格式（自定义 HTTP）

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

## HMAC-SHA256 签名

向自定义 HTTP 端点的每次投递都包含一个签名请求头，用于验证请求确实来自 KubeVision：

```
X-KubeVision-Signature: sha256=<hex digest>
```

Go 验证示例：

```go
mac := hmac.New(sha256.New, []byte(webhookSecret))
mac.Write(body)
expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
if !hmac.Equal([]byte(expected), []byte(r.Header.Get("X-KubeVision-Signature"))) {
    http.Error(w, "invalid signature", http.StatusUnauthorized)
}
```

:::tip
在接收端将 Webhook 密钥存储在环境变量或密钥管理服务中，切勿将其硬编码在源代码里。
:::

## API 参考

| 方法 | 路径 | 说明 |
|--------|------|-------------|
| `GET` | `/api/v1/webhooks` | 列出所有 Webhooks |
| `POST` | `/api/v1/webhooks` | 创建 Webhook |
| `PUT` | `/api/v1/webhooks/:id` | 更新 Webhook |
| `DELETE` | `/api/v1/webhooks/:id` | 删除 Webhook |
| `POST` | `/api/v1/webhooks/:id/test` | 发送测试事件 |

## 投递保证

KubeVision 采用指数退避重试策略进行投递（最多重试 3 次，最大延迟 30 秒）。投递失败的记录连同 HTTP 状态码和响应正文会记录在界面中该 Webhook 的 **Delivery History** 标签页内。

:::warning
Webhooks 采用尽力投递机制。若接收端长时间不可用，耗尽重试次数的事件将永久丢失。如需保证投递，请改为通过审计日志导出来消费事件。
:::

## 相关文档

- [审计日志](/docs/admin-guide/audit-logging) — Webhooks 不足时的完整事件历史
- [RBAC](/docs/admin-guide/rbac) — 仅 `admin` 和 `ops` 角色可管理 Webhooks
