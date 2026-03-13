---
title: 路线图
---

# 路线图

KubeVision 遵循分阶段发布计划。每个阶段都在前一阶段的基础上构建——首先是核心稳定性，其次是高级集成，最后是智能自动化。

:::note
本路线图反映当前规划，可能会根据社区反馈和贡献者的可用性而调整。请在 [GitHub Discussions](https://github.com/gocronx/kubevision/discussions) 中跟踪进展并为功能投票。
:::

---

## 第一阶段 — v0.2（近期）

重点：**开发者体验打磨与常用日常功能。**

### Helm Chart 管理

直接从控制台管理 Helm Release，无需切换到终端进行常规的 Chart 操作。

| 功能 | 详情 |
|-----------|--------|
| 列出 Release | 跨命名空间的所有 Release，包含 Chart 名称和版本 |
| 升级 Release | 上传或选择新的 Chart 版本，在线编辑 values |
| 回滚 Release | 回滚到任意历史版本 |
| 卸载 Release | 可选使用 `--purge` 清理 CRD |
| Release 历史 | 带版本时间线，支持 values 间的差异对比 |

### YAML Schema 验证

内置的 YAML 编辑器将在提交前根据官方 Kubernetes JSON Schema 验证资源 manifest，为未知字段、缺少的必填字段和类型不匹配提供内联红色下划线提示——无需 API Server 往返。

```yaml
# 示例：应用前捕获拼写错误
spec:
  replicas: "3"   # Error: must be integer, not string
  tempalte:       # Error: unknown field (did you mean 'template'?)
```

### Pod 文件浏览器

无需打开完整的终端会话，即可浏览运行中容器的文件系统。文件浏览器提供只读访问，并支持下载单个文件——便于检查配置文件、写入磁盘的日志或生成的构件。

```
/etc/nginx/
  ├── nginx.conf        [download]
  ├── conf.d/
  │   └── default.conf  [download]
/var/log/nginx/
  ├── access.log        [download]
  └── error.log         [download]
```

---

## 第二阶段 — v0.3（中期）

重点：**面向平台工程团队的 GitOps 与可观测性集成。**

### GitOps 集成（ArgoCD / Flux）

将 ArgoCD 和 Flux 的同步状态直接展示在资源卡片上——无需切换到独立的 GitOps 界面。

| 集成 | 功能 |
|------------|---------|
| ArgoCD | 应用同步状态、健康状态、与 Git 的差异、手动触发同步 |
| Flux | Kustomization 和 HelmRelease 状态，触发协调 |
| 两者 | Deployment 和 Service 上的未同步徽标，链接到源 Git commit |

```yaml
# config.yaml
plugins:
  argocd:
    enabled: true
    endpoint: "http://argocd-server.argocd.svc.cluster.local"
    token: "<argocd-api-token>"
```

### 网络策略可视化

渲染一个交互式图表，展示基于当前 `NetworkPolicy` 资源，哪些 Pod 可以与哪些其他 Pod 通信。快速识别过于宽松的策略或意外的隔离情况。

- 带命名空间泳道的节点连接图
- 点击 Pod 高亮显示所有允许的入站/出站路径
- 差异视图：对比变更前后的策略图（底层使用预演差异）

### 成本分析（OpenCost）

与 [OpenCost](https://www.opencost.io/) 集成，在资源利用率指标旁展示按命名空间和按工作负载的成本估算。

| 视图 | 新增成本数据 |
|------|----------------|
| 命名空间列表 | 每个命名空间的月度成本估算 |
| Deployment 详情 | 每副本成本、预计月度成本 |
| 节点详情 | 每小时节点成本与利用率效率对比 |
| 全局概览 | 成本最高的前 10 个工作负载 |

:::tip
OpenCost 需要单独安装到集群中。KubeVision 从其 API 读取数据，不会将成本数据复制到自己的数据库中。
:::

---

## 第三阶段 — v0.4+（长期）

重点：**智能自动化与更广泛的可访问性。**

### AI 助手

在控制台中内嵌一个自然语言助手，帮助运维人员在不离开界面的情况下诊断和解决问题。

计划中的功能：

- **解释资源** — "这个 CrashLoopBackOff 是什么意思，我该如何修复？"
- **生成 YAML** — "为一个带 /healthz 就绪探针的 3 副本 nginx 编写 Deployment"
- **分析日志** — 对最近 500 行日志进行摘要并高亮异常
- **建议操作** — 针对失败的 Pod，推荐最可能的修复步骤

助手通过可配置的 LLM 端点运行查询（自托管的 Ollama 或兼容 OpenAI 的 API）。除非你配置了外部端点，否则集群数据不会发送到任何外部服务。

### 多语言扩展

UI 目前仅提供英文版。计划新增的语言区域：

| 语言区域 | 状态 |
|--------|--------|
| 英文（`en`） | 已上线 |
| 简体中文（`zh-CN`） | 计划中 |
| 日文（`ja`） | 计划中 |
| 西班牙文（`es`） | 计划中 |

翻译通过 `i18next` 管理，社区贡献的 JSON 文件存放于 `web/src/locales/` 目录下，欢迎贡献。

### OAuth / OIDC（通过 Dex）

使用 [Dex](https://dexidp.io/) 作为 OIDC 代理，将当前仅限本地的登录替换为完整的联合身份认证流程。

| 身份提供商 | 通过 Dex |
|---------|---------|
| GitHub | 是 |
| Google | 是 |
| Microsoft Azure AD | 是 |
| LDAP / Active Directory | 是 |
| SAML 2.0 | 是（Dex connector） |

对于无法联合身份的环境，现有的本地账号和双因素认证将继续与 OIDC 并存支持。

---

## 参与路线图建设

有未列出的功能想法？影响路线图的最佳方式是：

1. 在 [GitHub Discussions](https://github.com/gocronx/kubevision/discussions) 中提出，描述使用场景。
2. 如果有社区兴趣，维护者将创建一个跟踪 Issue 并将其加入路线图。
3. 如果你想自行实现，请参阅[贡献指南](/docs/development/contributing)了解如何开始。

## 相关文档

- [简介](/docs/intro) — KubeVision 的现有功能
- [产品对比](/docs/comparison) — KubeVision 与竞品的比较
- [贡献指南](/docs/development/contributing) — 参与构建这些功能
