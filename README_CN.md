<div align="center">

# KubeVision

**现代化、实时 Kubernetes 管理面板**

*Kube + Vision — 洞察集群全貌，掌控每一处变更。*

Go + React + shadcn/ui &nbsp;|&nbsp; Informer 缓存 + WebSocket 推送 &nbsp;|&nbsp; 26 种资源类型

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev) [![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev) [![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://typescriptlang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)

<br>

[English](./README.md) &nbsp;&nbsp;|&nbsp;&nbsp; **中文**

<br>

<!-- TODO: 替换为实际截图 -->
<!-- <img src="docs/screenshot.png" width="800" alt="KubeVision Dashboard"> -->

</div>

<br>

## 🚀 为什么选择 KubeVision？

大多数 K8s Dashboard 要么**每种资源类型需要数百行重复代码**，要么完全放弃了实时更新。KubeVision 采用了不同的思路：

### 🧩 零模板代码

新增一种 K8s 资源类型，只需后端 **1 行** Go 代码 + 前端 **1 段** 配置。不需要新建文件，不需要重复 Handler。

```go
// 就这一行
r.resources["cronjobs"] = Meta{
  Name: "cronjobs",
  GVR:  schema.GroupVersionResource{...},
  ...
}
```

### ⚡ 真正的实时

Informer Watch 检测变更 → WebSocket Hub 推送到浏览器 → TanStack Query 缓存自动失效。**亚秒级更新，零轮询。**

```
K8s API ──→ Informer ──→ WS Hub ──→ 浏览器
  写入         监听        推送     自动刷新
```

<br>

## ✨ 核心特性

- 🔄 **实时更新** — Informer → WebSocket → 自动刷新
- 📦 **26 种资源类型** — 8 种缓存 + 18 种按需，统一 Handler
- 🌐 **多集群管理** — 单一面板管理所有集群
- 🔐 **JWT + Token 吊销** — Access/Refresh 令牌，版本号即时吊销
- 🌙 **暗色模式** — 跟随系统，一键切换
- 🌍 **国际化** — 内置中文和英文
- ⚙️ **泛型 CRUD** — List、Get、Create、Update、Delete、Patch
- 🏷️ **命名空间过滤** — 下拉快速切换命名空间
- 🖥️ **Pod 终端和日志** — xterm.js 终端，实时日志流带搜索
- 🚀 **Deployment 操作** — 扩缩容、重启、回滚，支持版本历史
- 🔍 **全局搜索** — Cmd+K 跨资源搜索，带相关性评分
- 📋 **Dry-Run 预览** — 应用前预览变更，并排 Diff 视图
- 💡 **kubectl 提示** — 每个 UI 操作自动生成对应的 kubectl 命令
- ⭐ **资源收藏** — 固定常用资源，支持拖拽排序
- 📊 **资源配额** — CPU/内存/Pod 使用量可视化进度条
- 🛡️ **RBAC 五级角色** — super-admin / admin / editor / viewer / custom，中间件级别强制执行
- 🔑 **2FA (TOTP)** — 基于时间的一次性密码，QR 码设置 + 恢复码
- 📝 **审计日志** — 异步批量写入所有变更操作，支持保留策略
- 🗝️ **API Key 认证** — 生成和撤销 API Key，用于编程访问
- 🗺️ **资源拓扑图** — 可视化所有权和选择器关系图
- ⚡ **批量操作** — 多选删除和重启，带确认对话框
- 🔀 **跨集群 Diff** — 并排对比不同集群中的同名资源
- 🔔 **Webhook** — 事件驱动通知到外部端点
- 🎬 **终端录屏** — Pod 终端会话录制与回放
- 🐘 **PostgreSQL** — 生产级数据库驱动，兼容 SQLite

<br>

## 🏗️ 架构

```
                    浏览器
            ┌────────┴────────┐
            REST          WebSocket
            │                 │
┌───────────▼─────────────────▼──────────────┐
│  中间件: RequestID → Logger → Auth          │
│                                             │
│  Handler ──→ Service ──→ K8sRepo            │
│                            │                │
│                 ┌──────────┴──────────┐     │
│                 │                     │     │
│          Informer 缓存          API Server  │
│                 │                            │
│     Informer Watch ──→ EventListener        │
│                            │                │
│                        WS Hub ──→ 浏览器     │
│                                             │
│  数据库: SQLite (开发) / PostgreSQL (生产)    │
└─────────────────────────────────────────────┘
```

> **读链路**: Informer 缓存优先，未命中降级 API Server
>
> **写链路**: API Server → Informer Watch → WS Hub → 浏览器自动刷新

<br>

## ⚡ 快速开始

### 环境要求

| 依赖 | 版本 |
|------|------|
| Go | 1.23+ |
| Node.js | 20+ |
| pnpm | 9+ |
| Kubernetes | 1.28+（kubeconfig） |

### 本地开发

```bash
git clone https://github.com/kubevision/kubevision.git
cd kubevision

# 后端 — 运行在 :8080
go mod tidy && make dev

# 前端 — 运行在 :5173，/api 自动代理到 :8080
cd web && pnpm install && pnpm dev
```

> 默认账号：**admin** / **admin123**

### Docker 部署

```bash
docker build -f deploy/Dockerfile -t kubevision:latest .
docker run -p 8080:8080 kubevision:latest
```

<br>

## 🛠️ 技术栈

| | 技术 | 用途 |
|---|-----|------|
| **后端** | Go 1.23 &middot; Gin &middot; GORM &middot; client-go &middot; zap | API 服务、K8s 集成 |
| **前端** | React 19 &middot; TypeScript &middot; Vite 7 &middot; shadcn/ui &middot; TanStack Query v5 | 现代 UI + 实时数据 |
| **数据库** | SQLite / PostgreSQL | 用户、集群、审计持久化 |
| **认证** | JWT（15 分钟 access + 7 天 refresh）&middot; bcrypt | 通过版本号实现 Token 吊销 |
| **实时** | Informer &middot; gorilla/websocket | 亚秒级推送到浏览器 |

<br>

## 📋 支持的资源类型

<table>
<tr>
<td><b>缓存资源</b>（Informer，实时推送）</td>
<td>

`Pods` `Deployments` `StatefulSets` `DaemonSets` `ReplicaSets` `Services` `Nodes` `Namespaces`

</td>
</tr>
<tr>
<td><b>按需资源</b>（API Server 查询）</td>
<td>

`Jobs` `CronJobs` `Ingresses` `ConfigMaps` `Secrets` `Events` `PVs` `PVCs` `StorageClasses` `ServiceAccounts` `Roles` `ClusterRoles` `RoleBindings` `ClusterRoleBindings` `HPAs` `Endpoints` `NetworkPolicies`

</td>
</tr>
</table>

<br>

## 📡 API 概览

```
认证
  POST   /api/v1/auth/login                          登录
  POST   /api/v1/auth/refresh                        刷新 Token
  GET    /api/v1/users/me                             当前用户
  POST   /api/v1/auth/2fa/verify                     验证 TOTP 码
  POST   /api/v1/auth/2fa/recovery                   使用恢复码
  POST   /api/v1/auth/2fa/setup                      生成 TOTP 密钥（需登录）
  POST   /api/v1/auth/2fa/enable                     启用 2FA（需登录）
  POST   /api/v1/auth/2fa/disable                    禁用 2FA（需登录）

集群
  GET    /api/v1/clusters                             集群列表
  POST   /api/v1/clusters                             添加集群
  DELETE /api/v1/clusters/:id                         删除集群

资源（泛型 CRUD）
  GET    /api/v1/clusters/:id/resources/:res          列表
  GET    /api/v1/clusters/:id/resources/:res/:name    详情
  POST   /api/v1/clusters/:id/resources/:res          创建
  PUT    /api/v1/clusters/:id/resources/:res/:name    更新
  DELETE /api/v1/clusters/:id/resources/:res/:name    删除
  PATCH  /api/v1/clusters/:id/resources/:res/:name    Patch
  POST   /api/v1/clusters/:id/resources/:res/dry-run  Dry-run 创建预览
  PUT    /api/v1/clusters/:id/resources/:res/:name/dry-run  Dry-run 更新预览
  POST   /api/v1/clusters/:id/resources/batch-delete  批量删除
  POST   /api/v1/clusters/:id/batch-restart           批量重启

工作负载操作
  PUT    /api/v1/clusters/:id/namespaces/:ns/:kind/:name/scale     扩缩容
  POST   /api/v1/clusters/:id/namespaces/:ns/:kind/:name/restart   重启
  GET    /api/v1/clusters/:id/namespaces/:ns/deployments/:name/history  发布历史
  POST   /api/v1/clusters/:id/namespaces/:ns/deployments/:name/rollback 回滚

拓扑
  GET    /api/v1/clusters/:id/namespaces/:ns/topology  资源关系图

搜索
  GET    /api/v1/clusters/:id/search                 全局搜索

收藏
  GET    /api/v1/favorites                            收藏列表
  POST   /api/v1/favorites                            添加收藏
  DELETE /api/v1/favorites/:id                        删除收藏

审计与安全
  GET    /api/v1/audit-logs                           审计日志列表
  GET    /api/v1/api-keys                             API Key 列表
  POST   /api/v1/api-keys                             生成 API Key
  DELETE /api/v1/api-keys/:id                         撤销 API Key

Webhook
  GET    /api/v1/webhooks                             Webhook 列表
  POST   /api/v1/webhooks                             创建 Webhook
  PUT    /api/v1/webhooks/:id                         更新 Webhook
  DELETE /api/v1/webhooks/:id                         删除 Webhook
  POST   /api/v1/webhooks/:id/test                    测试 Webhook

终端录屏
  GET    /api/v1/terminal-sessions                    录屏列表
  GET    /api/v1/terminal-sessions/:id                录屏详情
  GET    /api/v1/terminal-sessions/:id/play           录屏回放

跨集群
  POST   /api/v1/compare                              资源对比

实时
  GET    /api/v1/ws/watch                             WebSocket 事件推送
  GET    /api/v1/clusters/:id/.../pods/:name/exec     Pod 终端（WS）
  GET    /api/v1/clusters/:id/.../pods/:name/logs     Pod 日志流（WS）

健康检查
  GET    /healthz                                     健康检查
```

<br>

## 📁 项目结构

```
kubevision/
├── cmd/kubevision/main.go         入口，依赖注入
├── internal/
│   ├── config/                    YAML + 环境变量配置
│   ├── server/                    HTTP Server + 路由注册
│   ├── middleware/                 Auth、RequestID、Logger、Audit
│   ├── handler/                   HTTP 处理器（auth、cluster、resource、ws）
│   ├── service/                   业务逻辑层
│   ├── repository/                数据访问层（GORM + K8s Dynamic Client）
│   ├── model/                     数据库模型（11 张表）
│   ├── kubernetes/
│   │   ├── cluster/               多集群连接管理
│   │   ├── informer/              Informer 生命周期 + EventListener
│   │   └── resource/              资源注册表（26 种）
│   ├── auth/                      JWT + bcrypt + TOTP
│   └── pkg/                       统一响应、业务错误码
├── web/src/
│   ├── components/ui/             shadcn/ui（17 个组件）
│   ├── components/shared/         DataTable、StatusBadge、NamespaceSelector
│   ├── hooks/                     useResource、useCluster、useWebSocket
│   ├── pages/                     Login、Overview、ResourceList、ResourceDetail
│   ├── lib/                       API 客户端、WebSocket 客户端、K8s 工具函数
│   └── config/                    资源 UI 配置（图标、列定义）
├── deploy/Dockerfile              三阶段构建：Node → Go → Alpine
└── Makefile                       dev、build、test、lint、tidy
```

<br>

## ⚙️ 配置

```yaml
server:
  port: 8080

database:
  driver: sqlite           # sqlite | postgres
  dsn: kubevision.db

auth:
  jwt_secret: ""           # 留空自动生成
  access_token_ttl: 15m
  refresh_token_ttl: 168h

kubernetes:
  kubeconfig: ""           # 留空使用 in-cluster
  informer_resync: 30m
```

所有配置项均可通过环境变量覆盖：

| 变量 | 说明 |
|------|------|
| `KUBEVISION_SERVER_PORT` | HTTP 端口（默认 8080） |
| `KUBEVISION_DB_DRIVER` | `sqlite` 或 `postgres` |
| `KUBEVISION_DB_DSN` | 数据库连接字符串 |
| `KUBEVISION_JWT_SECRET` | JWT 签名密钥 |
| `KUBECONFIG` | kubeconfig 文件路径 |

<br>

## 🗺️ 开发路线

| 阶段 | 范围 | 状态 |
|------|------|------|
| **P1** | 单集群、泛型 CRUD、Informer + WebSocket、JWT 认证、基础 UI | 已完成 |
| **P2** | Pod 终端/日志、Deployment 操作、全局搜索、Dry-run Diff、kubectl 提示、收藏、资源配额 | 已完成 |
| **P3** | RBAC、2FA (TOTP)、审计日志、资源拓扑图、批量操作、跨集群 Diff、Webhook、终端录屏、PostgreSQL | 已完成 |
| **P4** | 插件系统（Prometheus/Grafana/ArgoCD）、OAuth/OIDC、CRD 动态发现、Helm Chart | 计划中 |

<br>

## 🧪 测试

```
12 个测试包 · 466 个测试用例 · 57 个测试文件
```

| 层级 | 包 | 覆盖范围 |
|------|---|---------|
| **中间件** | auth、rbac、audit、logger | JWT 验证、RBAC 规则、审计捕获、请求日志 |
| **Handler** | auth、cluster、resource、apikey、audit、compare、topology、webhook、terminal_session、ws/hub | 全部 HTTP 端点 + WebSocket Hub |
| **Service** | auth、apikey、audit、cluster、compare、terminal_session、topology、webhook | 业务逻辑 + Mock 仓库 |
| **Repository** | user、role、cluster、apikey、audit、webhook、terminal_session | GORM CRUD + 内存 SQLite |
| **Server** | router、server | 路由注册、健康检查 |
| **其他** | config、auth/jwt、auth/totp、resource registry、errors、response | 解析、签名、加密、工具函数 |

```bash
go test ./... -count=1      # 运行全部测试
go test ./... -cover         # 带覆盖率
```

<br>

## 🤝 参与贡献

欢迎贡献代码！请先提 Issue 讨论你想做的改动。
