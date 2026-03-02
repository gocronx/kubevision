# KubeVision vs Kite vs KubePolaris

---

## 架构

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| 资源 Handler | Go 泛型，逐个注册实例 | 每资源独立 Handler（大量重复） | **统一 Handler + Registry，新增资源加一行** |
| 分层 | Handler 直调 K8s | Handler→Service，DB 注入 Handler | **Handler→Service→Repo，职责清晰** |
| 依赖注入 | 部分接口化 | 具体类型注入 | **全接口注入，可 mock** |
| Informer 缓存 | 无 | 有 | **有，分级（核心缓存/敏感不缓存）** |
| 实时推送 | WS 仅日志/终端 | WS 仅日志/终端 | **Informer→WS Hub→前端自动刷新** |
| 插件系统 | 无 | 无 | **Plugin 接口，按需启用** |
| 新增资源成本 | 1 文件 1 行 | 6-8 文件 300+ 行 | **0 新文件，2 处配置** |

---

## 安全

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| 本地密码 | Yes | Yes | Yes |
| OAuth/OIDC | Yes（自行实现） | No | Yes（Dex 集成） |
| LDAP | No | Yes | Yes（Dex 集成） |
| **2FA (TOTP)** | No | No | **Yes，敏感操作二次验证** |
| API Key | Yes | No | Yes |
| **Token 吊销** | 无 | 无 | **TokenVersion，即时生效** |
| **Secrets 脱敏** | 直接暴露 | 直接暴露 | **默认脱敏** |
| RBAC | 自定义角色 | 五种权限 | 五级角色，**权限在 JWT 不查库** |
| 审计日志 | ResourceHistory | 含终端审计 | 异步批量 + 自动清理 |
| **终端会话录制** | No | 记录命令文本 | **完整录制 + 回放（asciinema 格式）** |
| **操作通知** | No | No | **Webhook 通知（Slack/钉钉/飞书/企业微信）** |

---

## 功能

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| 多集群 | Yes | Yes | Yes |
| 集群概览 | Yes | Yes | Yes |
| **集群健康诊断** | No | No | **证书过期/组件状态/资源压力** |
| Pod 终端 | Yes | Yes | Yes |
| Node 终端 | DaemonSet 代理 | SSH | kubectl debug（更安全） |
| Pod 日志 | Yes | Yes | Yes |
| 全局搜索 | Yes | No | Yes |
| **Dry-run + Diff** | No | 部分资源 | **全资源 + 可视化 Diff** |
| 资源变更历史 | Yes | No | Yes |
| **资源拓扑图** | 关联列表 | No | **Deployment→RS→Pod→Node 可视化** |
| **批量操作** | No | No | **批量删除/Label/Annotation** |
| 资源模板 | Yes | No | Yes |
| CRD 支持 | 手动 | No | **动态发现，自动注册** |
| **kubectl 命令生成** | No | No | **每个 UI 操作显示等效 kubectl 命令** |
| **跨集群资源对比** | No | No | **同一资源跨集群/环境 Side-by-side Diff** |
| **资源配额可视化** | No | No | **Namespace 配额用量进度条 + 超限告警** |
| **收藏夹** | No | No | **常用资源/集群/命名空间一键收藏** |
| Gateway API | Yes | No | Yes（后期） |

---

## 监控与集成

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| Prometheus | Yes（硬编码） | Yes（硬编码） | Yes（**插件化**） |
| Grafana | No | Yes（硬编码） | Yes（插件化） |
| AlertManager | No | Yes（硬编码） | Yes（插件化） |
| ArgoCD | No | Yes（硬编码） | Yes（插件化） |
| **自身 Metrics** | No | No | **/metrics Prometheus 格式** |
| **Webhook 通知** | No | No | **资源变更→Slack/钉钉/飞书** |

---

## 前端

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| UI 框架 | Radix + Tailwind | Ant Design | **shadcn/ui** |
| 暗色主题 | Yes | No | Yes |
| i18n | 中/英 | 中/英 | 中/英 |
| **资源定义动态获取** | 前端硬编码 | 前端硬编码 | **后端 API 动态获取** |
| **泛型组件** | 部分 | No | **Table/Detail/YAML/Diff 统一组件** |
| **kubectl 提示** | No | No | **操作时显示等效命令，可复制** |

---

## 响应格式

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| HTTP 状态码 | 语义化(200/400/500) | 语义化 | **统一 200，业务码在 body** |
| 错误码 | 字符串 message | 简单数字 | **五位业务码（401xx/403xx...）** |
| 缓存元信息 | No | No | **source/stale 标记** |

---

## 部署

| 维度 | Kite | KubePolaris | **KubeVision** |
|------|------|-------------|----------------|
| 单二进制 | Yes | Yes | Yes |
| Docker / Helm | Yes | Yes | Yes |
| 数据库 | SQLite/MySQL/PG | MySQL/SQLite | SQLite/PostgreSQL |
| **多副本说明** | 未说明 | 未说明 | **SQLite 单副本，PG 多副本** |

