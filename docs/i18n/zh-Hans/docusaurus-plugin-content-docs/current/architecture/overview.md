---
sidebar_position: 1
title: 架构概览
---

# 架构概览

KubeVision 采用清晰的分层架构，并支持实时数据流。

AI 助手是建立在仪表盘现有 Repository、授权模型和审计系统之上的编排层，不持有
独立的高权限 Kubernetes 客户端。

## 系统架构图

```text
                    Browser (React + shadcn/ui)
            ┌────────┴────────┐
            REST          WebSocket
            │                 │
┌───────────▼─────────────────▼──────────────┐
│         KubeVision Backend (Go + Gin)       │
│                                             │
│  Middleware: RequestID → Logger → Auth       │
│             → RBAC → Audit                  │
│                                             │
│  Handler ──→ Service ──→ K8sRepo            │
│  (params)    (logic)     (cache+fallback)   │
│                            │                │
│                 ┌──────────┴──────────┐     │
│                 │                     │     │
│          Informer Cache         API Server  │
│                 │                            │
│     Informer Watch ──→ EventListener        │
│                            │                │
│                        WS Hub ──→ Browser   │
│                                             │
│  Plugins: Prometheus │ Grafana │ ArgoCD     │
│  DB: SQLite (dev) / PostgreSQL (prod)       │
└─────────────────────────────────────────────┘
```

## 设计原则

| # | 原则 | 说明 |
|---|------|------|
| 1 | **简单优先** | 尽量使用标准库，减少第三方依赖 |
| 2 | **渐进式增强** | 先实现 MVP 核心功能，高级特性以插件方式接入 |
| 3 | **泛型资源** | 单一 Handler 加注册表覆盖所有 K8s 资源 |
| 4 | **实时反馈** | Informer 缓存 + WebSocket，更新延迟低于一秒 |
| 5 | **接口驱动** | 所有依赖通过接口注入 |
| 6 | **默认安全** | 敏感信息脱敏、审计默认开启、最小权限原则 |
| 7 | **受控 AI** | 有界工具、继承 RBAC、明确批准、修改可审计 |

## 各层说明

### Handler 层
HTTP 请求入口。负责解析参数、校验输入，并将请求委托给 Service 层。单一泛型 `ResourceHandler` 处理所有 Kubernetes 资源类型的增删改查。

### Service 层
业务逻辑层：输入校验、错误映射、缓存策略。`ResourceService` 提供统一的增删改查接口，读取时优先命中缓存，未命中则回退到 API Server。

### Repository 层
数据访问抽象层。`K8sRepo` 封装 Informer 缓存与动态客户端；`UserRepo`、`ClusterRepo` 等封装 GORM 以操作数据库。

### Middleware 栈
请求依次经过：`RequestID → Logger → Auth (JWT) → RBAC → Audit`

每个 Middleware 均可独立组合，并按路由粒度灵活挂载。

## 核心接口

```go
type K8sResourceRepo interface {
    List(ctx, cluster, ns, resource) (items, stale, error)
    Get(ctx, cluster, ns, resource, name) (*Unstructured, error)
    Create(ctx, cluster, ns, resource, obj) (*Unstructured, error)
    Update(ctx, cluster, ns, resource, name, obj) (*Unstructured, error)
    Delete(ctx, cluster, ns, resource, name) error
    DryRun(ctx, cluster, ns, resource, name, obj) (*Unstructured, error)
}

type ClusterManager interface {
    DynamicClient(cluster string) (dynamic.Interface, error)
    RESTConfig(cluster string) (*rest.Config, error)
}

type ResourceRegistry interface {
    Get(name string) (*ResourceInfo, error)
    All() []ResourceInfo
}
```
