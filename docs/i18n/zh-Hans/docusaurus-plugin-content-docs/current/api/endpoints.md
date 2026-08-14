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
| `POST` | `/clusters/:id/package-releases/preview/:operation` | Dry Run `install` 或 `upgrade` 并签发确认令牌 |
| `POST` | `/clusters/:id/package-releases/install` | 安装预检中确认的同一请求 |
| `POST` | `/clusters/:id/package-releases/upgrade` | 升级预检中确认的同一请求 |
| `POST` | `/clusters/:id/package-releases/:namespace/:name/check-upgrade` | 从已记录或本次提供的索引 Helm 来源检查新的稳定 Chart 版本 |
| `POST` | `/clusters/:id/package-releases/:namespace/:name/rollback` | 回滚 |
| `DELETE` | `/clusters/:id/package-releases/:namespace/:name` | 移除 Release |
| `GET`, `POST` | `/clusters/:id/helm/repositories` | 列出或创建托管 Helm/OCI 仓库 |
| `PUT`, `DELETE` | `/clusters/:id/helm/repositories/:repositoryId` | 更新或删除托管仓库 |
| `POST` | `/clusters/:id/helm/repositories/:repositoryId/test` | 测试托管仓库连接 |
| `GET` | `/clusters/:id/helm/repositories/:repositoryId/charts` | 浏览带索引的 Helm 仓库 |
| `GET` | `/clusters/:id/helm/artifact-hub/search` | 在 Artifact Hub 搜索公开 Chart |
| `POST` | `/clusters/:id/helm/charts/inspect` | 检查 Chart 元数据和内容 |
| `POST` | `/clusters/:id/helm/charts/upload` | 上传临时打包 Chart |
| `GET`, `POST` | `/clusters/:id/helm/upgrade-policies` | 列出或创建自动升级策略 |
| `PUT`, `DELETE` | `/clusters/:id/helm/upgrade-policies/:policyId` | 更新或删除升级策略 |
| `POST` | `/clusters/:id/helm/upgrade-policies/:policyId/check` | 立即执行升级策略 |
| `GET` | `/registry-tags` | 查询镜像标签 |

安装和升级请求与预检使用相同请求体，并且必须包含预检返回的一次性
`confirmationToken`。令牌会绑定用户、集群、命名空间、Release、Chart 来源、
Values 和操作类型。
Release 已记录来源时，检查更新请求体可以为空；旧 Release 可以提交
`{"source":{"chart":"name","repoUrl":"https://..."}}`，验证并关联带索引的
Helm 仓库。Release 来源记录只保存 Chart 坐标，不保存仓库凭据。
仓库管理、私有凭据和升级策略需要管理员权限。Chart 上传使用
`multipart/form-data`，30 分钟后过期，并绑定上传用户。

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
