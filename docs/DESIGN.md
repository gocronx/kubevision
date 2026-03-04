# KubeVision 架构精要

> Go + React + shadcn/ui | 简单易用，功能完整

---

## 技术栈

**后端** Go 1.23 / Gin / GORM / client-go / zap / pquerna/otp(TOTP)
**前端** React 19 / TypeScript / Vite / shadcn/ui / TanStack Query / Monaco Editor / xterm.js
**数据库** SQLite（开发）/ PostgreSQL（生产）

---

## 架构

```
浏览器 (React + shadcn/ui + TanStack Query)
  │ REST                    │ WebSocket
  ▼                         ▼
┌─────────────────────────────────────────────────┐
│ 中间件: Auth → RBAC → Audit                      │
│                                                   │
│ Handler ──→ Service ──→ K8sRepo ──→ K8s API      │
│   (参数解析)   (业务逻辑)   (缓存降级)    ↑        │
│                                    Informer Cache │
│                                        │          │
│ Informer Watch ──→ EventListener ──→ WS Hub      │
│                                                   │
│ 插件(可选): Prometheus │ Grafana │ ArgoCD         │
│ 数据库:     SQLite │ PostgreSQL                   │
└─────────────────────────────────────────────────┘
```

**核心链路：**
- 读: Handler → Service → K8sRepo（Informer 缓存优先，miss 降级 API Server）
- 写: Handler → Service → K8sRepo → API Server → Informer Watch → WS Hub → 浏览器自动刷新
- 新增资源: 后端 Registry 加一行 + 前端 UI Config 加一段，零新文件

---

## 目录结构

```
kubevision/
├── cmd/kubevision/main.go           # 入口
├── internal/
│   ├── config/config.go             # 配置
│   ├── server/
│   │   ├── server.go                # HTTP Server
│   │   └── router.go                # 路由注册
│   ├── middleware/                   # Auth / RBAC / Audit / Logger
│   ├── handler/
│   │   ├── resource_handler.go      # ★ 泛型资源 CRUD
│   │   ├── resource_action_handler.go # scale/cordon/drain 等特殊操作
│   │   ├── auth_handler.go
│   │   ├── cluster_handler.go
│   │   ├── search_handler.go
│   │   ├── webhook_handler.go       # ★ Webhook 通知
│   │   ├── favorite_handler.go      # ★ 收藏夹
│   │   ├── terminal_session_handler.go # ★ 终端录制/回放
│   │   └── ws/                      # Hub / Terminal / Logs / Watch
│   ├── service/
│   │   ├── resource_service.go      # ★ 缓存降级 + 输入校验 + 错误映射
│   │   ├── auth_service.go
│   │   ├── cluster_service.go
│   │   ├── audit_service.go         # 异步批量写入
│   │   ├── rbac_service.go
│   │   ├── webhook_service.go       # ★ 事件匹配 + HTTP 推送
│   │   ├── favorite_service.go      # ★ 收藏夹管理
│   │   └── terminal_session_service.go # ★ 录制存储 + 回放
│   ├── repository/
│   │   ├── interfaces.go            # ★ 全部接口定义
│   │   ├── k8s_repo.go              # Informer + Dynamic Client
│   │   └── db.go / user_repo.go ... # GORM
│   ├── model/                       # User / Cluster / Role / AuditLog / Webhook / Favorite / TerminalSession
│   ├── kubernetes/
│   │   ├── cluster/manager.go       # 多集群管理
│   │   ├── informer/manager.go      # Informer 生命周期 + EventListener
│   │   ├── resource/registry.go     # ★ 资源注册表
│   │   └── exec/terminal.go         # Pod exec / Node debug
│   ├── auth/
│   │   ├── jwt.go                   # JWT + Token 吊销(版本号)
│   │   └── totp.go                  # 2FA: TOTP 生成/验证/恢复码
│   └── plugin/                      # Prometheus / Grafana / ArgoCD
├── web/                             # React 前端
│   └── src/
│       ├── components/
│       │   ├── ui/                  # shadcn/ui
│       │   ├── resource/            # ★ 泛型: Table / Detail / YAML / Diff / Topology
│       │   └── specialized/         # Pod Terminal / Logs / Deploy Scale
│       │       ├── kubectl-hint.tsx     # ★ kubectl 命令生成
│       │       ├── cross-cluster-diff.tsx # ★ 跨集群对比
│       │       ├── terminal-player.tsx  # ★ 终端回放
│       │       ├── quota-overview.tsx   # ★ 配额可视化
│       │       └── favorites-panel.tsx  # ★ 收藏夹
│       ├── hooks/
│       │   ├── use-resource-list.ts # ★ 泛型资源 Hook
│       │   └── use-websocket.ts     # 自动重连 + 心跳
│       ├── pages/                   # resource-list / resource-detail / overview ...
│       └── config/resource-ui-config.ts  # 仅 UI 配置(icon/columns/actions)
├── deploy/                          # Dockerfile / Helm / install.yaml
└── Makefile
```

