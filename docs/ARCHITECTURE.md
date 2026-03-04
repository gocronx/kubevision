# KubeVision - Kubernetes 多集群管理平台架构设计 v2

> 设计原则：**简单易用，功能完整**
>
> 融合 Kite 和 KubePolaris 两个项目的最佳实践，去除过度设计，补齐关键遗漏

---

## 1. 设计原则

| # | 原则 | 说明 |
|---|------|------|
| 1 | **简单优先** | 能用一行配置解决的不写代码，能用标准库的不引第三方 |
| 2 | **渐进增强** | MVP 先跑通核心链路，监控/GitOps/高级认证按需插拔 |
| 3 | **统一泛型** | 一个 Handler + 注册表覆盖所有 K8s 资源，新增资源零文件 |
| 4 | **实时反馈** | Informer 缓存 + WebSocket 推送，亚秒级数据更新 |
| 5 | **接口驱动** | 所有依赖通过接口注入，方便 mock 测试 |
| 6 | **安全默认** | Secrets 默认脱敏、审计日志默认开启、最小权限原则 |

---

## 2. 技术栈

| 层级 | 选型 | 说明 |
|------|------|------|
| **后端** | Go 1.23+ / Gin / GORM | 成熟稳定，生态完善 |
| **前端** | React 19 / TypeScript / Vite 7 | 现代化开发体验 |
| **UI** | shadcn/ui (Radix + Tailwind) | 轻量可定制，现代审美 |
| **状态** | TanStack Query v5 | 服务端状态管理 + 缓存失效 |
| **K8s** | client-go 0.31+ / dynamic client | 官方客户端，Unstructured 动态类型 |
| **WebSocket** | gorilla/websocket | 成熟方案（后续可迁移 nhooyr.io/websocket） |
| **数据库** | SQLite（默认）/ PostgreSQL（生产） | GORM 抽象，按需切换 |
| **日志** | zap | 结构化高性能日志 |
| **编辑器** | Monaco Editor | YAML 编辑 + 语法高亮 |
| **终端** | xterm.js | Web 终端仿真 |
| **图表** | Recharts（概览）/ uPlot（监控） | 简单场景用 Recharts，时序数据用 uPlot |
| **国际化** | i18next | 中英文支持 |

---

## 3. 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                       浏览器 (React + shadcn/ui)                │
│   REST API ──────────────────── WebSocket ─────────────────────│
└───────────┬─────────────────────────┬──────────────────────────┘
            │                         │
            ▼                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    KubeVision 后端 (Go + Gin)                   │
│                                                                 │
│  ┌───────────────── 中间件链 ──────────────────────────────┐    │
│  │ CORS → RequestID → Logger → Metrics → Auth → RBAC → Audit│    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─── Handler 层 ───┐  ┌─── Service 层 ───┐  ┌─ Repository ─┐ │
│  │ GenericResource   │→│ ResourceService   │→│ K8sRepo      │ │
│  │ AuthHandler       │→│ AuthService       │→│ (Informer/   │ │
│  │ ClusterHandler    │→│ ClusterService    │→│  DynClient)  │ │
│  │ SearchHandler     │→│ SearchService     │→│              │ │
│  │ WS: Terminal/Logs │ │ AuditService      │→│ DBRepo       │ │
│  │ WS: Watch         │ │ RBACService       │→│ (GORM)       │ │
│  └───────────────────┘  └──────────────────┘  └──────────────┘ │
│                                                                 │
│  ┌─── Informer 缓存 ────────────┐  ┌─── 插件系统 ──────────┐  │
│  │ Per-Cluster Informer Factory  │  │ Prometheus (可选)      │  │
│  │ → EventListener 接口          │  │ Grafana    (可选)      │  │
│  │ → WebSocket Hub 订阅          │  │ ArgoCD     (可选)      │  │
│  └───────────────────────────────┘  └────────────────────────┘  │
│                                                                 │
│  ┌─── 数据库 ───────────────────────────────────────────────┐  │
│  │                SQLite (默认) │ PostgreSQL (生产)          │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
          │                              │
          ▼                              ▼
   K8s Cluster 1                  K8s Cluster N
```

### 核心数据流

```
[用户操作] → Handler → Service → K8sRepo → K8s API Server
                                     ↑
                              Informer Cache (读优先)

[实时更新] K8s Watch → Informer → EventListener → WebSocket Hub → 浏览器
                                                                    ↓
                                                    TanStack Query 缓存失效 → UI 刷新
```

---

## 4. 后端目录结构

```
kubevision/
├── cmd/kubevision/
│   └── main.go                         # 入口：配置加载 → DI 组装 → 启动
│
├── internal/
│   ├── config/
│   │   └── config.go                   # 全部配置项（环境变量 + YAML）
│   │
│   ├── server/
│   │   ├── server.go                   # HTTP Server + 优雅关闭
│   │   ├── router.go                   # 路由注册（核心）
│   │   └── metrics.go                  # /metrics 自身指标暴露
│   │
│   ├── middleware/
│   │   ├── auth.go                     # JWT 验证 → 注入用户上下文
│   │   ├── rbac.go                     # 权限检查（从 JWT Claims 读取，不查库）
│   │   ├── audit.go                    # 写操作审计（异步批量写入）
│   │   ├── cluster.go                  # 集群上下文注入
│   │   ├── requestid.go               # 请求追踪 ID
│   │   └── logger.go                   # 请求日志
│   │
│   ├── handler/                        # HTTP 处理器（薄层，只做参数解析和响应）
│   │   ├── resource_handler.go         # ★ 泛型资源 CRUD
│   │   ├── resource_action_handler.go  # ★ 特殊操作（scale/cordon/drain/restart）
│   │   ├── auth_handler.go             # 登录/登出/刷新
│   │   ├── cluster_handler.go          # 集群增删改查
│   │   ├── user_handler.go             # 用户管理
│   │   ├── search_handler.go           # 全局搜索
│   │   ├── overview_handler.go         # 集群概览
│   │   ├── template_handler.go         # 资源模板
│   │   ├── webhook_handler.go          # ★ Webhook 通知配置
│   │   ├── favorite_handler.go         # ★ 收藏夹
│   │   ├── terminal_session_handler.go # ★ 终端会话录制/回放
│   │   └── ws/
│   │       ├── hub.go                  # WebSocket Hub（单 goroutine + channel）
│   │       ├── terminal.go             # Pod/Node 终端
│   │       ├── logs.go                 # 日志流
│   │       └── watch.go                # 资源变更推送
│   │
│   ├── service/                        # 业务逻辑层（核心）
│   │   ├── resource_service.go         # ★ K8s 资源 CRUD（缓存降级 + 错误映射）
│   │   ├── auth_service.go             # 认证逻辑
│   │   ├── cluster_service.go          # 集群管理
│   │   ├── user_service.go             # 用户管理
│   │   ├── rbac_service.go             # 角色权限
│   │   ├── audit_service.go            # 审计日志（带异步批量写入）
│   │   ├── search_service.go           # 全局搜索
│   │   ├── template_service.go         # 资源模板
│   │   ├── webhook_service.go          # ★ Webhook 通知（事件匹配 + HTTP 推送）
│   │   ├── favorite_service.go         # ★ 收藏夹管理
│   │   └── terminal_session_service.go # ★ 终端录制存储 + 回放
│   │
│   ├── repository/                     # 数据访问层
│   │   ├── interfaces.go              # ★ 所有 Repository 接口定义
│   │   ├── k8s_repo.go                # K8s 资源访问（Informer + Dynamic Client）
│   │   ├── db.go                       # 数据库初始化 + 迁移
│   │   ├── user_repo.go
│   │   ├── cluster_repo.go
│   │   ├── role_repo.go
│   │   ├── audit_repo.go
│   │   ├── template_repo.go
│   │   ├── webhook_repo.go
│   │   ├── favorite_repo.go
│   │   └── terminal_session_repo.go
│   │
│   ├── model/                          # 数据模型（GORM + K8s 类型）
│   │   ├── user.go
│   │   ├── cluster.go
│   │   ├── role.go
│   │   ├── audit.go
│   │   ├── template.go
│   │   ├── apikey.go
│   │   ├── setting.go
│   │   ├── webhook.go               # ★ Webhook 配置模型
│   │   ├── favorite.go              # ★ 收藏夹模型
│   │   └── terminal_session.go      # ★ 终端会话录制模型
│   │
│   ├── kubernetes/                     # K8s 集成
│   │   ├── cluster/
│   │   │   ├── manager.go             # 多集群管理（RWMutex 保护）
│   │   │   └── client.go              # 客户端创建（kubeconfig/token/in-cluster）
│   │   │
│   │   ├── informer/
│   │   │   ├── manager.go             # Informer 管理（含 stopCh 生命周期）
│   │   │   ├── listener.go            # ★ EventListener 接口（解耦 Hub）
│   │   │   └── cache.go               # 缓存查询 + 新鲜度检测
│   │   │
│   │   ├── resource/
│   │   │   ├── registry.go            # 资源注册表
│   │   │   ├── discovery.go           # CRD 动态发现
│   │   │   └── related.go             # 关联资源查询
│   │   │
│   │   └── exec/
│   │       ├── terminal.go            # Pod exec
│   │       ├── node_debug.go          # Node 调试（kubectl debug 方式）
│   │       └── logs.go                # 日志流
│   │
│   ├── auth/
│   │   ├── jwt.go                     # JWT Token 管理（Access + Refresh）
│   │   ├── password.go                # 密码认证（bcrypt）
│   │   ├── apikey.go                  # API Key 认证
│   │   └── totp.go                    # ★ 2FA: TOTP 生成/验证/恢复码
│   │
│   ├── plugin/                         # 可插拔集成
│   │   ├── plugin.go                  # Plugin 接口
│   │   ├── manager.go                 # 插件管理器
│   │   ├── prometheus/
│   │   ├── grafana/
│   │   └── argocd/
│   │
│   └── pkg/                            # 内部共享
│       ├── response/response.go       # 统一响应格式
│       ├── errors/errors.go           # 错误定义 + K8s 错误码映射
│       └── crypto/crypto.go           # 加密工具
│
├── web/                                # 前端（见第 7 节）
├── deploy/                             # 部署配置
├── Makefile
└── docker-compose.yaml
```

---

## 5. 核心设计（修复审视报告中的所有 P0/P1 问题）

### 5.1 接口定义（解决依赖注入和可测试性）

```go
// internal/repository/interfaces.go
// ★ 所有依赖通过接口注入，方便 mock 测试

