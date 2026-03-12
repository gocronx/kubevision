---
sidebar_position: 3
title: 目录结构
---

# 目录结构

```text
kubevision/
├── cmd/kubevision/main.go           # 入口文件
├── internal/
│   ├── config/config.go             # 配置
│   ├── server/
│   │   ├── server.go                # HTTP Server
│   │   └── router.go                # 路由注册
│   ├── middleware/                   # Auth / RBAC / Audit / Logger
│   ├── handler/
│   │   ├── resource_handler.go      # 泛型资源增删改查
│   │   ├── resource_action_handler.go # scale/cordon/drain
│   │   ├── auth_handler.go
│   │   ├── cluster_handler.go
│   │   ├── search_handler.go
│   │   ├── webhook_handler.go       # Webhook 通知
│   │   ├── favorite_handler.go      # 收藏夹
│   │   ├── terminal_session_handler.go # 终端录制
│   │   └── ws/                      # Hub / Terminal / Logs / Watch
│   ├── service/
│   │   ├── resource_service.go      # 缓存回退 + 校验
│   │   ├── auth_service.go
│   │   ├── cluster_service.go
│   │   ├── audit_service.go         # 异步批量写入
│   │   ├── rbac_service.go
│   │   ├── webhook_service.go       # 事件匹配 + HTTP 推送
│   │   ├── favorite_service.go
│   │   └── terminal_session_service.go
│   ├── repository/
│   │   ├── interfaces.go            # 所有接口定义
│   │   ├── k8s_repo.go              # Informer + Dynamic Client
│   │   └── db.go / user_repo.go     # GORM 仓储
│   ├── model/                       # User / Cluster / Role / AuditLog
│   ├── kubernetes/
│   │   ├── cluster/manager.go       # 多集群管理
│   │   ├── informer/manager.go      # Informer 生命周期
│   │   ├── resource/registry.go     # 资源注册表
│   │   └── exec/terminal.go         # Pod exec / Node debug
│   ├── auth/
│   │   ├── jwt.go                   # JWT + Token 吊销
│   │   └── totp.go                  # 双因素认证：TOTP 生成与校验
│   └── plugin/                      # Prometheus / Grafana / ArgoCD
├── web/                             # React 前端
│   └── src/
│       ├── components/
│       │   ├── ui/                  # shadcn/ui 组件
│       │   ├── resource/            # 泛型组件：Table / Detail / YAML / Diff
│       │   └── specialized/         # Terminal / Logs / kubectl-hint 等
│       ├── hooks/
│       │   ├── use-resource-list.ts # 泛型资源 Hook
│       │   └── use-websocket.ts     # 自动重连 + 心跳
│       ├── pages/                   # 路由页面
│       └── config/resource-ui-config.ts  # UI 配置（图标/列/操作）
├── docs/                            # 文档站点 (Docusaurus)
├── deploy/                          # Dockerfile / Helm / install.yaml
└── Makefile
```

## 核心模式

### 泛型资源 Handler
单一 Handler 管理所有 Kubernetes 资源的增删改查。新增资源类型只需：
1. 在后端资源注册表中添加一行配置
2. 在前端 UI 配置文件中添加一个配置块

### 接口驱动设计
所有依赖均通过接口注入，使代码库完全可测试：

```go
// Every service accepts interfaces, not concrete types
func NewResourceService(
    k8sRepo K8sResourceRepo,
    registry ResourceRegistry,
    auditSvc AuditService,
) *ResourceService
```
