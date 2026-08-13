---
title: 常见问题
---

# 常见问题

KubeVision 的常见问题解答。

---

### KubeVision 支持哪些 Kubernetes 版本？

KubeVision 需要 **Kubernetes 1.24 或更高版本**。最低版本要求由服务端应用（Server-Side Apply）和稳定版 `apps/v1` API 组的使用所决定。大多数功能可在任何 1.24+ 集群上正常工作；部分功能（例如通过 `--dry-run=server` 进行预演差异）需要实时连接 API Server，在非常旧的补丁版本上行为可能有所不同。

---

### KubeVision 可以与托管 Kubernetes 服务配合使用吗？

可以。KubeVision 可以与任何通过标准 kubeconfig 或 API 令牌可达的集群配合使用：

| 服务商 | 说明 |
|----------|-------|
| Amazon EKS | 使用 `aws eks update-kubeconfig` 生成凭证 |
| Google GKE | 使用 `gcloud container clusters get-credentials` |
| Azure AKS | 使用 `az aks get-credentials` |
| DigitalOcean DOKS | 使用 `doctl kubernetes cluster kubeconfig save` |
| 自建 / 裸金属 | 任何具有有效 kubeconfig 的集群 |

---

### KubeVision 最多可以同时管理多少个集群？

代码中没有强制的硬性上限。KubeVision 已在同时注册 **10 个以上集群**的环境中经过测试。每个集群拥有独立的 Informer 缓存和 WebSocket 订阅。超过 10-15 个集群后，内存占用会按比例增加——根据资源数量的不同，每个活跃集群大约需要额外 50-150 MB 的内存。

---

### KubeVision 是否已准备好用于生产环境？

是的，当使用 **PostgreSQL** 作为数据库后端时可用于生产。默认的 SQLite 仅适用于本地开发和单用户评估。

| 部署模式 | 适用场景 |
|-----------------|-------------|
| SQLite | 本地开发、演示、单用户 |
| PostgreSQL | 生产、多用户、高可用部署 |

在生产部署中，还需配置：强随机的 `jwtSecret`、TLS 终止（Ingress 或负载均衡器）、带备份的外部 PostgreSQL，以及对管理员账号强制启用双因素认证。

---

### KubeVision 会自动修改我的集群资源吗？

不会。在没有明确用户操作（点击应用、删除、扩缩容等）的情况下，KubeVision 绝不会修改集群状态。后端没有任何会向集群写入数据的后台任务。只读操作（列出、监听、日志流）不会产生任何写入。

:::note
唯一的例外是 Informer 缓存，它会向 API Server 建立一个长连接 watch 请求。这是只读的，也是 Kubernetes 控制器和控制台的标准做法。
:::

---

### Kubernetes RBAC 怎么办？KubeVision 会遵循它吗？

KubeVision 有两个独立的访问控制层：

1. **KubeVision RBAC** — 控制谁可以登录，以及在控制台内可以看到什么（用户、角色、集群/命名空间分配）。
2. **Kubernetes RBAC** — KubeVision 后端用于与集群通信的 `ServiceAccount` 或凭证，必须拥有相应的 Kubernetes RBAC 权限。

KubeVision 在 `deploy/rbac.yaml` 中提供了一份示例 `ClusterRole` manifest，授予完整控制台功能所需的最小权限。对于只读部署，你可以进一步缩小其权限范围。

---

### 如何备份 KubeVision？

需要备份以下两项内容：

```
1. KubeVision 数据库（用户账号、集群注册信息、审计日志）
2. 存储在数据库中的 kubeconfig 文件（如果使用基于文件的导入，则保存在磁盘上）
```

对于 PostgreSQL：

```bash
# 转储数据库
pg_dump -U kubevision kubevision_db > kubevision_backup_$(date +%Y%m%d).sql
```

对于 SQLite（开发环境）：

```bash
sqlite3 kubevision.db ".backup kubevision_backup_$(date +%Y%m%d).db"
```

集群 kubeconfig 以加密形式存储在数据库中。一份数据库备份足以恢复所有集群连接。

---

### 我可以使用 SSO 或 OIDC 登录吗？

可以。管理员可以在本地 JWT 登录、TOTP、目录账户和通行密钥之外配置标准
OAuth 或 OIDC 提供商。OIDC 通过提供商的 Issuer 进行发现；标准 OAuth 集成也
可以显式配置授权、令牌和用户信息端点。详见
[认证提供商](/docs/admin-guide/authentication-providers)。

---

### KubeVision 有托管的 SaaS 版本吗？

没有。KubeVision 是**纯自托管**的。没有云托管版本，没有数据上报遥测，也没有许可证密钥。该项目完全以 MIT 协议开源。

---

### KubeVision 在大型集群上的性能如何？

KubeVision 使用 Kubernetes **Informer 缓存**来响应列表和获取请求。初始同步完成后，所有读取请求均从进程内存中提供服务——常规的列出操作不会产生 API Server 调用。

| 集群规模 | 行为 |
|-------------|---------|
| 每种类型资源数 < 500 | 启动时完成 Informer 同步，查询即时响应 |
| 500 – 5,000 个资源 | Informer 缓存持有全部资源；启动同步耗时数秒 |
| > 5,000 个资源 | 考虑使用命名空间范围的 Informer（可配置）以限制内存占用 |

在一个跨 50 个命名空间拥有 3,000 个 Pod 的集群上进行的基准测试中，从缓存响应的 list-pods API 调用耗时不超过 5 毫秒。

---

## 还有疑问？

- 在 [GitHub Discussions](https://github.com/gocronx/kubevision/discussions) 提出一般性问题。
- 在 [GitHub Issues](https://github.com/gocronx/kubevision/issues) 提交已确认的 bug。
- 如果你想参与贡献，请参阅[贡献指南](/docs/development/contributing)。