package repository

// K8sResourceRepo K8s 资源数据访问（Informer + Dynamic Client）
type K8sResourceRepo interface {
    List(ctx context.Context, clusterID, namespace, resource string) (
        items []unstructured.Unstructured, stale bool, err error)
    Get(ctx context.Context, clusterID, namespace, resource, name string) (
        *unstructured.Unstructured, error)
    Create(ctx context.Context, clusterID, namespace, resource string,
        obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
    Update(ctx context.Context, clusterID, namespace, resource, name string,
        obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
    Delete(ctx context.Context, clusterID, namespace, resource, name string) error
    // Dry-run（变更预览）
    DryRun(ctx context.Context, clusterID, namespace, resource string,
        obj *unstructured.Unstructured) error
}

// ResourceRegistry 资源元数据注册表
type ResourceRegistry interface {
    Get(name string) (ResourceMeta, bool)
    All() map[string]ResourceMeta
}

// ClusterManager 多集群管理
type ClusterManager interface {
    DynamicClient(clusterID string) (dynamic.Interface, error)
    RESTConfig(clusterID string) (*rest.Config, error)
    ListClusters() []ClusterInfo
    HealthCheck(clusterID string) error
}
```

### 5.2 路由设计（★ 修复路由冲突 — P0）

```go
// internal/server/router.go
// 关键改动：泛型资源路由使用 /resources/ 前缀，与特化路由物理隔离

func (s *Server) setupRoutes() {
    v1 := s.engine.Group("/api/v1")

    // ── 公开路由 ──
    v1.POST("/auth/login", s.authHandler.Login)
    v1.POST("/auth/refresh", s.authHandler.Refresh)
    // ★ 2FA
    v1.POST("/auth/2fa/verify", s.authHandler.Verify2FA)    // 登录时提交 TOTP
    v1.POST("/auth/2fa/recovery", s.authHandler.UseRecovery) // 使用恢复码

    // ── 需要认证的路由 ──
    authed := v1.Group("", s.middleware.Auth())
    {
        // 用户
        authed.GET("/users/me", s.userHandler.Me)
        authed.PUT("/users/me/password", s.userHandler.ChangePassword)
        // ★ 2FA 管理
        authed.POST("/auth/2fa/setup", s.authHandler.Setup2FA)    // 生成 TOTP secret + 二维码
        authed.POST("/auth/2fa/enable", s.authHandler.Enable2FA)  // 确认启用
        authed.POST("/auth/2fa/disable", s.authHandler.Disable2FA)

        // 集群管理
        authed.GET("/clusters", s.clusterHandler.List)
        authed.POST("/clusters", s.clusterHandler.Create)
        authed.GET("/clusters/:cluster", s.clusterHandler.Get)
        authed.DELETE("/clusters/:cluster", s.clusterHandler.Delete)

        // 集群概览
        authed.GET("/clusters/:cluster/overview", s.overviewHandler.Get)

        // 资源发现 API（前端动态获取后端支持的资源列表）
        authed.GET("/clusters/:cluster/resource-definitions", s.resourceHandler.Definitions)

        // ★ 泛型资源 CRUD — 使用 /resources/ 前缀避免路由冲突
        // 命名空间级
        ns := authed.Group("/clusters/:cluster/namespaces/:namespace/resources")
        {
            ns.GET("/:resource", s.resourceHandler.List)
            ns.GET("/:resource/:name", s.resourceHandler.Get)
            ns.POST("/:resource", s.resourceHandler.Create)
            ns.PUT("/:resource/:name", s.resourceHandler.Update)
            ns.DELETE("/:resource/:name", s.resourceHandler.Delete)
            // Dry-run 变更预览
            ns.POST("/:resource/:name/dry-run", s.resourceHandler.DryRun)
        }
        // 集群级
        cl := authed.Group("/clusters/:cluster/resources")
        {
            cl.GET("/:resource", s.resourceHandler.List)
            cl.GET("/:resource/:name", s.resourceHandler.Get)
            cl.POST("/:resource", s.resourceHandler.Create)
            cl.PUT("/:resource/:name", s.resourceHandler.Update)
            cl.DELETE("/:resource/:name", s.resourceHandler.Delete)
        }

        // ★ 特殊操作 — 独立路由，不走泛型
        actions := authed.Group("/clusters/:cluster")
        {
            // Pod 操作
            actions.GET("/namespaces/:namespace/pods/:name/logs", s.wsHandler.Logs)
            // Deployment 操作
            actions.POST("/namespaces/:namespace/deployments/:name/scale",
                s.actionHandler.Scale)
            actions.POST("/namespaces/:namespace/deployments/:name/restart",
                s.actionHandler.Restart)
            actions.POST("/namespaces/:namespace/deployments/:name/rollback",
                s.actionHandler.Rollback)
            // Node 操作
            actions.POST("/nodes/:name/cordon", s.actionHandler.Cordon)
            actions.POST("/nodes/:name/uncordon", s.actionHandler.Uncordon)
            actions.POST("/nodes/:name/drain", s.actionHandler.Drain)
        }

        // 全局搜索
        authed.GET("/search", s.searchHandler.Search)

        // 资源模板
        authed.GET("/templates", s.templateHandler.List)
        authed.POST("/templates", s.templateHandler.Create)

        // 审计日志
        authed.GET("/audit/logs", s.auditHandler.List)

        // RBAC
        authed.GET("/roles", s.rbacHandler.List)
        authed.POST("/roles", s.rbacHandler.Create)

        // 系统设置
        authed.GET("/settings", s.settingHandler.List)
        authed.PUT("/settings/:key", s.settingHandler.Update)

        // ★ 收藏夹
        authed.GET("/favorites", s.favoriteHandler.List)
        authed.POST("/favorites", s.favoriteHandler.Create)
        authed.DELETE("/favorites/:id", s.favoriteHandler.Delete)

        // ★ Webhook 通知
        authed.GET("/webhooks", s.webhookHandler.List)
        authed.POST("/webhooks", s.webhookHandler.Create)
        authed.PUT("/webhooks/:id", s.webhookHandler.Update)
        authed.DELETE("/webhooks/:id", s.webhookHandler.Delete)
        authed.POST("/webhooks/:id/test", s.webhookHandler.Test)

        // ★ 终端会话录制
        authed.GET("/terminal-sessions", s.terminalSessionHandler.List)
        authed.GET("/terminal-sessions/:id", s.terminalSessionHandler.Get)
        authed.GET("/terminal-sessions/:id/play", s.terminalSessionHandler.Play)

        // ★ 跨集群资源对比
        authed.POST("/compare", s.resourceHandler.Compare)

        // ★ 资源配额
        authed.GET("/clusters/:cluster/namespaces/:namespace/quota", s.overviewHandler.Quota)
    }

    // ── WebSocket 路由 ──
    ws := v1.Group("/ws", s.middleware.Auth())
    {
        ws.GET("/watch", s.wsHandler.Watch)                           // 资源变更订阅
        ws.GET("/terminal/:cluster/:namespace/:pod", s.wsHandler.Terminal)  // Pod 终端
        ws.GET("/node-debug/:cluster/:node", s.wsHandler.NodeDebug)        // Node 调试
    }

    // ── 健康检查 ──
    s.engine.GET("/healthz", s.healthHandler.Liveness)
    s.engine.GET("/readyz", s.healthHandler.Readiness)
    s.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // ── 插件路由（动态注册）──
    s.pluginManager.RegisterAllRoutes(authed.Group("/plugins"))
}
```

### 5.3 ResourceService（★ 修复 Handler 跳过 Service 层 — P1）

```go
// internal/service/resource_service.go
// Handler 不再直接调用 K8s 客户端，所有逻辑走 Service

package service

type ResourceService struct {
    registry ResourceRegistry
    k8sRepo  K8sResourceRepo
    auditSvc *AuditService
}

func NewResourceService(
    registry ResourceRegistry,
    k8sRepo K8sResourceRepo,
    auditSvc *AuditService,
) *ResourceService {
    return &ResourceService{
        registry: registry,
        k8sRepo:  k8sRepo,
        auditSvc: auditSvc,
    }
}

// List 列出资源（缓存降级 + 新鲜度标记）
func (s *ResourceService) List(ctx context.Context, clusterID, namespace, resource string) (
    items []unstructured.Unstructured, stale bool, err error,
) {
    // 验证资源类型是否存在
    meta, ok := s.registry.Get(resource)
    if !ok {
        return nil, false, ErrUnknownResource(resource)
    }

    // 验证 scope：命名空间级资源必须提供 namespace
    if meta.Scope == NamespaceScoped && namespace == "" {
        return nil, false, ErrNamespaceRequired(resource)
    }

    return s.k8sRepo.List(ctx, clusterID, namespace, resource)
}

// Create 创建资源（含输入校验 + 审计）
func (s *ResourceService) Create(ctx context.Context, clusterID, namespace, resource string,
    obj *unstructured.Unstructured,
) (*unstructured.Unstructured, error) {
    meta, ok := s.registry.Get(resource)
    if !ok {
        return nil, ErrUnknownResource(resource)
    }

    // 输入校验：apiVersion/kind 必须与 URL 中的 resource 匹配
    if err := s.validateObject(obj, meta); err != nil {
        return nil, err
    }

    return s.k8sRepo.Create(ctx, clusterID, namespace, resource, obj)
}

// DryRun 变更预览（服务端 dry-run）
func (s *ResourceService) DryRun(ctx context.Context, clusterID, namespace, resource string,
    obj *unstructured.Unstructured,
) error {
    return s.k8sRepo.DryRun(ctx, clusterID, namespace, resource, obj)
}

// validateObject 校验请求体与 URL 资源类型是否匹配
func (s *ResourceService) validateObject(obj *unstructured.Unstructured, meta ResourceMeta) error {
    kind := obj.GetKind()
    if kind != "" && kind != meta.GVK.Kind {
        return fmt.Errorf("kind mismatch: URL expects %s, got %s", meta.GVK.Kind, kind)
    }
    return nil
}
```

### 5.4 K8s Repository（缓存降级 + ★ 错误码正确映射 — P1）

```go
// internal/repository/k8s_repo.go

package repository

import (
    apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type k8sRepo struct {
    registry    ResourceRegistry
    clusterMgr  ClusterManager
    informerMgr *informer.Manager
}

// List 优先走 Informer 缓存，缓存未命中降级到 API Server
func (r *k8sRepo) List(ctx context.Context, clusterID, namespace, resource string) (
    []unstructured.Unstructured, bool /* stale */, error,
) {
    meta, _ := r.registry.Get(resource)

    // 优先 Informer 缓存
    if meta.Cached {
        items, stale, err := r.informerMgr.List(clusterID, meta.GVR, namespace)
        if err == nil {
            return items, stale, nil
        }
        // 缓存失败，降级到直接查询（不返回错误）
    }

    // 直接查询 API Server
    client, err := r.clusterMgr.DynamicClient(clusterID)
    if err != nil {
        return nil, false, r.mapError(err)
    }

    var result *unstructured.UnstructuredList
    if namespace != "" {
        result, err = client.Resource(meta.GVR).Namespace(namespace).
            List(ctx, metav1.ListOptions{})
    } else {
        result, err = client.Resource(meta.GVR).List(ctx, metav1.ListOptions{})
    }
    if err != nil {
        return nil, false, r.mapError(err) // ★ 正确映射 K8s 错误码
    }

    return result.Items, false, nil
}

// ★ mapError 将 K8s API 错误映射为业务状态码（HTTP 统一返回 200）
func (r *k8sRepo) mapError(err error) error {
    if apierrors.IsNotFound(err) {
        return NewBizError(40400, err.Error())  // 资源不存在
    }
    if apierrors.IsForbidden(err) {
        return NewBizError(40300, err.Error())  // 无权限
    }
    if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) {
        return NewBizError(40900, err.Error())  // 冲突/已存在
    }
    if apierrors.IsInvalid(err) {
        return NewBizError(42200, err.Error())  // 校验失败
    }
    return NewBizError(50200, err.Error())      // K8s API 不可达
}
```

### 5.5 Informer Manager（★ 修复资源泄漏 + 并发安全 — P0）

```go
// internal/kubernetes/informer/manager.go

package informer

// EventListener 事件监听接口（★ 解耦 WebSocket Hub — P1）
type EventListener interface {
    OnResourceEvent(event ResourceEvent)
}

type ResourceEvent struct {
    Type      string // ADDED | MODIFIED | DELETED
    ClusterID string
    Resource  string
    Namespace string
    Name      string
    Object    map[string]interface{}
}

type clusterRuntime struct {
    factory dynamicinformer.DynamicSharedInformerFactory
    cancel  context.CancelFunc // ★ 用于停止 Informer
    synced  bool               // 缓存是否已同步
    lastSync time.Time         // 最后同步时间
}

type Manager struct {
    mu        sync.RWMutex
    clusters  map[string]*clusterRuntime
    listeners []EventListener // ★ 通过接口解耦，不直接依赖 ws.Hub
    logger    *zap.Logger
}

func NewManager(logger *zap.Logger) *Manager {
    return &Manager{
        clusters: make(map[string]*clusterRuntime),
        logger:   logger,
    }
}

// AddListener 注册事件监听器（WebSocket Hub 实现此接口）
func (m *Manager) AddListener(l EventListener) {
    m.listeners = append(m.listeners, l)
}

// StartForCluster 启动集群 Informer
// ★ 修复：锁持有时间最小化，WaitForCacheSync 在锁外异步执行
func (m *Manager) StartForCluster(
    clusterID string,
    client dynamic.Interface,
    resources []schema.GroupVersionResource,
    resyncPeriod time.Duration,
) {
    ctx, cancel := context.WithCancel(context.Background())

    factory := dynamicinformer.NewDynamicSharedInformerFactory(client, resyncPeriod)

    for _, gvr := range resources {
        informer := factory.ForResource(gvr).Informer()
        gvrCopy := gvr
        informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
            AddFunc:    func(obj interface{}) { m.notify(clusterID, "ADDED", gvrCopy.Resource, obj) },
            UpdateFunc: func(_, obj interface{}) { m.notify(clusterID, "MODIFIED", gvrCopy.Resource, obj) },
            DeleteFunc: func(obj interface{}) { m.notify(clusterID, "DELETED", gvrCopy.Resource, obj) },
        })
    }

    factory.Start(ctx.Done())

    rt := &clusterRuntime{factory: factory, cancel: cancel}

    // ★ 最小化锁持有时间：只锁 map 写操作
    m.mu.Lock()
    m.clusters[clusterID] = rt
    m.mu.Unlock()

    // ★ 异步等待缓存同步，不阻塞其他集群
    go func() {
        synced := factory.WaitForCacheSync(ctx.Done())
        allSynced := true
        for _, s := range synced {
            if !s {
                allSynced = false
                break
            }
        }
        m.mu.Lock()
        rt.synced = allSynced
        rt.lastSync = time.Now()
        m.mu.Unlock()
        m.logger.Info("informer cache synced",
            zap.String("cluster", clusterID), zap.Bool("allSynced", allSynced))
    }()
}

