---
title: FAQ
---

# FAQ

Frequently asked questions about KubeVision.

---

### What Kubernetes versions are supported?

KubeVision requires **Kubernetes 1.24 or later**. The minimum version is set by the use of server-side apply and the stable `apps/v1` API group. Most features work on any 1.24+ cluster; some features (e.g., dry-run diff via `--dry-run=server`) require a live API Server connection and may behave differently on very old patch releases.

---

### Can I use KubeVision with managed Kubernetes services?

Yes. KubeVision works with any cluster reachable via a standard kubeconfig or API token:

| Provider | Notes |
|----------|-------|
| Amazon EKS | Use `aws eks update-kubeconfig` to generate credentials |
| Google GKE | Use `gcloud container clusters get-credentials` |
| Azure AKS | Use `az aks get-credentials` |
| DigitalOcean DOKS | Use `doctl kubernetes cluster kubeconfig save` |
| On-prem / bare metal | Any cluster with a valid kubeconfig |

---

### How many clusters can KubeVision manage at once?

There is no hard limit enforced in the code. KubeVision has been tested with **10+ clusters** registered simultaneously. Each cluster gets its own Informer cache and WebSocket feed. Beyond 10–15 clusters, memory usage increases proportionally — plan for approximately 50–150 MB of extra memory per active cluster depending on resource count.

---

### Is KubeVision production-ready?

Yes, when deployed with **PostgreSQL** as the database backend. The SQLite default is intended for local development and single-user evaluation only.

| Deployment mode | Suitable for |
|-----------------|-------------|
| SQLite | Local dev, demos, single user |
| PostgreSQL | Production, multi-user, HA setups |

For production deployments, also configure: a strong `jwtSecret`, TLS termination (ingress or load balancer), external PostgreSQL with backups, and 2FA enforcement for admin accounts.

---

### How does KubeVision compare to kubectl?

KubeVision is **complementary to kubectl**, not a replacement. The two tools serve different use cases:

| kubectl | KubeVision |
|---------|-----------|
| Terminal / scripting / CI | Browser / team / visual |
| Full API surface coverage | Curated resource views |
| No access control layer | 5-level RBAC + 2FA |
| No audit log | Full audit log |

KubeVision's **kubectl Hints** feature shows the equivalent `kubectl` command for every action taken in the UI, so you can use both tools fluidly.

---

### Does KubeVision modify my cluster resources automatically?

No. KubeVision never modifies cluster state without an explicit user action (clicking Apply, Delete, Scale, etc.). The backend does not run any background jobs that write to the cluster. Read-only operations (listing, watching, log streaming) generate no writes.

:::note
The one exception is the Informer cache, which opens a long-lived watch connection to the API Server. This is read-only and standard practice for Kubernetes controllers and dashboards alike.
:::

---

### What about Kubernetes RBAC? Does KubeVision respect it?

KubeVision has two separate access control layers:

1. **KubeVision RBAC** — Controls who can log in and what they can see within the dashboard (users, roles, cluster/namespace assignments).
2. **Kubernetes RBAC** — The `ServiceAccount` or credentials used by the KubeVision backend to communicate with the cluster must have appropriate Kubernetes RBAC permissions.

KubeVision ships with a sample `ClusterRole` manifest in `deploy/rbac.yaml` that grants the minimum permissions needed for full dashboard functionality. You can scope it down further for read-only deployments.

---

### How do I back up KubeVision?

Two things need to be backed up:

```
1. The KubeVision database (user accounts, cluster registrations, audit log)
2. The kubeconfig files stored in the database (or on disk if using file-based import)
```

For PostgreSQL:

```bash
# Dump the database
pg_dump -U kubevision kubevision_db > kubevision_backup_$(date +%Y%m%d).sql
```

For SQLite (dev):

```bash
sqlite3 kubevision.db ".backup kubevision_backup_$(date +%Y%m%d).db"
```

Cluster kubeconfigs are stored encrypted in the database. A database backup is sufficient to restore all cluster connections.

---

### Can I use SSO or OIDC for login?

OIDC / SSO support via **Dex** is planned for a future release (see [Roadmap](/docs/roadmap)). As of the current version, authentication is handled by KubeVision's own JWT-based login with optional 2FA (TOTP).

If OIDC is a hard requirement today, Headlamp is the recommended alternative — it has mature OIDC support via its plugin system.

---

### Is there a hosted SaaS version of KubeVision?

No. KubeVision is **self-hosted only**. There is no cloud-hosted version, no phone-home telemetry, and no license keys. The project is fully open source under Apache 2.0.

---

### How does KubeVision perform with large clusters?

KubeVision uses Kubernetes **Informer caches** to serve list and get requests. After the initial sync, all reads are served from in-process memory — no API Server calls are made for routine listing operations.

| Cluster size | Behavior |
|-------------|---------|
| < 500 resources per type | Full informer sync at startup, instant queries |
| 500 – 5,000 resources | Informer cache holds all resources; startup sync takes a few seconds |
| > 5,000 resources | Consider namespace-scoped informers (configurable) to limit memory |

In benchmarks on a cluster with 3,000 pods across 50 namespaces, list-pods API calls return in under 5 ms from cache.

---

## Still Have Questions?

- Open a [GitHub Discussion](https://github.com/gocronx/kubevision/discussions) for general questions.
- File a [GitHub Issue](https://github.com/gocronx/kubevision/issues) for confirmed bugs.
- See [Contributing](/docs/development/contributing) if you want to help improve KubeVision.
