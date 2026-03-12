---
title: 产品对比
---

# 产品对比

KubeVision 与其他 Kubernetes 控制台和命令行工具的比较。Star 数为 2026 年初的 GitHub 近似值。

## 项目概览

| 项目 | Star 数 | 类型 | 主要受众 |
|---------|-------|------|-----------------|
| **KubeVision** | — | Web 控制台 | 开发者 + 运维 + 企业 |
| [Headlamp](https://github.com/headlamp-k8s/headlamp) | ~5.9k | Web 控制台 | SIG UI 官方项目，插件生态 |
| [K9s](https://github.com/derailed/k9s) | ~33k | 终端 UI | 高级用户，CLI 优先的工作流 |
| [Kuboard](https://github.com/eip-work/kuboard-press) | ~23.5k | Web 控制台 | 企业级（中国市场） |
| [Skooner](https://github.com/skooner-k8s/skooner) | ~1.4k | Web 控制台 | CNCF 沙箱项目，轻量级 |
| [Kite](https://github.com/kubeguard/kite) | — | Web 控制台 | 轻量级单集群 |
| [KubePolaris](https://github.com/fairwindsops/polaris) | — | Web 控制台 + 审计 | 策略与配置审计 |

---

## 集群管理

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| 多集群 | 是 | 是 | 是（context） | 是 | 否 |
| 通过 kubeconfig 添加集群 | 是 | 是 | 是 | 是 | 是 |
| 通过 API 令牌添加集群 | 是 | 否 | 否 | 是 | 是 |
| 跨集群差异对比 | **是** | 否 | 否 | 否 | 否 |
| 集群健康概览 | 是 | 是 | 是 | 是 | 部分 |

## 安全与合规

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| 内置用户管理 | 是 | 否 | 否 | 是 | 否 |
| 五级 RBAC | **是** | 否 | 否 | 是（三级） | 否 |
| 双因素认证（TOTP） | **是** | 否 | 否 | 否 | 否 |
| 审计日志 | 是 | 否 | 否 | 是 | 否 |
| Secret 默认脱敏 | **是** | 否 | 否 | 否 | 否 |
| 基于 JWT 的认证 | 是 | OIDC | N/A | 是 | 是 |

## 实时性与搜索

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| 实时资源更新 | 是（WebSocket） | 是（轮询） | 是（watch） | 是 | 轮询 |
| 全局搜索（Cmd+K） | **是** | 是 | 是（`:`） | 部分 | 否 |
| Informer 缓存后端 | 是 | 否 | 是 | 否 | 否 |
| 亚秒级更新延迟 | 是 | 否 | 是 | 否 | 否 |

## 高级 DevOps 功能

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| 应用前预演差异 | **是** | 否 | 否 | 否 | 否 |
| 终端（exec 进入 Pod） | 是 | 是 | 是 | 是 | 否 |
| 终端会话录制 | **是** | 否 | 否 | 否 | 否 |
| 日志流 | 是 | 是 | 是 | 是 | 是 |
| UI 操作的 kubectl 提示 | **是** | 否 | N/A | 否 | 否 |
| 资源拓扑图 | 是 | 否 | 否 | 是 | 否 |

## 集成与扩展

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| Prometheus 指标 | 是 | 通过插件 | 是 | 是 | 否 |
| Grafana 嵌入 | 是 | 通过插件 | 否 | 是 | 否 |
| ArgoCD 集成 | 计划中 | 通过插件 | 否 | 部分 | 否 |
| 插件 / 扩展 API | 部分 | 是（完整 SDK） | 仅皮肤 | 否 | 否 |
| Helm Chart 管理 | 计划中 | 通过插件 | 是 | 是 | 否 |

## 前端体验

| 功能 | KubeVision | Headlamp | K9s | Kuboard | Skooner |
|---------|:----------:|:--------:|:---:|:-------:|:-------:|
| 深色模式 | 是 | 是 | 是 | 是 | 否 |
| 响应式 / 移动端 | 是 | 是 | 否 | 部分 | 是 |
| 键盘快捷键 | 是 | 部分 | 是（主要交互方式） | 否 | 否 |
| 资源收藏夹 | 是 | 否 | 是 | 否 | 否 |
| 带 Schema 的 YAML 编辑器 | 是 | 是 | 是 | 是 | 否 |

---

## KubeVision 独有功能

以下功能在其他任何 Kubernetes 控制台中均未提供：

| 功能 | 描述 |
|---------|-------------|
| **双因素认证（TOTP）** | 二维码设置、恢复码、按角色强制执行 |
| **预演差异（Dry-Run Diff）** | 在任何写操作前，由 API Server 验证并展示变更前后的并排差异 |
| **跨集群差异对比** | 对比同一资源（例如某个 Deployment）在预发布和生产集群之间的差异 |
| **终端录制** | 每个 exec 会话均可以 asciinema 格式录制，并可从审计日志中回放 |
| **kubectl 提示** | 在 UI 中执行的每个操作都会显示等效的 `kubectl` 命令——边点击边学习 |
| **Secret 脱敏** | 所有资源视图和日志中 Secret 值默认隐藏，查看需要明确的权限 |

---

## 与各竞品的对比总结

**对比 Headlamp** — Headlamp 拥有成熟的插件 SDK，是官方 SIG UI 项目。KubeVision 在此基础上增加了内置用户管理、双因素认证、RBAC 以及高级 DevOps 工具（预演差异、终端录制）——这些功能在 Headlamp 中要么依赖插件，要么完全缺失。

**对比 K9s** — K9s 是基于终端的集群管理的黄金标准，拥有庞大的用户群。KubeVision 面向浏览器工作流以及需要访问控制、审计日志和对新用户友好 UI 的团队——这些领域在 K9s 的设计定位之外。

**对比 Kuboard** — Kuboard 是中国市场企业级部署的有力选择。KubeVision 提供同等的企业级功能（RBAC、审计、多集群），同时增加了双因素认证、预演差异，以及面向国际受众的更现代化的 React 前端。

**对比 Skooner** — Skooner 是 CNCF 沙箱项目，专注于简洁与轻量。它缺乏用户管理、访问控制和实时推送能力。如果需要上述任何功能，KubeVision 是合适的选择。

**对比 Kite** — Kite 轻量级，适合单集群个人使用。KubeVision 从设计之初就面向多集群、多用户和生产环境。

**对比 KubePolaris** — Polaris 专注于配置审计和策略执行。KubeVision 专注于日常运维工作流，两者相辅相成，而非竞争关系。

## 相关文档

- [简介](/docs/intro) — KubeVision 概览及核心差异化特性
- [RBAC](/docs/admin-guide/rbac) — 内置五级角色体系
- [双因素认证](/docs/admin-guide/two-factor-auth) — TOTP 设置与强制执行