// StopForCluster ★ 修复：正确关闭 stopCh，防止 goroutine 泄漏
func (m *Manager) StopForCluster(clusterID string) {
    m.mu.Lock()
    rt, ok := m.clusters[clusterID]
    if ok {
        delete(m.clusters, clusterID)
    }
    m.mu.Unlock()

    if ok && rt != nil {
        rt.cancel() // ★ 触发 context 取消，Informer goroutine 会退出
    }
}

// List 从缓存查询，返回新鲜度标记
func (m *Manager) List(clusterID string, gvr schema.GroupVersionResource, namespace string) (
    []unstructured.Unstructured, bool /* stale */, error,
) {
    m.mu.RLock()
    rt, ok := m.clusters[clusterID]
    m.mu.RUnlock()
    if !ok {
        return nil, false, fmt.Errorf("no informer for cluster %s", clusterID)
    }

    // 新鲜度检测：超过 5 分钟未同步则标记为 stale
    stale := time.Since(rt.lastSync) > 5*time.Minute

    lister := rt.factory.ForResource(gvr).Lister()
    var objs []interface{}
    var err error
    if namespace != "" {
        objs, err = lister.ByNamespace(namespace).List(labels.Everything())
    } else {
        objs, err = lister.List(labels.Everything())
    }
    if err != nil {
        return nil, stale, err
    }

    result := make([]unstructured.Unstructured, 0, len(objs))
    for _, obj := range objs {
        if u, ok := obj.(*unstructured.Unstructured); ok {
            result = append(result, *u)
        }
    }
    return result, stale, nil
}

