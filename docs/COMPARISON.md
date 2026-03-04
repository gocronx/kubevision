# Kubernetes Dashboard 横向对比

> 对比对象：GitHub 上主流 K8s 可视化管理工具，数据截至 2026-03。

---

## 项目概览

| | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner | Kite | KubePolaris |
|---|---|---|---|---|---|---|---|---|
| Stars | — | ~5.9k | ~33k | ~23.5k | ~15.5k (已归档) | ~1.4k | — | — |
| 类型 | Web | Web + Desktop | 终端 | Web | Web | Web | Web | Web |
| 技术栈 | Go + React | Go + React | Go | Vue + JS | Go + TS | React + Node | React + Go | React + Go |
| 许可证 | Apache 2.0 | Apache 2.0 | Apache 2.0 | 部分开源 | Apache 2.0 | Apache 2.0 | Apache 2.0 | Apache 2.0 |
| 状态 | 活跃 | 活跃 (SIG UI) | 活跃 | 活跃 | **已归档** | CNCF Sandbox | 活跃 | 活跃 |

---

## 核心功能对比

### 集群管理

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| 多集群 | **Yes** | Yes | Yes | Yes | No | No |
| Pod 终端 | **Yes + 录制回放** | Yes | Yes | Yes | Yes | Yes |
| Pod 日志 | **Yes** | Yes | Yes | Yes | Yes | Yes |
| CRD 支持 | **自动发现** | Yes | Yes | Yes | Yes | 部分 |
| 批量操作 | **Yes** | No | 部分 | 部分 | No | No |
| Helm 管理 | 🗓️ 规划中 | Yes | No | No | No | No |

### 安全与合规

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| RBAC | **5 级角色** | K8s RBAC | K8s RBAC | 内置用户 | K8s RBAC | Token/OIDC |
| 2FA (TOTP) | **Yes** | No | — | No | No | No |
| 审计日志 | **Yes** | No | No | Yes | No | No |
| 终端录制 | **asciinema 回放** | No | No | No | No | No |
| Secrets 脱敏 | **默认脱敏** | No | No | No | No | No |
| Token 吊销 | **即时生效** | No | — | No | No | No |

### 实时性与搜索

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| 实时推送 | **Informer → WS** | Watch API | Watch | Watch | Watch | WebSocket |
| 全局搜索 | **Cmd+K** | Yes | `/` 模糊搜索 | No | 部分 | No |
| 收藏夹 | **Yes** | No | Yes | No | No | No |

### 高级 DevOps

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| Dry-Run Diff | **全资源** | No | No | No | No | No |
| 跨集群 Diff | **Yes** | No | No | No | No | No |
| 资源拓扑图 | **Yes** | Resource Map | XRay | 微服务分层 | No | No |
| kubectl 提示 | **Yes** | No | — | No | No | No |
| 资源模版 | **Yes** | No | No | Yes | 部分 | No |
| 资源配额可视化 | **进度条 + 告警** | 部分 | Yes | Yes | 部分 | 部分 |

### 集成与扩展

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| Webhook 通知 | **Slack/钉钉/飞书** | No | No | 邮件/微信 | No | No |
| 插件系统 | **Yes** | Yes (丰富) | Yes (CLI) | 部分 | No | No |
| Prometheus | **插件化** | 插件化 | Yes | Yes | 部分 | 部分 |
| GitOps | 🗓️ 规划中 | Flux 插件 | 部分 Flux | No | No | No |
| 成本分析 | 🗓️ 规划中 | OpenCost | No | No | No | No |
| 网络策略可视化 | 🗓️ 规划中 | No | No | No | No | No |

### 前端体验

| 功能 | **KubeVision** | Headlamp | K9s | Kuboard | K8s Dashboard | Skooner |
|------|---|---|---|---|---|---|
| 暗色模式 | **Yes** | Yes | Yes | No | Yes | No |
| i18n | **中/英** | 10 语言 | No | 中/英 | 6 语言 | No |

---

## KubeVision 独有功能

这些功能在所有对比项目中**均未实现**：

| 功能 | 说明 |
|------|------|
| **2FA (TOTP)** | 二次验证 + 恢复码，所有竞品均不支持 |
| **Dry-Run Diff** | 全资源变更预览，API Server 验证 |
| **跨集群 Diff** | 同资源跨环境 YAML 并排对比 |
| **终端会话录制** | asciinema 格式完整录制回放 |
| **kubectl 命令生成** | 每个 UI 操作显示等效命令 |
| **Secrets 默认脱敏** | 其他项目默认暴露明文 |

---

## 开发路线图

以下功能来自竞品分析，计划逐步实现以打造最全面的 K8s Dashboard：

### Phase 1 — 短期 (v0.2)

| 功能 | 参考 | 优先级 | 说明 |
|------|------|--------|------|
| **Helm Chart 管理** | Headlamp | 高 | 安装/升级/回滚/卸载 Release，浏览 Chart 仓库 |
| **YAML Schema 验证** | Skooner, Headlamp | 高 | 编辑器内联校验 + 自动补全 |
| **Pod 文件浏览器** | Kuboard | 中 | 容器内文件查看/上传/下载 |

### Phase 2 — 中期 (v0.3)

| 功能 | 参考 | 优先级 | 说明 |
|------|------|--------|------|
| **GitOps 集成** | Headlamp (Flux) | 高 | ArgoCD / Flux 状态可视化与同步操作 |
| **网络策略可视化** | 无竞品实现 | 高 | Pod 间流量关系图 + 策略影响分析（差异化功能） |
| **成本分析** | Headlamp (OpenCost) | 中 | Namespace/工作负载维度成本统计与优化建议 |

### Phase 3 — 长期 (v0.4+)

| 功能 | 参考 | 优先级 | 说明 |
|------|------|--------|------|
| **AI 助手** | Headlamp | 中 | 自然语言查询集群状态、故障诊断建议 |
| **多语言扩展** | Headlamp (10 语言), K8s Dashboard (6 语言) | 低 | 日语/韩语/法语等社区贡献 |

---

## 总结

| 维度 | KubeVision 定位 |
|------|----------------|
| vs Headlamp | 更强的安全体系 (2FA/审计/脱敏)，Dry-Run Diff 独有；Headlamp 插件生态更成熟 |
| vs K9s | Web 端团队协作，终端录制/审计/RBAC；K9s 个人效率更高 |
| vs Kuboard | 完全开源，架构更现代 (Informer→WS)；Kuboard 企业版功能更全 |
| vs K8s Dashboard | 已归档，官方推荐迁移到 Headlamp |
| vs Skooner | 功能全面性碾压；Skooner 胜在极简轻量 |
| vs Kite | 同为 React + Go，KubeVision 安全/实时/批量操作全面领先；Kite 更轻量 |
| vs KubePolaris | 同技术栈，KubeVision 多集群/2FA/审计/拓扑图均为差异化优势 |
