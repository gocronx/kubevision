---
sidebar_position: 2
title: Data Flow
---

# Data Flow

## Read Path

```text
Browser → Handler → Service → K8sRepo
                                  │
                     ┌────────────┴────────────┐
                     │                          │
              Informer Cache              API Server
              (sub-ms, 8 types)        (fallback, all types)
```

1. Handler receives GET request
2. Service delegates to K8sRepo
3. K8sRepo checks Informer cache first (8 core resources)
4. If cache miss or resource not cached, falls back to API Server
5. Response includes `meta.source` ("cache" or "apiserver") and `meta.stale` flag

## Write Path

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

1. Handler receives POST/PUT/DELETE request
2. Service validates and delegates to K8sRepo
3. K8sRepo calls API Server directly
4. API Server applies the change
5. Informer Watch detects the change
6. EventListener notifies WS Hub
7. Hub broadcasts to all connected browsers
8. Browsers update automatically (no polling needed)

## Real-time Push

The real-time system is fully decoupled:

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

## Informer Cache Strategy

| Cached (8 types) | On-Demand | Never Cached |
|-------------------|-----------|--------------|
| Pods | Jobs | Secrets (security) |
| Deployments | CronJobs | Events (volume) |
| StatefulSets | ConfigMaps | |
| DaemonSets | PVs, PVCs | |
| Services | StorageClasses | |
| Ingresses | NetworkPolicies | |
| Nodes | Roles, RoleBindings | |
| Namespaces | ServiceAccounts | |