// notify 通知所有监听器（★ 非阻塞，带超时保护）
func (m *Manager) notify(clusterID, eventType, resource string, obj interface{}) {
    u, ok := obj.(*unstructured.Unstructured)
    if !ok {
        return
    }
    event := ResourceEvent{
        Type:      eventType,
        ClusterID: clusterID,
        Resource:  resource,
        Namespace: u.GetNamespace(),
        Name:      u.GetName(),
        Object:    u.Object,
    }
    for _, l := range m.listeners {
        l.OnResourceEvent(event) // 监听器自身保证非阻塞
    }
}
```

### 5.6 WebSocket Hub（★ 修复并发安全 — P0）

```go
// internal/handler/ws/hub.go
// ★ 修复：全部通过 channel 通信，单 goroutine 内操作 map，无需锁

package ws

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte       // 缓冲大小 1024
    register   chan *Client
    unregister chan *Client
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte, 1024), // 加大缓冲区
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

// Run 单 goroutine 事件循环 — ★ 所有 map 操作都在同一 goroutine 内，无竞态
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }

        case data := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- data:
                default:
                    // 客户端缓冲区满 → 断开
                    delete(h.clients, client)
                    close(client.send)
                }
            }
        }
    }
}

// OnResourceEvent 实现 informer.EventListener 接口
// ★ 非阻塞：如果 broadcast channel 满，丢弃事件（前端有兜底轮询）
func (h *Hub) OnResourceEvent(event informer.ResourceEvent) {
    data, err := json.Marshal(event)
    if err != nil {
        return
    }
    select {
    case h.broadcast <- data:
    default:
        // channel 满，丢弃（不阻塞 Informer）
    }
}
```

### 5.7 资源注册表（分级缓存策略 — P1）

```go
// internal/kubernetes/resource/registry.go

func (r *Registry) registerBuiltins() {
    // ★ 分级缓存策略：
    // - 核心高频资源 → Informer 缓存 (Cached: true)
    // - 低频/大量/敏感资源 → 按需查询 (Cached: false)
    //
    // 新增资源只需追加一行

    // === 缓存的核心资源 ===
    r.reg("pods",         "v1", "",     "Pod",         Namespace, true)
    r.reg("deployments",  "v1", "apps", "Deployment",  Namespace, true)
    r.reg("statefulsets", "v1", "apps", "StatefulSet", Namespace, true)
    r.reg("daemonsets",   "v1", "apps", "DaemonSet",   Namespace, true)
    r.reg("replicasets",  "v1", "apps", "ReplicaSet",  Namespace, true)
    r.reg("services",     "v1", "",     "Service",     Namespace, true)
    r.reg("nodes",        "v1", "",     "Node",        Cluster,   true)
    r.reg("namespaces",   "v1", "",     "Namespace",   Cluster,   true)

    // === 按需查询的资源 ===
    r.reg("jobs",         "v1", "batch", "Job",        Namespace, false)
    r.reg("cronjobs",     "v1", "batch", "CronJob",    Namespace, false)
    r.reg("ingresses",    "v1", "networking.k8s.io", "Ingress",       Namespace, false)
    r.reg("networkpolicies", "v1", "networking.k8s.io", "NetworkPolicy", Namespace, false)
    r.reg("endpoints",    "v1", "",     "Endpoints",   Namespace, false)
    r.reg("configmaps",   "v1", "",     "ConfigMap",   Namespace, false)
    r.reg("secrets",      "v1", "",     "Secret",      Namespace, false) // ★ Secrets 不缓存（安全+内存）
    r.reg("events",       "v1", "",     "Event",       Namespace, false) // ★ Events 不缓存（量大）

    // 存储
    r.reg("persistentvolumes",      "v1", "", "PersistentVolume",      Cluster,   false)
    r.reg("persistentvolumeclaims", "v1", "", "PersistentVolumeClaim", Namespace, false)
    r.reg("storageclasses", "v1", "storage.k8s.io", "StorageClass",    Cluster,   false)

    // RBAC
    r.reg("serviceaccounts",     "v1", "",     "ServiceAccount",  Namespace, false)
    r.reg("roles",               "v1", "rbac.authorization.k8s.io", "Role",               Namespace, false)
    r.reg("clusterroles",        "v1", "rbac.authorization.k8s.io", "ClusterRole",        Cluster,   false)
    r.reg("rolebindings",        "v1", "rbac.authorization.k8s.io", "RoleBinding",        Namespace, false)
    r.reg("clusterrolebindings", "v1", "rbac.authorization.k8s.io", "ClusterRoleBinding", Cluster,   false)

    // HPA
    r.reg("horizontalpodautoscalers", "v2", "autoscaling", "HorizontalPodAutoscaler", Namespace, false)

    // Gateway API（后期按需启用）
    r.reg("gateways",   "v1", "gateway.networking.k8s.io", "Gateway",   Namespace, false)
    r.reg("httproutes", "v1", "gateway.networking.k8s.io", "HTTPRoute", Namespace, false)
}
```

### 5.8 Secrets 脱敏（★ P1 安全）

```go
// internal/service/resource_service.go

