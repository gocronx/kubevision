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
- 🛡️ **RBAC 五级角色** — admin / ops / dev / readonly / custom

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

实时
  GET    /api/v1/ws/watch                             WebSocket 事件推送

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
│   ├── auth/                      JWT + bcrypt
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
| **P2** | Pod 终端/日志、Deployment 操作、多集群、全局搜索、Dry-run Diff | 计划中 |
| **P3** | RBAC、2FA (TOTP)、审计、资源拓扑图、跨集群对比、Webhook 通知 | 计划中 |
| **P4** | 插件系统（Prometheus/Grafana/ArgoCD）、OAuth/OIDC、CRD 动态发现、Helm Chart | 计划中 |

<br>

## 🤝 参与贡献

欢迎贡献代码！请先提 Issue 讨论你想做的改动。