---

## 独有功能详解

### 1. kubectl 命令生成

每个 UI 操作旁显示等效 kubectl 命令，可一键复制：

```
┌─────────────────────────────────────────────────────┐
│  删除 Pod: nginx-7d5b8c-x2k9f                       │
│                                                     │
│  [确认删除]  [取消]                                   │
│                                                     │
│  ┌─ kubectl ──────────────────────────────────────┐ │
│  │ kubectl delete pod nginx-7d5b8c-x2k9f \       │ │
│  │   -n production --cluster=prod-cluster    📋  │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**价值**: 用户从 Dashboard 操作中学习 kubectl，降低 K8s 学习曲线。

### 2. 跨集群资源对比

选择两个集群/环境，对同一资源做 Side-by-side YAML Diff：

```
┌──── staging ────────────┬──── production ───────────┐
│ apiVersion: apps/v1     │ apiVersion: apps/v1       │
│ kind: Deployment        │ kind: Deployment          │
│ spec:                   │ spec:                     │
│   replicas: 1          →│   replicas: 3             │
│   image: nginx:1.26    →│   image: nginx:1.25       │
│   resources:            │   resources:              │
│     cpu: 100m          →│     cpu: 500m             │
└─────────────────────────┴───────────────────────────┘
```

**价值**: 多集群管理的核心场景——"为什么 staging 正常但 prod 挂了？"

### 3. 终端会话录制回放

Pod/Node 终端操作全程录制，管理员可回放审查：

```
录制格式: asciinema v2（行业标准，可在终端回放）
存储: 数据库 + 对象存储（可选）
回放: 内置 Web 播放器，支持倍速/暂停/搜索
保留: 可配置（默认 30 天）
```

**价值**: 安全合规要求（等保/SOC2），比 KubePolaris 的命令文本记录更完整。

### 4. Webhook 通知

资源变更事件推送到团队协作工具：

```yaml
webhooks:
  - name: prod-alerts
    url: https://hooks.slack.com/services/xxx
    events: [delete, scale]          # 只关注删除和扩缩容
    clusters: [prod]                 # 只关注生产集群
    resources: [deployments, pods]   # 只关注工作负载
```

**价值**: 团队实时感知集群变更，无需每个人都盯着 Dashboard。

### 5. 资源配额可视化

Namespace 资源配额用量一目了然：

```
┌─ production 命名空间配额 ──────────────────────────┐
│                                                     │
│  CPU    ████████████░░░░░░  12/20 cores  (60%)     │
│  Memory ██████████████░░░░  14/20 Gi     (70%)     │
│  Pods   ██████░░░░░░░░░░░░  45/150       (30%)     │
│                                                     │
│  ⚠️ Memory 接近限额，建议扩容或清理空闲 Pod          │
└─────────────────────────────────────────────────────┘
```

**价值**: 资源规划和成本控制的核心视图。

### 6. 收藏夹

常用集群/命名空间/资源一键收藏，快速访问：

```
★ 收藏夹
├── prod / default / deployment/nginx
├── prod / monitoring / statefulset/prometheus
└── staging / default / pods
```

**价值**: 管理多个集群时快速定位高频资源。

---

## 优势总结

| # | 功能 | Kite | KubePolaris | KubeVision | 价值 |
|---|------|------|-------------|------------|------|
| 1 | 统一泛型架构 | 部分 | 无 | **全栈泛型** | 新增资源零文件 |
| 2 | Informer→WS 全链路 | 无 | 未打通前端 | **亚秒推送** | 实时体验 |
| 3 | 2FA | 无 | 无 | **TOTP** | 安全合规 |
| 4 | Secrets 脱敏 | 无 | 无 | **默认脱敏** | 安全默认 |
| 5 | kubectl 命令生成 | 无 | 无 | **每操作生成** | 降低学习成本 |
| 6 | 跨集群对比 | 无 | 无 | **Side-by-side Diff** | 多集群运维利器 |
| 7 | 终端会话录制 | 无 | 命令文本 | **完整录制回放** | 安全审计 |
| 8 | Webhook 通知 | 无 | 无 | **Slack/钉钉/飞书** | 团队协作 |
| 9 | 资源配额可视化 | 无 | 无 | **进度条+告警** | 成本管控 |
| 10 | Dry-run + Diff | 无 | 部分 | **全资源** | 生产安全 |
| 11 | 资源拓扑图 | 列表 | 无 | **可视化** | 故障排查 |
| 12 | CRD 动态发现 | 手动 | 无 | **自动注册** | 开箱即用 |
| 13 | 插件化集成 | 无 | 无 | **按需启用** | 轻量部署 |
| 14 | 收藏夹 | 无 | 无 | **快速访问** | 效率提升 |
| 15 | 业务码体系 | 无 | 简单 | **五位码+缓存元信息** | 前端友好 |