// Get 获取单个资源（Secrets 自动脱敏）
func (s *ResourceService) Get(ctx context.Context, clusterID, namespace, resource, name string,
    showSecrets bool, // 需要显式传 true 才返回明文
) (*unstructured.Unstructured, error) {
    obj, err := s.k8sRepo.Get(ctx, clusterID, namespace, resource, name)
    if err != nil {
        return nil, err
    }

    // ★ Secrets 默认脱敏
    if resource == "secrets" && !showSecrets {
        s.maskSecretData(obj)
    }

    return obj, nil
}

func (s *ResourceService) maskSecretData(obj *unstructured.Unstructured) {
    data, found, _ := unstructured.NestedMap(obj.Object, "data")
    if !found {
        return
    }
    masked := make(map[string]interface{}, len(data))
    for k, v := range data {
        if str, ok := v.(string); ok {
            masked[k] = fmt.Sprintf("(%d bytes)", len(str))
        }
    }
    unstructured.SetNestedField(obj.Object, masked, "data")
}
```

---

## 6. 认证与授权（简化版）

### 6.1 认证方式

MVP 阶段只实现两种，后期通过插件扩展：

| 方式 | 阶段 | 说明 |
|------|------|------|
| **JWT + 本地密码** | Phase 1 | Access Token 15min + Refresh Token 7天 |
| **API Key** | Phase 2 | 程序化访问 |
| OAuth/OIDC | Phase 5 | 通过集成 Dex 实现，不自行实现协议 |
| LDAP | Phase 5 | 通过 Dex 的 LDAP connector |

### 6.2 JWT Token 管理

```go
// Access Token: 15 分钟，携带用户信息和权限
type TokenClaims struct {
    jwt.RegisteredClaims
    UserID       string            `json:"uid"`
    Username     string            `json:"username"`
    Role         string            `json:"role"`          // 全局角色
    ClusterRoles map[string]string `json:"clusterRoles"`  // clusterID → role
    TokenVersion int               `json:"tv"`            // ★ 用于主动吊销
}

// ★ Token 吊销机制：
// 用户密码修改/被禁用时，递增 user.TokenVersion
// 中间件校验时比对 Claims.tv 与数据库中的 TokenVersion
// 不需要 Redis 黑名单，简单可靠
```

### 6.3 RBAC 权限模型

```
预设角色（简洁五级）:
┌────────────┬────────────────────────────────────────────┐
│ admin      │ 全部权限（用户管理、集群管理、系统设置）     │
│ ops        │ 所有集群读写，不含用户管理                   │
│ dev        │ 指定集群/命名空间的读写                     │
│ readonly   │ 只读                                       │
│ custom     │ 自定义权限组合                              │
└────────────┴────────────────────────────────────────────┘

权限格式: {resource}:{action}
- 支持通配符: *:* 表示全部权限
- action 枚举: get | list | create | update | delete | exec | logs

RBAC 检查在中间件层完成，权限信息从 JWT Claims 读取，不查数据库。
权限变更通过 Token 刷新生效。
```

### 6.4 2FA (TOTP)

使用 `pquerna/otp` 库，兼容 Google Authenticator / Authy。

```go
// internal/auth/totp.go
type TOTPManager struct {
    issuer string // "KubeVision"
}

func (t *TOTPManager) GenerateSecret(username string) (*otp.Key, error)
func (t *TOTPManager) Validate(secret, code string) bool
func (t *TOTPManager) GenerateRecoveryCodes(count int) []string
```

**用户表新增字段：**
```
totp_secret_enc    -- 加密存储的 TOTP Secret
totp_enabled       -- 是否已启用
recovery_codes_enc -- 加密存储的恢复码(JSON 数组)
```

**流程：**
```
首次启用: 用户设置页 → 生成 TOTP Secret → 显示二维码 → 用户扫码 → 输入验证码确认 → 生成恢复码
登录流程: 用户名密码 → 验证通过 → 检查 2FA 启用? → 返回 {code: 40102, temp_token} → 用户输入 TOTP → 签发 JWT
敏感操作: 删除集群/修改 RBAC/查看 Secret 明文 → 要求再次输入 TOTP 验证码
```

### 6.5 审计日志

```go
// 异步批量写入，不影响请求性能
// 仅审计写操作（POST/PUT/PATCH/DELETE）
// Secrets 的 data 字段不记入审计日志
// 自动清理：默认保留 90 天，可配置
```

---

## 7. 前端架构

### 7.1 目录结构

```
web/src/
├── main.tsx
├── App.tsx
├── routes.tsx
│
├── components/
│   ├── ui/                         # shadcn/ui 组件
│   ├── layout/
│   │   ├── app-layout.tsx          # 主布局
│   │   ├── app-sidebar.tsx         # 侧边栏（根据资源定义动态生成）
│   │   └── app-header.tsx          # 顶栏：集群切换 + 用户菜单
│   │
│   ├── resource/                   # ★ 泛型资源组件
│   │   ├── resource-table.tsx      # 通用列表（根据 columns 配置渲染）
│   │   ├── resource-detail.tsx     # 通用详情
│   │   ├── resource-yaml.tsx       # YAML 编辑器（Monaco）
│   │   ├── resource-diff.tsx       # ★ 变更 Diff 预览（Dry-run 结果）
│   │   └── resource-topology.tsx   # ★ 关联资源拓扑图
│   │
│   ├── specialized/                # 特化组件
│   │   ├── pod-terminal.tsx
│   │   ├── pod-logs.tsx
│   │   ├── deployment-scale.tsx
│   │   ├── cluster-overview.tsx
│   │   ├── kubectl-hint.tsx        # ★ kubectl 命令生成提示
│   │   ├── cross-cluster-diff.tsx  # ★ 跨集群资源对比
│   │   ├── terminal-player.tsx     # ★ 终端会话回放（asciinema 播放器）
│   │   ├── quota-overview.tsx      # ★ 资源配额可视化（进度条 + 告警）
│   │   └── favorites-panel.tsx     # ★ 收藏夹面板
│   │
│   └── shared/
│       ├── global-search.tsx       # ⌘K 全局搜索
│       ├── cluster-switcher.tsx
│       ├── namespace-selector.tsx
│       ├── status-badge.tsx
│       └── confirm-dialog.tsx
│
├── pages/
│   ├── overview/
│   ├── resource-list.tsx           # ★ 通用资源列表页（读取 resource-config）
│   ├── resource-detail.tsx         # ★ 通用资源详情页
│   ├── monitoring/                 # 监控（插件启用时显示）
│   ├── gitops/                     # GitOps（插件启用时显示）
│   ├── audit/
│   ├── settings/
│   └── auth/
│
├── hooks/
│   ├── use-resource-list.ts        # 泛型资源列表
│   ├── use-resource-detail.ts
│   ├── use-resource-mutations.ts
│   ├── use-resource-definitions.ts # ★ 从后端拉取资源定义
│   ├── use-websocket.ts            # WebSocket + 自动重连
│   ├── use-cluster.ts
│   └── use-auth.ts
│
├── config/
│   └── resource-ui-config.ts       # ★ 仅 UI 相关配置（icon, columns, actions）
│                                   #   API 元数据从后端 /resource-definitions 获取
│
├── lib/
│   ├── api.ts                      # Axios 实例
│   ├── ws.ts                       # WebSocket 客户端（指数退避重连 + 心跳）
│   └── utils.ts
│
├── i18n/
│   ├── zh.json
│   └── en.json
│
└── styles/
    └── globals.css
