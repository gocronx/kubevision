---
sidebar_position: 3
title: Directory Structure
---

# Directory Structure

```text
kubevision/
├── cmd/kubevision/main.go           # Entry point
├── internal/
│   ├── config/config.go             # Configuration
│   ├── server/
│   │   ├── server.go                # HTTP Server
│   │   └── router.go                # Route registration
│   ├── middleware/                   # Auth / RBAC / Audit / Logger
│   ├── handler/
│   │   ├── resource_handler.go      # Generic resource CRUD
│   │   ├── resource_action_handler.go # scale/cordon/drain
│   │   ├── auth_handler.go
│   │   ├── cluster_handler.go
│   │   ├── search_handler.go
│   │   ├── webhook_handler.go       # Webhook notifications
│   │   ├── favorite_handler.go      # Favorites
│   │   ├── terminal_session_handler.go # Terminal recording
│   │   └── ws/                      # Hub / Terminal / Logs / Watch
│   ├── service/
│   │   ├── resource_service.go      # Cache fallback + validation
│   │   ├── auth_service.go
│   │   ├── cluster_service.go
│   │   ├── audit_service.go         # Async batch writes
│   │   ├── rbac_service.go
│   │   ├── webhook_service.go       # Event matching + HTTP push
│   │   ├── favorite_service.go
│   │   └── terminal_session_service.go
│   ├── repository/
│   │   ├── interfaces.go            # All interface definitions
│   │   ├── k8s_repo.go              # Informer + Dynamic Client
│   │   └── db.go / user_repo.go     # GORM repositories
│   ├── model/                       # User / Cluster / Role / AuditLog
│   ├── kubernetes/
│   │   ├── cluster/manager.go       # Multi-cluster management
│   │   ├── informer/manager.go      # Informer lifecycle
│   │   ├── resource/registry.go     # Resource registry
│   │   └── exec/terminal.go         # Pod exec / Node debug
│   ├── auth/
│   │   ├── jwt.go                   # JWT + token revocation
│   │   └── totp.go                  # 2FA: TOTP generation/verification
│   └── plugin/                      # Prometheus / Grafana / ArgoCD
├── web/                             # React frontend
│   └── src/
│       ├── components/
│       │   ├── ui/                  # shadcn/ui components
│       │   ├── resource/            # Generic: Table / Detail / YAML / Diff
│       │   └── specialized/         # Terminal / Logs / kubectl-hint / etc.
│       ├── hooks/
│       │   ├── use-resource-list.ts # Generic resource hook
│       │   └── use-websocket.ts     # Auto-reconnect + heartbeat
│       ├── pages/                   # Route pages
│       └── config/resource-ui-config.ts  # UI config (icon/columns/actions)
├── docs/                            # Documentation site (Docusaurus)
├── deploy/                          # Dockerfile / Helm / install.yaml
└── Makefile
```

## Key Patterns

### Generic Resource Handler
A single handler manages CRUD for all Kubernetes resources. New resources only need:
1. One line in the backend resource registry
2. One config block in the frontend UI config

### Interface-Driven Design
All dependencies are injected via interfaces, making the codebase fully testable:

```go
// Every service accepts interfaces, not concrete types
func NewResourceService(
    k8sRepo K8sResourceRepo,
    registry ResourceRegistry,
    auditSvc AuditService,
) *ResourceService
```
