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
  max_open_conns: 0        # 驱动默认值：SQLite 1，PostgreSQL 25
  max_idle_conns: 0        # 驱动默认值：SQLite 1，PostgreSQL 5
  conn_max_lifetime: 0s    # PostgreSQL 默认值：30m
  conn_max_idle_time: 0s   # PostgreSQL 默认值：5m
  ping_timeout: 5s

auth:
  jwt_secret: ""           # auto-generated if empty
  access_token_ttl: 30m
  refresh_token_ttl: 12h
  public_key:
    enabled: false
    rp_id: ""
    rp_display_name: KubeVision
    origins: []
    user_verification: required
    counter_policy: deny
    challenge_ttl: 5m

kubernetes:
  kubeconfig: ""           # empty = in-cluster mode
  informer_resync: 30m
  crd_discovery_interval: 5m

oauth:
  enabled: false
  providers: []
```

## 环境变量

所有配置项均可通过环境变量进行覆盖：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `KUBEVISION_SERVER_PORT` | HTTP 端口 | `8080` |
| `KUBEVISION_DB_DRIVER` | 数据库驱动（`sqlite` 或 `postgres`）| `sqlite` |
| `KUBEVISION_DB_DSN` | 数据库连接字符串 | `kubevision.db` |
| `KUBEVISION_DB_MAX_OPEN_CONNS` | 数据库最大连接数 | 按驱动默认 |
| `KUBEVISION_DB_MAX_IDLE_CONNS` | 数据库最大空闲连接数 | 按驱动默认 |
| `KUBEVISION_DB_CONN_MAX_LIFETIME` | 连接最长生命周期 | PostgreSQL：`30m` |
| `KUBEVISION_DB_CONN_MAX_IDLE_TIME` | 空闲连接最长时间 | PostgreSQL：`5m` |
| `KUBEVISION_DB_PING_TIMEOUT` | 启动时数据库探测超时 | `5s` |
| `KUBEVISION_JWT_SECRET` | JWT 签名密钥 | 自动生成 |
| `KUBEVISION_ACCESS_TOKEN_TTL` | 访问令牌有效期 | `30m` |
| `KUBEVISION_REFRESH_TOKEN_TTL` | 刷新令牌有效期 | `12h` |
| `KUBEVISION_PUBLIC_KEY_ENABLED` | 启用通行密钥/安全密钥登录 | `false` |
| `KUBEVISION_PUBLIC_KEY_RP_ID` | WebAuthn 依赖方域名 | 空 |
| `KUBEVISION_PUBLIC_KEY_ORIGINS` | 允许的浏览器来源（逗号分隔）| 空 |
| `KUBECONFIG` | kubeconfig 文件路径 | 集群内模式 |
| `KUBEVISION_INFORMER_RESYNC` | Informer 重同步周期 | `30m` |
| `KUBEVISION_CRD_DISCOVERY_INTERVAL` | CRD 发现刷新周期 | `5m` |
| `KUBEVISION_OAUTH_ENABLED` | 启用已配置的 OAuth/OIDC 提供商 | `false` |
| `KUBEVISION_ENCRYPT_KEY` | 持久化凭据加密密钥 | 自动生成 |
| `KUBEVISION_ALLOWED_ORIGINS` | WebSocket 来源白名单（逗号分隔）| `*` |

## 数据库

### SQLite（开发环境）

默认配置，无需额外设置：

```yaml
database:
  driver: sqlite
  dsn: kubevision.db
```

SQLite 仅允许一个 KubeVision 进程使用。Kubernetes 中使用 SQLite 时必须
保持单副本并关闭自动扩容；需要横向扩容时应先迁移到 PostgreSQL。

### PostgreSQL（生产环境）

```yaml
database:
  driver: postgres
  dsn: "host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
```

或通过环境变量设置：

```bash
export KUBEVISION_DB_DRIVER=postgres
export KUBEVISION_DB_DSN="host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
```

KubeVision 会记录数据库结构版本，并在启动时通过 PostgreSQL 锁串行执行迁移，
避免多副本同时修改结构。升级前仍应备份数据库。`/healthz` 表示进程存活，
`/readyz` 会检查数据库连接，应配置为 Kubernetes readiness probe。
Helm 多副本部署必须通过 `existingSecret` 提供共享的 `KUBEVISION_DB_DSN`、
`KUBEVISION_JWT_SECRET` 和 `KUBEVISION_ENCRYPT_KEY`。

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