```

### 7.2 资源发现 — 前后端配置不再重复

```typescript
// src/hooks/use-resource-definitions.ts
// ★ 资源 API 元数据从后端获取，前端只维护 UI 配置

export function useResourceDefinitions() {
  const { cluster } = useCluster()
  return useQuery({
    queryKey: ['resource-definitions', cluster],
    queryFn: () => api.get(`/api/v1/clusters/${cluster}/resource-definitions`),
    staleTime: 5 * 60 * 1000, // 5 分钟缓存
  })
}

// 使用时合并后端元数据和前端 UI 配置
export function useResourceConfig(resource: string) {
  const { data: definitions } = useResourceDefinitions()
  const uiConfig = resourceUIConfigs[resource] || defaultUIConfig
  const apiMeta = definitions?.[resource]

  return {
    ...apiMeta,    // name, scope, group, version, kind, cached, verbs
    ...uiConfig,   // icon, columns, actions, detailTabs
  }
}
```

```typescript
// src/config/resource-ui-config.ts
// ★ 仅 UI 展示相关配置，不包含 API 元数据

export const resourceUIConfigs: Record<string, Partial<ResourceUIConfig>> = {
  pods: {
    icon: Box,
    category: 'workloads',
    columns: [
      { key: 'metadata.name', label: 'Name' },
      { key: 'status.phase', label: 'Status', render: 'status-badge' },
      { key: 'spec.nodeName', label: 'Node' },
      { key: 'status.podIP', label: 'IP' },
      { key: 'metadata.creationTimestamp', label: 'Age', render: 'age' },
    ],
    actions: ['terminal', 'logs', 'delete'],
    detailTabs: ['overview', 'containers', 'events', 'logs', 'terminal', 'yaml'],
  },
  deployments: {
    icon: Server,
    category: 'workloads',
    columns: [
      { key: 'metadata.name', label: 'Name' },
      { key: 'status.readyReplicas', label: 'Ready', render: 'replicas' },
      { key: 'status.updatedReplicas', label: 'Up-to-date' },
      { key: 'metadata.creationTimestamp', label: 'Age', render: 'age' },
    ],
    actions: ['scale', 'restart', 'rollback', 'delete'],
  },
  // 未配置的资源自动使用 defaultUIConfig（显示 name + namespace + age）
}

export const defaultUIConfig: ResourceUIConfig = {
  icon: FileText,
  category: 'other',
  columns: [
    { key: 'metadata.name', label: 'Name' },
    { key: 'metadata.namespace', label: 'Namespace' },
    { key: 'metadata.creationTimestamp', label: 'Age', render: 'age' },
  ],
}
```

### 7.3 WebSocket 客户端（断线重连 + 心跳）

```typescript
// src/lib/ws.ts

class WebSocketClient {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private heartbeatTimer: number | null = null
  private subscriptions = new Map<string, Set<() => void>>()

  connect(url: string) {
    this.ws = new WebSocket(url)
    this.ws.onopen = () => {
      this.reconnectAttempts = 0
      this.startHeartbeat()
      this.resubscribeAll() // 重连后重新订阅
    }
    this.ws.onmessage = (e) => this.handleMessage(JSON.parse(e.data))
    this.ws.onclose = () => this.reconnect()
  }

  // 指数退避重连
  private reconnect() {
    this.stopHeartbeat()
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    setTimeout(() => {
      this.reconnectAttempts++
      this.connect(this.url)
    }, delay)
  }

  // 心跳检测
  private startHeartbeat() {
    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000)
  }

  // 订阅资源变更
  subscribe(topic: string, callback: () => void): () => void {
    if (!this.subscriptions.has(topic)) {
      this.subscriptions.set(topic, new Set())
    }
    this.subscriptions.get(topic)!.add(callback)
    return () => this.subscriptions.get(topic)?.delete(callback)
  }
}
```

---

## 8. 统一响应格式

HTTP 状态码统一返回 200，业务状态通过 body 中的 `code` 字段判断：

```go
// internal/pkg/response/response.go

type Response struct {
    Code    int         `json:"code"`              // 业务状态码，0=成功
    Message string      `json:"message"`           // 描述信息
    Data    interface{} `json:"data,omitempty"`    // 响应数据
    Meta    *Meta       `json:"meta,omitempty"`    // 元信息(可选)
}

type Meta struct {
    Source    string `json:"source,omitempty"`     // "cache" | "apiserver"
    Stale     bool   `json:"stale,omitempty"`      // 缓存是否过期
    Total     int64  `json:"total,omitempty"`      // 分页总数
    RequestID string `json:"requestId,omitempty"`  // 请求追踪 ID
}

// 成功
func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{Code: 0, Message: "success", Data: data})
}

// 成功(带元信息)
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
    c.JSON(200, Response{Code: 0, Message: "success", Data: data, Meta: meta})
}

// 失败
func Error(c *gin.Context, code int, message string) {
    c.JSON(200, Response{Code: code, Message: message})
}
```

**业务状态码规范：**

```
0       成功
400xx   参数/请求错误  (40001 参数缺失, 40002 格式错误)
401xx   认证错误      (40100 未登录, 40101 Token过期, 40102 需要2FA, 40103 2FA验证失败)
403xx   权限错误      (40300 无权限, 40301 集群无权限, 40302 命名空间无权限)
404xx   资源不存在    (40400 K8s资源不存在, 40401 用户不存在, 40402 集群不存在)
409xx   冲突         (40900 资源已存在, 40901 版本冲突)
422xx   校验失败      (42200 YAML无效, 42201 Dry-run失败)
500xx   服务端错误    (50000 内部错误, 50200 K8s API不可达)
```

**响应示例：**

```json
// 成功
{"code": 0, "message": "success", "data": {"name": "nginx", "status": "Running"}}

// 成功(列表 + 缓存元信息)
{"code": 0, "message": "success", "data": [...], "meta": {"source": "cache", "stale": false, "total": 150}}

// 业务错误
{"code": 40101, "message": "token expired"}
{"code": 40300, "message": "no permission: delete deployments in prod"}
{"code": 40400, "message": "deployment nginx not found"}

// 需要 2FA
{"code": 40102, "message": "2fa required", "data": {"temp_token": "xxx"}}
```

---

## 9. 数据库设计

### 9.1 核心表

```
users                          clusters
├── id (PK)                    ├── id (PK)
├── username (unique)          ├── name (unique)
├── email                      ├── display_name
├── password_hash              ├── api_server
├── role                       ├── auth_type (kubeconfig|token|in-cluster)
├── auth_provider (local)      ├── kubeconfig_enc (加密)
├── token_version (★吊销用)    ├── token_enc (加密)
├── totp_secret_enc (★2FA)     ├── status
├── totp_enabled               ├── version
├── recovery_codes_enc         └── created_at
├── is_active
├── last_login_at
├── created_at
└── updated_at

