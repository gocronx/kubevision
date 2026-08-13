---
sidebar_position: 2
title: 数据流
---

# 数据流

## AI 运维链路

```text
页面上下文 + 用户问题
          │
          ▼
   AI Chat Handler ──→ OpenAI 兼容提供商
          ▲                       │
          │                    工具请求
          │                       ▼
     流式结果 ← Authorizer → Kubernetes/日志/Prometheus
                          │
                     是否为修改？
                          │
                    暂停并等待批准
                          │
                 重新授权 → 执行 → 审计
```

1. 浏览器发送当前集群、命名空间、页面、资源和对话历史。
2. 提供商只能请求 KubeVision 已注册的工具。
3. 服务端根据工具和参数检查当前用户的 RBAC 权限。
4. 读取工具通过授权后立即执行，并返回有长度限制的结果。
5. 修改请求不会立即执行，而是创建一个短期、归属于当前用户的待确认操作。
6. 用户确认后会消费该操作、重新检查权限、仅执行一次，并记录审计结果。

Agent 循环次数和工具输出都有上限。模型不能使用任意 Shell 或任意出站 HTTP 工具。

## 读取路径

```text
Browser → Handler → Service → K8sRepo
                                  │
                     ┌────────────┴────────────┐
                     │                          │
              Informer Cache              API Server
              (sub-ms, 8 types)        (fallback, all types)
```

1. Handler 接收 GET 请求
2. Service 将请求委托给 K8sRepo
3. K8sRepo 优先查询 Informer 缓存（8 种核心资源）
4. 若缓存未命中或该资源未被缓存，则回退到 API Server
5. 响应中包含 `meta.source`（值为 "cache" 或 "apiserver"）及 `meta.stale` 标志

## 写入路径

```text
Browser → Handler → Service → K8sRepo → API Server
                                              │
                                        Informer Watch
                                              │
                                       EventListener
                                              │
                                          WS Hub
                                              │
                                         All Browsers
```

1. Handler 接收 POST/PUT/DELETE 请求
2. Service 校验后将请求委托给 K8sRepo
3. K8sRepo 直接调用 API Server
4. API Server 应用变更
5. Informer Watch 检测到变更
6. EventListener 通知 WS Hub
7. Hub 向所有已连接的浏览器广播消息
8. 浏览器自动更新（无需轮询）

## 实时推送

实时系统完全解耦：

```go
// EventListener interface — Hub implements this
type EventListener interface {
    OnResourceEvent(event ResourceEvent)
}

// Hub: non-blocking receive, drop if channel full
// (frontend has 30s fallback polling)
func (h *Hub) OnResourceEvent(event ResourceEvent) {
    select {
    case h.broadcast <- marshal(event):
    default: // drop, don't block Informer
    }
}
```

## Informer 缓存策略

| 已缓存（8 种） | 按需查询 | 不缓存 |
|---------------|----------|--------|
| Pods | Jobs | Secrets（安全考量） |
| Deployments | CronJobs | Events（数据量过大） |
| StatefulSets | ConfigMaps | |
| DaemonSets | PVs, PVCs | |
| Services | StorageClasses | |
| Ingresses | NetworkPolicies | |
| Nodes | Roles, RoleBindings | |
| Namespaces | ServiceAccounts | |
