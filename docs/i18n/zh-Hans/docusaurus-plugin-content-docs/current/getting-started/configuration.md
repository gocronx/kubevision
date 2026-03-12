---
sidebar_position: 3
title: 配置
---

# 配置

KubeVision 支持通过 YAML 文件或环境变量进行配置。

## 配置文件

```yaml
server:
  port: 8080

database:
  driver: sqlite           # sqlite | postgres
  dsn: kubevision.db       # file path or connection string

auth:
  jwt_secret: ""           # auto-generated if empty
  access_token_ttl: 15m
  refresh_token_ttl: 168h  # 7 days

kubernetes:
  kubeconfig: ""           # empty = in-cluster mode
  informer_resync: 30m
```

## 环境变量

所有配置项均可通过环境变量进行覆盖：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `KUBEVISION_SERVER_PORT` | HTTP 端口 | `8080` |
| `KUBEVISION_DB_DRIVER` | 数据库驱动（`sqlite` 或 `postgres`）| `sqlite` |
| `KUBEVISION_DB_DSN` | 数据库连接字符串 | `kubevision.db` |
| `KUBEVISION_JWT_SECRET` | JWT 签名密钥 | 自动生成 |
| `KUBEVISION_ACCESS_TOKEN_TTL` | 访问令牌有效期 | `15m` |
| `KUBEVISION_REFRESH_TOKEN_TTL` | 刷新令牌有效期 | `168h` |
| `KUBECONFIG` | kubeconfig 文件路径 | 集群内模式 |
| `KUBEVISION_INFORMER_RESYNC` | Informer 重同步周期 | `30m` |
| `KUBEVISION_ALLOWED_ORIGINS` | WebSocket 来源白名单（逗号分隔）| `*` |

## 数据库

### SQLite（开发环境）

默认配置，无需额外设置：

```yaml
database:
  driver: sqlite
  dsn: kubevision.db
```

### PostgreSQL（生产环境）

```yaml
database:
  driver: postgres
  dsn: "host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
```

或通过环境变量设置：

```bash
export KUBEVISION_DB_DRIVER=postgres
export KUBEVISION_DB_DSN="host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
```

## Kubernetes 连接

### 集群内模式

当 KubeVision 部署在 Kubernetes 集群内部时，会自动使用 ServiceAccount 令牌，无需任何配置。

### 集群外模式

指定 kubeconfig 文件路径：

```bash
export KUBECONFIG=/path/to/kubeconfig
```

或在配置文件中指定：

```yaml
kubernetes:
  kubeconfig: /path/to/kubeconfig
```

## Informer 缓存

KubeVision 使用 Kubernetes Informer 对高频访问资源进行缓存，实现亚毫秒级读取：

**已缓存资源（8 种）：** Pods、Deployments、StatefulSets、DaemonSets、Services、Ingresses、Nodes、Namespaces

**按需加载资源（18 种以上）：** Jobs、CronJobs、ConfigMaps、PVs、PVCs 等

**永不缓存：** Secrets（安全考量）、Events（数据量过大）

`informer_resync` 配置项控制缓存与 API Server 进行全量重同步的频率。