user_cluster_roles             roles
├── id (PK)                    ├── id (PK)
├── user_id (FK)               ├── name (unique)
├── cluster_id (FK)            ├── display_name
├── role_id (FK)               ├── is_system (内置不可删)
└── namespaces (逗号分隔)      └── permissions (JSON)

audit_logs                     templates
├── id (PK, auto)              ├── id (PK)
├── user_id                    ├── name
├── username                   ├── category
├── action                     ├── resource_type
├── resource                   ├── content (YAML)
├── name                       ├── is_builtin
├── namespace                  └── created_at
├── cluster
├── status_code                settings
├── duration_ms                ├── key (PK)
├── client_ip                  ├── value
├── request_body (≤4KB)        └── category
└── created_at

api_keys                       ★ webhooks
├── id (PK)                    ├── id (PK)
├── user_id (FK)               ├── name
├── name                       ├── url
├── key_hash (unique)          ├── secret (签名用)
├── key_prefix (显示用)        ├── events (JSON: ["delete","scale"])
├── expires_at                 ├── clusters (JSON: ["prod"])
└── created_at                 ├── resources (JSON: ["deployments"])
                               ├── is_active
                               └── created_at

★ favorites                    ★ terminal_sessions
├── id (PK)                    ├── id (PK)
├── user_id (FK)               ├── user_id (FK)
├── cluster_id                 ├── cluster
├── namespace                  ├── namespace
├── resource_type              ├── pod
├── resource_name              ├── container
├── display_name               ├── recording (BLOB/TEXT, asciinema v2)
├── sort_order                 ├── duration_ms
└── created_at                 ├── created_at
                               └── expires_at (默认 30 天后)
```

### 9.2 数据库选择

| 场景 | 推荐 | 原因 |
|------|------|------|
| 开发/个人/小团队 | **SQLite** | 零配置，单文件 |
| 生产/多副本 | **PostgreSQL** | 并发写入、JSON 字段、完善的生态 |

MySQL 不作为首选支持，减少测试矩阵。如有需求后期通过 GORM 方言扩展。

---

## 10. 功能完整性矩阵

### 10.1 完整功能列表

| 功能 | 阶段 | 说明 |
|------|------|------|
| **核心 CRUD** | | |
| 泛型资源列表/详情/创建/编辑/删除 | P1 | 一个 Handler 覆盖所有 |
| YAML 编辑器（Monaco） | P1 | 语法高亮 + 校验 |
| ★ Dry-run 变更预览 + Diff | P2 | K8s server-side dry-run |
| ★ 资源变更历史 | P2 | 记录每次 apply 前后的 YAML diff |
| **工作负载** | | |
| Deployment 扩缩容/重启/回滚 | P2 | 特殊操作路由 |
| Pod 终端（exec） | P2 | xterm.js + WebSocket |
| Pod 日志（流式 + 搜索 + 下载） | P2 | 支持多容器选择 |
| CronJob 手动触发 | P2 | |
| **集群管理** | | |
| 多集群增删改查 | P1 | kubeconfig / token / in-cluster |
| 集群概览仪表盘 | P1 | 资源统计 + 健康状态 |
| ★ 集群健康诊断 | P3 | 证书过期、组件状态、资源压力 |
| 命名空间管理 | P1 | |
| **节点管理** | | |
| 节点列表 + 详情 + 指标 | P1 | |
| Cordon/Uncordon/Drain | P2 | |
| ★ Node 调试（kubectl debug 方式） | P3 | 安全替代 SSH |
| Label/Taint 编辑 | P2 | |
| **网络** | | |
| Service/Ingress 管理 | P1 | |
| NetworkPolicy | P2 | |
| Gateway API（Gateways/HTTPRoutes） | P4 | 后期按需 |
| **存储** | | |
| PV/PVC/StorageClass | P1 | |
| **配置** | | |
| ConfigMap 编辑 | P1 | |
| Secret 查看（★ 默认脱敏，点击解码） | P1 | |
| **RBAC** | | |
| 用户管理 | P1 | |
| 角色管理（五级预设 + 自定义） | P3 | |
| K8s RBAC 查看（Role/ClusterRole/Binding） | P1 | |
| **搜索** | | |
| ★ 全局搜索（⌘K） | P2 | 按 name 搜索跨集群资源 |
| ★ Label 选择器过滤 | P2 | |
| **可视化** | | |
| ★ 资源拓扑图（Deployment→RS→Pod→Node） | P3 | |
| ★ 关联资源查询 | P2 | Pod 的 Service/ConfigMap/Secret |
| **实时** | | |
| Informer 缓存 + WebSocket 推送 | P1 | 亚秒级更新 |
| 缓存新鲜度标记（stale 提示） | P1 | |
| **批量操作** | | |
| ★ 批量删除 | P3 | 表格多选 |
| ★ 批量添加 Label/Annotation | P3 | |
| **模板** | | |
| 资源模板（内置 + 自定义） | P3 | |
| YAML 导出/下载 | P2 | |
| **监控（插件）** | | |
| Prometheus 指标查询 | P4 | |
| Grafana 仪表盘嵌入 | P4 | |
| AlertManager 告警管理 | P4 | |
| **GitOps（插件）** | | |
| ArgoCD Application 列表/同步 | P4 | 只做查看和 Sync |
| **安全** | | |
| JWT + 本地密码 | P1 | |
| ★ 2FA (TOTP) | P3 | Google Authenticator 兼容，敏感操作二次验证 |
| API Key | P2 | |
| ★ Secrets 默认脱敏 | P1 | 显式请求才返回明文 |
| ★ 终端会话录制回放 | P3 | asciinema v2 格式，Web 播放器，30 天保留 |
| OAuth/OIDC（通过 Dex） | P4 | |
| 审计日志（自动清理） | P1 | |
| **独有功能** | | |
| ★ kubectl 命令生成 | P2 | 每个 UI 操作显示等效 kubectl 命令，可复制 |
| ★ 跨集群资源对比 | P3 | 同一资源跨集群 Side-by-side YAML Diff |
| ★ Webhook 通知 | P3 | 资源变更→Slack/钉钉/飞书/企业微信 |
| ★ 资源配额可视化 | P2 | Namespace 配额进度条 + 超限告警 |
| ★ 收藏夹 | P2 | 常用集群/命名空间/资源一键收藏 |
| **其他** | | |
| 暗色/亮色/系统主题 | P1 | shadcn/ui 原生支持 |
| 中英文 i18n | P2 | |
| 用户偏好持久化 | P2 | 存后端 settings 表 |
| ★ 快捷键系统 | P3 | |

### 10.2 明确不做的功能

| 功能 | 原因 |
|------|------|
| Node SSH 终端 | 安全风险大，用 kubectl debug node 替代 |
| ArgoCD Repository 管理 | 超出 Dashboard 职责 |
| Helm Release 管理 | 复杂度高，建议用 Helm Dashboard 等专用工具 |
| AI 助手 | 可作为后期插件，MVP 不含 |
| 移动端适配 | K8s Dashboard 主要桌面使用 |

---

## 11. 分阶段实施计划

```
Phase 1: MVP 骨架 ─────────────────────────────── 3-4 周
  单集群 + 泛型 CRUD + SQLite + JWT 认证 + 基础前端
  ✅ 能看到资源列表、详情、YAML
  ✅ 能创建/编辑/删除资源
  ✅ Informer 缓存 + WebSocket 实时推送

