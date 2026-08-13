---
sidebar_position: 3
title: Configuration
---

# Configuration

KubeVision is configured via a YAML file or environment variables.

## Configuration File

```yaml
server:
  port: 8080

database:
  driver: sqlite           # sqlite | postgres
  dsn: kubevision.db       # file path or connection string
  max_open_conns: 0        # driver default: SQLite 1, PostgreSQL 25
  max_idle_conns: 0        # driver default: SQLite 1, PostgreSQL 5
  conn_max_lifetime: 0s    # PostgreSQL default: 30m
  conn_max_idle_time: 0s   # PostgreSQL default: 5m
  ping_timeout: 5s

auth:
  jwt_secret: ""           # auto-generated if empty
  access_token_ttl: 15m
  refresh_token_ttl: 168h  # 7 days

kubernetes:
  kubeconfig: ""           # empty = in-cluster mode
  informer_resync: 30m
```

## Environment Variables

All settings can be overridden via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBEVISION_SERVER_PORT` | HTTP port | `8080` |
| `KUBEVISION_DB_DRIVER` | Database driver (`sqlite` or `postgres`) | `sqlite` |
| `KUBEVISION_DB_DSN` | Database connection string | `kubevision.db` |
| `KUBEVISION_DB_MAX_OPEN_CONNS` | Maximum open database connections | driver default |
| `KUBEVISION_DB_MAX_IDLE_CONNS` | Maximum idle database connections | driver default |
| `KUBEVISION_DB_CONN_MAX_LIFETIME` | Maximum connection lifetime | PostgreSQL: `30m` |
| `KUBEVISION_DB_CONN_MAX_IDLE_TIME` | Maximum idle connection time | PostgreSQL: `5m` |
| `KUBEVISION_DB_PING_TIMEOUT` | Startup database ping timeout | `5s` |
| `KUBEVISION_JWT_SECRET` | JWT signing secret | auto-generated |
| `KUBEVISION_ACCESS_TOKEN_TTL` | Access token lifetime | `15m` |
| `KUBEVISION_REFRESH_TOKEN_TTL` | Refresh token lifetime | `168h` |
| `KUBECONFIG` | Path to kubeconfig file | in-cluster |
| `KUBEVISION_INFORMER_RESYNC` | Informer resync period | `30m` |
| `KUBEVISION_ALLOWED_ORIGINS` | WebSocket origin whitelist (comma-separated) | `*` |

## Database

### SQLite (Development)

Default configuration, zero setup:

```yaml
database:
  driver: sqlite
  dsn: kubevision.db
```

SQLite is limited to one KubeVision process. Kubernetes deployments using
SQLite must use one replica with autoscaling disabled. Use PostgreSQL before
scaling horizontally.

### PostgreSQL (Production)

```yaml
database:
  driver: postgres
  dsn: "host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 30m
  conn_max_idle_time: 5m
```

Or via environment variable:

```bash
export KUBEVISION_DB_DRIVER=postgres
export KUBEVISION_DB_DSN="host=localhost port=5432 user=kubevision password=secret dbname=kubevision sslmode=disable"
```

KubeVision records schema versions and serializes PostgreSQL migrations during
startup, preventing multiple replicas from changing the schema concurrently.
Back up the database before upgrading. `/healthz` reports process liveness;
`/readyz` checks database connectivity and should be used for readiness probes.
Multi-replica Helm deployments must use `existingSecret` to provide one shared
`KUBEVISION_DB_DSN`, `KUBEVISION_JWT_SECRET`, and `KUBEVISION_ENCRYPT_KEY`.

## Kubernetes Connection

### In-Cluster Mode

When deployed inside a Kubernetes cluster, KubeVision automatically uses the service account token. No configuration needed.

### External Mode

Point to your kubeconfig:

```bash
export KUBECONFIG=/path/to/kubeconfig
```

Or in the config file:

```yaml
kubernetes:
  kubeconfig: /path/to/kubeconfig
```

## Informer Cache

KubeVision uses Kubernetes Informers to cache frequently accessed resources for sub-millisecond reads:

**Cached resources (8):** Pods, Deployments, StatefulSets, DaemonSets, Services, Ingresses, Nodes, Namespaces

**On-demand resources (18+):** Jobs, CronJobs, ConfigMaps, PVs, PVCs, etc.

**Never cached:** Secrets (security), Events (volume)

The `informer_resync` setting controls how often the cache is fully re-synced with the API Server.