---

## 路由

```
# 认证 + 2FA
POST  /api/v1/auth/login              # 用户名密码 → 返回 requires_2fa 或 JWT
POST  /api/v1/auth/2fa/verify         # 提交 TOTP 验证码 → 签发 JWT
POST  /api/v1/auth/refresh            # 刷新 Token
POST  /api/v1/auth/2fa/setup          # 生成 TOTP secret + 二维码
POST  /api/v1/auth/2fa/enable         # 确认启用(验证一次 TOTP)
POST  /api/v1/auth/2fa/disable        # 关闭 2FA
POST  /api/v1/auth/2fa/recovery       # 使用恢复码登录

# 泛型资源 CRUD（/resources/ 前缀避免冲突）
GET|POST        /api/v1/clusters/:c/namespaces/:ns/resources/:res
GET|PUT|DELETE   /api/v1/clusters/:c/namespaces/:ns/resources/:res/:name
GET|POST        /api/v1/clusters/:c/resources/:res
GET|PUT|DELETE   /api/v1/clusters/:c/resources/:res/:name

# 特殊操作（独立路由）
POST  .../deployments/:name/scale|restart|rollback
POST  .../nodes/:name/cordon|uncordon|drain
POST  .../resources/:res/:name/dry-run

# WebSocket
/api/v1/ws/watch                          # 资源变更订阅
/api/v1/ws/terminal/:cluster/:ns/:pod     # Pod 终端
/api/v1/ws/logs/:cluster/:ns/:pod         # 日志流

# 独有功能
GET|POST|DELETE  /api/v1/favorites        # ★ 收藏夹
GET|POST|PUT|DELETE  /api/v1/webhooks     # ★ Webhook 通知配置
POST  /api/v1/webhooks/:id/test           # ★ 测试 Webhook
GET   /api/v1/terminal-sessions           # ★ 终端会话录制列表
GET   /api/v1/terminal-sessions/:id/play  # ★ 回放
POST  /api/v1/compare                     # ★ 跨集群资源对比
GET   /api/v1/clusters/:c/namespaces/:ns/quota  # ★ 资源配额

# 其他
GET  /api/v1/clusters/:c/overview         # 集群概览
GET  /api/v1/clusters/:c/resource-definitions  # 资源发现(前端动态获取)
GET  /api/v1/search?q=                    # 全局搜索
GET  /healthz | /readyz | /metrics
```

---

## 关键设计

### 1. 泛型资源注册表

```go
// 新增资源 = 加一行
r.reg("pods",        "v1", "",     "Pod",        Namespace, true)   // cached
r.reg("deployments", "v1", "apps", "Deployment", Namespace, true)   // cached
r.reg("secrets",     "v1", "",     "Secret",     Namespace, false)  // 不缓存(安全+内存)
r.reg("events",      "v1", "",     "Event",      Namespace, false)  // 不缓存(量大)
```

