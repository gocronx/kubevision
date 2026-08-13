---
sidebar_position: 4
title: 接口索引
---

# 接口索引

本索引覆盖当前后端注册的非资源接口。路径均相对于 `/api/v1`；受保护接口需要
Bearer Token 和对应 RBAC 权限。

## 账户与身份

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/users/me` | 当前用户 |
| `PUT` | `/users/me/password` | 修改自己的密码 |
| `POST` | `/auth/2fa/setup` | 开始配置 TOTP |
| `POST` | `/auth/2fa/enable` | 启用 TOTP |
| `POST` | `/auth/2fa/disable` | 禁用 TOTP |
| `POST` | `/auth/public-key/register/begin` | 开始注册通行密钥 |
| `POST` | `/auth/public-key/register/finish` | 完成通行密钥注册 |
| `GET` | `/auth/public-key/credentials` | 列出自己的凭据 |
| `PUT` | `/auth/public-key/credentials/:id` | 重命名凭据 |
| `DELETE` | `/auth/public-key/credentials/:id` | 撤销凭据 |

公开登录、OAuth、刷新令牌和通行密钥登录接口见[认证](/docs/api/authentication)。

## 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET`, `POST` | `/users` | 用户列表或创建用户 |
| `GET`, `PUT`, `DELETE` | `/users/:id` | 查看、更新或删除用户 |
| `PUT` | `/users/:id/reset-password` | 重置密码 |
| `DELETE` | `/users/:id/public-key-credentials/:credentialId` | 撤销用户凭据 |
| `GET`, `PUT` | `/directory/config` | 查看或更新目录设置 |
| `POST` | `/directory/test` | 测试目录设置 |
| `POST` | `/directory/preview` | 预览组到角色映射 |
| `GET` | `/audit-logs` | 查询审计事件 |
| `GET` | `/terminal-sessions` | 终端会话列表 |
| `GET` | `/terminal-sessions/:id` | 会话元数据 |
| `GET` | `/terminal-sessions/:id/play` | 回放数据 |

## 集群与软件包

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET`, `POST` | `/clusters` | 集群列表或添加集群 |
| `GET`, `DELETE` | `/clusters/:id` | 查看或移除集群 |
| `GET` | `/clusters/:id/package-releases` | Helm Release 列表 |
| `GET` | `/clusters/:id/package-releases/:namespace/:name` | Release 详情 |
| `GET` | `/clusters/:id/package-releases/:namespace/:name/history` | 修订历史 |
| `POST` | `/clusters/:id/package-releases/:namespace/:name/rollback` | 回滚 |
| `DELETE` | `/clusters/:id/package-releases/:namespace/:name` | 移除 Release |
| `GET` | `/registry-tags` | 查询镜像标签 |

## AI、插件与自动化

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET`, `PUT` | `/ai/config` | 查看或更新 AI 配置 |
| `POST` | `/ai/models` | 查询提供商模型 |
| `POST` | `/ai/chat` | 通过 SSE 流式返回回答 |
| `POST` | `/ai/continue-action` | 继续已确认的工具操作 |
| `GET` | `/plugins` | 插件列表 |
| `GET`, `PUT` | `/plugins/:name` | 查看或更新插件配置 |
| `GET` | `/plugins/:name/health` | 插件健康检查 |
| `GET` | `/plugins/prometheus/query` | 执行受限的即时 PromQL 查询 |
| `GET` | `/plugins/grafana/dashboards` | Grafana Dashboard 列表 |
| `GET` | `/plugins/argocd/applications` | Argo CD 应用列表 |
| `GET`, `POST` | `/webhooks` | Webhook 列表或创建 |
| `PUT`, `DELETE` | `/webhooks/:id` | 更新或删除 Webhook |
| `POST` | `/webhooks/:id/test` | 发送测试事件 |

收藏夹（`/favorites`）和模板（`/templates`）也提供对应的列表、创建和删除接口。

## Kubernetes HTTP 访问

`GET` 和 `HEAD`
`/clusters/:id/namespaces/:namespace/http/:kind/:name/*path` 会向 Pod 或
Service 转发受限且经过资源授权的请求。它不是通用 URL 代理，详见
[扩展访问](/docs/admin-guide/extended-access)。
