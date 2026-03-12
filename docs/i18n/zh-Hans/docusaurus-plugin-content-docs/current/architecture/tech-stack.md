---
sidebar_position: 4
title: 技术栈
---

# 技术栈

## 后端

| 技术 | 用途 |
|------|------|
| **Go 1.23** | 主要编程语言 |
| **Gin** | HTTP 框架 |
| **GORM** | ORM（支持 SQLite 与 PostgreSQL） |
| **client-go** | Kubernetes API 客户端与 Informer |
| **gorilla/websocket** | WebSocket 连接 |
| **zap** | 结构化日志 |
| **pquerna/otp** | TOTP 双因素认证实现 |

## 前端

| 技术 | 用途 |
|------|------|
| **React 19** | UI 框架 |
| **TypeScript** | 类型安全 |
| **Vite** | 构建工具与开发服务器 |
| **shadcn/ui** | 组件库（Radix + Tailwind） |
| **TanStack Query v5** | 服务端状态管理与缓存 |
| **Monaco Editor** | YAML 编辑与语法高亮 |
| **xterm.js** | Web 终端模拟器 |
| **Recharts** | 概览图表 |
| **i18next** | 国际化（中文/英文） |

## 数据库

| 模式 | 引擎 | 适用场景 |
|------|------|----------|
| 开发环境 | SQLite | 零配置，基于文件 |
| 生产环境 | PostgreSQL | 高扩展性，支持并发访问 |

两种引擎均通过 GORM 抽象，修改一处配置即可切换。

## 选型理由

- **Go + Gin**：编译速度快、内存占用低，拥有出色的 Kubernetes 生态（client-go）
- **React + shadcn/ui**：现代组件库，支持通过 Tailwind 进行完全自定义
- **TanStack Query**：消除手动缓存管理，自动处理数据重新获取与失效
- **Informer + WebSocket**：实时 K8s 仪表盘的业界标准模式
- **GORM**：数据库无关的 ORM，在 SQLite 与 PostgreSQL 之间迁移简便