缓存策略：8 种核心高频资源走 Informer，其余按需查询 API Server。

### 2. 接口驱动

```go
// 所有依赖通过接口注入，mock 测试一行代码
type K8sResourceRepo interface {
    List(ctx, cluster, ns, resource) (items, stale, err)
    Get(ctx, cluster, ns, resource, name) (*Unstructured, err)
    Create / Update / Delete / DryRun
}
type ClusterManager interface { DynamicClient(cluster) / RESTConfig(cluster) }
type ResourceRegistry interface { Get(name) / All() }
```

### 3. Informer → WebSocket 实时链

```go
// 解耦：Informer 不直接依赖 WebSocket Hub
type EventListener interface {
    OnResourceEvent(event ResourceEvent)  // Hub 实现此接口
}

// Hub 非阻塞接收，channel 满则丢弃（前端有 30s 兜底轮询）
func (h *Hub) OnResourceEvent(event) {
    select {
    case h.broadcast <- marshal(event):
    default: // 丢弃，不阻塞 Informer
    }
}
```

### 4. 安全

- **认证**: JWT (Access 15min + Refresh 7天) + API Key
- **2FA**: TOTP（Google Authenticator / Authy），管理员可强制全员开启
- **吊销**: user.TokenVersion 递增，中间件校验版本号，无需 Redis
- **RBAC**: 五级角色 (admin/ops/dev/readonly/custom)，权限存 JWT Claims，不查库
- **Secrets**: 默认脱敏，显式请求才返回明文
- **审计**: 写操作异步批量记录，90 天自动清理

**2FA 流程:**
```
首次启用: 用户设置页 → 生成 TOTP Secret → 显示二维码 → 用户扫码 → 输入验证码确认 → 生成恢复码
登录流程: 用户名密码 → 验证通过 → 检查 2FA 启用? → 要求输入 TOTP 验证码 → 签发 JWT
敏感操作: 删除集群/修改 RBAC/查看 Secret 明文 → 要求再次输入 TOTP 验证码
```

### 5. 统一响应格式

HTTP 状态码统一 200，业务状态通过 `code` 判断：

```go
type Response struct {
    Code    int         `json:"code"`              // 业务状态码
    Message string      `json:"message"`           // 描述
    Data    interface{} `json:"data,omitempty"`    // 数据
    Meta    *Meta       `json:"meta,omitempty"`    // 元信息(可选)
}

type Meta struct {
    Source    string `json:"source,omitempty"`    // "cache" | "apiserver"
    Stale    bool   `json:"stale,omitempty"`     // 缓存是否过期
    Total    int64  `json:"total,omitempty"`     // 分页总数
    RequestID string `json:"requestId,omitempty"`
}
```

```json
// 成功
{"code": 0, "message": "success", "data": {...}}

// 成功(带缓存元信息)
{"code": 0, "message": "success", "data": [...], "meta": {"source": "cache", "stale": false, "total": 150}}

// 业务错误
{"code": 40100, "message": "token expired"}
{"code": 40300, "message": "no permission: delete deployments in prod"}
{"code": 40400, "message": "deployment nginx not found"}
{"code": 42200, "message": "invalid YAML: line 12"}

// 需要 2FA
{"code": 40102, "message": "2fa required", "data": {"temp_token": "xxx"}}
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

前端统一拦截：
```typescript
// api.ts — Axios 响应拦截器
api.interceptors.response.use((res) => {
  const { code, message } = res.data
  if (code === 0) return res.data                    // 成功
  if (code === 40101) { refreshToken(); return }     // Token过期 → 刷新
  if (code === 40102) { router.push('/2fa'); return } // 需要2FA
  if (code >= 40000) { toast.error(message); return Promise.reject(res.data) }
  return res.data
})
```

---

## 数据库

```
users           clusters          roles             audit_logs
─────           ────────          ─────             ──────────
id              id                id                id (auto)
username        name              name              user_id
password_hash   api_server        permissions(JSON) action/resource/name
role            auth_type         is_system         cluster/namespace
token_version   kubeconfig_enc                      status_code
totp_secret_enc status                              created_at
totp_enabled
recovery_codes_enc