Phase 2: 交互能力 ─────────────────────────────── 2-3 周
  Pod 终端 + 日志 + Deployment 操作 + 全局搜索
  多集群管理 + 命名空间选择器
  Dry-run 变更预览 + 资源变更历史
  ★ kubectl 命令生成 + 收藏夹 + 资源配额可视化

Phase 3: 安全管控 ─────────────────────────────── 2-3 周
  完整 RBAC + 审计日志 + API Key + ★ 2FA(TOTP)
  资源拓扑图 + 批量操作
  集群健康诊断 + Node 调试
  ★ 跨集群资源对比 + Webhook 通知 + 终端会话录制
  PostgreSQL 支持

Phase 4: 生态集成 ─────────────────────────────── 3-4 周
  插件系统 + Prometheus + Grafana + ArgoCD
  OAuth/OIDC (Dex) + CRD 动态发现
  Gateway API + i18n
  Helm Chart + 生产级部署文档
```

---

## 12. 部署

### 12.1 开发

```bash
# 先决条件：Go 1.23+, Node.js 22+, 一个 K8s 集群
git clone https://github.com/kubevision/kubevision.git && cd kubevision

# 后端
go mod download && air  # 热重载

# 前端（另一个终端）
cd web && pnpm install && pnpm dev

# 访问 http://localhost:5173
```

### 12.2 生产

```bash
# Docker（最简单）
docker run -d -p 8080:8080 \
  -v kubevision-data:/data \
  -e KUBECONFIG=/data/kubeconfig \
  ghcr.io/kubevision/kubevision:latest

# Helm
helm install kubevision kubevision/kubevision \
  --namespace kubevision --create-namespace \
  --set config.database.driver=postgresql \
  --set config.database.dsn="host=pg port=5432 ..."

# 单二进制
./kubevision serve --config config.yaml
```

### 12.3 多副本部署注意事项

- **SQLite 模式仅限单副本**（文件锁限制）
- 多副本必须使用 PostgreSQL
- WebSocket 需要 Ingress 配置 sticky session
- 每个副本独立维护 Informer 缓存（无状态）
- JWT 无状态认证天然支持多副本

### 12.4 Docker 多阶段构建

```
Stage 1: node:22-alpine  → pnpm install --frozen-lockfile + vite build
Stage 2: golang:1.23     → go:embed 前端产物 + go build
Stage 3: alpine:3.20     → 仅含二进制 (30-50MB)，非 root 运行
```

---

## 13. 测试策略

### 13.1 后端

| 层级 | 工具 | 覆盖 |
|------|------|------|
| 单元测试 | `go test` + testify | Service/Repository 层，mock K8s 接口 |
| 集成测试 | `envtest` (controller-runtime) | Informer + K8s API 端到端 |
| API 测试 | httptest + Gin test mode | Handler 层 |
| 覆盖率 | `go test -cover` | 目标 ≥70% |

接口驱动的设计使得 mock 测试非常简单：

```go
// 示例：mock K8sResourceRepo 测试 ResourceService
type mockK8sRepo struct { items []unstructured.Unstructured }
func (m *mockK8sRepo) List(...) ([]unstructured.Unstructured, bool, error) {
    return m.items, false, nil
}
```

### 13.2 前端

| 层级 | 工具 |
|------|------|
| 组件测试 | Vitest + Testing Library |
| Hook 测试 | renderHook + MSW (API mock) |
| E2E | Playwright |

### 13.3 CI 流水线

```yaml
# .github/workflows/ci.yaml
jobs:
  backend:
    - golangci-lint run ./...
    - go test ./... -cover -coverprofile=coverage.out
    - go build ./cmd/kubevision/
  frontend:
    - pnpm lint
    - pnpm test
    - pnpm build
```

---

## 14. 配置项清单

```yaml
# config.yaml（或通过环境变量）

server:
  port: 8080                       # PORT
  base_path: ""                    # 反向代理路径前缀

database:
  driver: sqlite                   # DB_DRIVER: sqlite | postgresql
  dsn: "kubevision.db"            # DB_DSN

auth:
  jwt_secret: ""                   # JWT_SECRET（留空自动生成）
  access_token_ttl: 15m           # ACCESS_TOKEN_TTL
  refresh_token_ttl: 168h         # REFRESH_TOKEN_TTL (7天)
  totp_issuer: "KubeVision"      # TOTP_ISSUER（2FA 显示名称）
  totp_force: false               # TOTP_FORCE（强制全员开启 2FA）

kubernetes:
  kubeconfig: ""                   # KUBECONFIG（留空使用 in-cluster）
  informer_resync: 30m            # INFORMER_RESYNC_PERIOD
  informer_stale_threshold: 5m   # 缓存过期提示阈值

websocket:
  broadcast_buffer: 1024          # WS_BROADCAST_BUFFER
  heartbeat_interval: 30s         # WS_HEARTBEAT_INTERVAL

audit:
  enabled: true                    # AUDIT_ENABLED
  retention_days: 90              # AUDIT_RETENTION_DAYS
  batch_size: 100                 # 批量写入大小
  flush_interval: 5s             # 批量写入间隔

encrypt_key: ""                    # ENCRYPT_KEY（加密 kubeconfig/token）

terminal_recording:
  enabled: true                    # 启用终端会话录制
  retention_days: 30              # 录制保留天数
  max_size_mb: 10                 # 单次录制最大体积

webhook:
  timeout: 10s                     # HTTP 推送超时
  retry_count: 3                  # 失败重试次数
  retry_interval: 5s             # 重试间隔

plugins:
  prometheus:
    enabled: false
    url: ""
  grafana:
    enabled: false
    url: ""
  argocd:
    enabled: false
    url: ""
    token: ""
```

---

## 附录：审视报告回应

> 本次 v2 架构针对 Planner 和 Architect 两位专家的审视报告做了以下修正：

| 问题 | 级别 | 修正 |
|------|------|------|
| 路由冲突 | P0 | 泛型资源路由加 `/resources/` 前缀（§5.2） |
| WebSocket Hub 并发安全 | P0 | 改为纯 channel 通信，单 goroutine 操作 map（§5.6） |
| Informer StopForCluster 泄漏 | P0 | 使用 context.Cancel 正确关闭（§5.5） |
| 缺少测试策略 | P0 | 新增完整测试策略（§13） |
| Handler 跳过 Service 层 | P1 | 引入 ResourceService（§5.3） |
| Informer 反向依赖 Hub | P1 | EventListener 接口解耦（§5.5） |
| Handler 依赖具体类型 | P1 | 全部通过接口注入（§5.1） |
| Informer 内存消耗 | P1 | 分级缓存策略，Secrets/Events 不缓存（§5.7） |
| K8s 错误码统一 500 | P1 | mapError 正确映射 K8s 状态码（§5.4） |
| Secrets 无脱敏 | P1 | 默认脱敏，显式请求才返回明文（§5.8） |
| JWT 无法吊销 | P1 | TokenVersion 机制（§6.2） |
| 前后端配置重复 | P2 | 资源发现 API，前端只维护 UI 配置（§7.2） |
| 缓存新鲜度 | P2 | stale 标记 + 响应元数据（§5.5, §8） |
| 认证过度设计 | P2 | MVP 只做 JWT+密码，后期通过 Dex 集成（§6.1） |
| 缺少 Dry-run/Diff | P2 | 新增 DryRun 接口和 resource-diff 组件（§5.3, §7.1） |
| 遗漏：拓扑图/批量操作/历史 | - | 全部纳入功能矩阵（§10.1） |
| 过度设计：Gateway/多DB/SSH | - | Gateway 延后，MySQL 去除，SSH 改 kubectl debug（§10.2） |
| 对比矩阵过于乐观 | - | 已移除，待项目实现后基于事实重新评估 |
