---
sidebar_position: 1
slug: /intro
title: 简介
---

# KubeVision

**清晰洞察集群，即时付诸行动。**

KubeVision 是一款基于 Go 和 React 构建的现代化实时 Kubernetes 多集群管理仪表板，将企业级安全性与直观的开发者体验融为一体。

## 为什么选择 KubeVision？

- **实时架构** — Informer Watch → WebSocket Push，实现亚秒级更新，无需轮询
- **多集群管理** — 通过单一仪表板管理所有集群
- **企业级安全** — 双因素认证（TOTP）、5 级 RBAC、审计日志、Secrets 脱敏
- **开发者友好** — kubectl 命令提示、全局搜索（Cmd+K）、收藏夹、资源模板
- **DevOps 工具** — Dry-run 变更预览、终端录制、资源拓扑图

## 核心差异化特性

以下功能是 **KubeVision 独有的** — 其他 Kubernetes 仪表板均不具备：

| 特性 | 说明 |
|------|------|
| **双因素认证（TOTP）** | 支持 QR 码配置与恢复码的双因素身份认证 |
| **Dry-Run 变更预览** | 在应用变更前预览内容，由 API Server 校验 |
| **终端录制** | 以 asciinema 格式录制和回放终端会话 |
| **kubectl 命令提示** | 为每个界面操作自动生成对应的 CLI 命令 |
| **Secrets 脱敏** | 所有视图中默认隐藏 Secrets 内容 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23、Gin、GORM、client-go |
| 前端 | React 19、TypeScript、Vite、shadcn/ui |
| 状态管理 | TanStack Query v5 |
| 数据库 | SQLite（开发环境）/ PostgreSQL（生产环境）|
| 实时通信 | gorilla/websocket、Informer 缓存 |

## 下一步

- [安装](/docs/getting-started/installation) — 部署并运行 KubeVision
- [快速开始](/docs/getting-started/quick-start) — 安装后的第一步操作
- [架构](/docs/architecture/overview) — 了解系统设计
