---
sidebar_position: 1
slug: /intro
title: 简介
---

# KubeVision

**AI 原生 Kubernetes 运维，同时保留人工控制。**

KubeVision 是一款基于 Go 和 React 构建的 AI 原生 Kubernetes 仪表盘。AI 助手
使用实时集群上下文和 Kubernetes 工具，让运维人员无需离开控制台即可通过自然
语言调查故障并准备变更。

系统坚持人机协同：AI 工具继承当前用户的 RBAC 权限，修改操作需要明确批准，
已批准的变更会写入审计记录。实时仪表盘始终可用于直接检查和手动操作。

![包含 AI 操作确认流程的 KubeVision 概览](/img/screenshots/overview.png)

_集群概览与 AI 助手可同时使用，修改操作执行前会等待用户明确确认。_

## 为什么选择 KubeVision？

- **AI 运维工作区** — 使用实时上下文检查资源和日志、解释故障、查询指标并准备变更
- **受控执行** — 工具级 RBAC、修改确认、执行前重新授权和审计记录
- **模型选择** — 可连接支持的 OpenAI 兼容端点，不把集群操作绑定到单一厂商
- **实时架构** — Informer Watch → WebSocket Push，实现亚秒级更新，无需轮询
- **多集群管理** — 通过单一仪表板管理所有集群
- **企业级安全** — 双因素认证（TOTP）、5 级 RBAC、审计日志、Secrets 脱敏
- **开发者友好** — 响应式桌面端与移动端操作流程、kubectl 命令提示、全局搜索（Cmd+K）、收藏夹和资源模板
- **DevOps 工具** — Dry-run 变更预览、终端录制、资源拓扑图

## AI 运作模型

| 阶段 | 行为 |
|------|------|
| **上下文** | AI 助手获取当前集群、命名空间、页面和资源信息 |
| **调查** | 可读取 Kubernetes 资源、Pod 日志、集群健康和已配置的 Prometheus 数据 |
| **授权** | 每次工具调用都检查当前用户角色 |
| **批准** | 创建、更新、Patch 和删除操作暂停并等待明确确认 |
| **执行** | 已批准修改在执行前再次检查权限 |
| **审计** | AI 修改记录操作人、目标、工具、关联 ID 和结果 |

KubeVision 不在后台运行自治修复，也不会绕过 Kubernetes 或应用 RBAC。它是运维
人员的 Copilot，而不是无人值守的集群控制器。

## 运维基础能力

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
| 后端 | Go 1.26.6、Gin、GORM、client-go |
| 前端 | React 19、TypeScript、Vite、shadcn/ui |
| 状态管理 | TanStack Query v5 |
| 数据库 | SQLite（开发环境）/ PostgreSQL（生产环境）|
| 实时通信 | gorilla/websocket、Informer 缓存 |

## 下一步

- [安装](/docs/getting-started/installation) — 部署并运行 KubeVision
- [快速开始](/docs/getting-started/quick-start) — 安装后的第一步操作
- [架构](/docs/architecture/overview) — 了解系统设计
