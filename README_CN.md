<div align="center">

<img src="docs/assets/logo.svg" width="120" alt="KubeVision Logo">

# KubeVision

现代化的实时 Kubernetes 仪表盘

**Kube + Vision — 洞察集群全貌，即时响应变更。**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev) [![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black)](https://react.dev) [![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://typescriptlang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)

[English](./README.md) &nbsp;&nbsp;|&nbsp;&nbsp; **中文**

<br>

<!-- TODO: 替换为实际动图，展示 15-20 秒的操作流程 -->
<!-- <img src="docs/assets/demo.gif" width="800" alt="KubeVision Demo"> -->
<img src="assets/overview.png" width="800" alt="KubeVision Dashboard">
<!-- TODO: 替换为实际截图 -->

</div>

<br>

## 为什么选择 KubeVision？

<table>
<tr>
<td width="33%" align="center">

**实时架构**

Informer Watch → WebSocket 推送。

亚秒级更新，零轮询。

</td>
<td width="33%" align="center">

**高度可扩展**

每种资源只需 1 行 Go + 1 段配置。

CRD 运行时自动发现。

</td>
<td width="33%" align="center">

**生产就绪**

多集群、RBAC、2FA、审计日志。

为团队构建，而非仅供演示。

</td>
</tr>
</table>

<br>

## 截图

<table>
<tr>
<td width="50%" align="center">
<strong>Pod 终端</strong><br><br>
<img src="docs/assets/screenshot-terminal.png" width="100%" alt="Pod 终端">
</td>
<td width="50%" align="center">
<strong>资源拓扑</strong><br><br>
<img src="docs/assets/screenshot-topology.png" width="100%" alt="资源拓扑">
</td>
</tr>
<tr>
<td width="50%" align="center">
<strong>资源 YAML</strong><br><br>
<img src="docs/assets/screenshot-diff.png" width="100%" alt="资源 YAML">
</td>
<td width="50%" align="center">
<strong>全局搜索</strong><br><br>
<img src="docs/assets/screenshot-search.png" width="100%" alt="全局搜索">
</td>
</tr>
<tr>
<td width="50%" align="center">
<strong>终端审计录频</strong><br><br>
<img src="docs/assets/screenshot-audit.png" width="100%" alt="终端审计录频">
</td>
<td width="50%" align="center">
<strong>暗色模式</strong><br><br>
<img src="docs/assets/screenshot-dark.png" width="100%" alt="暗色模式">
</td>
</tr>
</table>

<br>

## 核心特性

- :zap: **实时同步** — Informer → WebSocket，亚秒级更新
- :globe_with_meridians: **多集群** — 一个面板管理所有集群
- :computer: **Pod 终端与日志** — 会话录制与回放
- :rocket: **工作负载操作** — 扩缩容、重启、回滚
- :eyes: **Dry-Run Diff** — 应用前预览变更
- :twisted_rightwards_arrows: **跨集群 Diff** — 快速发现配置漂移
- :world_map: **资源拓扑图** — 可视化资源关系
- :mag: **全局搜索** — `Cmd+K` 模糊搜索
- :keyboard: **kubectl 提示** — 自动生成等效命令
- :jigsaw: **资源模版** — 一键部署
- :shield: **RBAC** — 5 级内置角色 + 自定义
- :key: **2FA (TOTP)** — QR 码 + 恢复码
- :memo: **审计日志** — 异步写入 + 保留策略
- :bell: **Webhook** — Slack、Discord、自定义端点
- :package: **26+ 资源类型** — 8 缓存 + 18 按需 + CRD
- :zap: **批量操作** — 多选删除与重启
- :earth_africa: **国际化** — 中英双语
- :crescent_moon: **暗色模式** — 跟随系统主题

<br>

## 快速开始

### Helm（推荐）

```bash
helm install kubevision deploy/helm/kubevision
```

### Docker

```bash
docker build -f deploy/Dockerfile -t kubevision:latest .
docker run -p 8080:8080 kubevision:latest
```

> 打开 http://localhost:8080
>
> 默认账号：`admin` / `admin123`

### 本地开发

```bash
git clone https://github.com/gocronx/kubevision.git
cd kubevision

# 后端 — :8080
go mod tidy && make dev

# 前端 — :5173，/api 自动代理到 :8080
cd web && pnpm install && pnpm dev
```

<br>

## 架构

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
> **写链路**: API Server → Informer → WS Hub → 浏览器

<br>

## 核心依赖

1. [Gin](https://github.com/gin-gonic/gin) — HTTP 框架
2. [client-go](https://github.com/kubernetes/client-go) — Kubernetes API 客户端 & Informer
3. [GORM](https://gorm.io) — ORM（SQLite + PostgreSQL）
4. [shadcn/ui](https://ui.shadcn.com) — 基于 [Radix UI](https://radix-ui.com) 的组件库
5. [TanStack Query](https://tanstack.com/query) — 服务端状态管理 & 缓存
6. [xterm.js](https://xtermjs.org) — Pod 终端模拟器

<br>

## 配置

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

| 变量 | 说明 |
|------|------|
| `KUBEVISION_SERVER_PORT` | HTTP 端口（默认 8080） |
| `KUBEVISION_DB_DRIVER` | `sqlite` 或 `postgres` |
| `KUBEVISION_DB_DSN` | 数据库连接字符串 |
| `KUBEVISION_JWT_SECRET` | JWT 签名密钥 |
| `KUBECONFIG` | kubeconfig 文件路径 |

<br>

## 文档

**[kubevision-docs](https://kubevision-docs.pages.dev/)** — 完整文档站，包含使用指南、API 参考和架构详解。

| | |
|---|---|
| [快速开始](https://kubevision.github.io/docs/getting-started/installation) | 安装、快速上手、配置说明 |
| [架构设计](https://kubevision.github.io/docs/architecture/overview) | 系统设计、数据流、组件交互 |
| [使用指南](https://kubevision.github.io/docs/user-guide/cluster-management) | 功能详解与操作指引 |
| [API 参考](https://kubevision.github.io/docs/api/overview) | REST & WebSocket API 文档 |
| [方案对比](https://kubevision.github.io/docs/comparison) | KubeVision 与其他方案的对比 |

<br>

## 参与贡献

欢迎贡献代码！请先提 Issue 讨论你想做的改动。

<!-- ## 贡献者 -->
<!-- <a href="https://github.com/gocronx/kubevision/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=gocronx/kubevision" />
</a> -->