user_cluster_roles    api_keys          templates         settings
──────────────────    ────────          ─────────         ────────
user_id → cluster_id  user_id           name/category     key → value
role_id / namespaces  key_hash          content(YAML)

★ webhooks            ★ favorites       ★ terminal_sessions
──────────            ─────────         ───────────────────
id/name/url           id/user_id        id/user_id
secret(签名)          cluster/ns        cluster/ns/pod
events(JSON)          resource_type     recording(asciinema v2)
clusters/resources    resource_name     duration_ms
is_active             sort_order        expires_at(30天)
```

SQLite 开发用，PostgreSQL 生产用。

---

## 独有功能

### 1. kubectl 命令生成
每个 UI 操作旁显示等效 kubectl 命令，可一键复制。降低 K8s 学习曲线。
```
前端: KubectlHint 组件，根据 action + resource + params 拼接命令
后端: 无额外 API，纯前端生成
```

### 2. 跨集群资源对比
选择两个集群/环境，对同一资源做 Side-by-side YAML Diff。
```
API: POST /api/v1/compare {source: {cluster, ns, resource, name}, target: {cluster, ns, resource, name}}
返回: 两个集群的资源 YAML，前端用 Monaco Diff Editor 渲染
```

### 3. 终端会话录制回放
Pod/Node 终端操作全程录制，管理员可回放审查。
```
格式: asciinema v2（行业标准）
存储: terminal_sessions 表
回放: 内置 Web 播放器，支持倍速/暂停
保留: 可配置（默认 30 天）
```

### 4. Webhook 通知
资源变更事件推送到团队协作工具（Slack/钉钉/飞书/企业微信）。
```
触发: Informer Watch 事件 → WebhookService 匹配规则 → HTTP POST
配置: 按事件类型(delete/scale) + 集群 + 资源类型过滤
签名: HMAC-SHA256，接收方可验证来源
```

### 5. 资源配额可视化
Namespace 资源配额用量进度条 + 超限告警。
```
API: GET /clusters/:c/namespaces/:ns/quota
数据源: ResourceQuota + LimitRange 对象
展示: CPU/Memory/Pods 进度条 + 百分比 + 阈值告警
```

### 6. 收藏夹
常用集群/命名空间/资源一键收藏，快速访问。侧边栏顶部显示。

---

## 实施计划

```
P1 (3-4周) MVP
  单集群 + 泛型 CRUD + Informer + WebSocket + JWT + 基础 UI
  → 能看、能编辑、能删除、实时刷新

P2 (2-3周) 交互
  Pod 终端/日志 + Deployment 操作 + 多集群 + 全局搜索 + Dry-run Diff
  ★ kubectl 命令生成 + 收藏夹 + 资源配额可视化

P3 (2-3周) 管控
  RBAC + 审计 + API Key + ★ 2FA(TOTP) + 资源拓扑 + 批量操作
  ★ 跨集群资源对比 + Webhook 通知 + 终端会话录制
  PostgreSQL 支持

P4 (3-4周) 生态
  插件系统 + Prometheus/Grafana/ArgoCD + OAuth(Dex) + CRD 动态发现 + Helm Chart
```

---

## 不做

| 功能 | 原因 |
|------|------|
| Node SSH | 安全风险，用 kubectl debug 替代 |
| Helm 管理 | 复杂度高，用专用工具 |
| AI 助手 | 后期插件 |
| MySQL | 减少测试矩阵，SQLite + PG 足够 |
| 移动端 | Dashboard 主要桌面使用 |
